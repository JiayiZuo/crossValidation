package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"crossValidationDemo/internal/config"
	"crossValidationDemo/internal/domain"
)

// PostgresStore 提供 PostgreSQL 持久化实现，适合事件审计和告警归档。
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(cfg config.Config) (*PostgresStore, error) {
	if cfg.PostgresURL == "" {
		cfg.PostgresURL = "postgres://postgres:postgres@localhost:5432/cross_validation?sslmode=disable"
	}

	db, err := sql.Open("postgres", cfg.PostgresURL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := initPostgresSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &PostgresStore{db: db}, nil
}

func initPostgresSchema(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS rfid_events (
			id SERIAL PRIMARY KEY,
			tag_id TEXT NOT NULL,
			device_id TEXT,
			gate_id TEXT,
			zone_id TEXT,
			antenna_id TEXT,
			event_type TEXT,
			event_time TIMESTAMPTZ NOT NULL,
			signal_strength DOUBLE PRECISION,
			confidence DOUBLE PRECISION
		);`,
		`CREATE TABLE IF NOT EXISTS cctv_events (
			id SERIAL PRIMARY KEY,
			camera_id TEXT,
			channel_id TEXT,
			track_id TEXT,
			target_type TEXT,
			object_id TEXT,
			zone_id TEXT,
			direction TEXT,
			event_time TIMESTAMPTZ NOT NULL,
			confidence DOUBLE PRECISION
		);`,
		`CREATE TABLE IF NOT EXISTS alert_records (
			id TEXT PRIMARY KEY,
			tag_id TEXT,
			camera_id TEXT,
			severity TEXT,
			message TEXT,
			created_at TIMESTAMPTZ NOT NULL
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresStore) AppendRFID(event domain.RFIDEvent) {
	if p == nil || p.db == nil {
		return
	}

	_, err := p.db.Exec(
		`INSERT INTO rfid_events (tag_id, device_id, gate_id, zone_id, antenna_id, event_type, event_time, signal_strength, confidence)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		event.TagID,
		event.DeviceID,
		event.GateID,
		event.ZoneID,
		event.AntennaID,
		event.EventType,
		event.Timestamp,
		event.SignalStrength,
		event.Confidence,
	)
	if err != nil {
		fmt.Println("postgres rfid insert error:", err)
	}
}

func (p *PostgresStore) AppendCCTV(event domain.CCTVEvent) {
	if p == nil || p.db == nil {
		return
	}

	_, err := p.db.Exec(
		`INSERT INTO cctv_events (camera_id, channel_id, track_id, target_type, object_id, zone_id, direction, event_time, confidence)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		event.CameraID,
		event.ChannelID,
		event.TrackID,
		event.TargetType,
		event.ObjectID,
		event.ZoneID,
		event.Direction,
		event.Timestamp,
		event.Confidence,
	)
	if err != nil {
		fmt.Println("postgres cctv insert error:", err)
	}
}

func (p *PostgresStore) RecentCCTV(window time.Duration) []domain.CCTVEvent {
	if p == nil || p.db == nil {
		return nil
	}
	cutoff := time.Now().Add(-window)
	rows, err := p.db.Query(
		`SELECT camera_id, channel_id, track_id, target_type, object_id, zone_id, direction, event_time, confidence
		FROM cctv_events WHERE event_time >= $1 ORDER BY event_time DESC`,
		cutoff,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]domain.CCTVEvent, 0)
	for rows.Next() {
		var evt domain.CCTVEvent
		var ts time.Time
		if err := rows.Scan(&evt.CameraID, &evt.ChannelID, &evt.TrackID, &evt.TargetType, &evt.ObjectID, &evt.ZoneID, &evt.Direction, &ts, &evt.Confidence); err == nil {
			evt.Timestamp = ts
			result = append(result, evt)
		}
	}
	return result
}

func (p *PostgresStore) RecentRFIDByTag(tagID string, window time.Duration) []domain.RFIDEvent {
	if p == nil || p.db == nil {
		return nil
	}
	cutoff := time.Now().Add(-window)
	rows, err := p.db.Query(
		`SELECT tag_id, device_id, gate_id, zone_id, antenna_id, event_type, event_time, signal_strength, confidence
		FROM rfid_events WHERE tag_id = $1 AND event_time >= $2 ORDER BY event_time DESC`,
		tagID, cutoff,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]domain.RFIDEvent, 0)
	for rows.Next() {
		var evt domain.RFIDEvent
		var ts time.Time
		if err := rows.Scan(&evt.TagID, &evt.DeviceID, &evt.GateID, &evt.ZoneID, &evt.AntennaID, &evt.EventType, &ts, &evt.SignalStrength, &evt.Confidence); err == nil {
			evt.Timestamp = ts
			result = append(result, evt)
		}
	}
	return result
}

func (p *PostgresStore) LatestRFIDByTag(tagID string) (domain.RFIDEvent, bool) {
	items := p.RecentRFIDByTag(tagID, 7*24*time.Hour)
	if len(items) == 0 {
		return domain.RFIDEvent{}, false
	}
	return items[0], true
}

func (p *PostgresStore) AddAlert(alert domain.AlertRecord) {
	if p == nil || p.db == nil {
		return
	}
	_, err := p.db.Exec(
		`INSERT INTO alert_records (id, tag_id, camera_id, severity, message, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (id) DO NOTHING`,
		alert.ID,
		alert.TagID,
		alert.CameraID,
		alert.Severity,
		alert.Message,
		alert.CreatedAt,
	)
	if err != nil {
		fmt.Println("postgres alert insert error:", err)
	}
}

func (p *PostgresStore) ListAlerts() []domain.AlertRecord {
	if p == nil || p.db == nil {
		return nil
	}
	rows, err := p.db.Query(`SELECT id, tag_id, camera_id, severity, message, created_at FROM alert_records ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]domain.AlertRecord, 0)
	for rows.Next() {
		var alert domain.AlertRecord
		var ts time.Time
		if err := rows.Scan(&alert.ID, &alert.TagID, &alert.CameraID, &alert.Severity, &alert.Message, &ts); err == nil {
			alert.CreatedAt = ts
			result = append(result, alert)
		}
	}
	return result
}
