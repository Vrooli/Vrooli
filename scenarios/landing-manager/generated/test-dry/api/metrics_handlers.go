package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleMetricsTrack handles POST /api/v1/metrics/track
// Tracks analytics events with variant_id for A/B testing
// Implements OT-P0-019: Event variant tagging
func handleMetricsTrack(service *MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var event MetricEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
			return
		}

		// Validate required fields
		if event.EventType == "" {
			http.Error(w, `{"error": "event_type is required"}`, http.StatusBadRequest)
			return
		}
		if event.VariantID == 0 {
			http.Error(w, `{"error": "variant_id is required"}`, http.StatusBadRequest)
			return
		}
		if event.SessionID == "" {
			http.Error(w, `{"error": "session_id is required"}`, http.StatusBadRequest)
			return
		}

		// Track event (idempotent)
		if err := service.TrackEvent(event); err != nil {
			http.Error(w, `{"error": "Failed to track event"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Event tracked successfully",
		})
	}
}

// handleMetricsSummary handles GET /api/v1/metrics/summary
// Returns analytics dashboard summary with variant stats
// Implements OT-P0-023: Analytics summary (total visitors, conversion rate per variant, top CTAs)
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
			http.Error(w, `{"error": "Failed to fetch analytics summary"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summary)
	}
}

// handleMetricsVariantStats handles GET /api/v1/metrics/variants
// Returns detailed stats for variants with optional filtering
// Implements OT-P0-020: Analytics variant filtering (by variant slug and time range)
// Implements OT-P0-024: Variant detail view (views, clicks, conversions, rate, trend)
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
			http.Error(w, `{"error": "Failed to fetch variant stats"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"start_date": startDate.Format("2006-01-02"),
			"end_date":   endDate.Format("2006-01-02"),
			"stats":      stats,
		})
	}
}
