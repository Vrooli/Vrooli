package resources

import (
	"context"
	"testing"
	"time"
)

func TestManagedUpdateRequiredRollbackProbePrecedesTargetDispatch(t *testing.T) {
	u := updateTestInitial(t)
	u.RollbackQualificationRequired = true
	bad := updateTestClone(t, u)
	bad.Candidate.Intent = updateTestIntent("candidate", bad.Target)
	if _, err := transitionManagedUpdate(bad, UpdateLaunchCandidate, bad.UpdatedAt.Add(time.Second)); err == nil {
		t.Fatal("target dispatch bypassed required rollback qualification")
	}
	s := updateTestStore(t, u)
	if _, _, err := s.Create(context.Background(), u, false); err != nil {
		t.Fatal(err)
	}
	u.RollbackProbe.Intent = updateTestIntent("rollback-probe", u.Rollback)
	s.now = func() time.Time { return u.UpdatedAt.Add(time.Second) }
	var err error
	u, err = s.Commit(context.Background(), u, UpdateLaunchRollbackProbe)
	if err != nil {
		t.Fatal(err)
	}
	updateTestObserveStarted(&u.RollbackProbe, u.Rollback, u.UpdatedAt)
	u, err = s.Commit(context.Background(), u, UpdateRollbackProbeStarted)
	if err != nil {
		t.Fatal(err)
	}
	u.RollbackProbe.Proof = updateTestProof(*u.RollbackProbe.State, u.UpdatedAt)
	u, err = s.Commit(context.Background(), u, UpdateRollbackProbeVerified)
	if err != nil {
		t.Fatal(err)
	}
	if u.Phase != UpdatePrepared || u.RollbackQualifiedAt == nil {
		t.Fatal("qualification did not return to target preparation")
	}
	changed := updateTestClone(t, u)
	changed.RollbackProbe.Proof.ObservedAt = changed.UpdatedAt.Add(time.Second)
	changed.Candidate.Intent = updateTestIntent("candidate", changed.Target)
	if _, err := s.Commit(context.Background(), changed, UpdateLaunchCandidate); err == nil {
		t.Fatal("historical qualification proof was replaced")
	}
	u.Candidate.Intent = updateTestIntent("candidate", u.Target)
	u, err = s.Commit(context.Background(), u, UpdateLaunchCandidate)
	if err != nil {
		t.Fatal(err)
	}
	if u.Phase != UpdateCandidateStarting {
		t.Fatal("qualified rollback did not admit the target candidate")
	}
}

func TestManagedUpdateRollbackProbeFailurePreservesOriginal(t *testing.T) {
	u := updateTestInitial(t)
	u.RollbackQualificationRequired = true
	u.RollbackProbe.Intent = updateTestIntent("rollback-probe", u.Rollback)
	u, err := transitionManagedUpdate(u, UpdateLaunchRollbackProbe, u.UpdatedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	u.Error = "rollback qualification failed"
	u, err = transitionManagedUpdate(u, UpdateFailure, u.UpdatedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if u.Phase != UpdatePreserving || u.PrimaryStopVerified || u.Candidate.Intent != nil {
		t.Fatal("failed rollback qualification reached serving mutation")
	}
}
