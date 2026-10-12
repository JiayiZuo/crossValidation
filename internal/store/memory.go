package store

import (
	"sync"
	"time"

	"crossValidationDemo/internal/domain"
)

// MemoryStore 提供一个内存级的实时事件缓存和告警记录表。
type MemoryStore struct {
	mu         sync.RWMutex
	rfidEvents []domain.RFIDEvent
	cctvEvents []domain.CCTVEvent
	alerts     []domain.AlertRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rfidEvents: make([]domain.RFIDEvent, 0, 256),
		cctvEvents: make([]domain.CCTVEvent, 0, 512),
		alerts:     make([]domain.AlertRecord, 0, 64),
	}
}

func (s *MemoryStore) AppendRFID(event domain.RFIDEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rfidEvents = append(s.rfidEvents, event)
	if len(s.rfidEvents) > 2048 {
		s.rfidEvents = s.rfidEvents[len(s.rfidEvents)-2048:]
	}
}

func (s *MemoryStore) AppendCCTV(event domain.CCTVEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cctvEvents = append(s.cctvEvents, event)
	if len(s.cctvEvents) > 4096 {
		s.cctvEvents = s.cctvEvents[len(s.cctvEvents)-4096:]
	}
}

func (s *MemoryStore) RecentCCTV(window time.Duration) []domain.CCTVEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	result := make([]domain.CCTVEvent, 0, len(s.cctvEvents))
	for _, event := range s.cctvEvents {
		if event.Timestamp.After(cutoff) {
			result = append(result, event)
		}
	}
	return result
}

func (s *MemoryStore) RecentRFIDByTag(tagID string, window time.Duration) []domain.RFIDEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	result := make([]domain.RFIDEvent, 0, len(s.rfidEvents))
	for _, event := range s.rfidEvents {
		if event.TagID == tagID && event.Timestamp.After(cutoff) {
			result = append(result, event)
		}
	}
	return result
}

func (s *MemoryStore) LatestRFIDByTag(tagID string) (domain.RFIDEvent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest domain.RFIDEvent
	found := false
	for _, event := range s.rfidEvents {
		if event.TagID == tagID {
			if !found || event.Timestamp.After(latest.Timestamp) {
				latest = event
				found = true
			}
		}
	}
	return latest, found
}

func (s *MemoryStore) AddAlert(alert domain.AlertRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = append(s.alerts, alert)
	if len(s.alerts) > 256 {
		s.alerts = s.alerts[len(s.alerts)-256:]
	}
}

func (s *MemoryStore) ListAlerts() []domain.AlertRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.AlertRecord, len(s.alerts))
	copy(result, s.alerts)
	return result
}
