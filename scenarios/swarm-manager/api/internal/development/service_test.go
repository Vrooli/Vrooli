package development

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	_ "modernc.org/sqlite"
	"swarm-manager/internal/identity"
)

func operatorContext() context.Context {
	return coreidentity.WithPrincipal(context.Background(), coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "operator-test", Verified: true, Scopes: []string{"swarm-manager:write"}})
}

func openRepo(t *testing.T, filename string) *SQLiteRepository {
	t.Helper()
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	return NewSQLiteRepository(database.NewFromPrimary(db))
}

func setupService(t *testing.T) (*Service, Proposal, string) {
	t.Helper()
	r, p := fixture(t)
	filename := filepath.Join(t.TempDir(), "engagements.db")
	s := NewService(openRepo(t, filename), r, func(context.Context, string) error { return nil }, nil)
	review, err := r.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Approve(operatorContext(), p, review.ProposalDigest, 0, "bounded test approval"); err != nil {
		t.Fatal(err)
	}
	return s, p, filename
}

// [REQ:SWM-P0-004] [REQ:SWM-P0-015]
func TestApprovedBytesSurviveSourceEditsAndRepositoryReopen(t *testing.T) {
	s, p, filename := setupService(t)
	before, err := s.Get(context.Background(), p.WorkItem)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, s.reviewer.RepoRoot, "scenarios/example/PRD.md", "weakened outcome")
	reopened := openRepo(t, filename)
	snapshot, err := reopened.Snapshot(context.Background(), before.Digest)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, content := range snapshot.Contents {
		if content.Path == "scenarios/example/PRD.md" {
			found = true
			if string(content.Bytes) != "Protected product outcome" {
				t.Fatal("approved bytes changed with live source")
			}
		}
	}
	if !found {
		t.Fatal("approved PRD bytes not retained")
	}
	if _, err := s.Approve(operatorContext(), p, before.Digest, before.Version, "approve stale view"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale approval: %v", err)
	}
	after, _ := s.Get(context.Background(), p.WorkItem)
	if after.Version != before.Version || after.Digest != before.Digest {
		t.Fatal("stale approval changed authority")
	}
}

func TestApprovalCannotMoveAnEngagementToAnotherCanonicalPlan(t *testing.T) {
	s, p, _ := setupService(t)
	s.SetPlanReferenceValidator(func(context.Context, string, PlanReference) error { return nil })
	state, err := s.Get(context.Background(), p.WorkItem)
	if err != nil {
		t.Fatal(err)
	}
	p.PlanRef = &PlanReference{Provider: PlanManagerProvider, PlanID: "different-plan", Slug: "different-plan", Role: ExecutionSpecRole}
	review, err := s.reviewer.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Approve(operatorContext(), p, review.ProposalDigest, state.Version, "attempt to move plan"); !errors.Is(err, ErrConflict) {
		t.Fatalf("plan replacement was admitted: %v", err)
	}
}

// [REQ:SWM-P0-015]
func TestTwoRepairsShareApprovalAndAccountingAcrossRestart(t *testing.T) {
	s, p, filename := setupService(t)
	ctx := operatorContext()
	approved, _ := s.Get(ctx, p.WorkItem)
	if approved.Campaign.ApprovalDigest != approved.Digest || !approved.Campaign.Pending || len(approved.Campaign.RemainingOutcomeIDs) != len(p.Outcomes) {
		t.Fatalf("approval did not initialize campaign checkpoint: %+v", approved.Campaign)
	}
	for i, key := range []string{"repair-one", "repair-two"} {
		state, err := s.Reserve(ctx, p.WorkItem, approved.Digest, key, "workflow-fallback", Usage{5000, 300})
		if err != nil {
			t.Fatal(err)
		}
		if state.Reserved.Tokens != 5000 {
			t.Fatal("reservation not held")
		}
		if state.Campaign.ApprovalDigest != approved.Digest || state.Campaign.AttemptKey != key || !state.Campaign.Pending {
			t.Fatalf("campaign reservation checkpoint: %+v", state.Campaign)
		}
		if _, err := s.BindExecution(ctx, p.WorkItem, key, "owner-"+key); err != nil {
			t.Fatal(err)
		}
		// Reconstruct both service and repository while an attempt is active.
		s = NewService(openRepo(t, filename), s.reviewer, s.validateWork, nil)
		state, err = s.Settle(ctx, p.WorkItem, key, "owner-"+key, &Usage{4500, 250}, "repair checked; next outcome remains")
		if err != nil {
			t.Fatal(err)
		}
		if state.Used.Tokens != int64(i+1)*4500 || state.Used.WallSeconds != int64(i+1)*250 || len(state.Approvals) != 1 {
			t.Fatalf("accounting/approval reset: %+v", state)
		}
		if state.Campaign.ApprovalDigest != approved.Digest || state.Campaign.OwnerExecutionID != "owner-"+key || state.Campaign.Pending || state.Campaign.LastOutcome == "" || state.Campaign.Digest == "" {
			t.Fatalf("campaign checkpoint was not reconstructed and settled: %+v", state.Campaign)
		}
		if state.Campaign.LastCheckpoint == nil || state.Campaign.LastCheckpoint.Kind != "owner-verification" || state.Campaign.NoProgressCycles != i {
			t.Fatalf("checkpoint reference/no-progress state: %+v", state.Campaign)
		}
		version := state.Version
		again, err := s.Settle(ctx, p.WorkItem, key, "owner-"+key, &Usage{4500, 250}, "repair checked; next outcome remains")
		if err != nil || again.Version != version {
			t.Fatalf("settlement retry charged twice: %v", err)
		}
	}
	if _, err := s.Reserve(ctx, p.WorkItem, approved.Digest, "repair-three", "native-goal", Usage{5000, 300}); !errors.Is(err, ErrDenied) {
		t.Fatalf("fallback-to-native reset allowance: %v", err)
	}
	state, _ := s.Get(ctx, p.WorkItem)
	if state.Used.Tokens != 9000 || state.Reserved.Tokens != 0 || state.Status != "paused" {
		t.Fatal("budget refusal mutated accounting")
	}
}

// [REQ:SWM-P0-015]
func TestCampaignRetainsIndependentUnmetOutcomesAcrossRepairs(t *testing.T) {
	r, p := fixture(t)
	p.Outcomes = append(p.Outcomes, Outcome{ID: "streaming", Criterion: "A later interval remains available", EvidenceSource: p.Outcomes[0].EvidenceSource})
	filename := filepath.Join(t.TempDir(), "campaign.db")
	s := NewService(openRepo(t, filename), r, func(context.Context, string) error { return nil }, nil)
	review, err := r.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	approved, err := s.Approve(operatorContext(), p, review.ProposalDigest, 0, "approve one campaign")
	if err != nil {
		t.Fatal(err)
	}
	for i, key := range []string{"repair-one", "repair-two"} {
		if _, err := s.Reserve(context.Background(), p.WorkItem, approved.Digest, key, "workflow-fallback", Usage{2000, 100}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.BindExecution(context.Background(), p.WorkItem, key, "campaign-owner-"+key); err != nil {
			t.Fatal(err)
		}
		state, err := s.Settle(context.Background(), p.WorkItem, key, "campaign-owner-"+key, &Usage{1500, 80}, "verified repair "+key)
		if err != nil {
			t.Fatal(err)
		}
		if len(state.Approvals) != 1 || state.Campaign.ApprovalDigest != approved.Digest || len(state.Campaign.RemainingOutcomeIDs) != 2 || len(state.Campaign.CompletedOutcomeIDs) != 0 {
			t.Fatalf("repair %d changed the protected campaign inventory: %+v", i, state.Campaign)
		}
	}
	state, err := s.Get(context.Background(), p.WorkItem)
	if err != nil {
		t.Fatal(err)
	}
	if state.Used != (Usage{3000, 160}) || state.Status != "paused" {
		t.Fatalf("campaign accounting reset or status changed: %+v", state)
	}
}

// [REQ:SWM-P0-015]
func TestLostStartCancellationAndOverrunRemainAccounted(t *testing.T) {
	s, p, _ := setupService(t)
	ctx := operatorContext()
	approved, _ := s.Get(ctx, p.WorkItem)
	state, err := s.Reserve(ctx, p.WorkItem, approved.Digest, "start-one", "native-goal", Usage{5000, 300})
	if err != nil {
		t.Fatal(err)
	}
	retry, err := s.Reserve(ctx, p.WorkItem, approved.Digest, "start-one", "native-goal", Usage{5000, 300})
	if err != nil || retry.Version != state.Version || len(retry.Attempts) != 1 {
		t.Fatal("lost response reservation duplicated")
	}
	if _, err := s.Reserve(ctx, p.WorkItem, approved.Digest, "start-two", "workflow-fallback", Usage{5000, 300}); err == nil {
		t.Fatal("concurrent start admitted")
	}
	state, err = s.Revoke(ctx, p.WorkItem, state.Version, "operator cancelled")
	if err != nil {
		t.Fatal(err)
	}
	if state.Reserved.Tokens != 5000 {
		t.Fatal("cancellation discarded unknown usage")
	}
	if _, err := s.BindExecution(ctx, p.WorkItem, "start-one", "late-owner-response"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Settle(ctx, p.WorkItem, "start-one", "late-owner-response", nil, "unknown receipt"); !errors.Is(err, ErrDenied) {
		t.Fatal("unknown usage was settled")
	}
	unknown, _ := s.Get(ctx, p.WorkItem)
	if unknown.Reserved.Tokens != 5000 || unknown.Attempts[0].SettledAt != nil {
		t.Fatal("unknown usage released reservation")
	}
	state, err = s.Settle(ctx, p.WorkItem, "start-one", "late-owner-response", &Usage{5200, 310}, "cancel reconciled")
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "cancelled" || state.Used.Tokens != 5200 || state.Reserved != (Usage{}) || state.StopReason == "" {
		t.Fatal("overrun was hidden")
	}
	if _, err := s.Reserve(ctx, p.WorkItem, approved.Digest, "after-cancel", "native-goal", Usage{100, 10}); !errors.Is(err, ErrDenied) {
		t.Fatal("cancelled engagement admitted more work")
	}
}

// [REQ:SWM-P0-004] [REQ:SWM-P0-015]
func TestAmendmentCannotDiscardUsageAndRetainsPriorRevision(t *testing.T) {
	s, p, _ := setupService(t)
	ctx := operatorContext()
	state, _ := s.Get(ctx, p.WorkItem)
	original := state.Digest
	_, err := s.Reserve(ctx, p.WorkItem, state.Digest, "repair", "workflow-fallback", Usage{5000, 300})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.BindExecution(ctx, p.WorkItem, "repair", "owner-run"); err != nil {
		t.Fatal(err)
	}
	state, err = s.Settle(ctx, p.WorkItem, "repair", "owner-run", &Usage{4000, 200}, "checkpoint")
	if err != nil {
		t.Fatal(err)
	}
	p.MaxTokens = 3999
	r, _ := s.reviewer.Preview(p)
	if _, err := s.Approve(ctx, p, r.ProposalDigest, state.Version, "reduce ceiling"); !errors.Is(err, ErrInvalid) {
		t.Fatal("amendment erased incurred cost")
	}
	p.MaxTokens = 12000
	r, _ = s.reviewer.Preview(p)
	state, err = s.Approve(ctx, p, r.ProposalDigest, state.Version, "operator extends same engagement")
	if err != nil {
		t.Fatal(err)
	}
	if state.Used.Tokens != 4000 || len(state.Approvals) != 2 || state.Digest == original {
		t.Fatal("amendment did not preserve identity/accounting")
	}
	if _, err := s.Snapshot(ctx, original); err != nil {
		t.Fatal("old snapshot lost")
	}
}

// [REQ:SWM-P0-004]
func TestConcurrentDecisionsHaveOneWinnerAndImmutableSnapshot(t *testing.T) {
	s, p, _ := setupService(t)
	ctx := operatorContext()
	state, _ := s.Get(ctx, p.WorkItem)
	var wg sync.WaitGroup
	errorsOut := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Revoke(ctx, p.WorkItem, state.Version, "cancel decision")
			errorsOut <- err
		}()
	}
	wg.Wait()
	close(errorsOut)
	success, conflict := 0, 0
	for err := range errorsOut {
		if err == nil {
			success++
		} else if errors.Is(err, ErrConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("decision winners=%d conflicts=%d", success, conflict)
	}
	snapshot, _ := s.Snapshot(ctx, state.Digest)
	snapshot.Contents[0].Bytes = []byte("replacement")
	state, _ = s.Get(ctx, p.WorkItem)
	expected := state.Version
	state.Version++
	if err := s.repo.Commit(ctx, expected, state, &snapshot); !errors.Is(err, ErrConflict) {
		t.Fatal("immutable snapshot overwritten")
	}
	after, _ := s.Get(ctx, p.WorkItem)
	if after.Version != expected {
		t.Fatal("snapshot conflict partially committed aggregate")
	}
}

type ownerEvidence struct {
	value    Evidence
	revision string
	err      error
}

func (r *ownerEvidence) Resolve(context.Context, string, string) (Evidence, error) {
	return r.value, r.err
}

func (r *ownerEvidence) CurrentRevision(context.Context, string) (string, error) {
	return r.revision, r.err
}

// [REQ:SWM-P0-016]
func TestAcceptanceRequiresOwnerEvidenceForCurrentProductAndApprovedTarget(t *testing.T) {
	for _, mutation := range []string{"missing", "failed", "wrong-target", "stale-product", "wrong-run", "wrong-outcome", "wrong-source", "wrong-receipt", "wrong-resolver", "wrong-schema", "wrong-cohort", "stale-receipt", "replayed", "superseded", "old", "future", "owner-unavailable", "valid"} {
		t.Run(mutation, func(t *testing.T) {
			s, p, _ := setupService(t)
			ctx := operatorContext()
			state, _ := s.Get(ctx, p.WorkItem)
			state, err := s.Reserve(ctx, p.WorkItem, state.Digest, "repair", "workflow-fallback", Usage{5000, 300})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.BindExecution(ctx, p.WorkItem, "repair", "owner-run"); err != nil {
				t.Fatal(err)
			}
			state, err = s.Settle(ctx, p.WorkItem, "repair", "owner-run", &Usage{4000, 200}, "agent says complete")
			if err != nil {
				t.Fatal(err)
			}
			if state.Status == "accepted" {
				t.Fatal("harness completion became acceptance")
			}
			now := time.Now().UTC()
			owner := &ownerEvidence{revision: "product-sha", value: Evidence{OutcomeID: "tail", Source: p.Outcomes[0].EvidenceSource, ResolverID: "testgenie.audio", ReceiptSchema: "v1", ReceiptID: "receipt-1", Digest: state.Digest, ExecutionID: "owner-run", SubjectRevision: "product-sha", Cohort: "linux-amd64-chromium", Passed: true, Status: "passed", ObservedAt: now, FreshUntil: now.Add(time.Hour)}}
			refs := map[string]string{"tail": "receipt-1"}
			switch mutation {
			case "missing":
				refs = nil
			case "failed":
				owner.value.Passed = false
			case "wrong-target":
				owner.value.Digest = "another-target"
			case "stale-product":
				owner.value.SubjectRevision = "old-product-sha"
			case "wrong-run":
				owner.value.ExecutionID = "unrelated-run"
			case "wrong-outcome":
				owner.value.OutcomeID = "different-outcome"
			case "wrong-source":
				owner.value.Source = "agent-report"
			case "wrong-receipt":
				owner.value.ReceiptID = "other-receipt"
			case "wrong-resolver":
				owner.value.ResolverID = "other.resolver"
			case "wrong-schema":
				owner.value.ReceiptSchema = "v2"
			case "wrong-cohort":
				owner.value.Cohort = "other-cohort"
			case "stale-receipt":
				owner.value.FreshUntil = time.Now().UTC().Add(-time.Second)
			case "replayed":
				owner.value.Status = "replayed"
			case "superseded":
				owner.value.Status = "superseded"
			case "old":
				owner.value.ObservedAt = state.Attempts[0].StartedAt.Add(-time.Second)
			case "future":
				owner.value.ObservedAt = time.Now().Add(time.Hour)
			case "owner-unavailable":
				owner.err = errors.New("owner offline")
			}
			s.evidence = owner
			accepted, err := s.Accept(ctx, p.WorkItem, state.Version, refs)
			if mutation == "valid" {
				if err != nil || accepted.Status != "accepted" || len(accepted.Evidence) != 1 || accepted.AcceptedAt == nil {
					t.Fatalf("valid owner evidence refused: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("invalid evidence accepted")
				}
				after, _ := s.Get(ctx, p.WorkItem)
				if after.Version != state.Version || after.Status != "paused" || len(after.Evidence) != 0 {
					t.Fatal("refusal changed acceptance state")
				}
			}
		})
	}
}

// [REQ:SWM-P0-004]
func TestAgentAndFailedVerificationCannotApproveOrRevoke(t *testing.T) {
	s, p, _ := setupService(t)
	state, _ := s.Get(context.Background(), p.WorkItem)
	if _, err := s.Approve(context.Background(), p, state.Digest, state.Version, "anonymous approval"); !errors.Is(err, ErrDenied) {
		t.Fatal("anonymous caller approved target")
	}
	for _, identityValue := range []identity.Provenance{
		{Actor: identity.TypeAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run"},
		{Actor: identity.TypeOperator, VerificationStatus: provenance.VerificationInvalid},
		{Actor: identity.TypeOperator, VerificationStatus: provenance.VerificationUnavailable},
	} {
		ctx := identity.NewContext(operatorContext(), identityValue)
		if _, err := s.Approve(ctx, p, state.Digest, state.Version, "self approve"); !errors.Is(err, ErrDenied) {
			t.Fatal("non-operator approval accepted")
		}
		if _, err := s.Revoke(ctx, p.WorkItem, state.Version, "revoke"); !errors.Is(err, ErrDenied) {
			t.Fatal("non-operator revocation accepted")
		}
	}
	after, _ := s.Get(context.Background(), p.WorkItem)
	if after.Version != state.Version {
		t.Fatal("denied caller changed authority")
	}
}
