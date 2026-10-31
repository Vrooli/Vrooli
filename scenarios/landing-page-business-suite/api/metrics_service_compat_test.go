package main

import (
	"time"

	domainmetrics "landing-page-business-suite-api/internal/metrics"
)

func generateEventID(event domainmetrics.Event) string {
	return domainmetrics.GenerateEventIDAt(event, time.Now())
}
