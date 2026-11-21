package codingjobs_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"client-rehearsal/internal/codingjobs"
	localdb "client-rehearsal/internal/database"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
)

func newService(t *testing.T) codingjobs.Service {
	t.Helper()
	store := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), store,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(codingjobs.Schema),
	))
	require.NoError(t, codingjobs.EnsureReplayKeySchema(context.Background(), store))
	clock := scheduletest.New(time.Date(2026, 11, 14, 1, 0, 0, 0, time.UTC))
	return codingjobs.NewService(codingjobs.NewSQLiteRepository(store), clock.Now)
}

func TestCodingJobRequiresBlindVisibleFieldsAndStoresOnlySealReference(t *testing.T) {
	service := newService(t)
	ctx := context.Background()
	_, err := service.Create(ctx, codingjobs.CreateInput{ReplayKey: "replay-1", CaseID: "case-1", Project: "org/repo", Title: "Fix the parser", Task: "Handle the documented empty-input case", SourceSealRef: "seal:source-1"})
	require.NoError(t, err)
	items, err := service.List(ctx, 5)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, codingjobs.StateReady, items[0].State)
	require.Equal(t, "seal:source-1", items[0].SourceSealRef)
	require.Empty(t, items[0].EvaluatorReceiptRef, "solver-visible job has no evaluation or reference payload")
	require.Empty(t, items[0].Score)
	require.Len(t, items[0].Events, 1)
}

func TestCodingJobCreateIsIdempotentAndRejectsReplayKeyReuse(t *testing.T) {
	service := newService(t)
	ctx := context.Background()
	input := codingjobs.CreateInput{ReplayKey: "case-1:attempt-1", CaseID: "case-1", Project: "org/repo", Title: "Fix the parser", Task: "Handle the documented empty-input case", SourceSealRef: "seal:source-1"}
	first, err := service.Create(ctx, input)
	require.NoError(t, err)
	second, err := service.Create(ctx, input)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID, "retry after a lost create response must return the durable job")
	require.Equal(t, first.Revision, second.Revision)
	require.Len(t, second.Events, 1)

	items, err := service.List(ctx, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)

	changed := input
	changed.Task = "Different task"
	_, err = service.Create(ctx, changed)
	var conflict codingjobs.ErrConflict
	require.ErrorAs(t, err, &conflict, "one replay key cannot identify two different case payloads")
}

func TestCodingJobEnforcesDistinctSolverScorerAndOneFrozenOutcome(t *testing.T) {
	service := newService(t)
	ctx := context.Background()
	job, err := service.Create(ctx, codingjobs.CreateInput{ReplayKey: "replay-2", CaseID: "case-1", Project: "org/repo", Title: "Fix", Task: "Repair the bug", SourceSealRef: "seal:source-1"})
	require.NoError(t, err)

	_, err = service.RecordStage(ctx, codingjobs.StageInput{ID: job.ID, Stage: "solver_started", Actor: "agent-manager-scorer", EvidenceRef: "run:wrong", ExpectedRevision: job.Revision})
	var conflict codingjobs.ErrConflict
	require.ErrorAs(t, err, &conflict, "scorer cannot start the solver phase")

	job, err = service.RecordStage(ctx, codingjobs.StageInput{ID: job.ID, Stage: "solver_started", Actor: "agent-manager-solver", EvidenceRef: "run:solver", ExpectedRevision: job.Revision})
	require.NoError(t, err)
	require.Equal(t, codingjobs.StateSolving, job.State)
	job, err = service.RecordStage(ctx, codingjobs.StageInput{ID: job.ID, Stage: "solver_completed", Actor: "agent-manager-solver", EvidenceRef: "run:solver", EvidenceSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ExpectedRevision: job.Revision})
	require.NoError(t, err)
	require.Equal(t, codingjobs.StateScoring, job.State)
	_, err = codingjobs.CandidateSealDigest(job)
	require.Error(t, err, "candidate source is not available until the durable job is reviewable")
	job, err = service.RecordStage(ctx, codingjobs.StageInput{ID: job.ID, Stage: "scoring_started", Actor: "agent-manager-scorer", EvidenceRef: "run:scorer", ExpectedRevision: job.Revision})
	require.NoError(t, err)
	job, err = service.RecordEvaluation(ctx, codingjobs.EvaluationInput{ID: job.ID, ReceiptRef: "test-genie:receipt-1", ReceiptSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Score: "strict_pass", ExpectedRevision: job.Revision})
	require.NoError(t, err)
	require.Equal(t, codingjobs.StateReviewable, job.State)
	require.Equal(t, "strict_pass", job.Score)
	require.Len(t, job.Events, 5)
	digest, err := codingjobs.CandidateSealDigest(job)
	require.NoError(t, err)
	require.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", digest)

	for name, mutate := range map[string]func(*codingjobs.Job){
		"wrong actor":          func(copy *codingjobs.Job) { copy.Events[2].Actor = "agent-manager-scorer" },
		"wrong run binding":    func(copy *codingjobs.Job) { copy.Events[2].EvidenceRef = "run:other" },
		"bad digest":           func(copy *codingjobs.Job) { copy.Events[2].EvidenceSHA256 = "not-a-digest" },
		"duplicate completion": func(copy *codingjobs.Job) { copy.Events = append(copy.Events, copy.Events[2]) },
	} {
		copy := job
		copy.Events = append([]codingjobs.Event(nil), job.Events...)
		mutate(&copy)
		if _, err := codingjobs.CandidateSealDigest(copy); err == nil {
			t.Errorf("%s job binding accepted", name)
		}
	}

	_, err = service.RecordEvaluation(ctx, codingjobs.EvaluationInput{ID: job.ID, ReceiptRef: "test-genie:receipt-2", ReceiptSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Score: "candidate_failed", ExpectedRevision: job.Revision})
	require.ErrorAs(t, err, &conflict, "the accepted evaluation is immutable")

	loaded, err := service.Get(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, job, loaded, "job and append-only event history survive repository readback")
}

func TestCodingJobEvaluationAcceptsOnlyClosedThreeWayOutcomes(t *testing.T) {
	service := newService(t)
	ctx := context.Background()
	job, err := service.Create(ctx, codingjobs.CreateInput{ReplayKey: "replay-3", CaseID: "case-2", Project: "org/repo", Title: "Fix", Task: "Repair", SourceSealRef: "seal:source-2"})
	require.NoError(t, err)
	job, err = service.RecordStage(ctx, codingjobs.StageInput{ID: job.ID, Stage: "solver_started", Actor: "agent-manager-solver", EvidenceRef: "run:solver", ExpectedRevision: job.Revision})
	require.NoError(t, err)
	job, err = service.RecordStage(ctx, codingjobs.StageInput{ID: job.ID, Stage: "solver_completed", Actor: "agent-manager-solver", EvidenceRef: "run:solver", EvidenceSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ExpectedRevision: job.Revision})
	require.NoError(t, err)
	job, err = service.RecordStage(ctx, codingjobs.StageInput{ID: job.ID, Stage: "scoring_started", Actor: "agent-manager-scorer", EvidenceRef: "run:scorer", ExpectedRevision: job.Revision})
	require.NoError(t, err)
	_, err = service.RecordEvaluation(ctx, codingjobs.EvaluationInput{ID: job.ID, ReceiptRef: "receipt", ReceiptSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Score: "pass", ExpectedRevision: job.Revision})
	var invalid codingjobs.ErrInvalid
	require.ErrorAs(t, err, &invalid, "a generic pass does not represent fail/pass/pass")
}

func TestCodingJobCASRejectsStaleRevision(t *testing.T) {
	service := newService(t)
	ctx := context.Background()
	job, err := service.Create(ctx, codingjobs.CreateInput{ReplayKey: "replay-4", CaseID: "case-3", Project: "org/repo", Title: "Fix", Task: "Repair", SourceSealRef: "seal:source-3"})
	require.NoError(t, err)
	first, err := service.RecordStage(ctx, codingjobs.StageInput{ID: job.ID, Stage: "solver_started", Actor: "agent-manager-solver", EvidenceRef: "run:solver", ExpectedRevision: job.Revision})
	require.NoError(t, err)
	_, err = service.RecordStage(ctx, codingjobs.StageInput{ID: job.ID, Stage: "solver_started", Actor: "agent-manager-solver", EvidenceRef: "run:solver", ExpectedRevision: job.Revision})
	var conflict codingjobs.ErrConflict
	require.ErrorAs(t, err, &conflict)
	require.Greater(t, first.Revision, job.Revision)
	require.False(t, errors.Is(err, context.Canceled))
}
