# Cross Validation Demo

基于 Go 语言实现的 RFID 与 CCTV 交叉验证后端服务，用于在门禁/闸机/视频监控场景下，对 RFID 读卡事件和 CCTV 目标检测事件进行时间、区域、方向和身份的一致性核验，并在必要时触发报警。

## 功能概览

- 接收 RFID 采集事件
- 接收 CCTV 目标检测/跟踪事件
- 在时间窗口内执行交叉比对
- 计算匹配分数并输出结论
- 对异常事件进行分级报警
- 提供 HTTP API 接口进行数据接入与查询
- 使用内存缓存实现近实时校验

## 系统目标

该模块本质上是一个“多源感知一致性判定引擎”，重点解决以下问题：

- RFID 记录了人员进入，但 CCTV 没有对应轨迹
- CCTV 记录了摄像头监测对象，但 RFID 没有对应门禁记录
- RFID 与 CCTV 发生在不同区域、方向不一致或时间偏差过大
- 目标重复检测、重复读卡、绕过门禁等异常场景

## 模块结构

```text
crossValidationDemo/
├── README.md
├── go.mod
├── docs/
│   └── technical-design.md
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── domain/
│   │   └── event.go
│   ├── matcher/
│   │   └── engine.go
│   ├── store/
│   │   └── memory.go
│   ├── service/
│   │   └── validation_service.go
│   └── httpapi/
│       └── handler.go
└── .gitignore
```

## 代码设计

### 1. 事件模型

- RFID 事件：TagID、DeviceID、GateID、ZoneID、EventType、Timestamp、Confidence
- CCTV 事件：CameraID、TrackID、ObjectID、ZoneID、Direction、Timestamp、Confidence

### 2. 核心匹配逻辑

系统采用“加权评分模型”进行交叉验证：

- 时间接近度：30%
- 区域一致性：25%
- 方向一致性：20%
- 目标可信度：15%
- 连续性：10%

判定规则：

- Score >= 0.80：正常匹配
- 0.60 <= Score < 0.80：待确认
- 0.35 <= Score < 0.60：异常/可疑
- Score < 0.35：严重异常

### 3. 报警策略

- info：轻微偏差，通常不需要人工报警
- warning：待确认事件，建议记录
- critical：异常事件，可附加人工复核
- emergency：高风险事件，强制报警

## 运行方式

### 1）编译

```bash
go build ./...
```

### 2）启动服务

```bash
go run ./cmd/server
```

默认监听地址：

```text
:8080
```

### 3）发送 RFID 事件

```bash
curl -X POST http://localhost:8080/api/v1/rfid-events \
  -H "Content-Type: application/json" \
  -d '{
    "tag_id": "RFID-1001",
    "device_id": "gate-a-01",
    "gate_id": "gate-A",
    "zone_id": "zone-1",
    "event_type": "enter",
    "timestamp": "2026-10-12T10:00:00Z",
    "signal_strength": 78.2,
    "confidence": 0.96
  }'
```

### 4）发送 CCTV 事件

```bash
curl -X POST http://localhost:8080/api/v1/cctv-events \
  -H "Content-Type: application/json" \
  -d '{
    "camera_id": "cam-01",
    "track_id": "track-77",
    "target_type": "person",
    "object_id": "obj-77",
    "zone_id": "zone-1",
    "direction": "in",
    "timestamp": "2026-10-12T10:00:02Z",
    "confidence": 0.93
  }'
```

### 5）执行一次交叉验证

```bash
curl -X POST http://localhost:8080/api/v1/validate \
  -H "Content-Type: application/json" \
  -d '{
    "tag_id": "RFID-1001",
    "window_seconds": 10
  }'
```

### 6）查看告警数据

```bash
curl http://localhost:8080/api/v1/alerts
```

## API 说明

### POST /api/v1/rfid-events

新增 RFID 事件并立即进行校验。

### POST /api/v1/cctv-events

新增 CCTV 事件并加入近期窗口数据。

### POST /api/v1/validate

用于直接根据 TagID 执行一次交叉比对。

### GET /api/v1/alerts

获取当前报警记录。

### GET /health

健康检查接口。

## Redis / PostgreSQL 接入

当前版本已加入两种生产级持久化方案：

- Redis：作为热点事件缓存和实时状态存储
- PostgreSQL：作为事件审计、告警归档和长期存储

可直接复制环境变量模板：

```bash
cp .env.example .env
```

然后按需要启用：

```bash
export CROSS_VALIDATION_USE_REDIS=true
export CROSS_VALIDATION_REDIS_ADDR=localhost:6379
export CROSS_VALIDATION_REDIS_PASSWORD=
export CROSS_VALIDATION_REDIS_DB=0

export CROSS_VALIDATION_USE_POSTGRES=true
export CROSS_VALIDATION_POSTGRES_URL='postgres://postgres:postgres@localhost:5432/cross_validation?sslmode=disable'
```

本地快速启动：

```bash
docker compose up -d redis postgres
```

数据库初始化脚本：

- [scripts/init-postgres.sql](scripts/init-postgres.sql)

## Postman 导入

可直接导入以下文件：

- [postman/CrossValidationDemo.postman_collection.json](postman/CrossValidationDemo.postman_collection.json)

导入步骤：

1. 打开 Postman
2. 点击 Import
3. 选择上面的 collection 文件
4. 选择本地服务地址 `http://localhost:8080`
5. 执行接口用例即可

## 生产部署增强版

在原有本地 demo 能力基础上，本项目已补齐生产部署所需的几项关键能力：

- 容器化构建：支持 Docker 镜像打包
- 优雅停机：SIGTERM / Ctrl+C 时安全关闭 HTTP 服务
- 健康检查：`/health` 和 `/ready` 可用于容器探针、Ingress 和负载均衡
- 配置外部化：通过 `.env` / 环境变量管理运行参数
- 多环境部署：同时支持本地开发、Docker Compose、Kubernetes

### 1. Docker 镜像构建

```bash
docker build -t cross-validation:latest .
```

### 2. Docker Compose 生产模式

```bash
docker compose -f docker-compose.prod.yml up --build -d
```

该配置包含：

- App 容器
- Redis 容器
- PostgreSQL 容器
- 自动挂载初始化 SQL

### 3. Kubernetes 部署示例

部署清单位于：

- [deploy/kubernetes/deployment.yaml](deploy/kubernetes/deployment.yaml)
- [deploy/kubernetes/service.yaml](deploy/kubernetes/service.yaml)

可直接应用：

```bash
kubectl apply -f deploy/kubernetes/
```

### 4. Makefile 常用命令

```bash
make build
make test
make run
make docker-up
make docker-down
```

## 扩展方向

- 增加 Kafka/NATS 消息总线集成
- 替换内存存储为 Redis/PostgreSQL
- 增加黑名单/白名单规则
- 对接视频分析平台与门禁平台
- 增加事件回放和审计日志
- 进一步补充 OpenAPI/Swagger 文档和 CI/CD 流程

## 说明

该项目是一个后端算法和架构示例，重点覆盖“交叉验证 + 规则判定 + 报警”，适合作为安全监控、门禁管理、资产追踪等场景的基础模块。
