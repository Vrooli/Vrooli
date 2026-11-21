package supervision

import (
	"context"
	"errors"
	"strings"
	"testing"

	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type fakeOutcomeLedger struct {
	err   error
	calls int
	ids   []string
}

func (f *fakeOutcomeLedger) AppendSupervisionOutcome(_ context.Context, outcome SupervisionOutcome) (string, error) {
	f.calls++
	f.ids = append(f.ids, outcome.ID)
	return "ledger-1", f.err
}

func policyFixture(version string) SupervisionPolicy {
	return SupervisionPolicy{Version: version, EventCount: 3, QuietSeconds: 30, FrictionThreshold: .8, Terminal: true, AllowedActions: []string{"escalate", "observe", "park", "wake_parent"}, ClassifierRevision: "classifier-v1"}
}

func TestCandidateFreezesResolvedArtifactBeforeCollectingHoldout(t *testing.T) {
	repo, db := testRepository(t)
	store := NewPolicyStore(db, nil)
	service := NewService(repo, &cohortSource{})
	service.SetPolicyStore(store)
	req := &domainpb.CreateSupervisionPolicyCandidateRequest{Policy: policyRecordProto(PolicyRecord{Policy: policyFixture("frozen")}).Policy, CreatedBy: "reviewer"}
	if _, err := service.CreatePolicyCandidate(context.Background(), req); err == nil {
		t.Fatal("candidate created without a resolvable artifact")
	}
	digest := strings.Repeat("a", 64)
	service.SetPolicyArtifactResolver(func(context.Context) (string, error) { return digest, nil })
	record, err := service.CreatePolicyCandidate(context.Background(), req)
	if err != nil || record.GetPolicy().GetEvaluatorDigest() != digest {
		t.Fatalf("artifact was not frozen: %v %v", record, err)
	}
	digest = strings.Repeat("b", 64)
	if _, err := service.CreatePolicyCandidate(context.Background(), req); err == nil {
		t.Fatal("candidate version silently adopted changed evaluator content")
	}
	req.Policy.Version = "mismatched"
	req.Policy.EvaluatorDigest = strings.Repeat("c", 64)
	if _, err := service.CreatePolicyCandidate(context.Background(), req); err == nil {
		t.Fatal("caller-supplied mismatched artifact was accepted")
	}
}

func TestPolicyVersionsAreImmutableAndOutcomeLedgerFailureDegradesSafely(t *testing.T) {
	repo, db := testRepository(t)
	ledger := &fakeOutcomeLedger{err: errors.New("source ledger unavailable")}
	store := NewPolicyStore(db, ledger)
	store.now = repo.now
	created, err := store.CreateCandidate(context.Background(), policyFixture("policy-v1"), "", "reviewer-a")
	if err != nil || created.State != "candidate" || created.Digest == "" {
		t.Fatalf("candidate=%+v err=%v", created, err)
	}
	changed := policyFixture("policy-v1")
	changed.EventCount = 4
	if _, err := store.CreateCandidate(context.Background(), changed, "", "reviewer-a"); err == nil {
		t.Fatal("existing policy version was mutated")
	}
	outcome := labelledDecision(t, repo, store, "policy-v1", "quiet")
	written, err := store.RecordOutcome(context.Background(), outcome)
	if err != nil || written.LedgerError == nil || written.Outcome.ID == "" || ledger.calls != 1 {
		t.Fatalf("outcome=%+v calls=%d err=%v", written, ledger.calls, err)
	}
	replayed, err := store.RecordOutcome(context.Background(), outcome)
	if err != nil || !replayed.Reused || ledger.calls != 2 || replayed.Outcome.ID != written.Outcome.ID || ledger.ids[0] != ledger.ids[1] {
		t.Fatalf("dedupe=%+v calls=%d err=%v", replayed, ledger.calls, err)
	}
}

func TestNewWatchBindsActivePolicyOnceAndEmergencyDisablePreservesCursor(t *testing.T) {
	repo, db := testRepository(t)
	store := NewPolicyStore(db, nil)
	store.now = repo.now
	active, err := store.EnsureInitialActive(context.Background(), DefaultSupervisionPolicy(), "bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, &cohortSource{retention: eventlog.RetentionState{Generation: 1}})
	service.SetPolicyStore(store)
	watch, _, err := service.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(uuid.New()), IdempotencyKey: "policy-bound"})
	if err != nil || watch.GetSpec().GetPolicyVersion() != active.Policy.Version {
		t.Fatalf("watch=%+v err=%v", watch, err)
	}
	if err := store.SetDisabled(context.Background(), true, "operator stop", "operator"); err != nil {
		t.Fatal(err)
	}
	called := false
	evaluator := PolicyControlledEvaluator{Store: store, Delegate: evaluatorFunc(func(context.Context, EvaluationInput) (*domainpb.WatchDecision, error) {
		called = true
		return nil, nil
	})}
	decision, err := evaluator.Evaluate(context.Background(), EvaluationInput{Watch: watch, Now: repo.now()})
	if err != nil || called || decision.GetDisposition() != domainpb.WatchDisposition_WATCH_DISPOSITION_UNAVAILABLE || !strings.Contains(decision.GetClassification(), "supervision_disabled") {
		t.Fatalf("decision=%+v called=%v err=%v", decision, called, err)
	}
}
