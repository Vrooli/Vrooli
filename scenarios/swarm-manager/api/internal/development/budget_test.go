package development

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// [REQ:SWM-P0-004] [REQ:SWM-P0-017]
func TestBudgetPolicyIsReviewedAuthority(t *testing.T) {
	r, p := fixture(t)
	implicit, err := r.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	p.BudgetPolicy = BudgetMetered
	explicit, err := r.Preview(p)
	if err != nil || explicit.ProposalDigest != implicit.ProposalDigest || !strings.Contains(explicit.GoalMessage, "in-flight usage may overshoot") {
		t.Fatalf("default metered review: %v", err)
	}
	p.BudgetPolicy = BudgetHard
	hard, err := r.Preview(p)
	if err != nil || hard.ProposalDigest == explicit.ProposalDigest || !strings.Contains(hard.GoalMessage, "Refuse unsupported paths") {
		t.Fatalf("hard review: %v", err)
	}
	p.BudgetPolicy = "unlimited"
	if _, err := r.Preview(p); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown policy: %v", err)
	}
	s, p, _ := setupService(t)
	state, _ := s.Get(context.Background(), p.WorkItem)
	snapshot, _ := s.repo.Snapshot(context.Background(), state.Digest)
	if snapshot.Proposal.BudgetPolicy != BudgetMetered {
		t.Fatal("new approval failed to retain explicit policy")
	}
	p.BudgetPolicy = BudgetHard
	if _, err := s.Approve(operatorContext(), p, state.Digest, state.Version, "change mode without review"); !errors.Is(err, ErrConflict) {
		t.Fatalf("policy changed under stale approval: %v", err)
	}
}

// [REQ:SWM-P0-015] [REQ:SWM-P0-016]
func TestMeteredOvershootRetainsUsageAndStillRequiresOutcomeEvidence(t *testing.T) {
	s, p, _ := setupService(t)
	ctx := operatorContext()
	state, _ := s.Get(ctx, p.WorkItem)
	_, err := s.Reserve(ctx, p.WorkItem, state.Digest, "repair", "workflow-fallback", Usage{10000, 300})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.BindExecution(ctx, p.WorkItem, "repair", "owner-run"); err != nil {
		t.Fatal(err)
	}
	state, err = s.Settle(ctx, p.WorkItem, "repair", "owner-run", &Usage{10123, 250}, "late meter reconciled")
	if err != nil || state.Status != "budget_exhausted" || state.Used.Tokens != 10123 || state.Reserved != (Usage{}) {
		t.Fatalf("metered settlement: %+v %v", state, err)
	}
	if _, err = s.Reserve(ctx, p.WorkItem, state.Digest, "next", "workflow-fallback", Usage{1, 1}); !errors.Is(err, ErrDenied) {
		t.Fatal("exhausted engagement dispatched again")
	}
	if _, err = s.Accept(ctx, p.WorkItem, state.Version, map[string]string{"tail": "receipt"}); !errors.Is(err, ErrDenied) {
		t.Fatal("missing evidence owner accepted")
	}
	now := time.Now().UTC()
	s.evidence = &ownerEvidence{revision: "product", value: Evidence{OutcomeID: "tail", Source: p.Outcomes[0].EvidenceSource, ResolverID: "testgenie.audio", ReceiptSchema: "v1", ReceiptID: "receipt", Digest: state.Digest, ExecutionID: "owner-run", SubjectRevision: "product", Cohort: "linux-amd64-chromium", Passed: true, Status: "passed", ObservedAt: now, FreshUntil: now.Add(time.Hour)}}
	accepted, err := s.Accept(ctx, p.WorkItem, state.Version, map[string]string{"tail": "receipt"})
	if err != nil || accepted.Status != "accepted" || accepted.Used.Tokens != 10123 {
		t.Fatalf("evidenced outcome with authorized overshoot: %+v %v", accepted, err)
	}
}

// [REQ:SWM-P0-015]
func TestMeteredAttemptOvershootReducesTheNextReservation(t *testing.T) {
	s, p, _ := setupService(t)
	ctx := operatorContext()
	state, _ := s.Get(ctx, p.WorkItem)
	if _, err := s.Reserve(ctx, p.WorkItem, state.Digest, "first", "workflow-fallback", Usage{5000, 300}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.BindExecution(ctx, p.WorkItem, "first", "run"); err != nil {
		t.Fatal(err)
	}
	state, err := s.Settle(ctx, p.WorkItem, "first", "run", &Usage{5200, 250}, "meter stopped first attempt")
	if err != nil || state.Status != "paused" {
		t.Fatalf("aggregate still has allowance: %+v %v", state, err)
	}
	if _, err = s.Reserve(ctx, p.WorkItem, state.Digest, "too-much", "workflow-fallback", Usage{5000, 300}); !errors.Is(err, ErrDenied) {
		t.Fatal("overshoot did not reduce remaining budget")
	}
	state, err = s.Reserve(ctx, p.WorkItem, state.Digest, "remaining", "workflow-fallback", Usage{4800, 300})
	if err != nil || state.Used.Tokens != 5200 || state.Reserved.Tokens != 4800 || len(state.Approvals) != 1 {
		t.Fatalf("continuation allowance reset: %+v %v", state, err)
	}
}

// Legacy retained snapshots never gain the new overshoot permission by default.
// [REQ:SWM-P0-004] [REQ:SWM-P0-015]
func TestLegacyAndHardReservationsRetainStrictDisposition(t *testing.T) {
	for _, policy := range []string{"", BudgetHard, BudgetMetered} {
		t.Run("policy_"+policy, func(t *testing.T) {
			s, p, _ := setupService(t)
			ctx := operatorContext()
			p.WorkItem = "execute/retained-policy"
			p.BudgetPolicy = policy
			snapshot := Snapshot{Digest: "retained", Proposal: p, GoalMessage: "historical approved goal"}
			state := Engagement{WorkItem: p.WorkItem, WorkShape: "contract-development", Version: 1, Digest: snapshot.Digest, Status: "approved"}
			if err := s.repo.Commit(ctx, 0, state, &snapshot); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Reserve(ctx, p.WorkItem, state.Digest, "attempt", "workflow-fallback", Usage{100, 100}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.BindExecution(ctx, p.WorkItem, "attempt", "run"); err != nil {
				t.Fatal(err)
			}
			usage := Usage{110, 90}
			if policy == BudgetMetered {
				usage.WallSeconds = 101
			} // Token policy does not authorize time overrun.
			state, err := s.Settle(ctx, p.WorkItem, "attempt", "run", &usage, "retained usage")
			if err != nil || state.Status != "cancelled" || state.Used != usage {
				t.Fatalf("strict settlement: %+v %v", state, err)
			}
		})
	}
}
