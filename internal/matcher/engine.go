package matcher

import (
	"fmt"
	"math"
	"time"

	"crossValidationDemo/internal/config"
	"crossValidationDemo/internal/domain"
)

// Engine 使用加权评分模型来判定 RFID 与 CCTV 的一致性。
type Engine struct {
	cfg config.Config
}

func NewEngine(cfg config.Config) *Engine {
	return &Engine{cfg: cfg}
}

// Evaluate 对一个 RFID 事件，寻找最近时间窗口内的 CCTV 事件并给出评分结论。
func (e *Engine) Evaluate(rfid domain.RFIDEvent, cctvEvents []domain.CCTVEvent) domain.MatchResult {
	if len(cctvEvents) == 0 {
		return domain.MatchResult{
			RFID:           &rfid,
			Score:          0,
			Status:         domain.MatchStatusMissing,
			Severity:       domain.AlertSeverityEmergency,
			Reason:         "no CCTV evidence found within configured time window",
			GeneratedAt:    time.Now(),
			AlertTriggered: true,
		}
	}

	best := domain.MatchResult{
		RFID:           &rfid,
		Score:          0,
		Status:         domain.MatchStatusMismatch,
		GeneratedAt:    time.Now(),
		AlertTriggered: true,
	}

	for _, cctv := range cctvEvents {
		score := e.score(rfid, cctv)
		result := domain.MatchResult{
			RFID:           &rfid,
			CCTV:           &cctv,
			Score:          score,
			Status:         classifyScore(score, e.cfg.WarningThreshold, e.cfg.MinScore),
			Severity:       severityByScore(score),
			GeneratedAt:    time.Now(),
			AlertTriggered: shouldTriggerAlert(score, classifyScore(score, e.cfg.WarningThreshold, e.cfg.MinScore)),
		}
		result.Reason = buildReason(rfid, cctv, score)

		if score > best.Score {
			best = result
		}
	}

	if best.Score == 0 {
		best.Reason = "RFID event has no valid CCTV correlation in the current window"
		best.Severity = domain.AlertSeverityEmergency
		best.AlertTriggered = true
		best.Status = domain.MatchStatusMissing
	}

	return best
}

func (e *Engine) score(rfid domain.RFIDEvent, cctv domain.CCTVEvent) float64 {
	timeScore := timeScore(rfid.Timestamp, cctv.Timestamp, e.cfg.TimeWindow)
	zoneScore := zoneScore(rfid.ZoneID, cctv.ZoneID)
	directionScore := directionScore(rfid.EventType, cctv.Direction)
	identityScore := identityScore(rfid.Confidence, cctv.Confidence)
	continuityScore := continuityScore(rfid.EventType, cctv.Direction)

	total := 0.30*timeScore +
		0.25*zoneScore +
		0.20*directionScore +
		0.15*identityScore +
		0.10*continuityScore

	return clamp(total, 0, 1)
}

func timeScore(rfidTime, cctvTime time.Time, window time.Duration) float64 {
	delta := absDuration(rfidTime.Sub(cctvTime))
	if delta <= 0 {
		return 1
	}
	if delta >= window {
		return 0
	}
	return 1 - (delta.Seconds() / window.Seconds())
}

func zoneScore(rfidZone, cctvZone string) float64 {
	if rfidZone == "" || cctvZone == "" {
		return 0.4
	}
	if rfidZone == cctvZone {
		return 1
	}
	return 0.25
}

func directionScore(rfidType domain.EventType, cctvDirection string) float64 {
	normalizedDirection := normalizeDirection(cctvDirection)
	switch rfidType {
	case domain.EventTypeEnter:
		if normalizedDirection == "in" {
			return 1
		}
	case domain.EventTypeExit:
		if normalizedDirection == "out" {
			return 1
		}
	}
	if normalizedDirection == "unknown" {
		return 0.5
	}
	return 0
}

func identityScore(rfidConfidence, cctvConfidence float64) float64 {
	if rfidConfidence <= 0 || cctvConfidence <= 0 {
		return 0.5
	}
	return clamp((rfidConfidence+cctvConfidence)/2, 0, 1)
}

func continuityScore(eventType domain.EventType, cctvDirection string) float64 {
	normalizedDirection := normalizeDirection(cctvDirection)
	switch eventType {
	case domain.EventTypeEnter:
		if normalizedDirection == "in" || normalizedDirection == "unknown" {
			return 0.8
		}
	case domain.EventTypeExit:
		if normalizedDirection == "out" || normalizedDirection == "unknown" {
			return 0.8
		}
	}
	return 0.5
}

func classifyScore(score, warningThreshold, minScore float64) domain.MatchStatus {
	if score >= warningThreshold {
		return domain.MatchStatusMatched
	}
	if score >= minScore {
		return domain.MatchStatusSuspicious
	}
	return domain.MatchStatusMismatch
}

func severityByScore(score float64) domain.AlertSeverity {
	switch {
	case score >= 0.8:
		return domain.AlertSeverityInfo
	case score >= 0.6:
		return domain.AlertSeverityWarning
	case score >= 0.35:
		return domain.AlertSeverityCritical
	default:
		return domain.AlertSeverityEmergency
	}
}

func shouldTriggerAlert(score float64, status domain.MatchStatus) bool {
	if status == domain.MatchStatusMismatch || status == domain.MatchStatusSuspicious {
		return score < 0.8
	}
	return false
}

func buildReason(rfid domain.RFIDEvent, cctv domain.CCTVEvent, score float64) string {
	if score >= 0.8 {
		return fmt.Sprintf("RFID tag %s matched CCTV track %s with score %.2f", rfid.TagID, cctv.TrackID, score)
	}
	if score >= 0.6 {
		return fmt.Sprintf("RFID tag %s showed weak correlation with CCTV track %s (score %.2f)", rfid.TagID, cctv.TrackID, score)
	}
	return fmt.Sprintf("RFID tag %s deviated from CCTV track %s (score %.2f)", rfid.TagID, cctv.TrackID, score)
}

func normalizeDirection(direction string) string {
	switch direction {
	case "in", "enter", "IN", "Enter":
		return "in"
	case "out", "exit", "OUT", "Exit":
		return "out"
	default:
		return "unknown"
	}
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func isWithinWindow(ts, now time.Time, window time.Duration) bool {
	return now.Sub(ts) <= window && ts.Sub(now) <= window
}

func (e *Engine) MaxWindow() time.Duration {
	return e.cfg.TimeWindow
}

func (e *Engine) MatchThresholds() (warning float64, min float64) {
	return e.cfg.WarningThreshold, e.cfg.MinScore
}

func (e *Engine) TimeWindow() time.Duration {
	return e.cfg.TimeWindow
}

func (e *Engine) IsMatchScore(score float64) bool {
	return score >= e.cfg.WarningThreshold
}

func (e *Engine) IsCritical(score float64) bool {
	return score < e.cfg.MinScore
}

func (e *Engine) IsCandidate(score float64) bool {
	return score >= e.cfg.MinScore && score < e.cfg.WarningThreshold
}

func (e *Engine) EvaluateFromPair(rfid domain.RFIDEvent, cctv domain.CCTVEvent) domain.MatchResult {
	score := e.score(rfid, cctv)
	status := classifyScore(score, e.cfg.WarningThreshold, e.cfg.MinScore)
	result := domain.MatchResult{
		RFID:           &rfid,
		CCTV:           &cctv,
		Score:          score,
		Status:         status,
		Severity:       severityByScore(score),
		Reason:         buildReason(rfid, cctv, score),
		GeneratedAt:    time.Now(),
		AlertTriggered: shouldTriggerAlert(score, status),
	}
	return result
}

func (e *Engine) EvaluateMulti(rfid domain.RFIDEvent, cctv []domain.CCTVEvent) domain.MatchResult {
	if len(cctv) == 0 {
		return domain.MatchResult{
			RFID:           &rfid,
			Score:          0,
			Status:         domain.MatchStatusMissing,
			Severity:       domain.AlertSeverityEmergency,
			Reason:         "no CCTV evidence found",
			GeneratedAt:    time.Now(),
			AlertTriggered: true,
		}
	}

	bestScore := -1.0
	bestResult := domain.MatchResult{}
	for _, item := range cctv {
		result := e.EvaluateFromPair(rfid, item)
		if result.Score > bestScore {
			bestScore = result.Score
			bestResult = result
		}
	}
	return bestResult
}

var _ = math.Abs
var _ = isWithinWindow
