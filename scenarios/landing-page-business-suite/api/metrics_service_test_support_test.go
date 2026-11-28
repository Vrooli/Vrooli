package main

import domainmetrics "landing-page-business-suite-api/internal/metrics"

func NewMetricsService(db domainmetrics.Store) *domainmetrics.Service {
	return domainmetrics.NewService(db)
}
