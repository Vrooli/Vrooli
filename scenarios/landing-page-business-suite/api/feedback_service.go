package main

import domainmetrics "landing-page-business-suite-api/internal/metrics"

// Feedback business rules are owned by internal/metrics; the root package only
// composes the current HTTP transport during the incremental migration.
type (
	FeedbackServicer    = domainmetrics.FeedbackServicer
	FeedbackRequest     = domainmetrics.FeedbackRequest
	FeedbackService     = domainmetrics.FeedbackService
	CreateFeedbackInput = domainmetrics.CreateFeedbackInput
)

func NewFeedbackService(db domainmetrics.FeedbackStore) *FeedbackService {
	return domainmetrics.NewFeedbackService(db)
}
