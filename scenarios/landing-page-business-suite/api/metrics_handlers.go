package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// handleMetricsTrack handles POST /api/v1/metrics/track
// Tracks analytics events with variant_id for A/B testing
// Implements OT-P0-019: Event variant tagging
// [REQ:SIGNAL-FEEDBACK] Provides structured logging for observability
func handleMetricsTrack(service *MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var event MetricEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			logStructuredError("metrics_track_decode_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		// Track event (idempotent)
		if err := service.TrackEvent(event); err != nil {
			var validationErr *MetricValidationError
			if errors.As(err, &validationErr) {
				logStructured("metrics_track_validation_failed", map[string]interface{}{
					"event_type": event.EventType,
					"variant_id": event.VariantID,
					"reason":     validationErr.Reason,
				})
				writeJSONError(w, http.StatusBadRequest, validationErr.Reason, ApiErrorTypeValidation)
				return
			}
			logStructuredError("metrics_track_failed", map[string]interface{}{
				"event_type": event.EventType,
				"variant_id": event.VariantID,
				"error":      err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to track event. Please try again.", ApiErrorTypeServerError)
			return
		}

		// Log successful event tracking for observability
		logStructured("metrics_event_tracked", map[string]interface{}{
			"event_type": event.EventType,
			"variant_id": event.VariantID,
			"session_id": event.SessionID,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"message":    "Event tracked successfully",
			"event_type": event.EventType,
		}); err != nil {
			logStructuredError("metrics_track_encode_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// handleMetricsSummary handles GET /api/v1/metrics/summary
// Returns analytics dashboard summary with variant stats
// Implements OT-P0-023: Analytics summary (total visitors, conversion rate per variant, top CTAs)
// [REQ:SIGNAL-FEEDBACK] Provides structured logging for observability
func handleMetricsSummary(service *MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse time range from query params (default: last 7 days)
		endDate := time.Now()
		startDate := endDate.AddDate(0, 0, -7) // 7 days ago

		if startParam := r.URL.Query().Get("start_date"); startParam != "" {
			if parsed, err := time.Parse("2006-01-02", startParam); err == nil {
				startDate = parsed
			}
		}
		if endParam := r.URL.Query().Get("end_date"); endParam != "" {
			if parsed, err := time.Parse("2006-01-02", endParam); err == nil {
				endDate = parsed
			}
		}

		// Get analytics summary
		summary, err := service.GetAnalyticsSummary(startDate, endDate)
		if err != nil {
			logStructuredError("metrics_summary_failed", map[string]interface{}{
				"start_date": startDate.Format("2006-01-02"),
				"end_date":   endDate.Format("2006-01-02"),
				"error":      err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to fetch analytics summary. Please try again.", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summary); err != nil {
			logStructuredError("metrics_summary_encode_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// handleMetricsVariantStats handles GET /api/v1/metrics/variants
// Returns detailed stats for variants with optional filtering
// Implements OT-P0-020: Analytics variant filtering (by variant slug and time range)
// Implements OT-P0-024: Variant detail view (views, clicks, conversions, rate, trend)
// [REQ:SIGNAL-FEEDBACK] Provides structured logging for observability
func handleMetricsVariantStats(service *MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse time range from query params (default: last 7 days)
		endDate := time.Now()
		startDate := endDate.AddDate(0, 0, -7)

		if startParam := r.URL.Query().Get("start_date"); startParam != "" {
			if parsed, err := time.Parse("2006-01-02", startParam); err == nil {
				startDate = parsed
			}
		}
		if endParam := r.URL.Query().Get("end_date"); endParam != "" {
			if parsed, err := time.Parse("2006-01-02", endParam); err == nil {
				endDate = parsed
			}
		}

		// Optional variant filter
		variantSlug := r.URL.Query().Get("variant")

		// Get variant stats
		stats, err := service.GetVariantStats(startDate, endDate, variantSlug)
		if err != nil {
			logStructuredError("metrics_variant_stats_failed", map[string]interface{}{
				"start_date":   startDate.Format("2006-01-02"),
				"end_date":     endDate.Format("2006-01-02"),
				"variant_slug": variantSlug,
				"error":        err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to fetch variant stats. Please try again.", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"start_date": startDate.Format("2006-01-02"),
			"end_date":   endDate.Format("2006-01-02"),
			"stats":      stats,
		}); err != nil {
			logStructuredError("metrics_variant_stats_encode_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}
