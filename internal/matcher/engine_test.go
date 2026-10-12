package matcher

import (
	"testing"
	"time"

	"crossValidationDemo/internal/config"
	"crossValidationDemo/internal/domain"
)

func TestEvaluateMatchedCase(t *testing.T) {
	cfg := config.Config{
		TimeWindow:       10 * time.Second,
		WarningThreshold: 0.6,
		MinScore:         0.35,
	}
	engine := NewEngine(cfg)

	now := time.Now().UTC()
	rfid := domain.RFIDEvent{
		TagID:      "RFID-1001",
		ZoneID:     "zone-1",
		EventType:  domain.EventTypeEnter,
		Timestamp:  now,
		Confidence: 0.96,
	}
	cctv := []domain.CCTVEvent{{
		TrackID:    "track-77",
		ZoneID:     "zone-1",
		Direction:  "in",
		Timestamp:  now.Add(2 * time.Second),
		Confidence: 0.93,
	}}

	result := engine.Evaluate(rfid, cctv)
	if result.Score < 0.8 {
		t.Fatalf("expected matched score >= 0.8, got %.2f", result.Score)
	}
	if result.Status != domain.MatchStatusMatched {
		t.Fatalf("expected matched status, got %s", result.Status)
	}
}

func TestEvaluateMissingAlertCase(t *testing.T) {
	cfg := config.Config{
		TimeWindow:       10 * time.Second,
		WarningThreshold: 0.6,
		MinScore:         0.35,
	}
	engine := NewEngine(cfg)

	now := time.Now().UTC()
	rfid := domain.RFIDEvent{
		TagID:      "RFID-2001",
		ZoneID:     "zone-5",
		EventType:  domain.EventTypeExit,
		Timestamp:  now,
		Confidence: 0.9,
	}

	result := engine.Evaluate(rfid, nil)
	if result.Status != domain.MatchStatusMissing {
		t.Fatalf("expected status missing, got %s", result.Status)
	}
	if !result.AlertTriggered {
		t.Fatalf("expected alert triggered for missing CCTV evidence")
	}
}
