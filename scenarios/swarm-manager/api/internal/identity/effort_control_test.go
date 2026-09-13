package identity

import (
	"errors"
	"testing"
)

func testCandidatePolicy() CandidatePolicyBinding {
	return CandidatePolicyBinding{
		EconomicalRunner: "opencode",
		EconomicalModel:  "opencode-go/deepseek-v4.1-flash",
		EconomicalEffort: "runner-native",
		CredentialPool:   "aquila-shared",
		Capabilities:     []string{"tools", "files"},
		Withheld:         []string{"openrouter-paid"},
	}.Bind()
}

func testEffortControl(effortID string, revision int64) EffortControl {
	return EffortControl{
		EffortID:         effortID,
		Slug:             "aquila-launch-2026-09-17",
		Revision:         revision,
		FamilyID:         "4533de55-41ea-427a-839e-83e1a3d98d82",
		DelegatedActions: []string{DelegatedActionAuthorChildPlans, DelegatedActionDispatch, DelegatedActionRepair},
		Scope: EffortScope{
			Allow: []string{"scenarios/swarm-manager/**", "packages/proto/**"},
			Deny:  []string{"scenarios/swarm-manager/api/internal/auth/**"},
		},
		AggregateLimits: AggregateLimits{
			MaxWorkers:            3,
			MaxConcurrency:        3,
			MaxDepth:              2,
			MaxActiveDescendants:  3,
			MaxPremiumDescendants: 1,
		},
		RepairLimits: RepairLimits{
			PerFingerprint:         2,
			PerComponent:           4,
			PerEffort:              12,
			ComponentActiveMinutes: 90,
		},
		PolicyBinding: PolicyBinding{
			Source: "efforts/aquila-launch-2026-09-17/effort.json",
			Digest: "sha256:approved-revision-1",
		},
		CandidatePolicy: testCandidatePolicy(),
		WorkReferences: []WorkReference{
			{Owner: "plan-manager", Kind: "family", ID: "4533de55-41ea-427a-839e-83e1a3d98d82", Role: "topology"},
		},
	}
}

func TestEffortControlAdmitRejectsReplacementOfExistingEffort(t *testing.T) {
	current := testEffortControl("aquila-launch-2026-09-17", 1)

	replacement := testEffortControl("aquila-launch-2026-09-17", 1)
	if _, err := replacement.Admit(&current); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected revision conflict when admitting over an existing effort, got %v", err)
	}

	fresh := testEffortControl("aquila-other-effort", 1)
	if _, err := fresh.Admit(nil); err != nil {
		t.Fatalf("initial admission of a new effort failed: %v", err)
	}
}

func TestEffortControlAmendRequiresExactNextRevision(t *testing.T) {
	current := testEffortControl("aquila-launch-2026-09-17", 4)

	skipped := testEffortControl("aquila-launch-2026-09-17", 6)
	if _, err := skipped.Amend(current); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected revision conflict for a skipped revision, got %v", err)
	}

	next := testEffortControl("aquila-launch-2026-09-17", 5)
	if _, err := next.Amend(current); err != nil {
		t.Fatalf("expected revision 5 to amend revision 4, got %v", err)
	}
}

func TestEffortControlTwoEffortIsolation(t *testing.T) {
	effortA := testEffortControl("aquila-launch-2026-09-17", 1)
	effortB := testEffortControl("other-effort", 1)
	effortB.Scope.Allow = []string{"scenarios/content-desk/**"}
	effortB.DelegatedActions = []string{DelegatedActionRepair}

	if effortA.AuthorityDigest() == effortB.AuthorityDigest() {
		t.Fatal("two efforts with different policies shared an authority digest")
	}

	foreign := testEffortControl("other-effort", 2)
	if _, err := foreign.Amend(effortA); !errors.Is(err, ErrEffortIdentityMismatch) {
		t.Fatalf("expected identity mismatch amending across efforts, got %v", err)
	}

	// Amending one effort must not change the other's reviewed digest.
	before := effortA.AuthorityDigest()
	amended := testEffortControl("other-effort", 2)
	if _, err := amended.Amend(effortB); err != nil {
		t.Fatalf("amending effort B failed: %v", err)
	}
	if effortA.AuthorityDigest() != before {
		t.Fatal("amending a second effort mutated the first effort's digest")
	}
}

func TestEffortControlCompletionNeverBecomesHumanAccepted(t *testing.T) {
	control := testEffortControl("aquila-launch-2026-09-17", 1)

	control.Completion = control.Completion.MarkEvidenceComplete("evidence/aquila-wake139.md")
	if !control.Completion.EvidenceComplete {
		t.Fatal("evidence completion was not recorded")
	}
	if control.Completion.HumanAccepted {
		t.Fatal("evidence completion set human acceptance")
	}
	if control.Completion.HumanActor != "" {
		t.Fatal("evidence completion recorded a human actor")
	}

	if _, err := control.Completion.HumanAccept("   "); !errors.Is(err, ErrHumanAcceptanceRequired) {
		t.Fatalf("expected a blank human actor to be refused, got %v", err)
	}
	if control.Completion.HumanAccepted {
		t.Fatal("a refused product acceptance still set HumanAccepted")
	}

	accepted, err := control.Completion.HumanAccept("operator")
	if err != nil {
		t.Fatalf("authenticated human acceptance failed: %v", err)
	}
	if !accepted.HumanAccepted || accepted.HumanActor != "operator" {
		t.Fatalf("human acceptance was not recorded: %+v", accepted)
	}
}

func TestEffortControlAuthorityDigestExcludesCompletion(t *testing.T) {
	control := testEffortControl("aquila-launch-2026-09-17", 1)
	before := control.AuthorityDigest()

	control.Completion = control.Completion.MarkEvidenceComplete("evidence/one.md")
	if control.AuthorityDigest() != before {
		t.Fatal("evidence completion changed the authority digest")
	}

	control.DelegatedActions = append(control.DelegatedActions, DelegatedActionPlanRound)
	if control.AuthorityDigest() == before {
		t.Fatal("a changed delegated action did not change the authority digest")
	}
}

func TestEffortControlAuthorityDigestStableUnderOrdering(t *testing.T) {
	control := testEffortControl("aquila-launch-2026-09-17", 1)
	before := control.AuthorityDigest()

	reordered := testEffortControl("aquila-launch-2026-09-17", 1)
	reordered.DelegatedActions = []string{DelegatedActionRepair, DelegatedActionAuthorChildPlans, DelegatedActionDispatch}
	reordered.Scope.Allow = []string{"packages/proto/**", "scenarios/swarm-manager/**"}
	if reordered.AuthorityDigest() != before {
		t.Fatal("slice ordering changed the authority digest")
	}
}

func TestEffortControlRefusesUnknownDelegatedAction(t *testing.T) {
	control := testEffortControl("aquila-launch-2026-09-17", 1)
	control.DelegatedActions = []string{"change-global-approval-policy"}
	if err := control.Validate(); !errors.Is(err, ErrUnauthorizedAction) {
		t.Fatalf("expected an unknown delegated action to be refused, got %v", err)
	}
	if control.Authorizes(DelegatedActionDispatch) {
		t.Fatal("a revision with an unknown action authorized a known one")
	}
}

func TestEffortControlCandidatePolicyIsImmutable(t *testing.T) {
	control := testEffortControl("aquila-launch-2026-09-17", 1)
	if err := control.Validate(); err != nil {
		t.Fatalf("a bound candidate policy should validate: %v", err)
	}

	tampered := control
	tampered.CandidatePolicy.PremiumModel = "gpt-5.6-sol"
	if err := tampered.Validate(); !errors.Is(err, ErrCandidatePolicyImmutable) {
		t.Fatalf("expected a tampered candidate policy to be refused, got %v", err)
	}

	foreign := testCandidatePolicy()
	foreign.EconomicalModel = "some-other-model"
	if err := control.VerifyCandidatePolicy(foreign); !errors.Is(err, ErrCandidatePolicyImmutable) {
		t.Fatalf("expected a mismatched candidate policy digest to be refused, got %v", err)
	}
	if err := control.VerifyCandidatePolicy(control.CandidatePolicy); err != nil {
		t.Fatalf("the bound candidate policy should verify: %v", err)
	}
}

func TestEffortControlDescendantAllowanceIsShared(t *testing.T) {
	limits := testEffortControl("aquila-launch-2026-09-17", 1).AggregateLimits

	if err := limits.CanActivate(3, 0, false); !errors.Is(err, ErrAdditionalCapacityDenied) {
		t.Fatalf("expected the active-descendant cap to refuse a fourth child, got %v", err)
	}
	if err := limits.CanActivate(1, 1, true); !errors.Is(err, ErrAdditionalCapacityDenied) {
		t.Fatalf("expected the premium cap to refuse a second premium child, got %v", err)
	}
	if err := limits.CanActivate(1, 0, false); err != nil {
		t.Fatalf("a non-premium child under the allowance was refused: %v", err)
	}
}

func TestEffortScopeDenyWins(t *testing.T) {
	scope := testEffortControl("aquila-launch-2026-09-17", 1).Scope

	if !scope.Allows("scenarios/swarm-manager/api/internal/backlog/plan_acceptance.go") {
		t.Fatal("an allowed path was refused")
	}
	if scope.Allows("scenarios/swarm-manager/api/internal/auth/token.go") {
		t.Fatal("a denied path was allowed")
	}
	if scope.Allows("scenarios/content-desk/api/service.go") {
		t.Fatal("a path outside the effort scope was allowed")
	}
}

func TestEffortControlRepairLimitsRejectIncoherentTotals(t *testing.T) {
	limits := RepairLimits{PerFingerprint: 4, PerComponent: 2, PerEffort: 1, ComponentActiveMinutes: 90}
	if err := limits.Validate(); err == nil {
		t.Fatal("expected incoherent cumulative repair limits to be refused")
	}
}
