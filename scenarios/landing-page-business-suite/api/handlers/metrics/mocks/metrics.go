// Package mocks provides reusable metrics transport test doubles.
package mocks

import (
	"time"

	metricshttp "landing-page-business-suite-api/handlers/metrics"
	metrics "landing-page-business-suite-api/internal/metrics"
)

type FakeEventTracker struct {
	TrackEventFunc func(metrics.Event) error
	Events         []metrics.Event
}

func (f *FakeEventTracker) TrackEvent(event metrics.Event) error {
	f.Events = append(f.Events, event)
	if f.TrackEventFunc != nil {
		return f.TrackEventFunc(event)
	}
	return nil
}

var _ metricshttp.EventTracker = (*FakeEventTracker)(nil)

type FakeAnalyticsReader struct {
	GetAnalyticsSummaryFunc func(time.Time, time.Time) (*metrics.AnalyticsSummary, error)
	GetVariantStatsFunc     func(time.Time, time.Time, string) ([]metrics.VariantStats, error)
}

func (f *FakeAnalyticsReader) GetAnalyticsSummary(startDate, endDate time.Time) (*metrics.AnalyticsSummary, error) {
	if f.GetAnalyticsSummaryFunc != nil {
		return f.GetAnalyticsSummaryFunc(startDate, endDate)
	}
	return &metrics.AnalyticsSummary{}, nil
}

func (f *FakeAnalyticsReader) GetVariantStats(startDate, endDate time.Time, variantSlug string) ([]metrics.VariantStats, error) {
	if f.GetVariantStatsFunc != nil {
		return f.GetVariantStatsFunc(startDate, endDate, variantSlug)
	}
	return nil, nil
}

var _ metricshttp.AnalyticsReader = (*FakeAnalyticsReader)(nil)

type FakeFeedbackNotifier struct {
	Notifications []*metrics.FeedbackRequest
}

func (f *FakeFeedbackNotifier) Notify(feedback *metrics.FeedbackRequest) {
	f.Notifications = append(f.Notifications, feedback)
}

var _ metricshttp.FeedbackNotifier = (*FakeFeedbackNotifier)(nil)
