package planproto

import (
	"math"
	"testing"

	"plan-manager/internal/planmodel"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

func TestOrderToInt32ClampsOutOfRangeValues(t *testing.T) {
	t.Parallel()

	if got := OrderToInt32(-1); got != 0 {
		t.Fatalf("OrderToInt32(-1) = %d, want 0", got)
	}
	if got := OrderToInt32(math.MaxInt32 + 1); got != math.MaxInt32 {
		t.Fatalf("OrderToInt32(MaxInt32+1) = %d, want MaxInt32", got)
	}
	if got := OrderToInt32(42); got != 42 {
		t.Fatalf("OrderToInt32(42) = %d, want 42", got)
	}
}

func TestEnumConversionsRoundTripKnownValuesAndDefaultSafely(t *testing.T) {
	t.Parallel()

	if got := PlanStatusFromProto(PlanStatusToProto(planmodel.PlanStatusActive)); got != planmodel.PlanStatusActive {
		t.Fatalf("plan status round trip = %q", got)
	}
	if got := PhaseStatusFromProto(PhaseStatusToProto(planmodel.PhaseStatusBlocked)); got != planmodel.PhaseStatusBlocked {
		t.Fatalf("phase status round trip = %q", got)
	}
	if got := RefKindFromProto(RefKindToProto(planmodel.ReferenceDoc)); got != planmodel.ReferenceDoc {
		t.Fatalf("reference kind round trip = %q", got)
	}
	if got := RefResolutionFromProto(RefResolutionToProto(planmodel.ResolutionFuture)); got != planmodel.ResolutionFuture {
		t.Fatalf("reference resolution round trip = %q", got)
	}
	if got := StalenessFromProto(StalenessToProto(planmodel.StalenessDefinitelyStale)); got != planmodel.StalenessDefinitelyStale {
		t.Fatalf("staleness round trip = %q", got)
	}

	if got := PlanStatusFromProto(sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED); got != "" {
		t.Fatalf("unspecified plan status = %q, want empty", got)
	}
	if got := RefKindFromProto(sharedv1.ReferenceKind_REFERENCE_KIND_UNSPECIFIED); got != planmodel.ReferenceCode {
		t.Fatalf("unspecified reference kind = %q, want code floor", got)
	}
	if got := StalenessFromProto(sharedv1.StalenessTier_STALENESS_TIER_UNSPECIFIED); got != planmodel.StalenessUnknown {
		t.Fatalf("unspecified staleness = %q, want unknown", got)
	}
}

func TestPlanToProtoMapsNestedStructuredFields(t *testing.T) {
	t.Parallel()

	proto := PlanToProto(planmodel.Plan{
		ID:               "plan-1",
		Slug:             "migrate-auth",
		Title:            "Migrate auth",
		Status:           planmodel.PlanStatusActive,
		ContentHash:      "hash",
		CreatedAt:        "2026-06-25T10:00:00Z",
		UpdatedAt:        "2026-06-25T11:00:00Z",
		Purpose:          "Move auth",
		Scope:            "auth only",
		Constraints:      "no outage",
		NonGoals:         "billing",
		DefinitionOfDone: "all green",
		Supersedes:       []string{"old"},
		SupersededBy:     []string{"new"},
		RegressionAnchor: planmodel.RegressionAnchor{
			Strategy:       "scenario_baseline",
			Scenario:       "plan-manager",
			BaselineName:   "base",
			HeadSha:        "abc123",
			AllowlistPaths: []string{"scenarios/plan-manager/**"},
			Commands:       []string{"go test ./..."},
			CapturedAt:     "2026-06-25T09:00:00Z",
		},
		References: []planmodel.Reference{{
			ID:           "ref-1",
			Kind:         planmodel.ReferenceReq,
			Target:       "PM-PLAN-001",
			Resolution:   planmodel.ResolutionResolved,
			Staleness:    planmodel.StalenessFresh,
			ChangeFactor: 0.25,
			Note:         "ok",
		}},
		RelevantContext: []planmodel.RelevantContextItem{{
			Kind:         planmodel.RelevantContextSearch,
			Scope:        planmodel.RelevantContextScopeGlobal,
			Label:        "Recall auth work",
			Command:      "search-hub query auth --type record,doc",
			Required:     true,
			RepeatPolicy: planmodel.RelevantContextOnResume,
			Source:       planmodel.RelevantContextSourceAuthored,
			Status:       planmodel.RelevantContextStatusReady,
		}},
		Phases: []planmodel.Phase{{
			ID:              "phase-1",
			Order:           1,
			Title:           "Contracts",
			Intent:          "Define proto",
			RequiredReading: []string{"docs/concepts/PLAN-MODEL.md"},
			Reminders:       []string{"No silent drops"},
			BaselineScope:   []string{"api"},
			Acceptance:      "Generated clients compile",
			Status:          planmodel.PhaseStatusDone,
			References: []planmodel.Reference{{
				Kind:   planmodel.ReferenceCode,
				Target: "api/main.go",
			}},
			RelevantContext: []planmodel.RelevantContextItem{{
				Kind:         planmodel.RelevantContextDoc,
				Scope:        planmodel.RelevantContextScopePhase,
				PhaseID:      "phase-1",
				Label:        "Model docs",
				Target:       "docs/concepts/PLAN-MODEL.md",
				Required:     true,
				RepeatPolicy: planmodel.RelevantContextPhaseEntry,
			}},
		}},
	})

	if proto.GetId() != "plan-1" || proto.GetSlug() != "migrate-auth" || proto.GetStatus() != sharedv1.PlanStatus_PLAN_STATUS_ACTIVE {
		t.Fatalf("proto identity/status = %#v", proto)
	}
	if proto.GetRegressionAnchor().GetBaselineName() != "base" {
		t.Fatalf("anchor = %#v", proto.GetRegressionAnchor())
	}
	if got := len(proto.GetReferences()); got != 1 {
		t.Fatalf("len(proto.References) = %d, want 1", got)
	}
	if proto.GetReferences()[0].GetKind() != sharedv1.ReferenceKind_REFERENCE_KIND_REQ {
		t.Fatalf("reference kind = %v", proto.GetReferences()[0].GetKind())
	}
	if got := len(proto.GetPhases()); got != 1 {
		t.Fatalf("len(proto.Phases) = %d, want 1", got)
	}
	phase := proto.GetPhases()[0]
	if phase.GetOrder() != 1 || phase.GetStatus() != sharedv1.PhaseStatus_PHASE_STATUS_DONE {
		t.Fatalf("phase order/status = %#v", phase)
	}
	if got := len(phase.GetReferences()); got != 1 {
		t.Fatalf("len(phase.References) = %d, want 1", got)
	}
	if got := len(proto.GetRelevantContext()); got != 1 {
		t.Fatalf("len(proto.RelevantContext) = %d, want 1", got)
	}
	if proto.GetRelevantContext()[0].GetRepeatPolicy() != sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_ON_RESUME {
		t.Fatalf("plan context repeat policy = %v", proto.GetRelevantContext()[0].GetRepeatPolicy())
	}
	if got := len(phase.GetRelevantContext()); got != 1 {
		t.Fatalf("len(phase.RelevantContext) = %d, want 1", got)
	}
}

// TestPlanProtoRoundTripNewFields asserts every new professional plan/phase field
// survives a model -> proto -> model round trip (the conversion drift guard).
func TestPlanProtoRoundTripNewFields(t *testing.T) {
	t.Parallel()
	in := planmodel.Plan{
		Title:                   "RT",
		ProblemStatement:        "prob",
		TargetOutcome:           "out",
		Assumptions:             "assume",
		TechnicalApproach:       "approach",
		ValidationStrategy:      "valstrat",
		FinalValidationCommands: []string{"cmd a", "cmd b"},
		RisksHazards:            "risk",
		ProhibitedApproaches:    "prohib",
		WorkPosture:             planmodel.WorkPostureBrownfield,
		WorkPostureSource:       planmodel.WorkPostureSourceServiceMaturity,
		WorkPostureDetail:       "pilot",
		ImportProvenance:        &planmodel.ImportProvenance{SourcePath: "x.md", OriginalFormat: "legacy_markdown"},
		PreservedLegacySections: []planmodel.LegacySection{
			{Heading: "Old", Content: "body", PreservationReason: "unmapped_legacy_section"},
		},
		Phases: []planmodel.Phase{{
			Title:           "P1",
			AffectedAreas:   []string{"a", "b"},
			Steps:           []string{"s1", "s2"},
			ExpectedOutputs: []string{"o1"},
			Validation:      "go test",
			HandoffNotes:    "handoff",
			RisksHazards:    []string{"r1"},
		}},
	}
	got := PlanFromProto(PlanToProto(in))

	if got.ProblemStatement != in.ProblemStatement || got.TargetOutcome != in.TargetOutcome ||
		got.Assumptions != in.Assumptions || got.TechnicalApproach != in.TechnicalApproach ||
		got.ValidationStrategy != in.ValidationStrategy || got.RisksHazards != in.RisksHazards ||
		got.ProhibitedApproaches != in.ProhibitedApproaches {
		t.Fatalf("plan prose fields did not round-trip: %+v", got)
	}
	if len(got.FinalValidationCommands) != 2 || got.FinalValidationCommands[1] != "cmd b" {
		t.Fatalf("final_validation_commands round-trip: %v", got.FinalValidationCommands)
	}
	if got.WorkPosture != planmodel.WorkPostureBrownfield || got.WorkPostureSource != planmodel.WorkPostureSourceServiceMaturity || got.WorkPostureDetail != "pilot" {
		t.Fatalf("work posture round-trip: %q/%q/%q", got.WorkPosture, got.WorkPostureSource, got.WorkPostureDetail)
	}
	if got.ImportProvenance == nil || got.ImportProvenance.SourcePath != "x.md" {
		t.Fatalf("import provenance round-trip: %+v", got.ImportProvenance)
	}
	if len(got.PreservedLegacySections) != 1 || got.PreservedLegacySections[0].Heading != "Old" {
		t.Fatalf("preserved legacy round-trip: %+v", got.PreservedLegacySections)
	}
	ph := got.Phases[0]
	if len(ph.AffectedAreas) != 2 || len(ph.Steps) != 2 || len(ph.ExpectedOutputs) != 1 ||
		ph.Validation != "go test" || ph.HandoffNotes != "handoff" || len(ph.RisksHazards) != 1 {
		t.Fatalf("phase fields round-trip: %+v", ph)
	}
}
