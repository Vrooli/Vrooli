package mocks

import (
	"context"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// MonitorQuerier is a configurable test double for handlers.MonitorQuerier.
type MonitorQuerier struct {
	metrics          *models.MetricsResponse
	freshMetrics     *models.MetricsResponse
	detailedMetrics  *models.DetailedMetrics
	timelineResponse *models.MetricsTimelineResponse
	processData      *models.ProcessMonitorData
	infraData        *models.InfrastructureMonitorData
	active           bool
	err              error
}

func NewMonitorQuerier() *MonitorQuerier {
	return &MonitorQuerier{}
}

func (m *MonitorQuerier) WithCurrentMetrics(metrics *models.MetricsResponse) *MonitorQuerier {
	m.metrics = metrics
	return m
}

func (m *MonitorQuerier) WithFreshMetrics(metrics *models.MetricsResponse) *MonitorQuerier {
	m.freshMetrics = metrics
	return m
}

func (m *MonitorQuerier) WithDetailedMetrics(metrics *models.DetailedMetrics) *MonitorQuerier {
	m.detailedMetrics = metrics
	return m
}

func (m *MonitorQuerier) WithTimelineResponse(timeline *models.MetricsTimelineResponse) *MonitorQuerier {
	m.timelineResponse = timeline
	return m
}

func (m *MonitorQuerier) WithProcessData(data *models.ProcessMonitorData) *MonitorQuerier {
	m.processData = data
	return m
}

func (m *MonitorQuerier) WithInfrastructureData(data *models.InfrastructureMonitorData) *MonitorQuerier {
	m.infraData = data
	return m
}

func (m *MonitorQuerier) WithActive(active bool) *MonitorQuerier {
	m.active = active
	return m
}

func (m *MonitorQuerier) WithError(err error) *MonitorQuerier {
	m.err = err
	return m
}

func (m *MonitorQuerier) GetCurrentMetrics(_ context.Context) (*models.MetricsResponse, error) {
	return m.metrics, m.err
}

func (m *MonitorQuerier) GetCurrentMetricsFresh(_ context.Context) (*models.MetricsResponse, error) {
	if m.freshMetrics != nil {
		return m.freshMetrics, m.err
	}
	return m.metrics, m.err
}

func (m *MonitorQuerier) GetDetailedMetrics(_ context.Context) (*models.DetailedMetrics, error) {
	return m.detailedMetrics, m.err
}

func (m *MonitorQuerier) GetMetricsTimeline(_ context.Context, _, _ int) (*models.MetricsTimelineResponse, error) {
	return m.timelineResponse, m.err
}

func (m *MonitorQuerier) GetProcessMonitorData(_ context.Context) (*models.ProcessMonitorData, error) {
	return m.processData, m.err
}

func (m *MonitorQuerier) GetInfrastructureMonitorData(_ context.Context) (*models.InfrastructureMonitorData, error) {
	return m.infraData, m.err
}

func (m *MonitorQuerier) IsActive() bool {
	return m.active
}
