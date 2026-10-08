package main

import domainmetrics "landing-page-business-suite-api/internal/metrics"

func NewFeedbackService(db domainmetrics.FeedbackStore) *domainmetrics.FeedbackService {
	return domainmetrics.NewFeedbackService(db)
}
