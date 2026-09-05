package supervision

// [REQ:REQ-P2-011]

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/eventlog"
	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Each case is a real persisted watch/decision/input, rather than a disconnected
// outcome label that could manufacture a passing gate.
func labelledDecision(t *testing.T, repo *Repository, store *PolicyStore, version, observed string) SupervisionOutcome {
	t.Helper()
	ctx := context.Background()
	spec := validServiceSpec(uuid.New())
	spec.PolicyVersion = version
	watch, before, _, err := repo.Create(ctx, spec, uuid.NewString(), 1)
	if err != nil {
		t.Fatal(err)
	}
	after := before
	after.Token = uuid.NewString()
	decision := &domainpb.WatchDecision{IdempotencyKey: uuid.NewString(), Disposition: domainpb.WatchDisposition_WATCH_DISPOSITION_QUIET, Classification: "quiet", EvidenceIds: []string{"event-proof"}}
	input := EvaluationInput{Watch: watch, Now: repo.now(), ProposedCursor: after.Token, Subjects: []SubjectSummary{{RunID: spec.Subjects[0].RunId, Status: "running"}}, Events: []eventlog.CohortEvent{{ID: uuid.New(), Data: []byte(`{"private":"must-not-persist"}`)}}}
	if observed == "stalled" {
		input.Subjects[0].FrictionScore = .9
	}
	saved, err := repo.CommitDecision(ctx, watch.WatchId, watch.Revision, before, decision, after, input)
	if err != nil {
		t.Fatal(err)
	}
	outcomes, err := store.ListOutcomes(ctx, version, 500)
	if err != nil {
		t.Fatal(err)
	}
	var prior string
	for _, o := range outcomes {
		if o.DecisionID == saved.LastDecision.DecisionId {
			prior = o.ID
		}
	}
	if prior == "" {
		t.Fatal("decision must capture an unassessed outcome")
	}
	var raw string
	if err := store.db.QueryRowContext(ctx, `SELECT input_json FROM supervision_evaluation_inputs WHERE decision_id=?`, saved.LastDecision.DecisionId).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "must-not-persist") {
		t.Fatal("raw event data leaked into replay")
	}
	return SupervisionOutcome{IdempotencyKey: uuid.NewString(), PolicyVersion: version, FamilyExecutionID: spec.FamilyExecutionId, WatchID: watch.WatchId, DecisionID: saved.LastDecision.DecisionId, ChildRunID: spec.Subjects[0].RunId, EvidenceIDs: []string{"event-proof"}, PredictedClass: "quiet", ObservedClass: observed, CompletionImpact: .1, CompletionImpactObserved: true, Supersedes: prior}
}

func TestReplayRunsCandidateAndRolloutCannotBeClaimed(t *testing.T) {
	repo, db := testRepository(t)
	store := NewPolicyStore(db, nil)
	store.now = repo.now
	ctx := context.Background()
	_, err := store.CreateCandidate(ctx, policyFixture("candidate"), "", "author")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	store.SetReplayEvaluator(evaluatorFunc(func(ctx context.Context, in EvaluationInput) (*domainpb.WatchDecision, error) {
		calls++
		if in.Watch.Spec.PolicyVersion != "candidate" {
			t.Fatal("candidate policy was not replayed")
		}
		if err := store.BindEvaluator(ctx, "candidate", strings.Repeat("a", 64)); err != nil {
			return nil, err
		}
		klass := "quiet"
		if in.Subjects[0].FrictionScore > .7 {
			klass = "stalled"
		}
		return &domainpb.WatchDecision{Disposition: domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL, Classification: klass, RecommendedAction: domainpb.WatchActionKind_WATCH_ACTION_KIND_OBSERVE}, nil
	}))
	for i := 0; i < 20; i++ {
		klass := "quiet"
		if i%2 == 0 {
			klass = "stalled"
		}
		o := labelledDecision(t, repo, store, "candidate", klass)
		if _, err := store.RecordOutcome(ctx, o); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.EvaluateCandidate(ctx, "candidate", 1000, ReplayThresholds{}); err == nil {
		t.Fatal("caller rollout count was accepted")
	}
	report, err := store.EvaluateCandidate(ctx, "candidate", 0, ReplayThresholds{})
	if err != nil || !report.ReplayPassed || !report.RolloutPassed || calls != 20 {
		t.Fatalf("report=%+v calls=%d err=%v", report, calls, err)
	}
	if _, err := store.Promote(ctx, "candidate", "reviewer"); err != nil {
		t.Fatal(err)
	}
}

func TestReplayDetectsCandidateRegressionDespiteFavorableStoredPredictions(t *testing.T) {
	repo, db := testRepository(t)
	store := NewPolicyStore(db, nil)
	store.now = repo.now
	ctx := context.Background()
	_, _ = store.CreateCandidate(ctx, policyFixture("candidate"), "", "author")
	for i := 0; i < 20; i++ {
		klass := "quiet"
		if i%2 == 0 {
			klass = "stalled"
		}
		o := labelledDecision(t, repo, store, "candidate", klass)
		if _, err := store.RecordOutcome(ctx, o); err != nil {
			t.Fatal(err)
		}
	}
	store.SetReplayEvaluator(evaluatorFunc(func(ctx context.Context, in EvaluationInput) (*domainpb.WatchDecision, error) {
		_ = store.BindEvaluator(ctx, "candidate", strings.Repeat("a", 64))
		return &domainpb.WatchDecision{Disposition: domainpb.WatchDisposition_WATCH_DISPOSITION_QUIET, Classification: "quiet"}, nil
	}))
	report, err := store.EvaluateCandidate(ctx, "candidate", 0, ReplayThresholds{})
	if err != nil || report.ReplayPassed || report.FalseNegatives != 10 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if _, err := store.Promote(ctx, "candidate", "reviewer"); err == nil {
		t.Fatal("regressing candidate promoted")
	}
}

func TestUnknownAndDisconnectedOutcomesCannotProveImprovement(t *testing.T) {
	repo, db := testRepository(t)
	store := NewPolicyStore(db, nil)
	store.now = repo.now
	ctx := context.Background()
	_, _ = store.CreateCandidate(ctx, policyFixture("candidate"), "", "author")
	o := labelledDecision(t, repo, store, "candidate", "quiet")
	bad := o
	bad.ChildRunID = uuid.NewString()
	if _, err := store.RecordOutcome(ctx, bad); err == nil {
		t.Fatal("disconnected child accepted")
	}
	o.ObservedClass = ""
	if _, err := store.RecordOutcome(ctx, o); err != nil {
		t.Fatal(err)
	}
	store.SetReplayEvaluator(evaluatorFunc(func(context.Context, EvaluationInput) (*domainpb.WatchDecision, error) {
		t.Fatal("unknown label replayed")
		return nil, nil
	}))
	r, err := store.EvaluateCandidate(ctx, "candidate", 0, ReplayThresholds{})
	if err != nil || r.SampleCount != 0 || r.RolloutSamples != 0 || r.ReplayPassed || r.RolloutPassed {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestEvaluatorArtifactCannotChangeAndExpiryRemovesEvidence(t *testing.T) {
	repo, db := testRepository(t)
	store := NewPolicyStore(db, nil)
	store.now = repo.now
	ctx := context.Background()
	_, _ = store.CreateCandidate(ctx, policyFixture("candidate"), "", "author")
	if err := store.BindEvaluator(ctx, "candidate", strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if err := store.BindEvaluator(ctx, "candidate", strings.Repeat("b", 64)); err == nil {
		t.Fatal("policy artifact changed")
	}
	o := labelledDecision(t, repo, store, "candidate", "stalled")
	o.ExpiresAt = repo.now().Add(time.Minute)
	if _, err := store.RecordOutcome(ctx, o); err != nil {
		t.Fatal(err)
	}
	store.SetReplayEvaluator(evaluatorFunc(func(context.Context, EvaluationInput) (*domainpb.WatchDecision, error) {
		return &domainpb.WatchDecision{Disposition: domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL, Classification: "stalled"}, nil
	}))
	if _, err := store.EvaluateCandidate(ctx, "candidate", 0, ReplayThresholds{}); err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return repo.now().Add(2 * time.Minute) }
	if _, err := store.PruneExpired(ctx); err != nil {
		t.Fatal(fmt.Errorf("prune: %w", err))
	}
}

func TestFamilyCannotSwitchPolicyAcrossWatches(t *testing.T) {
	repo, _ := testRepository(t)
	ctx := context.Background()
	spec := validServiceSpec(uuid.New())
	spec.PolicyVersion = "first"
	if _, _, _, err := repo.Create(ctx, spec, "first-watch", 1); err != nil {
		t.Fatal(err)
	}
	spec.PolicyVersion = "second"
	if _, _, _, err := repo.Create(ctx, spec, "second-watch", 1); err == nil {
		t.Fatal("family silently changed its policy")
	}
}
