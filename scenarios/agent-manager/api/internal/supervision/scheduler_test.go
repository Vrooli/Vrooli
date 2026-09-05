package supervision

import (
	"context"
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
