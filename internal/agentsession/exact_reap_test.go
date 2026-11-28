package agentsession

import (
	"context"
	"errors"
	"testing"
	"time"
)

type exactFacts struct {
	path    string
	members []int
	facts   map[int]ProcessFact
	names   map[int]string
	err     error
}

func (f exactFacts) ScopeCgroupPath(string) (string, error) { return f.path, f.err }
func (f exactFacts) ScopeMembers(string) ([]int, error) {
	return append([]int(nil), f.members...), f.err
}
func (f exactFacts) Process(pid int) (ProcessFact, error) { return f.facts[pid], f.err }
func (f exactFacts) ProcessName(pid int) (string, error)  { return f.names[pid], f.err }

type exactApprovalVerifier struct {
	approval ExactReapApproval
	err      error
}

func (v exactApprovalVerifier) VerifyExactApproval(_ context.Context, proof ExactReapProof) (ExactReapApproval, error) {
	if v.err != nil {
		return ExactReapApproval{}, v.err
	}
	if proof.Digest != v.approval.ProofDigest || ExactReapAuthorizationTarget(proof) != v.approval.Target {
		return ExactReapApproval{}, errors.New("unexpected proof")
	}
	return v.approval, nil
}

type exactStopper struct{ calls int }

func (s *exactStopper) Stop(context.Context, Session) error { s.calls++; return nil }

func testExactSession() Session {
	return Session{
		SessionID: "(no lease)", Scope: "vrooli-agent-old.scope", ScopeHeld: true,
		Owner:    Owner{Kind: KindWebConsoleSession, Ref: "web-session-1", Source: SourceInferred, Evidence: Evidence{WebConsoleSessionID: "web-session-1"}},
		Liveness: Liveness{State: OwnerGone, Detail: "archived"}, State: StateOrphaned, Frozen: "no",
	}
}

func testExactFacts() exactFacts {
	return exactFacts{
		path:    "/user.slice/user-1000.slice/vrooli.slice/vrooli-agents.slice/vrooli-agent-old.scope",
		members: []int{42, 19}, facts: map[int]ProcessFact{19: {PID: 19, StartTime: "100"}, 42: {PID: 42, StartTime: "200"}},
		names: map[int]string{19: "mock-api", 42: "node"},
	}
}

func TestCaptureExactReapProofRequiresExactHeldOrphanAndReadableContainment(t *testing.T) {
	session, facts := testExactSession(), testExactFacts()
	proof, err := CaptureExactReapProof(session, session.Scope, facts)
	if err != nil || proof.Digest == "" || proof.Target != session.Scope || proof.Processes[0].PID != 19 || proof.Processes[1].PID != 42 {
		t.Fatalf("capture proof = %+v, %v", proof, err)
	}
	if _, err := CaptureExactReapProof(session, "vrooli-agent-old", facts); err == nil {
		t.Fatal("accepted scope without exact unit suffix")
	}
	wrongSlice := facts
	wrongSlice.path = "/user.slice/user-1000.slice/vrooli.slice/other.slice/" + session.Scope
	if _, err := CaptureExactReapProof(session, session.Scope, wrongSlice); err == nil {
		t.Fatal("accepted a cgroup path outside the canonical Vrooli agent slice")
	}
	for name, mutate := range map[string]func(*Session){
		"not orphaned": func(s *Session) { s.State = StateLive },
		"owner alive":  func(s *Session) { s.Liveness.State = OwnerAlive },
		"frozen":       func(s *Session) { s.Frozen = "yes" },
		"not held":     func(s *Session) { s.ScopeHeld = false },
	} {
		t.Run(name, func(t *testing.T) {
			changed := session
			mutate(&changed)
			if _, err := CaptureExactReapProof(changed, changed.Scope, facts); err == nil {
				t.Fatal("accepted unsafe state")
			}
		})
	}
}

func TestCaptureExactReapProofRejectsCodingAgentAndUnreadableMembers(t *testing.T) {
	session, facts := testExactSession(), testExactFacts()
	facts.names[42] = "codex"
	if _, err := CaptureExactReapProof(session, session.Scope, facts); err == nil {
		t.Fatal("accepted a protected coding-agent process")
	}
	facts = testExactFacts()
	facts.facts[19] = ProcessFact{}
	if _, err := CaptureExactReapProof(session, session.Scope, facts); err == nil {
		t.Fatal("accepted unreadable process incarnation")
	}
}

func TestExecuteExactReapRequiresBoundApprovalAndStableOwnerAndContainment(t *testing.T) {
	ctx := context.Background()
	session, facts := testExactSession(), testExactFacts()
	proof, err := CaptureExactReapProof(session, session.Scope, facts)
	if err != nil {
		t.Fatal(err)
	}
	approval := ExactReapApproval{DecisionID: "decision-1", ApprovedBy: "human-owner", Operation: proof.Operation, Target: ExactReapAuthorizationTarget(proof), ProofDigest: proof.Digest, VerifiedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour).UTC()}
	refresh := func(context.Context, string) (Session, error) { return session, nil }
	stopper := &exactStopper{}
	gotApproval, err := ExecuteExactReap(ctx, proof, facts, exactApprovalVerifier{approval: approval}, refresh, stopper)
	if err != nil || stopper.calls != 1 || gotApproval.DecisionID != approval.DecisionID {
		t.Fatalf("execute exact reap = (%+v, %v), stop calls %d", gotApproval, err, stopper.calls)
	}
	stopper.calls = 0
	wildcard := approval
	wildcard.Target = "agent-session:*"
	if _, err := ExecuteExactReap(ctx, proof, facts, exactApprovalVerifier{approval: wildcard}, refresh, stopper); err == nil || stopper.calls != 0 {
		t.Fatalf("wildcard approval accepted: err=%v calls=%d", err, stopper.calls)
	}
	facts.facts[42] = ProcessFact{PID: 42, StartTime: "reused-pid"}
	if _, err := ExecuteExactReap(ctx, proof, facts, exactApprovalVerifier{approval: approval}, refresh, stopper); err == nil || stopper.calls != 0 {
		t.Fatalf("changed process incarnation accepted: err=%v calls=%d", err, stopper.calls)
	}
	facts = testExactFacts()
	ownerReturned := func(context.Context, string) (Session, error) {
		updated := session
		updated.State = StateLive
		updated.Liveness = Liveness{State: OwnerAlive}
		return updated, nil
	}
	if _, err := ExecuteExactReap(ctx, proof, facts, exactApprovalVerifier{approval: approval}, ownerReturned, stopper); err == nil || stopper.calls != 0 {
		t.Fatalf("returning owner accepted: err=%v calls=%d", err, stopper.calls)
	}
}
