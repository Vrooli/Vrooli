package repository_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/sqlite"
)

type metricCycleRepository interface {
	SaveMetricCycle(context.Context, string, time.Time, []repository.MetricObservation) error
	GetMetrics(context.Context, repository.MetricsFilter) ([]*models.MetricsResponse, error)
	GetLatestMetrics(context.Context) (*models.MetricsResponse, error)
}

func TestMetricProjectionMatchesAcrossStorageBackends(t *testing.T) {
	sqliteRepo, err := sqlite.NewInMemoryRepository()
	if err != nil {
		t.Fatal(err)
	}
	defer sqliteRepo.Close()

	backends := []struct {
		name string
		repo metricCycleRepository
	}{
		{name: "memory", repo: memory.NewRepository()},
		{name: "sqlite", repo: sqliteRepo},
	}
	observedAt := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	observations := []repository.MetricObservation{
		{CollectorName: "cpu", Values: map[string]interface{}{
			"status": "measured", "usage_percent": 35.0,
			"context_switches_per_second": 4.5,
			"mode_breakdown":              map[string]float64{"iowait": 2.0, "steal": 1.0},
		}},
		{CollectorName: "memory", Values: map[string]interface{}{
			"status": "measured", "usage_percent": 55.0,
			"swap": map[string]interface{}{"percent": 12.0},
		}},
		{CollectorName: "network", Values: map[string]interface{}{
			"status": "measured", "tcp_connections": 17,
			"established_rate_per_second": 3.0,
		}},
		{CollectorName: "gpu", Values: map[string]interface{}{"status": "measured", "total_usage_percent": 61.0}},
		{CollectorName: "disk", Values: map[string]interface{}{
			"status": "measured", "usage": map[string]interface{}{"percent": 72.0},
		}},
		{CollectorName: "pressure", Values: map[string]interface{}{
			"cpu_psi_some_avg10": 0.5, "cpu_psi_status": "measured",
			"fragmentation_max_free_order": 3.0, "fragmentation_status": "measured",
		}},
	}

	var expected []*models.MetricsResponse
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			if err := backend.repo.SaveMetricCycle(context.Background(), "cycle-1", observedAt, observations); err != nil {
				t.Fatal(err)
			}
			actual, err := backend.repo.GetMetrics(context.Background(), repository.MetricsFilter{})
			if err != nil {
				t.Fatal(err)
			}
			if len(actual) != 1 {
				t.Fatalf("got %d samples, want one", len(actual))
			}
			latest, err := backend.repo.GetLatestMetrics(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(latest, actual[0]) {
				latestJSON, _ := json.MarshalIndent(latest, "", "  ")
				actualJSON, _ := json.MarshalIndent(actual[0], "", "  ")
				t.Fatalf("latest projection differs from cycle history\nlatest: %s\nhistory: %s", latestJSON, actualJSON)
			}
			if backend.name == "memory" {
				expected = actual
				return
			}
			if !reflect.DeepEqual(actual, expected) {
				actualJSON, _ := json.MarshalIndent(actual[0], "", "  ")
				expectedJSON, _ := json.MarshalIndent(expected[0], "", "  ")
				t.Fatalf("backend projection differs\nactual: %s\nexpected: %s", actualJSON, expectedJSON)
			}
		})
	}
}

func TestFailedMetricValuesAreNotProjectedAsMeasurements(t *testing.T) {
	sqliteRepo, err := sqlite.NewInMemoryRepository()
	if err != nil {
		t.Fatal(err)
	}
	defer sqliteRepo.Close()
	backends := []struct {
		name string
		repo metricCycleRepository
	}{
		{name: "memory", repo: memory.NewRepository()},
		{name: "sqlite", repo: sqliteRepo},
	}
	observations := []repository.MetricObservation{
		{CollectorName: "cpu", Values: map[string]interface{}{"status": "failed", "usage_percent": 99.0}},
		{CollectorName: "memory", Values: map[string]interface{}{
			"status": "failed", "usage_percent": 99.0, "swap": map[string]interface{}{"percent": 88.0},
		}},
		{CollectorName: "network", Values: map[string]interface{}{"status": "failed", "tcp_connections": 55}},
		{CollectorName: "gpu", Values: map[string]interface{}{"status": "failed", "total_usage_percent": 90.0}},
		{CollectorName: "disk", Values: map[string]interface{}{
			"status": "failed", "usage": map[string]interface{}{"percent": 99.0},
		}},
	}
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			if err := backend.repo.SaveMetricCycle(context.Background(), "failed-cycle", time.Now(), observations); err != nil {
				t.Fatal(err)
			}
			rows, err := backend.repo.GetMetrics(context.Background(), repository.MetricsFilter{})
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 {
				t.Fatalf("got %d rows, want one", len(rows))
			}
			row := rows[0]
			if row.CPUState.Status != "failed" || row.CPUUsage != 0 || row.MemoryState.Status != "failed" || row.MemoryUsage != 0 {
				t.Fatalf("failed cpu/memory values were projected: %+v", row)
			}
			if row.ConnectionsState.Status != "failed" || row.TCPConnections != 0 || row.GPUState.Status != "failed" || row.GPUUsage != nil {
				t.Fatalf("failed network/gpu values were projected: %+v", row)
			}
			if row.DiskState.Status != "failed" || row.DiskUsage != 0 || row.SwapUsage != nil {
				t.Fatalf("failed disk/swap values were projected: %+v", row)
			}
		})
	}
}
