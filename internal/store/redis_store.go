package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"crossValidationDemo/internal/config"
	"crossValidationDemo/internal/domain"
)

const (
	redisRFIDKeyPrefix = "cross-validation:rfid"
	redisCCTVKeyPrefix = "cross-validation:cctv"
	redisAlertKey      = "cross-validation:alerts"
)

// RedisStore 提供 Redis 事件缓存实现，作为生产方案的热数据层。
type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(cfg config.Config) (*RedisStore, error) {
	if cfg.RedisAddr == "" {
		cfg.RedisAddr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisStore{client: client}, nil
}

func (r *RedisStore) AppendRFID(event domain.RFIDEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	ctx := context.Background()
	key := fmt.Sprintf("%s:%s", redisRFIDKeyPrefix, event.TagID)
	_ = r.client.LPush(ctx, key, string(payload)).Err()
	_ = r.client.LTrim(ctx, key, 0, 2047).Err()
}

func (r *RedisStore) AppendCCTV(event domain.CCTVEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	ctx := context.Background()
	key := fmt.Sprintf("%s:%s", redisCCTVKeyPrefix, event.TrackID)
	_ = r.client.LPush(ctx, key, string(payload)).Err()
	_ = r.client.LTrim(ctx, key, 0, 2047).Err()
}

func (r *RedisStore) RecentCCTV(window time.Duration) []domain.CCTVEvent {
	keys, _, err := r.client.Scan(context.Background(), 0, "cross-validation:cctv:*", 200).Result()
	if err != nil || len(keys) == 0 {
		return nil
	}

	result := make([]domain.CCTVEvent, 0)
	cutoff := time.Now().Add(-window)
	for _, key := range keys {
		items, err := r.client.LRange(context.Background(), key, 0, -1).Result()
		if err != nil {
			continue
		}
		for _, item := range items {
			var evt domain.CCTVEvent
			if err := json.Unmarshal([]byte(item), &evt); err == nil && evt.Timestamp.After(cutoff) {
				result = append(result, evt)
			}
		}
	}
	return result
}

func (r *RedisStore) RecentRFIDByTag(tagID string, window time.Duration) []domain.RFIDEvent {
	key := fmt.Sprintf("%s:%s", redisRFIDKeyPrefix, tagID)
	items, err := r.client.LRange(context.Background(), key, 0, -1).Result()
	if err != nil {
		return nil
	}
	cutoff := time.Now().Add(-window)
	result := make([]domain.RFIDEvent, 0, len(items))
	for _, item := range items {
		var evt domain.RFIDEvent
		if err := json.Unmarshal([]byte(item), &evt); err == nil && evt.Timestamp.After(cutoff) {
			result = append(result, evt)
		}
	}
	return result
}

func (r *RedisStore) LatestRFIDByTag(tagID string) (domain.RFIDEvent, bool) {
	items := r.RecentRFIDByTag(tagID, 24*time.Hour)
	if len(items) == 0 {
		return domain.RFIDEvent{}, false
	}
	return items[0], true
}

func (r *RedisStore) AddAlert(alert domain.AlertRecord) {
	payload, err := json.Marshal(alert)
	if err != nil {
		return
	}
	ctx := context.Background()
	_ = r.client.LPush(ctx, redisAlertKey, string(payload)).Err()
	_ = r.client.LTrim(ctx, redisAlertKey, 0, 255).Err()
}

func (r *RedisStore) ListAlerts() []domain.AlertRecord {
	items, err := r.client.LRange(context.Background(), redisAlertKey, 0, -1).Result()
	if err != nil {
		return nil
	}
	result := make([]domain.AlertRecord, 0, len(items))
	for _, item := range items {
		var alert domain.AlertRecord
		if err := json.Unmarshal([]byte(item), &alert); err == nil {
			result = append(result, alert)
		}
	}
	return result
}
