package main

import domainmetrics "landing-page-business-suite-api/internal/metrics"

// Waitlist behavior is domain-owned by internal/metrics. These aliases retain
// the established HTTP composition contract while handlers are migrated.
type (
	WaitlistServicer = domainmetrics.WaitlistServicer
	WaitlistEmail    = domainmetrics.WaitlistEmail
	WaitlistService  = domainmetrics.WaitlistService
)

func NewWaitlistService(db domainmetrics.WaitlistStore) *WaitlistService {
	return domainmetrics.NewWaitlistService(db)
}
