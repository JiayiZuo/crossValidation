package store

import (
	"log"
	"time"

	"crossValidationDemo/internal/config"
	"crossValidationDemo/internal/domain"
)

// EventStore 定义事件存储与检索接口。
type EventStore interface {
	AppendRFID(event domain.RFIDEvent)
	AppendCCTV(event domain.CCTVEvent)
	RecentCCTV(window time.Duration) []domain.CCTVEvent
	RecentRFIDByTag(tagID string, window time.Duration) []domain.RFIDEvent
	LatestRFIDByTag(tagID string) (domain.RFIDEvent, bool)
	AddAlert(alert domain.AlertRecord)
	ListAlerts() []domain.AlertRecord
}

// CompositeStore 组合内存、Redis 和 PostgreSQL 层，优先走内存以保证低延迟。
type CompositeStore struct {
	memory   *MemoryStore
	redis    *RedisStore
	postgres *PostgresStore
}

func NewCompositeStore(cfg config.Config) EventStore {
	store := &CompositeStore{memory: NewMemoryStore()}

	if cfg.UseRedis {
		redisStore, err := NewRedisStore(cfg)
		if err != nil {
			log.Printf("redis not available, fallback to memory: %v", err)
		} else {
			store.redis = redisStore
		}
	}

	if cfg.UsePostgres {
		pgStore, err := NewPostgresStore(cfg)
		if err != nil {
			log.Printf("postgres not available, fallback to memory: %v", err)
		} else {
			store.postgres = pgStore
		}
	}

	return store
}

func (s *CompositeStore) AppendRFID(event domain.RFIDEvent) {
	s.memory.AppendRFID(event)
	if s.redis != nil {
		s.redis.AppendRFID(event)
	}
	if s.postgres != nil {
		s.postgres.AppendRFID(event)
	}
}

func (s *CompositeStore) AppendCCTV(event domain.CCTVEvent) {
	s.memory.AppendCCTV(event)
	if s.redis != nil {
		s.redis.AppendCCTV(event)
	}
	if s.postgres != nil {
		s.postgres.AppendCCTV(event)
	}
}

func (s *CompositeStore) RecentCCTV(window time.Duration) []domain.CCTVEvent {
	return s.memory.RecentCCTV(window)
}

func (s *CompositeStore) RecentRFIDByTag(tagID string, window time.Duration) []domain.RFIDEvent {
	return s.memory.RecentRFIDByTag(tagID, window)
}

func (s *CompositeStore) LatestRFIDByTag(tagID string) (domain.RFIDEvent, bool) {
	return s.memory.LatestRFIDByTag(tagID)
}

func (s *CompositeStore) AddAlert(alert domain.AlertRecord) {
	s.memory.AddAlert(alert)
	if s.redis != nil {
		s.redis.AddAlert(alert)
	}
	if s.postgres != nil {
		s.postgres.AddAlert(alert)
	}
}

func (s *CompositeStore) ListAlerts() []domain.AlertRecord {
	return s.memory.ListAlerts()
}
