package main

import (
	"time"

	domainmetrics "landing-page-business-suite-api/internal/metrics"
)

// This test-only alias keeps the existing transport characterization tests
// focused on observable behavior while the service moves into its domain.
func generateEventID(event MetricEvent) string {
	return domainmetrics.GenerateEventIDAt(event, time.Now())
}
