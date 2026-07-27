// DOC: docs/reference/api/metrics.md - Analytics and event tracking endpoints
// DOC: docs/concepts/CONCEPTS.md#ab-testing-system - Metrics integration with A/B testing
// DOC: PRD.md#OT-P0-019 - Event variant tagging requirement
package main

import (
	"encoding/json"
	"net/http"

	metricshttp "landing-page-business-suite-api/handlers/metrics"
)

var metricsHandlerDependencies = metricshttp.Dependencies{
	DecodeJSON: decodeJSONBody,
	WriteErrorType: func(w http.ResponseWriter, status int, message, errorType string) {
		writeJSONError(w, status, message, errorType)
	},
	WriteJSON: func(w http.ResponseWriter, value interface{}) error {
		return json.NewEncoder(w).Encode(value)
	},
	LogError: logStructuredError,
}

var (
	_ metricshttp.EventTracker    = (*MetricsService)(nil)
	_ metricshttp.AnalyticsReader = (*MetricsService)(nil)
)

func handleMetricsTrack(service *MetricsService) http.HandlerFunc {
	return metricsHandlerDependencies.Track(service)
}

func handleMetricsSummary(service *MetricsService) http.HandlerFunc {
	return metricsHandlerDependencies.Summary(service)
}

func handleMetricsVariantStats(service *MetricsService) http.HandlerFunc {
	return metricsHandlerDependencies.VariantStats(service)
}
