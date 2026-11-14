package main

import domainmetrics "landing-page-business-suite-api/internal/metrics"

func NewWaitlistService(db domainmetrics.WaitlistStore) *domainmetrics.WaitlistService {
	return domainmetrics.NewWaitlistService(db)
}
