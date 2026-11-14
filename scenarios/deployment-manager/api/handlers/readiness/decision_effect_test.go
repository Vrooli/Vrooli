package readinesshandler

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	internalreadiness "deployment-manager/internal/readiness"
	releasedecision "deployment-manager/internal/releasedecision"
	"deployment-manager/internal/testutil"
	domain "deployment-manager/readiness"

	"github.com/vrooli/api-core/operatordecision"
)

// approvableReview seeds a review whose mechanical + human evidence all pass and
// whose independent goal is closed, so approveReviewCore's revalidate succeeds.
// It returns the review and the correlation map the resolver reconstructs the
// identity from.
func approvableReview(t *testing.T, repo *internalreadiness.SQLRepository, now time.Time) (domain.Review, map[string]string) {
	t.Helper()
	policy := domain.DefaultChecklist()
	identity := domain.ReviewIdentity{Scenario: "demo", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: policy.Version}
	review := domain.Review{Identity: identity, ComparisonMode: domain.ComparisonFirstRelease}
	if _, err := repo.CreateOrGet(context.Background(), &review); err != nil {
		t.Fatal(err)
	}
	evidence := make([]domain.EvidenceItem, 0, len(policy.Items))
	for _, criterion := range policy.Items {
		status := domain.SignalPassed
		if criterion.HumanReview != nil {
			status = domain.SignalUnknown
		}
		evidence = append(evidence, domain.EvidenceItem{ReviewKey: review.Key, CriterionID: criterion.ID, Status: status, Applicability: "applicable", Producer: criterion.Owner, CandidateCommit: identity.CandidateCommit, ArtifactDigest: identity.ArtifactDigest, Target: "linux", Environment: identity.Channel, PolicyVersion: identity.PolicyVersion, ObservedAt: now, Reference: "evidence:" + criterion.ID})
	}
	if err := repo.ReplaceEvaluation(context.Background(), review.Key, evidence, nil, domain.ReviewAgentReview); err != nil {
		t.Fatal(err)
	}
	for _, criterion := range policy.Items {
		if criterion.HumanReview != nil {
			if err := repo.SaveHumanCheck(context.Background(), domain.HumanCheck{ReviewKey: review.Key, CriterionID: criterion.ID, Verdict: "passed", Actor: "reviewer", EvidenceReference: "journey:" + criterion.ID, ReviewedAt: now}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := repo.SetGoal(context.Background(), review.Key, "goal-1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordGoalClosure(context.Background(), review.Key, now); err != nil {
		t.Fatal(err)
	}
	correlation := map[string]string{
		"review_key":          review.Key,
		"scenario":            identity.Scenario,
		"profile_id":          identity.ProfileID,
		"candidate_commit":    identity.CandidateCommit,
		"artifact_digest":     identity.ArtifactDigest,
		"channel":             identity.Channel,
		"targets":             strings.Join(identity.Targets, ","),
		"authorization_epoch": "0",
		"policy_version":      strconv.Itoa(identity.PolicyVersion),
		"requesting_agent":    "run-agent-7",
	}
	return review, correlation
}

func newResolverFixture(t *testing.T, now time.Time) (*ReleaseDecisionResolver, *internalreadiness.SQLRepository, *releasedecision.SQLStore) {
	t.Helper()
	db := testutil.OpenSQLite(t)
	if _, err := db.Exec(internalreadiness.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(releasedecision.Schema()); err != nil {
		t.Fatal(err)
	}
	repo := internalreadiness.NewSQLRepository(db, "sqlite")
	handler := NewConnectHandler(nil, repo, nil)
	handler.now = func() time.Time { return now }
	store := releasedecision.NewSQLStore(db)
	resolver := NewReleaseDecisionResolver(handler, store, nil, nil)
	return resolver, repo, store
}

func seedPending(t *testing.T, store *releasedecision.SQLStore, correlation map[string]string, now time.Time) string {
	t.Helper()
	askID := operatordecision.CorrelationID("release", correlation["profile_id"], correlation["candidate_commit"], correlation["artifact_digest"], correlation["targets"])
	if err := store.Put(context.Background(), &operatordecision.PendingAsk{
		ID: askID, Subject: correlation["review_key"], State: operatordecision.PendingStatePending,
		RaisedAt: now, ExpiresAt: now.Add(24 * time.Hour), Extra: correlation,
	}); err != nil {
		t.Fatal(err)
	}
	return askID
}

// An approve resolution writes the review approval under the HUMAN subject
// (answered_by), NOT the requesting agent.
func TestResolverApproveRecordsHumanApproval(t *testing.T) {
	now := time.Now().UTC()
	resolver, repo, store := newResolverFixture(t, now)
	review, correlation := approvableReview(t, repo, now)
	seedPending(t, store, correlation, now)

	handled, err := resolver.Apply(context.Background(), operatordecision.DecisionResolution{
		State: operatordecision.StateAnswered, Answer: "approve", AnsweredBy: "operator-human",
		SourceEventType: operatordecision.DecisionRequestedEventType(ReleaseDecisionSource, ReleaseDecisionKind),
		Correlation:     correlation,
	})
	if err != nil || !handled {
		t.Fatalf("approve not handled: handled=%v err=%v", handled, err)
	}
	stored, _ := repo.Get(context.Background(), review.Key)
	if stored.Status != domain.ReviewApproved {
		t.Fatalf("review not approved, status=%q", stored.Status)
	}
	if stored.ApprovedBy != "operator-human" {
		t.Fatalf("approval must be attributed to the human, got %q", stored.ApprovedBy)
	}
	if stored.ApprovedBy == correlation["requesting_agent"] {
		t.Fatal("approval must NOT be attributed to the requesting agent")
	}
}

// A replay of an already-resolved ask is a benign no-op (single-use).
func TestResolverReplayIsNoOp(t *testing.T) {
	now := time.Now().UTC()
	resolver, repo, store := newResolverFixture(t, now)
	review, correlation := approvableReview(t, repo, now)
	askID := seedPending(t, store, correlation, now)

	res := operatordecision.DecisionResolution{State: operatordecision.StateAnswered, Answer: "approve", AnsweredBy: "operator-human", Correlation: correlation}
	if _, err := resolver.Apply(context.Background(), res); err != nil {
		t.Fatal(err)
	}
	// The ask is now resolved. Regress the evidence then replay: a single-use ask
	// must not re-approve, and because it is already resolved the replay is a
	// benign no-op regardless.
	ask, _ := store.Get(context.Background(), askID)
	if ask.State != operatordecision.PendingStateApproved {
		t.Fatalf("expected ask approved after first apply, got %q", ask.State)
	}
	handled, err := resolver.Apply(context.Background(), res)
	if err != nil || !handled {
		t.Fatalf("replay should be a benign handled no-op, handled=%v err=%v", handled, err)
	}
	stored, _ := repo.Get(context.Background(), review.Key)
	if stored.ApprovedBy != "operator-human" {
		t.Fatalf("replay must not change attribution, got %q", stored.ApprovedBy)
	}
}

// A correlation whose identity does not hash to the stored review key is refused
// and records no approval (artifact-binding).
func TestResolverRefusesArtifactMismatch(t *testing.T) {
	now := time.Now().UTC()
	resolver, repo, store := newResolverFixture(t, now)
	review, correlation := approvableReview(t, repo, now)
	// Seed the pending ask under the REAL identity so the resolver finds it (the
	// ask id is derived from profile/commit/digest/targets).
	seedPending(t, store, correlation, now)
	// The resolution tampers a field that is part of the review IDENTITY but not
	// the ask-locating correlation id (policy_version). The ask is still located,
	// but the recomputed review key no longer matches the stored review, so the
	// approval is refused — a resolution cannot be redirected to a different
	// artifact than the one that was reviewed.
	tampered := map[string]string{}
	for k, v := range correlation {
		tampered[k] = v
	}
	tampered["policy_version"] = strconv.Itoa(99999)

	handled, err := resolver.Apply(context.Background(), operatordecision.DecisionResolution{
		State: operatordecision.StateAnswered, Answer: "approve", AnsweredBy: "operator-human", Correlation: tampered,
	})
	if err != nil || !handled {
		t.Fatalf("mismatch should be handled (refused), handled=%v err=%v", handled, err)
	}
	stored, _ := repo.Get(context.Background(), review.Key)
	if stored.Status == domain.ReviewApproved {
		t.Fatal("an identity that does not match the stored review must NOT be approved")
	}
}

// A regressed-evidence approval is refused at the callback: revalidate fails so
// no approval is recorded.
func TestResolverRefusesRegressedEvidence(t *testing.T) {
	now := time.Now().UTC()
	resolver, repo, store := newResolverFixture(t, now)
	review, correlation := approvableReview(t, repo, now)
	seedPending(t, store, correlation, now)

	// Regress one mechanical signal to failed AFTER the ask was raised.
	evidence, _, err := repo.ListEvaluation(context.Background(), review.Key)
	if err != nil {
		t.Fatal(err)
	}
	for i := range evidence {
		if evidence[i].Status == domain.SignalPassed {
			evidence[i].Status = domain.SignalFailed
			break
		}
	}
	if err := repo.ReplaceEvaluation(context.Background(), review.Key, evidence, nil, domain.ReviewAgentReview); err != nil {
		t.Fatal(err)
	}

	handled, err := resolver.Apply(context.Background(), operatordecision.DecisionResolution{
		State: operatordecision.StateAnswered, Answer: "approve", AnsweredBy: "operator-human", Correlation: correlation,
	})
	if err != nil || !handled {
		t.Fatalf("regressed-evidence approval should be handled (refused), handled=%v err=%v", handled, err)
	}
	stored, _ := repo.Get(context.Background(), review.Key)
	if stored.Status == domain.ReviewApproved {
		t.Fatal("a regressed-evidence approval must be REFUSED at the callback")
	}
}

// A reject resolution ships nothing and records the refusal.
func TestResolverRejectShipsNothing(t *testing.T) {
	now := time.Now().UTC()
	resolver, repo, store := newResolverFixture(t, now)
	review, correlation := approvableReview(t, repo, now)
	askID := seedPending(t, store, correlation, now)

	handled, err := resolver.Apply(context.Background(), operatordecision.DecisionResolution{
		State: operatordecision.StateAnswered, Answer: "reject", AnsweredBy: "operator-human", Correlation: correlation,
	})
	if err != nil || !handled {
		t.Fatalf("reject not handled: handled=%v err=%v", handled, err)
	}
	stored, _ := repo.Get(context.Background(), review.Key)
	if stored.Status == domain.ReviewApproved {
		t.Fatal("a rejected decision must not approve the review")
	}
	ask, _ := store.Get(context.Background(), askID)
	if ask.State != operatordecision.PendingStateRejected {
		t.Fatalf("rejected ask state=%q", ask.State)
	}
}

// putFailingStore wraps a real store and fails the terminal Put (the one that
// flips the ask to resolved), while letting the initial Get and seed succeed.
type putFailingStore struct {
	inner     PendingResolver
	failOnPut bool
}

func (s *putFailingStore) Get(ctx context.Context, id string) (*operatordecision.PendingAsk, error) {
	return s.inner.Get(ctx, id)
}

func (s *putFailingStore) Put(ctx context.Context, ask *operatordecision.PendingAsk) error {
	if s.failOnPut {
		return errors.New("pending store write failed")
	}
	return s.inner.Put(ctx, ask)
}

// On the approve-SUCCESS path, a failed pending-store write must NOT be
// swallowed: the review is already approved, so if the ask is not flipped to
// resolved the single-use guard is defeated. The resolver surfaces the error
// (handled=false, err) so vrooli-events retries the delivery.
func TestResolverApproveSurfacesPendingStoreWriteFailure(t *testing.T) {
	now := time.Now().UTC()
	resolver, repo, store := newResolverFixture(t, now)
	review, correlation := approvableReview(t, repo, now)
	seedPending(t, store, correlation, now)

	failing := &putFailingStore{inner: store, failOnPut: true}
	resolver.pending = failing

	handled, err := resolver.Apply(context.Background(), operatordecision.DecisionResolution{
		State: operatordecision.StateAnswered, Answer: "approve", AnsweredBy: "operator-human",
		SourceEventType: operatordecision.DecisionRequestedEventType(ReleaseDecisionSource, ReleaseDecisionKind),
		Correlation:     correlation,
	})
	if err == nil {
		t.Fatal("a failed pending-store write on the approve path must surface an error, not be swallowed")
	}
	if handled {
		t.Fatal("a failed terminal write must report handled=false so the delivery is retried")
	}
	// The review was approved (the write that failed is the ask flip, not the
	// approval). On retry, the approved review makes approveReviewCore a safe no-op
	// and the Put can succeed.
	stored, _ := repo.Get(context.Background(), review.Key)
	if stored.Status != domain.ReviewApproved {
		t.Fatalf("review should be approved before the store write failed, got %q", stored.Status)
	}
}

// A resolution for an event type this effect does not own is skipped.
func TestResolverHandlesOnlyReleaseDecisions(t *testing.T) {
	resolver, _, _ := newResolverFixture(t, time.Now().UTC())
	if resolver.Handles("agent_manager.authorization.decision_requested.v1") {
		t.Fatal("resolver claimed a foreign event type")
	}
	if !resolver.Handles(operatordecision.DecisionRequestedEventType(ReleaseDecisionSource, ReleaseDecisionKind)) {
		t.Fatal("resolver did not claim its own event type")
	}
}
