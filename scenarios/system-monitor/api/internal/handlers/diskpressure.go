package handlers

// DOC: docs/operations/RUNBOOK.md#disk-pressure
//
// Plain-JSON operator surface, matching the convention already used by the
// forensics and logs handlers. This is the endpoint behind the single
// documented command for answering "how much disk pressure is there right
// now, and what has the system done about it".

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
)

// violationHistory is the read side of threshold violation storage.
type violationHistory interface {
	GetThresholdViolations(ctx context.Context, timeRange repository.TimeRange) ([]*models.ThresholdViolation, error)
}

// DiskPressureHandler serves the current disk-pressure picture.
type DiskPressureHandler struct {
	scheduler *services.ThresholdScheduler
	history   violationHistory
	log       *slog.Logger
}

// NewDiskPressureHandler creates the handler.
func NewDiskPressureHandler(scheduler *services.ThresholdScheduler, history violationHistory, log *slog.Logger) *DiskPressureHandler {
	if log == nil {
		log = slog.Default()
	}
	return &DiskPressureHandler{scheduler: scheduler, history: history, log: log}
}

// DiskPressureResponse is the operator-facing shape.
type DiskPressureResponse struct {
	Observed             bool                         `json:"observed"`
	ObservedAt           time.Time                    `json:"observed_at,omitempty"`
	MountPath            string                       `json:"mount_path,omitempty"`
	Band                 string                       `json:"band"`
	UsedPercent          float64                      `json:"used_percent"`
	UsedBytes            int64                        `json:"used_bytes"`
	AvailableBytes       int64                        `json:"available_bytes"`
	TotalBytes           int64                        `json:"total_bytes"`
	FillRateBytesPerHour int64                        `json:"fill_rate_bytes_per_hour"`
	HotWriters           []DiskPressureWriter         `json:"hot_writers,omitempty"`
	CheckInterval        string                       `json:"check_interval"`
	Violations           int64                        `json:"violations_recorded"`
	LastViolation        *models.ThresholdViolation   `json:"last_violation,omitempty"`
	LastTransition       *services.BandObservation    `json:"last_band_transition,omitempty"`
	LastRemediation      *services.RemediationResult  `json:"last_remediation,omitempty"`
	RecentHistory        []*models.ThresholdViolation `json:"recent_violations,omitempty"`
	LastError            string                       `json:"last_error,omitempty"`
}

type DiskPressureWriter struct {
	Root          string `json:"root"`
	CurrentBytes  int64  `json:"current_bytes"`
	BytesPerHour  int64  `json:"bytes_per_hour"`
	WindowSeconds int64  `json:"window_seconds"`
}

// Handle serves GET /api/v1/disk-pressure.
func (h *DiskPressureHandler) Handle(w http.ResponseWriter, r *http.Request) {
	status := h.scheduler.Status()

	resp := DiskPressureResponse{
		Observed:             status.HasRun,
		ObservedAt:           status.LastRunAt,
		MountPath:            status.LastUsage.MountPath,
		Band:                 status.Band.String(),
		UsedPercent:          status.LastUsage.UsedPercent,
		UsedBytes:            status.LastUsage.UsedBytes,
		AvailableBytes:       status.LastUsage.AvailableBytes,
		TotalBytes:           status.LastUsage.TotalBytes,
		FillRateBytesPerHour: status.LastUsage.FillRateBytesPerHour,
		CheckInterval:        status.NextInterval.String(),
		Violations:           status.Violations,
		LastViolation:        status.LastViolation,
		LastTransition:       status.LastTransition,
		LastRemediation:      status.LastRemediation,
		LastError:            status.LastError,
	}
	for _, writer := range status.LastWriters {
		if writer.Hot {
			resp.HotWriters = append(resp.HotWriters, DiskPressureWriter{Root: writer.Root, CurrentBytes: writer.Bytes, BytesPerHour: writer.BytesPerHour, WindowSeconds: int64(writer.DeltaHours * 3600)})
		}
	}

	if h.history != nil {
		// A day of history is enough to answer "did this fire overnight",
		// which is the question the incident actually raised.
		violations, err := h.history.GetThresholdViolations(r.Context(), repository.TimeRange{
			StartTime: time.Now().Add(-24 * time.Hour),
			EndTime:   time.Now(),
		})
		if err != nil {
			h.log.Error("read threshold violations", "error", err)
		} else {
			resp.RecentHistory = violations
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.Error("encode disk pressure response", "error", err)
	}
}
