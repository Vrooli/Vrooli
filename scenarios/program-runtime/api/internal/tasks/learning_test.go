package tasks

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDelayedFeedbackPersistsAndQuarantinesExactCandidate(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "learning.db")
	s := NewStore(taskDB(t, path))
	f, err := s.PutFragment(ctx, Fragment{StepKey: "example", Fragment: "def step(): return 1"}, 1, 0)
	require.NoError(t, err)
	other, err := s.PutFragment(ctx, Fragment{StepKey: "example", Fragment: "def step(): return 2"}, 1, 0)
	require.NoError(t, err)
	ref := "prt_feedback_v1_" + strings.Repeat("a", 43)
	r := LearningResult{FeedbackRef: ref, AttemptID: "attempt", Scope: "bas-usage", Provenance: "test", StepKey: f.StepKey, FragmentHash: f.FragmentHash}
	require.NoError(t, s.PutLearningResult(ctx, r))
	require.NoError(t, s.PutLearningResult(ctx, r))
	changed := r
	changed.FragmentHash = other.FragmentHash
	require.Error(t, s.PutLearningResult(ctx, changed))
	feedback := Feedback{ObservationID: "later-correction", FeedbackRef: ref, Disposition: "contradicted", Dimension: "verification", Evidence: []string{"test://downstream"}}
	_, err = s.ApplyFeedback(ctx, feedback, "agent")
	require.Error(t, err)
	finding, err := s.ApplyFeedback(ctx, feedback, "test")
	require.NoError(t, err)
	require.Equal(t, "browser-automation-studio", finding.Owner)
	_, err = s.ApplyFeedback(ctx, feedback, "test")
	require.NoError(t, err)
	reopened := NewStore(taskDB(t, path))
	got, err := reopened.GetFragment(ctx, "example")
	require.NoError(t, err)
	require.Equal(t, other.FragmentHash, got.FragmentHash)
	bad, err := reopened.fragmentByHash(ctx, f.StepKey, f.FragmentHash)
	require.NoError(t, err)
	require.Equal(t, 1, bad.ContradictedSinceEdit)
	rejected, err := reopened.RejectedFragmentHashes(ctx, "example")
	require.NoError(t, err)
	require.Equal(t, []string{f.FragmentHash}, rejected)
	findings, err := reopened.ListFindings(ctx, "browser-automation-studio")
	require.NoError(t, err)
	require.Len(t, findings, 1)
}

func TestContextFeedbackDoesNotCondemnCodeAndClaimsRequireReceipt(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "learning.db")))
	f, err := s.PutFragment(ctx, Fragment{StepKey: "example", Fragment: "def step(): return 1"}, 1, 0)
	require.NoError(t, err)
	ref := "prt_feedback_v1_" + strings.Repeat("b", 43)
	require.NoError(t, s.PutLearningResult(ctx, LearningResult{FeedbackRef: ref, AttemptID: "attempt", Scope: "bas-usage", Provenance: "test", StepKey: f.StepKey, FragmentHash: f.FragmentHash}))
	finding, err := s.ApplyFeedback(ctx, Feedback{ObservationID: "context", FeedbackRef: ref, Disposition: "contradicted", Dimension: "context", Evidence: []string{"test://changed-requirement"}}, "test")
	require.NoError(t, err)
	_, err = s.GetFragment(ctx, "example")
	require.NoError(t, err)
	ev := []string{"test://work-receipt"}
	_, err = s.TransitionFinding(ctx, finding.ID, "observed", "routed", "wrong-owner", "", ev)
	require.Error(t, err)
	_, err = s.TransitionFinding(ctx, finding.ID, "observed", "routed", finding.Owner, "", ev)
	require.NoError(t, err)
	claim, err := s.TransitionFinding(ctx, finding.ID, "routed", "claimed", finding.Owner, "", ev)
	require.NoError(t, err)
	require.NotEmpty(t, claim.ClaimRef)
	_, err = s.TransitionFinding(ctx, finding.ID, "claimed", "repaired", finding.Owner, "wrong", ev)
	require.Error(t, err)
	repaired, err := s.TransitionFinding(ctx, finding.ID, "claimed", "repaired", finding.Owner, claim.ClaimRef, ev)
	require.NoError(t, err)
	require.Len(t, repaired.History, 3)
	listed, err := s.ListFindings(ctx, "")
	require.NoError(t, err)
	require.Empty(t, listed[0].ClaimRef)
}

func TestEligibleCandidateNotHiddenByStaleOrNarrowWinner(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "learning.db")))
	seed := func(code string, n int) *Fragment {
		var f *Fragment
		var err error
		for i := 0; i < n; i++ {
			f, err = s.PutFragment(ctx, Fragment{StepKey: "fragment-v2:example", Fragment: code, AttemptID: string(rune('a' + i)), InputDigest: string(rune('a' + i)), Compatibility: map[string]any{"version": "1"}, Evidence: []string{"test://verified"}}, 1, 0)
			require.NoError(t, err)
		}
		return f
	}
	stale := seed("def step(): return 1", 5)
	fresh := seed("def step(): return 2", 3)
	_, err := s.db.Exec("UPDATE learning_fragment_evidence SET created_at='2000-01-01T00:00:00Z' WHERE fragment_hash=?", stale.FragmentHash)
	require.NoError(t, err)
	got, err := s.EligibleFragment(ctx, "fragment-v2:example", 3, 2, 30)
	require.NoError(t, err)
	require.Equal(t, fresh.FragmentHash, got.FragmentHash)
	_, err = s.EligibleFragment(ctx, "fragment-v2:example", 4, 2, 30)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestBaselineFeedbackIsRejectedBeforeFirstFragmentPersistence(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "learning.db")))
	ref := "prt_feedback_v1_" + strings.Repeat("c", 43)
	require.NoError(t, s.PutLearningResult(ctx, LearningResult{FeedbackRef: ref, AttemptID: "attempt", Provenance: "test", StepKey: "baseline", FragmentHash: "sha256:bad"}))
	_, err := s.ApplyFeedback(ctx, Feedback{ObservationID: "late", FeedbackRef: ref, Disposition: "contradicted", Dimension: "execution", Evidence: []string{"test://bad"}}, "test")
	require.NoError(t, err)
	hashes, err := s.RejectedFragmentHashes(ctx, "baseline")
	require.NoError(t, err)
	require.Equal(t, []string{"sha256:bad"}, hashes)
}

func TestArtifactCorrectionsRouteWithoutTaskAgentAndDoNotLeakClaims(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "learning.db")))
	artifact := map[string]any{"kind": "inference", "owner": "fixture", "id": "answer", "revision": "v1"}
	ref := "prt_feedback_v1_" + strings.Repeat("d", 43)
	require.NoError(t, s.PutLearningResult(ctx, LearningResult{FeedbackRef: ref, AttemptID: "attempt", StepAttemptID: "child", Scope: "fixture-usage", Provenance: "test", Artifact: artifact}))
	before, err := s.LearningResultStatus(ctx, "", artifact, "test")
	require.NoError(t, err)
	require.Equal(t, true, before["eligible"])
	feedback := Feedback{ObservationID: "useful", FeedbackRef: ref, Disposition: "contradicted", Dimension: "usefulness", Evidence: []string{"test://later-user-feedback"}}
	first, err := s.ApplyFeedback(ctx, feedback, "test")
	require.NoError(t, err)
	after, err := s.LearningResultStatus(ctx, "", artifact, "test")
	require.NoError(t, err)
	require.Equal(t, false, after["eligible"])
	require.Equal(t, "child", after["step_attempt_id"])
	require.NoError(t, (&Drainer{Store: s}).DrainOnce(ctx, time.Now()))
	again, err := s.ApplyFeedback(ctx, feedback, "test")
	require.NoError(t, err)
	require.Equal(t, first.ID, again.ID)
	require.Equal(t, "routed", again.State)
	findings, err := s.ListFindings(ctx, "fixture")
	require.NoError(t, err)
	require.Len(t, findings, 1)
	require.Equal(t, "routed", findings[0].State)
}

func TestFeedbackProjectionSurvivesRestartAndUsesOriginalScope(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "learning.db")
	s := NewStore(taskDB(t, path))
	root := seedRecord()
	root.Scope = "original-usage"
	_, _, err := s.Begin(ctx, root)
	require.NoError(t, err)
	_, err = s.Update(ctx, root.AttemptID, func(r *Record) error {
		r.State = "completed"
		r.FinishInputs = map[string]any{"scope": root.Scope, "attempt": map[string]any{"attempt_id": root.AttemptID}}
		return nil
	})
	require.NoError(t, err)
	ref := "prt_feedback_v1_" + strings.Repeat("e", 43)
	require.NoError(t, s.PutLearningResult(ctx, LearningResult{FeedbackRef: ref, AttemptID: root.AttemptID, StepAttemptID: "child", Scope: root.Scope, Provenance: "agent"}))
	_, err = s.ApplyFeedback(ctx, Feedback{ObservationID: "delayed", FeedbackRef: ref, Disposition: "contradicted", Dimension: "verification", Evidence: []string{"test://later"}}, "operator")
	require.NoError(t, err)
	status, err := s.LearningResultStatus(ctx, ref, nil, "operator")
	require.NoError(t, err)
	require.Equal(t, true, status["contradicted"])
	require.NoError(t, s.DrainFeedback(ctx, time.Now(), func(context.Context, Record) (map[string]any, error) { return nil, errors.New("offline") }))
	restarted := NewStore(taskDB(t, path))
	calls := 0
	require.NoError(t, restarted.DrainFeedback(ctx, time.Now().Add(time.Hour), func(_ context.Context, r Record) (map[string]any, error) {
		calls++
		require.Equal(t, root.Scope, r.FinishInputs["scope"])
		obs := r.FinishInputs["observations"].([]any)[0].(map[string]any)
		require.Equal(t, "child", obs["attempt_id"])
		return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete"}}, nil
	}))
	require.NoError(t, restarted.DrainFeedback(ctx, time.Now().Add(2*time.Hour), func(context.Context, Record) (map[string]any, error) {
		t.Fatal("acknowledged feedback must not be redelivered")
		return nil, nil
	}))
	require.Equal(t, 1, calls)
}

func TestOlderContradictionCannotBeHiddenByBoundedDetails(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "learning.db")))
	ref := "prt_feedback_v1_" + strings.Repeat("f", 43)
	require.NoError(t, s.PutLearningResult(ctx, LearningResult{FeedbackRef: ref, AttemptID: "attempt", Provenance: "agent"}))
	for i := 0; i < 103; i++ {
		disposition := "supported"
		if i == 0 {
			disposition = "contradicted"
		}
		_, err := s.ApplyFeedback(ctx, Feedback{ObservationID: fmt.Sprintf("observation-%d", i), FeedbackRef: ref, Disposition: disposition, Dimension: "verification", Evidence: []string{"test://later"}}, "agent")
		require.NoError(t, err)
	}
	status, err := s.LearningResultStatus(ctx, ref, nil, "operator")
	require.NoError(t, err)
	require.Equal(t, true, status["contradicted"])
	require.Equal(t, true, status["truncated"])
	isolated, err := s.LearningResultStatus(ctx, ref, nil, "test")
	require.NoError(t, err)
	require.Equal(t, false, isolated["found"])
}

func TestClaimLostResponseRetryAndExpiredClaimRecovery(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "learning.db")))
	ref := "prt_feedback_v1_" + strings.Repeat("g", 43)
	require.NoError(t, s.PutLearningResult(ctx, LearningResult{FeedbackRef: ref, AttemptID: "attempt", Scope: "fixture-usage", Provenance: "test"}))
	f, err := s.ApplyFeedback(ctx, Feedback{ObservationID: "claim", FeedbackRef: ref, Disposition: "contradicted", Dimension: "execution", Evidence: []string{"test://failure"}}, "test")
	require.NoError(t, err)
	require.NoError(t, s.RouteFindings(ctx))
	claimRef := "prt_claim_v1_" + strings.Repeat("a", 64)
	evidence := []string{"test://worker-one"}
	first, err := s.TransitionFinding(ctx, f.ID, "routed", "claimed", f.Owner, claimRef, evidence)
	require.NoError(t, err)
	retry, err := s.TransitionFinding(ctx, f.ID, "routed", "claimed", f.Owner, claimRef, evidence)
	require.NoError(t, err)
	require.Equal(t, first.History, retry.History)
	other := "prt_claim_v1_" + strings.Repeat("b", 64)
	_, err = s.TransitionFinding(ctx, f.ID, "claimed", "claimed", f.Owner, other, []string{"test://worker-two"})
	require.Error(t, err)
	first.ClaimExpiresAt = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano)
	body, err := json.Marshal(first)
	require.NoError(t, err)
	_, err = s.db.Exec("UPDATE learning_findings SET record=? WHERE finding_id=?", string(body), f.ID)
	require.NoError(t, err)
	recovered, err := s.TransitionFinding(ctx, f.ID, "claimed", "claimed", f.Owner, other, []string{"test://worker-two"})
	require.NoError(t, err)
	require.Equal(t, other, recovered.ClaimRef)
	_, err = s.TransitionFinding(ctx, f.ID, "claimed", "repaired", f.Owner, claimRef, evidence)
	require.Error(t, err)
	_, err = s.TransitionFinding(ctx, f.ID, "claimed", "repaired", f.Owner, other, evidence)
	require.NoError(t, err)
	_, err = s.TransitionFinding(ctx, f.ID, "claimed", "repaired", f.Owner, other, evidence)
	require.NoError(t, err)
	_, err = s.TransitionFinding(ctx, f.ID, "claimed", "repaired", f.Owner, other, []string{"different"})
	require.Error(t, err)
}

func TestArtifactFindingsDeduplicateAndReopenMeasuredRecurrence(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "learning.db")))
	artifact := map[string]any{"kind": "flow", "owner": "fixture", "id": "flow", "revision": "1"}
	makeFeedback := func(char string) *Finding {
		ref := "prt_feedback_v1_" + strings.Repeat(char, 43)
		require.NoError(t, s.PutLearningResult(ctx, LearningResult{FeedbackRef: ref, AttemptID: "attempt-" + char, Scope: "fixture-usage", Artifact: artifact, Provenance: "test"}))
		f, err := s.ApplyFeedback(ctx, Feedback{ObservationID: "obs-" + char, FeedbackRef: ref, Disposition: "contradicted", Dimension: "verification", Evidence: []string{"test://failure-" + char}}, "test")
		require.NoError(t, err)
		return f
	}
	first := makeFeedback("a")
	second := makeFeedback("b")
	require.Equal(t, first.ID, second.ID)
	claim := ""
	state := "observed"
	for _, next := range []string{"routed", "claimed", "repaired", "validated", "measured"} {
		f, err := s.TransitionFinding(ctx, first.ID, state, next, first.Owner, claim, []string{"test://" + next})
		require.NoError(t, err)
		claim = f.ClaimRef
		state = next
	}
	rows, err := s.ListFindings(ctx, "")
	require.NoError(t, err)
	require.Empty(t, rows)
	recurrence := makeFeedback("c")
	require.Equal(t, first.ID, recurrence.ID)
	require.Equal(t, "observed", recurrence.State)
	require.Len(t, recurrence.History, 6)
}

func TestFragmentCohortCannotBeForgedByReusingLiveKey(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "learning.db")))
	require.NoError(t, s.ClaimFragmentCohort(ctx, "live-key", "agent"))
	require.NoError(t, s.ClaimFragmentCohort(ctx, "live-key", "operator"))
	require.Error(t, s.ClaimFragmentCohort(ctx, "live-key", "test"))
	require.Error(t, s.ClaimFragmentCohort(ctx, "live-key", "replay"))
	require.NoError(t, s.ClaimFragmentCohort(ctx, "test-key", "test"))
	require.Error(t, s.ClaimFragmentCohort(ctx, "test-key", "agent"))
}

func TestInstallingOptionalMemoryPinsPreviouslyUnpinnedCaptureOnce(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "learning.db")))
	r := seedRecord()
	r.FinishDigest = ""
	_, _, err := s.Begin(ctx, r)
	require.NoError(t, err)
	_, err = s.Update(ctx, r.AttemptID, func(r *Record) error {
		r.State = "completed"
		r.Delivery = "blocked"
		r.LastError = "memory_not_installed"
		r.FinishInputs = map[string]any{"scope": r.Scope, "attempt": map[string]any{"attempt_id": r.AttemptID}}
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, s.RequeueUnpinned(ctx, "installed-digest"))
	stored, err := s.Get(ctx, r.AttemptID)
	require.NoError(t, err)
	require.Equal(t, "pending", stored.Delivery)
	require.Equal(t, "installed-digest", stored.FinishDigest)
	require.NoError(t, s.RequeueUnpinned(ctx, "newer-digest"))
	stored, err = s.Get(ctx, r.AttemptID)
	require.NoError(t, err)
	require.Equal(t, "installed-digest", stored.FinishDigest)
}

func TestNegativeEvidencePrecedesFragmentWriteWithoutLosingQuarantine(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "learning.db")))
	code := "def step(): return 1"
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(code)))
	ref := "prt_feedback_v1_" + strings.Repeat("h", 43)
	require.NoError(t, s.PutLearningResult(ctx, LearningResult{FeedbackRef: ref, AttemptID: "attempt", Provenance: "test", StepKey: "out-of-order", FragmentHash: hash}))
	_, err := s.ApplyFeedback(ctx, Feedback{ObservationID: "before-positive", FeedbackRef: ref, Disposition: "contradicted", Dimension: "verification", Evidence: []string{"test://later-result"}}, "test")
	require.NoError(t, err)
	_, err = s.PutFragment(ctx, Fragment{StepKey: "out-of-order", Fragment: code}, 1, 0)
	require.NoError(t, err)
	_, err = s.GetFragment(ctx, "out-of-order")
	require.ErrorIs(t, err, sql.ErrNoRows)
	rejected, err := s.FragmentRejected(ctx, "out-of-order", hash)
	require.NoError(t, err)
	require.True(t, rejected)
}
