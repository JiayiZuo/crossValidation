package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"crossValidationDemo/internal/config"
	"crossValidationDemo/internal/domain"
	"crossValidationDemo/internal/service"
)

// Handler 处理 HTTP 接口请求。
type Handler struct {
	service *service.ValidationService
	cfg     config.Config
}

func NewHandler(svc *service.ValidationService, cfg config.Config) *Handler {
	return &Handler{service: svc, cfg: cfg}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/ready", h.Ready)
	mux.HandleFunc("/api/v1/rfid-events", h.HandleRFIDEvent)
	mux.HandleFunc("/api/v1/cctv-events", h.HandleCCTVEvent)
	mux.HandleFunc("/api/v1/validate", h.HandleValidate)
	mux.HandleFunc("/api/v1/alerts", h.HandleAlerts)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "ok",
		"service":   "cross-validation",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"redis":    dependencyStatus(h.cfg.UseRedis),
			"postgres": dependencyStatus(h.cfg.UsePostgres),
		},
	})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	statusCode := http.StatusOK
	statusValue := "ready"
	if h.cfg.UseRedis && h.cfg.RedisAddr == "" {
		statusCode = http.StatusServiceUnavailable
		statusValue = "not_ready"
	}
	if h.cfg.UsePostgres && h.cfg.PostgresURL == "" {
		statusCode = http.StatusServiceUnavailable
		statusValue = "not_ready"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  statusValue,
		"service": "cross-validation",
		"dependencies": map[string]string{
			"redis":    dependencyStatus(h.cfg.UseRedis),
			"postgres": dependencyStatus(h.cfg.UsePostgres),
		},
	})
}

func dependencyStatus(enabled bool) string {
	if !enabled {
		return "disabled"
	}
	return "configured"
}

func (h *Handler) HandleRFIDEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var evt domain.RFIDEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		http.Error(w, "invalid RFID event payload", http.StatusBadRequest)
		return
	}

	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}

	result, err := h.service.HandleRFIDEvent(evt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) HandleCCTVEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var evt domain.CCTVEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		http.Error(w, "invalid CCTV event payload", http.StatusBadRequest)
		return
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}

	h.service.HandleCCTVEvent(evt)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (h *Handler) HandleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TagID         string `json:"tag_id"`
		WindowSeconds int    `json:"window_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid validate request", http.StatusBadRequest)
		return
	}
	if req.TagID == "" {
		http.Error(w, "tag_id is required", http.StatusBadRequest)
		return
	}

	window := time.Duration(req.WindowSeconds) * time.Second
	if req.WindowSeconds <= 0 {
		window = 10 * time.Second
	}

	result, err := h.service.ValidateByTag(req.TagID, window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) HandleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	alerts := h.service.Alerts()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"count": len(alerts), "items": alerts})
}

func parseIntParam(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
