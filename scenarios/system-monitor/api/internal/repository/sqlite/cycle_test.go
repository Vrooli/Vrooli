package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

func TestMetricCycleSurvivesFreshSchemaRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "system-monitor.db")
	when := time.Date(2026, 8, 20, 12, 0, 0, 456, time.UTC)

	repo, err := NewRepository(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveMetricCycle(context.Background(), "cycle-restart", when, []repository.MetricObservation{
		{CollectorName: "cpu", Values: map[string]interface{}{"usage_percent": 7.25}},
	}); err != nil {
		repo.Close()
		t.Fatal(err)
	}
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewRepository(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	rows, err := reopened.GetMetrics(context.Background(), repository.MetricsFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].CycleID != "cycle-restart" || !rows[0].Timestamp.Equal(when) {
		t.Fatalf("restart lost cycle: %#v", rows)
	}
}

func TestMetricsSamplingSelectsNewestCyclePerBucket(t *testing.T) {
	repo, err := NewInMemoryRepository()
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	base := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 12; i++ {
		if err := repo.SaveMetricCycle(context.Background(), fmt.Sprintf("cycle-%d", i), base.Add(time.Duration(i)*time.Minute), []repository.MetricObservation{
			{CollectorName: "cpu", Values: map[string]interface{}{"usage_percent": float64(i)}},
			{CollectorName: "memory", Values: map[string]interface{}{"usage_percent": float64(i + 20)}},
		}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := repo.GetMetrics(context.Background(), repository.MetricsFilter{
		TimeRange:     repository.TimeRange{StartTime: base, EndTime: base.Add(12 * time.Minute)},
		SampleBuckets: []repository.TimeRange{{StartTime: base, EndTime: base.Add(5 * time.Minute)}, {StartTime: base.Add(5 * time.Minute), EndTime: base.Add(10 * time.Minute)}, {StartTime: base.Add(10 * time.Minute), EndTime: base.Add(12 * time.Minute)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("sampled cycles = %d, want 3", len(rows))
	}
	for i, want := range []string{"cycle-5", "cycle-10", "cycle-12"} {
		if rows[i].CycleID != want || rows[i].MemoryState.Status != "measured" || rows[i].CPUState.Status != "measured" {
			t.Fatalf("sample %d = %+v, want %s with both collectors", i, rows[i], want)
		}
	}
}

func TestMetricsSamplingReachesPastDefaultNewestCycleLimit(t *testing.T) {
	repo, err := NewInMemoryRepository()
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	base := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 1100; i++ {
		if err := repo.SaveMetricCycle(ctx, fmt.Sprintf("cycle-%d", i), base.Add(time.Duration(i)*time.Minute), []repository.MetricObservation{{CollectorName: "memory", Values: map[string]interface{}{"usage_percent": float64(i)}}}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := repo.GetMetrics(ctx, repository.MetricsFilter{SampleBuckets: []repository.TimeRange{
		{StartTime: base, EndTime: base.Add(10 * time.Minute)},
		{StartTime: base.Add(1090 * time.Minute), EndTime: base.Add(1100 * time.Minute)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].CycleID != "cycle-10" || rows[1].CycleID != "cycle-1100" {
		t.Fatalf("history omitted old bucket: %+v", rows)
	}
	var buckets []repository.TimeRange
	for i := 0; i < 205; i++ {
		buckets = append(buckets, repository.TimeRange{StartTime: base.Add(time.Duration(i) * time.Minute), EndTime: base.Add(time.Duration(i+1) * time.Minute)})
	}
	rows, err = repo.GetMetrics(ctx, repository.MetricsFilter{SampleBuckets: buckets})
	if err != nil || len(rows) != 205 || rows[0].CycleID != "cycle-1" || rows[204].CycleID != "cycle-205" {
		t.Fatalf("chunked sampling failed: count=%d err=%v", len(rows), err)
	}
}

func TestSampledTimelineBreaksTimestampTiesConsistently(t *testing.T) {
	sqliteRepo, err := NewInMemoryRepository()
	if err != nil {
		t.Fatal(err)
	}
	defer sqliteRepo.Close()
	type metricRepo interface {
		SaveMetricCycle(context.Context, string, time.Time, []repository.MetricObservation) error
		GetMetrics(context.Context, repository.MetricsFilter) ([]*models.MetricsResponse, error)
	}
	for name, repo := range map[string]metricRepo{"sqlite": sqliteRepo, "memory": memory.NewRepository()} {
		t.Run(name, func(t *testing.T) {
			when := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
			for _, cycle := range []string{"cycle-a", "cycle-z"} {
				if err := repo.SaveMetricCycle(context.Background(), cycle, when, []repository.MetricObservation{{CollectorName: "memory", Values: map[string]interface{}{"usage_percent": float64(10)}}}); err != nil {
					t.Fatal(err)
				}
			}
			rows, err := repo.GetMetrics(context.Background(), repository.MetricsFilter{SampleBuckets: []repository.TimeRange{{StartTime: when.Add(-time.Minute), EndTime: when}}})
			if err != nil || len(rows) != 1 || rows[0].CycleID != "cycle-z" {
				t.Fatalf("unexpected tie selection: %+v, %v", rows, err)
			}
		})
	}
}

func TestMetricCyclePersistsOneLogicalSample(t *testing.T) {
	repo, err := NewInMemoryRepository()
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	when := time.Date(2026, 8, 20, 12, 0, 0, 456, time.UTC)
	if err := repo.SaveMetricCycle(context.Background(), "cycle-sqlite", when, []repository.MetricObservation{
		{CollectorName: "cpu", Values: map[string]interface{}{"usage_percent": 12.5}},
		{CollectorName: "network", Values: map[string]interface{}{"status": "unsupported", "reason": "backend unavailable"}},
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.GetMetrics(context.Background(), repository.MetricsFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].CycleID != "cycle-sqlite" || !rows[0].Timestamp.Equal(when) {
		t.Fatalf("unexpected cycle rows: %#v", rows)
	}
	if rows[0].ConnectionsState.Status != "unsupported" {
		t.Fatalf("network state = %#v", rows[0].ConnectionsState)
	}
	if rows[0].CPUState.Status != "measured" || rows[0].CPUState.Value != 12.5 {
		t.Fatalf("CPU state = %#v", rows[0].CPUState)
	}
	if err := repo.SaveMetricCycle(context.Background(), "cycle-sqlite", when, nil); err == nil {
		t.Fatal("duplicate cycle was accepted")
	}
}
