package service

import (
	"fmt"
	"time"

	"crossValidationDemo/internal/config"
	"crossValidationDemo/internal/domain"
	"crossValidationDemo/internal/matcher"
	"crossValidationDemo/internal/store"
)

// ValidationService 提供交叉验证核心服务。
type ValidationService struct {
	cfg     config.Config
	matcher *matcher.Engine
	store   store.EventStore
}

func NewValidationService(cfg config.Config) *ValidationService {
	return &ValidationService{
		cfg:     cfg,
		matcher: matcher.NewEngine(cfg),
		store:   store.NewCompositeStore(cfg),
	}
}

func (s *ValidationService) HandleRFIDEvent(event domain.RFIDEvent) (domain.MatchResult, error) {
	s.store.AppendRFID(event)
	return s.ValidateRFID(event)
}

func (s *ValidationService) HandleCCTVEvent(event domain.CCTVEvent) {
	s.store.AppendCCTV(event)
}

// ValidateRFID 使用最新的 CCTV 事件窗口进行交叉验证，并在必要时产生告警。
func (s *ValidationService) ValidateRFID(event domain.RFIDEvent) (domain.MatchResult, error) {
	cctvEvents := s.store.RecentCCTV(s.cfg.TimeWindow)
	result := s.matcher.Evaluate(event, cctvEvents)

	if result.ShouldAlert() {
		alert := domain.AlertRecord{
			ID:        fmt.Sprintf("alert-%d", time.Now().UnixNano()),
			TagID:     event.TagID,
			CameraID:  firstCameraID(cctvEvents),
			Severity:  result.Severity,
			Message:   result.Reason,
			CreatedAt: time.Now(),
		}
		s.store.AddAlert(alert)
	}

	return result, nil
}

// ValidateByTag 以 TagID 为入口读取最近的 RFID 事件并比对 CCTV。
func (s *ValidationService) ValidateByTag(tagID string, window time.Duration) (domain.MatchResult, error) {
	recent, found := s.store.LatestRFIDByTag(tagID)
	if !found {
		return domain.MatchResult{}, fmt.Errorf("no RFID record found for tag %s", tagID)
	}

	if window <= 0 {
		window = s.cfg.TimeWindow
	}

	list := s.store.RecentRFIDByTag(tagID, window)
	if len(list) == 0 {
		return domain.MatchResult{}, fmt.Errorf("no RFID event in the given window for tag %s", tagID)
	}

	latest := list[len(list)-1]
	cctvEvents := s.store.RecentCCTV(s.cfg.TimeWindow)
	result := s.matcher.Evaluate(latest, cctvEvents)

	if result.ShouldAlert() {
		s.store.AddAlert(domain.AlertRecord{
			ID:        fmt.Sprintf("alert-%d", time.Now().UnixNano()),
			TagID:     tagID,
			CameraID:  firstCameraID(cctvEvents),
			Severity:  result.Severity,
			Message:   result.Reason,
			CreatedAt: time.Now(),
		})
	}

	_ = recent
	return result, nil
}

func (s *ValidationService) Alerts() []domain.AlertRecord {
	return s.store.ListAlerts()
}

func firstCameraID(events []domain.CCTVEvent) string {
	if len(events) == 0 {
		return ""
	}
	return events[0].CameraID
}
