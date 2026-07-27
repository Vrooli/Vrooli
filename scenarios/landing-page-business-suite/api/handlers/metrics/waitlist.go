// Package metricshttp owns HTTP transport for the analytics domain. It receives
// generic response and validation behavior from the composition root so domain
// handlers never import the root package.
package metricshttp

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	metrics "landing-page-business-suite-api/internal/metrics"
)

type Dependencies struct {
	DecodeJSON         func(http.ResponseWriter, *http.Request, interface{}) bool
	ValidateEmail      func(http.ResponseWriter, string) (string, bool)
	PathInt64          func(http.ResponseWriter, *http.Request, string) (int64, bool)
	PathInt            func(http.ResponseWriter, *http.Request, string) (int, bool)
	WriteSuccess       func(http.ResponseWriter, string)
	WriteSuccessData   func(http.ResponseWriter, interface{})
	WriteSuccessSimple func(http.ResponseWriter)
	WriteError         func(http.ResponseWriter, int, string)
	WriteErrorType     func(http.ResponseWriter, int, string, string)
	WriteJSON          func(http.ResponseWriter, interface{}) error
	LogError           func(string, map[string]interface{})
}

// seam: EventTracker records a metrics event. The production metrics.Service is
// wired by the API composition root; tests use mocks.FakeEventTracker.
//
// EventTracker makes the HTTP edge testable without a database while keeping
// persistence inside the metrics domain.
type EventTracker interface {
	TrackEvent(metrics.Event) error
}

// seam: AnalyticsReader reads metrics aggregates. The production metrics.Service
// is wired by the API composition root; tests use mocks.FakeAnalyticsReader.
type AnalyticsReader interface {
	GetAnalyticsSummary(time.Time, time.Time) (*metrics.AnalyticsSummary, error)
	GetVariantStats(time.Time, time.Time, string) ([]metrics.VariantStats, error)
}

func (d Dependencies) CreateWaitlist(svc metrics.WaitlistServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email  string `json:"email"`
			Source string `json:"source,omitempty"`
		}
		if !d.DecodeJSON(w, r, &req) {
			return
		}
		email, ok := d.ValidateEmail(w, req.Email)
		if !ok {
			return
		}
		source := strings.TrimSpace(req.Source)
		if source == "" {
			source = "coming_soon"
		}
		if _, err := svc.Create(r.Context(), email, source); err != nil {
			d.LogError("waitlist_create_failed", map[string]interface{}{"email": email, "error": err.Error()})
			d.WriteError(w, http.StatusInternalServerError, "Failed to add email to waitlist")
			return
		}
		d.WriteSuccess(w, "Email added to waitlist")
	}
}

func (d Dependencies) ListWaitlist(svc metrics.WaitlistServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emails, err := svc.List(r.Context())
		if err != nil {
			d.LogError("waitlist_list_failed", map[string]interface{}{"error": err.Error()})
			d.WriteError(w, http.StatusInternalServerError, "Failed to list waitlist emails")
			return
		}
		if emails == nil {
			emails = []metrics.WaitlistEmail{}
		}
		d.WriteSuccessData(w, emails)
	}
}

func (d Dependencies) DeleteWaitlist(svc metrics.WaitlistServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := d.PathInt64(w, r, "id")
		if !ok {
			return
		}
		if err := svc.Delete(r.Context(), id); err != nil {
			d.LogError("waitlist_delete_failed", map[string]interface{}{"id": id, "error": err.Error()})
			d.WriteError(w, http.StatusInternalServerError, "Failed to delete email")
			return
		}
		d.WriteSuccessSimple(w)
	}
}

func (d Dependencies) ExportWaitlist(svc metrics.WaitlistServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emails, err := svc.List(r.Context())
		if err != nil {
			d.LogError("waitlist_export_failed", map[string]interface{}{"error": err.Error()})
			d.WriteError(w, http.StatusInternalServerError, "Failed to export waitlist")
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=waitlist.csv")
		writer := csv.NewWriter(w)
		defer writer.Flush()
		if err := writer.Write([]string{"ID", "Email", "Source", "Created At"}); err != nil {
			d.LogError("waitlist_export_write_header_failed", map[string]interface{}{"error": err.Error()})
			return
		}
		for _, email := range emails {
			if err := writer.Write([]string{fmt.Sprintf("%d", email.ID), email.Email, email.Source, email.CreatedAt.Format("2006-01-02 15:04:05")}); err != nil {
				d.LogError("waitlist_export_write_row_failed", map[string]interface{}{"id": email.ID, "error": err.Error()})
				return
			}
		}
	}
}

// Track receives one idempotent analytics event.
func (d Dependencies) Track(svc EventTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var event metrics.Event
		if !d.DecodeJSON(w, r, &event) {
			return
		}
		if err := svc.TrackEvent(event); err != nil {
			var validationErr *metrics.ValidationError
			if errors.As(err, &validationErr) {
				d.LogError("metrics_track_validation_failed", map[string]interface{}{
					"event_type": event.EventType, "variant_slug": event.VariantSlug, "reason": validationErr.Reason,
				})
				d.WriteErrorType(w, http.StatusBadRequest, validationErr.Reason, "validation")
				return
			}
			d.LogError("metrics_track_failed", map[string]interface{}{
				"event_type": event.EventType, "variant_slug": event.VariantSlug, "error": err.Error(),
			})
			d.WriteErrorType(w, http.StatusInternalServerError, "Failed to track event. Please try again.", "server_error")
			return
		}
		d.LogError("metrics_event_tracked", map[string]interface{}{
			"event_type": event.EventType, "variant_slug": event.VariantSlug, "session_id": event.SessionID,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := d.WriteJSON(w, map[string]interface{}{"success": true, "message": "Event tracked successfully", "event_type": event.EventType}); err != nil {
			d.LogError("metrics_track_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// Summary returns aggregate metrics for the requested range, defaulting to
// the last seven days when callers omit dates.
func (d Dependencies) Summary(svc AnalyticsReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startDate, endDate := metricDateRange(r)
		summary, err := svc.GetAnalyticsSummary(startDate, endDate)
		if err != nil {
			d.LogError("metrics_summary_failed", map[string]interface{}{"start_date": startDate.Format("2006-01-02"), "end_date": endDate.Format("2006-01-02"), "error": err.Error()})
			d.WriteErrorType(w, http.StatusInternalServerError, "Failed to fetch analytics summary. Please try again.", "server_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := d.WriteJSON(w, summary); err != nil {
			d.LogError("metrics_summary_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// VariantStats returns per-variant analytics for the requested range.
func (d Dependencies) VariantStats(svc AnalyticsReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startDate, endDate := metricDateRange(r)
		variantSlug := r.URL.Query().Get("variant")
		stats, err := svc.GetVariantStats(startDate, endDate, variantSlug)
		if err != nil {
			d.LogError("metrics_variant_stats_failed", map[string]interface{}{"start_date": startDate.Format("2006-01-02"), "end_date": endDate.Format("2006-01-02"), "variant_slug": variantSlug, "error": err.Error()})
			d.WriteErrorType(w, http.StatusInternalServerError, "Failed to fetch variant stats. Please try again.", "server_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := d.WriteJSON(w, map[string]interface{}{"start_date": startDate.Format("2006-01-02"), "end_date": endDate.Format("2006-01-02"), "stats": stats}); err != nil {
			d.LogError("metrics_variant_stats_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func metricDateRange(r *http.Request) (time.Time, time.Time) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)
	if value := r.URL.Query().Get("start_date"); value != "" {
		if parsed, err := time.Parse("2006-01-02", value); err == nil {
			startDate = parsed
		}
	}
	if value := r.URL.Query().Get("end_date"); value != "" {
		if parsed, err := time.Parse("2006-01-02", value); err == nil {
			endDate = parsed
		}
	}
	return startDate, endDate
}
