# RFID + CCTV 交叉验证功能技术设计文档

## 1. 背景

在智能安防、门禁控制、仓储管理等场景中，单独依赖 RFID 或 CCTV 都存在信息空白和误判问题：

- RFID 能记录门禁、标签识别与进出状态，但无法确认人员真实动作轨迹
- CCTV 能观察现场、目标轨迹和区域动态，但对门禁事件识别不完整

因此，需要建立一套“RFID 与 CCTV 交叉验证机制”，通过时间、区域、方向、身份等维度做一致性核验，并在必要时触发报警。

## 2. 设计目标

### 业务目标

- 识别 RFID 与 CCTV 是否存在对应关联
- 判断数据是否缺失、异常或伪造
- 对关键事件进行分级报警
- 提供日志与审计链路

### 非功能目标

- 可扩展：支持不同门禁设备、不同摄像头
- 可配置：支持不同时间窗口、不同阈值
- 高并发：适用于多设备并发上报
- 可观测：提供健康检查与报警查询接口

## 3. 总体架构

```text
RFID 采集源 ──┐
               ├── 事件归一化 ──> 事件缓存 ──> 匹配引擎 ──> 规则引擎 ──> 报警模块
CCTV 视频源 ──┘
```

## 4. 核心实体

### 4.1 RFID 事件

```go
type RFIDEvent struct {
    TagID       string
    DeviceID    string
    GateID      string
    ZoneID      string
    AntennaID   string
    EventType   string
    Timestamp   time.Time
    SignalStrength float64
    Confidence  float64
}
```

字段说明：

- TagID：RFID 标签唯一标识
- DeviceID：设备编号
- GateID：门禁点编码
- ZoneID：区域编码
- EventType：enter / exit / read
- Timestamp：事件时间
- Confidence：读卡可信度

### 4.2 CCTV 事件

```go
type CCTVEvent struct {
    CameraID    string
    ChannelID   string
    TrackID     string
    TargetType  string
    ObjectID    string
    ZoneID      string
    Direction   string
    Timestamp   time.Time
    Confidence  float64
}
```

### 4.3 匹配结果

```go
type MatchResult struct {
    RFID        *RFIDEvent
    CCTV        *CCTVEvent
    Score       float64
    Status      string
    Severity    string
    Reason      string
    GeneratedAt time.Time
    AlertTriggered bool
}
```

## 5. 关键逻辑设计

### 5.1 时间窗口

所有比对均发生在一个“滑动时间窗口”内，避免系统时钟偏差导致误判。

推荐窗口：

- 默认：10 秒
- 扩展：可配置为 5s / 15s / 30s

时间差越小，匹配得分越高。

### 5.2 匹配评分模型

采用加权评分公式：

$$
S = 0.30T + 0.25Z + 0.20D + 0.15I + 0.10C
$$

其中：

- T：时间接近度
- Z：区域一致性
- D：方向一致性
- I：身份可信度
- C：连续性评分

评分规则：

- 0.8 以上：正常匹配
- 0.6-0.8：待确认
- 0.35-0.6：异常
- 低于 0.35：严重异常

### 5.3 方向一致性

- RFID enter + CCTV in => 正常
- RFID exit + CCTV out => 正常
- RFID enter + CCTV out => 方向异常
- RFID exit + CCTV in => 方向异常

### 5.4 区域一致性

- 若 RFID gate zone 与 CCTV camera zone 相同，区域得分高
- 若跨区域，按距离或区域策略扣分

### 5.5 报警分级

| Level | Meaning | Action |
|---|---|---|
| info | 微小偏差 | 记录 |
| warning | 待确认 | 记录，人工核验 |
| critical | 明显异常 | 触发告警 |
| emergency | 重大风险 | 强制报警 |

## 6. 状态机设计

```text
NEW -> WAITING_MATCH -> MATCHED
                    \-> MISMATCH
                    \-> SUSPICIOUS -> ALERTED -> RESOLVED
```

说明：

- 新事件进入等待匹配状态
- 对相邻 CCTV 事件做关联
- 当分数过低时进入 MISMATCH 或 SUSPICIOUS
- 连续重复异常进入 ALERTED

## 7. 数据处理流程

1. RFID/视频事件被接收到后进入归一化模块
2. 归一化后写入内存缓存，保留近期窗口数据
3. 服务根据 RFID TagID 找到最近 CCTV 事件
4. 调用评分函数计算匹配分数
5. 根据规则引擎输出结论和报警等级
6. 存储日志与警报记录

## 8. 存储设计

### 8.1 热数据存储

- 在内存中保存最近 10-30 分钟事件
- 便于高频实时判定
- Redis 可作为热数据缓存层，用于最近窗口事件快速读取

### 8.2 持久化存储

- PostgreSQL：事件表、报警表、审计结果
- Redis：最近状态缓存
- SQL 初始化脚本可见 [scripts/init-postgres.sql](scripts/init-postgres.sql)

### 8.3 环境变量

项目支持如下配置：

- `CROSS_VALIDATION_USE_REDIS`
- `CROSS_VALIDATION_REDIS_ADDR`
- `CROSS_VALIDATION_REDIS_PASSWORD`
- `CROSS_VALIDATION_REDIS_DB`
- `CROSS_VALIDATION_USE_POSTGRES`
- `CROSS_VALIDATION_POSTGRES_URL`

模板文件： [.env.example](.env.example)

## 9. 接口设计

### API 1：接收 RFID 事件

```http
POST /api/v1/rfid-events
```

### API 2：接收 CCTV 事件

```http
POST /api/v1/cctv-events
```

### API 3：执行交叉验证

```http
POST /api/v1/validate
```

### API 4：查询告警

```http
GET /api/v1/alerts
```

## 10. 关键优点

- 规则清晰，易于扩展
- 评分模型可以适应不同业务场景
- 与低成本内存实现兼容
- 架构可直接演进到消息队列与数据库持久化

## 11. 后续扩展

- Kafka/NATS 消息总线
- Redis 集群缓存
- PostgreSQL 事件审计
- AI 视频识别增强
- 规则引擎参数化配置

## 12. 结论

该设计把监控与门禁两个异构数据源统一为一个判定引擎，提升了异常检测准确率，并且可持续扩展至更复杂的智慧安防场景。

## 13. 可交付材料说明

本文档为可编辑版本，适合作为项目设计说明、评审材料或汇报文档使用。该仓库同时提供了完整的 README 文档与 Postman 导入文件，便于开发、测试和联调。
