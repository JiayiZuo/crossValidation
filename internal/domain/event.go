package domain

import "time"

// EventType 表示 RFID 的门禁事件类型。
type EventType string

const (
	EventTypeEnter EventType = "enter"
	EventTypeExit  EventType = "exit"
	EventTypeRead  EventType = "read"
)

// RFIDEvent 表示门禁 RFID 数据。
type RFIDEvent struct {
	TagID          string    `json:"tag_id"`
	DeviceID       string    `json:"device_id"`
	GateID         string    `json:"gate_id"`
	ZoneID         string    `json:"zone_id"`
	AntennaID      string    `json:"antenna_id,omitempty"`
	EventType      EventType `json:"event_type"`
	Timestamp      time.Time `json:"timestamp"`
	SignalStrength float64   `json:"signal_strength,omitempty"`
	Confidence     float64   `json:"confidence"`
}

// CCTVEvent 表示视频识别结果。
type CCTVEvent struct {
	CameraID   string    `json:"camera_id"`
	ChannelID  string    `json:"channel_id,omitempty"`
	TrackID    string    `json:"track_id"`
	TargetType string    `json:"target_type"`
	ObjectID   string    `json:"object_id"`
	ZoneID     string    `json:"zone_id"`
	Direction  string    `json:"direction"`
	Timestamp  time.Time `json:"timestamp"`
	Confidence float64   `json:"confidence"`
}

// MatchStatus 定义校验状态。
type MatchStatus string

const (
	MatchStatusMatched    MatchStatus = "matched"
	MatchStatusMismatch   MatchStatus = "mismatch"
	MatchStatusSuspicious MatchStatus = "suspicious"
	MatchStatusMissing    MatchStatus = "missing"
)

// AlertSeverity 表示告警严重级别。
type AlertSeverity string

const (
	AlertSeverityInfo      AlertSeverity = "info"
	AlertSeverityWarning   AlertSeverity = "warning"
	AlertSeverityCritical  AlertSeverity = "critical"
	AlertSeverityEmergency AlertSeverity = "emergency"
)

// MatchResult 表示一次交叉验证结果。
type MatchResult struct {
	RFID           *RFIDEvent    `json:"rfid_event,omitempty"`
	CCTV           *CCTVEvent    `json:"cctv_event,omitempty"`
	Score          float64       `json:"score"`
	Status         MatchStatus   `json:"status"`
	Severity       AlertSeverity `json:"severity"`
	Reason         string        `json:"reason"`
	GeneratedAt    time.Time     `json:"generated_at"`
	AlertTriggered bool          `json:"alert_triggered"`
}

// ShouldAlert 返回当前结果是否需要报警。
func (r MatchResult) ShouldAlert() bool {
	return r.AlertTriggered || r.Severity == AlertSeverityCritical || r.Severity == AlertSeverityEmergency
}

// AlertRecord 表示告警信息。
type AlertRecord struct {
	ID        string        `json:"id"`
	TagID     string        `json:"tag_id,omitempty"`
	CameraID  string        `json:"camera_id,omitempty"`
	Severity  AlertSeverity `json:"severity"`
	Message   string        `json:"message"`
	CreatedAt time.Time     `json:"created_at"`
}
