package supervision

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"

	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestRestartRecoveryProcessesPersistedDueWatchWithoutMemoryState(t *testing.T) { // [REQ:REQ-P2-008]
	repo, _ := testRepository(t)
	runID := uuid.New()
	source := &cohortSource{retention: eventlog.RetentionState{Generation: 1}}
	creatingService := NewService(repo, source)
	watch, _, err := creatingService.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(runID), IdempotencyKey: "restart-recovery"})
	if err != nil {
		t.Fatal(err)
	}

	// Reconstruct every in-memory supervision object around the same database,
	// which models process restart after watch persistence.
	recoveredService := NewService(NewRepository(repo.db), source)
	recoveredProcessor := NewProcessor(recoveredService, summaryResolver{summaries: []SubjectSummary{{RunID: runID.String(), Status: "running"}}}, nil)
	recoveredProcessor.now = func() time.Time { return repo.now().Add(2 * time.Minute) }
	scheduler := NewScheduler(recoveredService.watches, recoveredProcessor, nil)
	scheduler.now = recoveredProcessor.now
	processed, err := scheduler.RecoverOnce(context.Background())
	if err != nil || processed != 1 {
		t.Fatalf("recovery processed=%d err=%v", processed, err)
	}
	loaded, err := recoveredService.Get(context.Background(), watch.GetWatchId())
	if err != nil || loaded.GetRevision() != 2 || loaded.GetLastDecision() == nil {
		t.Fatalf("recovered watch = %+v err=%v", loaded, err)
	}
}

func TestWatchScaleProtocolMeasuresBoundedSchedulerProcessing(t *testing.T) {
	for _, tier := range []struct {
		name  string
		count int
	}{
		{name: "100-watches", count: 100},
		{name: "1000-watches", count: 1000},
	} {
		t.Run(tier.name, func(t *testing.T) {
			runtime.GC()
			var before runtime.MemStats
			runtime.ReadMemStats(&before)
			goroutinesBefore := runtime.NumGoroutine()

			repo, _ := testRepository(t)
			source := &cohortSource{retention: eventlog.RetentionState{Generation: 1}}
			service := NewService(repo, source)
			ctx := context.Background()
			for i := 0; i < tier.count; i++ {
				if _, _, err := service.Create(ctx, &domainpb.CreateCohortWatchRequest{
					Spec:           validServiceSpec(uuid.New()),
					IdempotencyKey: fmt.Sprintf("processing-scale-%d", i),
				}); err != nil {
					t.Fatal(err)
				}
			}

			processor := NewProcessor(service, nil, nil)
			scheduler := NewScheduler(repo, processor, nil)
			now := time.Now().UTC()
			scheduler.now = func() time.Time { return now }
			processor.now = func() time.Time { return now }

			batchLatencies := make([]time.Duration, 0, (tier.count+99)/100)
			processed := 0
			for processed < tier.count {
				started := time.Now()
				batch, err := scheduler.RecoverOnce(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if batch == 0 {
					t.Fatalf("scheduler stopped making progress after %d/%d watches", processed, tier.count)
				}
				processed += batch
				batchLatencies = append(batchLatencies, time.Since(started))
			}
			if processed != tier.count {
				t.Fatalf("processed=%d, want %d", processed, tier.count)
			}

			sort.Slice(batchLatencies, func(i, j int) bool { return batchLatencies[i] < batchLatencies[j] })
			p50 := batchLatencies[len(batchLatencies)/2]
			p95 := batchLatencies[(len(batchLatencies)*95+99)/100-1]
			maxLatency := batchLatencies[len(batchLatencies)-1]
			limit := 10 * time.Second
			if tier.count == 1000 {
				limit = 30 * time.Second
			}
			if p95 > limit {
				t.Fatalf("p95 scheduler processing latency=%s exceeds %s", p95, limit)
			}

			runtime.GC()
			var after runtime.MemStats
			runtime.ReadMemStats(&after)
			heapDelta := int64(after.HeapInuse) - int64(before.HeapInuse)
			if heapDelta < 0 {
				heapDelta = 0
			}
			if heapDelta > 256<<20 {
				t.Fatalf("heap delta=%d exceeds 256 MiB synthetic processing budget", heapDelta)
			}
			if delta := runtime.NumGoroutine() - goroutinesBefore; delta > 10 {
				t.Fatalf("goroutine delta=%d; scheduler processing must remain one bounded loop", delta)
			}
			t.Logf("host=%s/%s go=%s gomaxprocs=%d watches=%d batches=%d p50=%s p95=%s max=%s heap_delta=%d goroutines_before=%d goroutines_after=%d", runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.GOMAXPROCS(0), tier.count, len(batchLatencies), p50, p95, maxLatency, heapDelta, goroutinesBefore, runtime.NumGoroutine())
		})
	}
}
