package config

import (
	"os"
	"strconv"
	"time"
)

// Config 定义服务配置。
type Config struct {
	HTTPAddr          string
	TimeWindow        time.Duration
	AlertWindow       time.Duration
	DuplicateWindow   time.Duration
	MinScore          float64
	WarningThreshold  float64
	CriticalThreshold float64
	UseRedis          bool
	UsePostgres       bool
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	PostgresURL       string
}

// Load 从环境变量加载配置，提供默认值。
func Load() Config {
	return Config{
		HTTPAddr:          getEnvString("CROSS_VALIDATION_HTTP_ADDR", ":8080"),
		TimeWindow:        getEnvDuration("CROSS_VALIDATION_TIME_WINDOW", 10*time.Second),
		AlertWindow:       getEnvDuration("CROSS_VALIDATION_ALERT_WINDOW", 60*time.Second),
		DuplicateWindow:   getEnvDuration("CROSS_VALIDATION_DUPLICATE_WINDOW", 3*time.Second),
		MinScore:          getEnvFloat64("CROSS_VALIDATION_MIN_SCORE", 0.35),
		WarningThreshold:  getEnvFloat64("CROSS_VALIDATION_WARNING_THRESHOLD", 0.6),
		CriticalThreshold: getEnvFloat64("CROSS_VALIDATION_CRITICAL_THRESHOLD", 0.8),
		UseRedis:          getEnvBool("CROSS_VALIDATION_USE_REDIS", false),
		UsePostgres:       getEnvBool("CROSS_VALIDATION_USE_POSTGRES", false),
		RedisAddr:         getEnvString("CROSS_VALIDATION_REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnvString("CROSS_VALIDATION_REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("CROSS_VALIDATION_REDIS_DB", 0),
		PostgresURL:       getEnvString("CROSS_VALIDATION_POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/cross_validation?sslmode=disable"),
	}
}

func getEnvString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if raw := os.Getenv(key); raw != "" {
		if value, err := time.ParseDuration(raw); err == nil {
			return value
		}
	}
	return fallback
}

func getEnvFloat64(key string, fallback float64) float64 {
	if raw := os.Getenv(key); raw != "" {
		if value, err := strconv.ParseFloat(raw, 64); err == nil {
			return value
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if raw := os.Getenv(key); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			return value
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if raw := os.Getenv(key); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			return value
		}
	}
	return fallback
}
