package main

import domainmetrics "landing-page-business-suite-api/internal/metrics"

// The HTTP composition package aliases the metrics domain contract. All
// metrics business rules live in internal/metrics.
type (
	MetricsService        = domainmetrics.Service
	MetricEvent           = domainmetrics.Event
	VariantStats          = domainmetrics.VariantStats
	AnalyticsSummary      = domainmetrics.AnalyticsSummary
	MetricValidationError = domainmetrics.ValidationError
)

func NewMetricsService(db domainmetrics.Store) *MetricsService { return domainmetrics.NewService(db) }
