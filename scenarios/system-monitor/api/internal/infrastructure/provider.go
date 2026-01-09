package infrastructure

import (
	"context"
	"time"

	"system-monitor-api/internal/models"
)

// Provider isolates infrastructure-level telemetry.
type Provider interface {
	GetInfrastructureMonitorData(ctx context.Context) (*models.InfrastructureMonitorData, error)
	CheckServiceDependencies() []models.ServiceHealth
}

// StaticProvider returns placeholder infrastructure data.
type StaticProvider struct{}

func NewStaticProvider() *StaticProvider {
	return &StaticProvider{}
}

func (p *StaticProvider) GetInfrastructureMonitorData(ctx context.Context) (*models.InfrastructureMonitorData, error) {
	result := &models.InfrastructureMonitorData{
		Timestamp: time.Now(),
	}

	result.DatabasePools = []models.ConnectionPool{
		{
			Name:     "postgres-main",
			Active:   8,
			Idle:     2,
			MaxSize:  10,
			Waiting:  0,
			Healthy:  true,
			LeakRisk: "low",
		},
	}

	result.HTTPClientPools = []models.ConnectionPool{
		{
			Name:     "scenario-api-1->ollama",
			Active:   3,
			Idle:     7,
			MaxSize:  10,
			Waiting:  0,
			Healthy:  true,
			LeakRisk: "low",
		},
	}

	result.MessageQueues = models.MessageQueueInfo{
		RedisPubSub: models.RedisPubSubInfo{
			Subscribers: 12,
			Channels:    5,
		},
		BackgroundJobs: models.BackgroundJobsInfo{
			Pending: 3,
			Active:  1,
			Failed:  0,
		},
	}

	result.StorageIO = models.StorageIOInfo{
		DiskQueueDepth: 0.2,
		IOWaitPercent:  2.5,
		ReadMBPerSec:   15.0,
		WriteMBPerSec:  8.0,
	}

	return result, nil
}

func (p *StaticProvider) CheckServiceDependencies() []models.ServiceHealth {
	return []models.ServiceHealth{
		{
			Name:      "postgres",
			Status:    "healthy",
			LatencyMs: 2.5,
			LastCheck: time.Now(),
			Endpoint:  "localhost:5432",
		},
		{
			Name:      "redis",
			Status:    "healthy",
			LatencyMs: 0.8,
			LastCheck: time.Now(),
			Endpoint:  "localhost:6379",
		},
		{
			Name:      "qdrant",
			Status:    "healthy",
			LatencyMs: 4.1,
			LastCheck: time.Now(),
			Endpoint:  "localhost:6333",
		},
		{
			Name:      "ollama",
			Status:    "healthy",
			LatencyMs: 15.2,
			LastCheck: time.Now(),
			Endpoint:  "localhost:11434",
		},
	}
}
