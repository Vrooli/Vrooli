package protoconv

import (
	"reflect"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

func TestRunAdmissionRoundTrip(t *testing.T) {
	want := &domain.RunAdmission{
		RequestedRunner:   "codex",
		RequestedModel:    "gpt-5.6-luna",
		RequestedRoleRef:  "code.default",
		RequestedEffort:   "high",
		RequestedTimeout:  45 * time.Minute,
		RequestedMaxTurns: 12,
		RequestedGoalMode: "until",
		EffectiveRunner:   string(domain.RunnerTypeCodex),
		EffectiveModel:    "gpt-5.6-luna",
		EffectiveEffort:   string(domain.EffortMedium),
		EffectiveTimeout:  time.Hour,
		EffectiveMaxTurns: 20,
		EffectiveUntil:    "done",
		CatalogDigest:     "catalog-abc",
		PolicyDigest:      "policy-def",
		PolicyPath:        "resources/codex/model-policy.json",
		SelectionReason:   "role_policy",
		PassedControlArgs: []string{"-m", "gpt-5.6-luna", "-c", "model_reasoning_effort=medium"},
		TranslationDiagnostics: []string{
			"model: gpt-5.6-luna -> -m gpt-5.6-luna",
		},
		RuntimeVersion:         "codex-cli 0.55.0",
		ProviderAcknowledgment: []string{"provider accepted model=gpt-5.6-luna effort=medium"},
	}

	got := RunAdmissionFromProto(RunAdmissionToProto(want))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("admission round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

// TestRunAdmissionRuntimeAndProviderLayersStayTruthfullyEmpty verifies the two
// live-only layers survive the round trip as empty rather than being backfilled
// from the requested, effective or passed layers. A reader must be able to tell
// "not observed yet" from "observed and equal".
func TestRunAdmissionRuntimeAndProviderLayersStayTruthfullyEmpty(t *testing.T) {
	want := &domain.RunAdmission{
		EffectiveRunner:   string(domain.RunnerTypeCodex),
		EffectiveModel:    "gpt-5.6-luna",
		EffectiveEffort:   string(domain.EffortMedium),
		PassedControlArgs: []string{"-m", "gpt-5.6-luna", "-c", "model_reasoning_effort=medium"},
	}

	got := RunAdmissionFromProto(RunAdmissionToProto(want))
	if got.RuntimeVersion != "" {
		t.Fatalf("runtime version should stay empty until a live launch observes it, got %q", got.RuntimeVersion)
	}
	if len(got.ProviderAcknowledgment) != 0 {
		t.Fatalf("provider acknowledgment should stay empty until provider evidence exists, got %#v", got.ProviderAcknowledgment)
	}
	if got.Receipt != nil {
		t.Fatalf("qualification receipt should stay nil until a live probe exists, got %#v", got.Receipt)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("empty-layer round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

// TestRunAdmissionQualificationReceiptRoundTrip verifies the route-keyed
// qualification receipt survives the proto boundary intact, including the
// requested/effective/passed layers, provider acknowledgment, operation
// identity, accepted output and measured usage. The receipt is the evidence the
// dependent-delegation gate compares against a requested identity, so it must
// round-trip without loss.
func TestRunAdmissionQualificationReceiptRoundTrip(t *testing.T) {
	usage := domain.QualificationUsage{
		State:        domain.QualificationUsageMeasured,
		InputTokens:  100,
		OutputTokens: 200,
		CostUSD:      0.01,
	}
	want := &domain.RunAdmission{
		EffectiveRunner:        string(domain.RunnerTypeOpenCode),
		EffectiveModel:         "opencode-go/deepseek-v4.1-flash",
		EffectiveEffort:        string(domain.EffortMedium),
		RuntimeVersion:         "opencode 1.2.3",
		ProviderAcknowledgment: []string{"provider accepted model=opencode-go/deepseek-v4.1-flash"},
		Receipt: &domain.QualificationReceipt{
			Route:                  "code.flatrate",
			RequestedRunner:        string(domain.RunnerTypeOpenCode),
			RequestedModel:         "opencode-go/deepseek-v4.1-flash",
			RequestedRoleRef:       "code.flatrate",
			RequestedEffort:        string(domain.EffortMedium),
			EffectiveRunner:        string(domain.RunnerTypeOpenCode),
			EffectiveModel:         "opencode-go/deepseek-v4.1-flash",
			EffectiveEffort:        string(domain.EffortMedium),
			PassedControlArgs:      []string{"--model", "opencode-go/deepseek-v4.1-flash"},
			TranslationDiagnostics: []string{"model: opencode-go/deepseek-v4.1-flash -> --model opencode-go/deepseek-v4.1-flash"},
			ProviderAcknowledgment: []string{"provider accepted model=opencode-go/deepseek-v4.1-flash effort=medium"},
			CatalogDigest:          "catalog-abc",
			PolicyDigest:           "policy-def",
			PolicyPath:             "resources/opencode/model-policy.json",
			RuntimeVersion:         "opencode 1.2.3",
			RunID:                  "run-123",
			OperationID:            "op-456",
			AcceptedOutput:         true,
			Usage:                  usage,
			Limitations:            []string{"single bounded probe"},
			CapturedAt:             time.Date(2026, 9, 12, 9, 30, 0, 0, time.UTC),
		},
	}

	got := RunAdmissionFromProto(RunAdmissionToProto(want))
	if got.Receipt == nil {
		t.Fatalf("qualification receipt should survive the round trip")
	}
	if !got.Receipt.CapturedAt.Equal(want.Receipt.CapturedAt) {
		t.Fatalf("captured_at mismatch: got %s want %s", got.Receipt.CapturedAt, want.Receipt.CapturedAt)
	}
	got.Receipt.CapturedAt = want.Receipt.CapturedAt
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("qualification receipt round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}
	if err := domain.AdmitDependentDelegation(got.Receipt, domain.DependentDelegationRequest{
		Runner: string(domain.RunnerTypeOpenCode),
		Model:  "opencode-go/deepseek-v4.1-flash",
		Effort: string(domain.EffortMedium),
	}); err != nil {
		t.Fatalf("round-tripped receipt should admit its matching identity: %v", err)
	}
}
