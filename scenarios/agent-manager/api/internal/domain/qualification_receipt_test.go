package domain

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func fullyQualifiedReceipt() *QualificationReceipt {
	return &QualificationReceipt{
		Route:                  "code.flatrate",
		RequestedRunner:        "opencode",
		RequestedRoleRef:       "code.flatrate",
		EffectiveRunner:        "opencode",
		EffectiveModel:         "opencode-go/deepseek-v4.1-flash",
		EffectiveEffort:        "medium",
		PassedControlArgs:      []string{"--model", "opencode-go/deepseek-v4.1-flash"},
		ProviderAcknowledgment: []string{"model=opencode-go/deepseek-v4.1-flash"},
		CatalogDigest:          "catalog-sha",
		PolicyDigest:           "policy-sha",
		RuntimeVersion:         "opencode-go/1.2.3",
		RunID:                  "run-1",
		OperationID:            "op-1",
		AcceptedOutput:         true,
		Usage: QualificationUsage{
			State:           QualificationUsageUnknownReserved,
			ReservedUnknown: true,
		},
		Limitations: []string{"exact account allowance unknown"},
	}
}

func TestAdmitDependentDelegation_AdmitsMatchingQualifiedRoute(t *testing.T) {
	req := DependentDelegationRequest{
		Runner: "opencode",
		Model:  "opencode-go/deepseek-v4.1-flash",
		Effort: "medium",
	}
	if err := AdmitDependentDelegation(fullyQualifiedReceipt(), req); err != nil {
		t.Fatalf("expected matching qualified route to be admitted, got %v", err)
	}
}

func TestAdmitDependentDelegation_ClosesOnMissingEvidence(t *testing.T) {
	req := DependentDelegationRequest{
		Runner: "opencode",
		Model:  "opencode-go/deepseek-v4.1-flash",
		Effort: "medium",
	}
	cases := []struct {
		name    string
		receipt *QualificationReceipt
	}{
		{"nil receipt", nil},
		{"no accepted output", func() *QualificationReceipt {
			r := fullyQualifiedReceipt()
			r.AcceptedOutput = false
			return r
		}()},
		{"no route", func() *QualificationReceipt {
			r := fullyQualifiedReceipt()
			r.Route = "  "
			return r
		}()},
		{"no runtime identity", func() *QualificationReceipt {
			r := fullyQualifiedReceipt()
			r.RuntimeVersion = ""
			return r
		}()},
		{"empty effective runner", func() *QualificationReceipt {
			r := fullyQualifiedReceipt()
			r.EffectiveRunner = ""
			return r
		}()},
		{"empty effective model", func() *QualificationReceipt {
			r := fullyQualifiedReceipt()
			r.EffectiveModel = ""
			return r
		}()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := AdmitDependentDelegation(tc.receipt, req); err == nil {
				t.Fatalf("expected dependent delegation to stay closed for %s", tc.name)
			}
		})
	}
}

func TestAdmitDependentDelegation_ClosesOnIdentityMismatch(t *testing.T) {
	cases := []struct {
		name string
		req  DependentDelegationRequest
	}{
		{"runner", DependentDelegationRequest{Runner: "codex", Model: "opencode-go/deepseek-v4.1-flash", Effort: "medium"}},
		{"model", DependentDelegationRequest{Runner: "opencode", Model: "openrouter/deepseek/deepseek-v4.1-flash", Effort: "medium"}},
		{"effort", DependentDelegationRequest{Runner: "opencode", Model: "opencode-go/deepseek-v4.1-flash", Effort: "high"}},
		{"requested effort omitted", DependentDelegationRequest{Runner: "opencode", Model: "opencode-go/deepseek-v4.1-flash"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := AdmitDependentDelegation(fullyQualifiedReceipt(), tc.req); err == nil {
				t.Fatalf("expected dependent delegation to stay closed on %s mismatch", tc.name)
			}
		})
	}
}

// A live parent may delegate to a strictly cheaper model tier at any effort
// (a Sol supervisor starting a Luna repair run), never to a pricier or
// unknown model.
func TestAdmitLiveDependentDelegation_ModelTiers(t *testing.T) {
	parent := &RunAdmission{EffectiveRunner: "codex", EffectiveModel: "gpt-6-sol", EffectiveEffort: "medium", RuntimeVersion: "codex/1", PassedControlArgs: []string{"--model", "gpt-6-sol"}}
	cases := []struct {
		name  string
		model string
		// effort is the child's requested effort.
		effort string
		admit  bool
	}{
		{"same model narrower effort", "gpt-6-sol", "low", true},
		{"same model wider effort", "gpt-6-sol", "high", false},
		{"cheaper tier any effort", "gpt-6-luna", "high", true},
		{"unknown model", "gpt-6-terra", "low", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := AdmitLiveDependentDelegation(parent, DependentDelegationRequest{Runner: "codex", Model: tc.model, Effort: tc.effort})
			if (err == nil) != tc.admit {
				t.Fatalf("admit=%t, err=%v", tc.admit, err)
			}
		})
	}
	cheaperParent := *parent
	cheaperParent.EffectiveModel = "gpt-6-luna"
	if err := AdmitLiveDependentDelegation(&cheaperParent, DependentDelegationRequest{Runner: "codex", Model: "gpt-6-sol", Effort: "low"}); err == nil {
		t.Fatal("a Luna parent delegated to the pricier Sol tier")
	}
}

func TestNewQualificationReceipt_CopiesAdmissionLayersAndLeavesLiveFieldsEmpty(t *testing.T) {
	admission := &RunAdmission{
		RequestedRunner:        "opencode",
		RequestedRoleRef:       "code.flatrate",
		EffectiveRunner:        "opencode",
		EffectiveModel:         "opencode-go/deepseek-v4.1-flash",
		EffectiveEffort:        "medium",
		PassedControlArgs:      []string{"--model", "opencode-go/deepseek-v4.1-flash"},
		TranslationDiagnostics: []string{"effort translated to medium"},
		CatalogDigest:          "catalog-sha",
		PolicyDigest:           "policy-sha",
		PolicyPath:             "/tmp/policy.json",
		RuntimeVersion:         "opencode-go/1.2.3",
	}
	receipt := NewQualificationReceipt("code.flatrate", admission)
	if receipt == nil {
		t.Fatal("expected receipt from non-nil admission")
	}
	if receipt.Route != "code.flatrate" || receipt.EffectiveModel != admission.EffectiveModel {
		t.Fatalf("expected route and effective model copied, got %+v", receipt)
	}
	if receipt.AcceptedOutput || receipt.RunID != "" || len(receipt.Limitations) != 0 {
		t.Fatalf("live-only fields must stay empty until observed, got %+v", receipt)
	}
	// The receipt must not alias the admission's slices.
	receipt.PassedControlArgs[0] = "mutated"
	if admission.PassedControlArgs[0] != "--model" {
		t.Fatal("expected passed control args to be copied, not aliased")
	}
}

func TestCaptureQualificationReceipt_DerivesFromAdmissionAndAppliesEvidence(t *testing.T) {
	captured := time.Date(2026, 9, 12, 8, 20, 0, 0, time.UTC)
	admission := &RunAdmission{
		RequestedRunner:        "opencode",
		RequestedRoleRef:       "code.flatrate",
		EffectiveRunner:        "opencode",
		EffectiveModel:         "opencode-go/deepseek-v4.1-flash",
		EffectiveEffort:        "medium",
		PassedControlArgs:      []string{"--model", "opencode-go/deepseek-v4.1-flash"},
		TranslationDiagnostics: []string{"effort translated to medium"},
		CatalogDigest:          "catalog-sha",
		PolicyDigest:           "policy-sha",
	}
	evidence := QualificationEvidence{
		Route:                  "code.flatrate",
		RuntimeVersion:         "opencode-go/1.2.3",
		ProviderAcknowledgment: []string{"model=opencode-go/deepseek-v4.1-flash"},
		AcceptedOutput:         true,
		Usage: QualificationUsage{
			State:           QualificationUsageUnknownReserved,
			ReservedUnknown: true,
		},
		Limitations: []string{"exact account allowance unknown"},
		RunID:       "run-1",
		OperationID: "op-1",
		CapturedAt:  captured,
	}

	receipt := CaptureQualificationReceipt(admission, evidence)
	if receipt == nil {
		t.Fatal("expected receipt from non-nil admission")
	}
	if receipt.Route != "code.flatrate" ||
		receipt.EffectiveRunner != admission.EffectiveRunner ||
		receipt.EffectiveModel != admission.EffectiveModel ||
		receipt.RequestedRoleRef != admission.RequestedRoleRef {
		t.Fatalf("expected route and admission layers to carry over, got %+v", receipt)
	}
	if receipt.RuntimeVersion != "opencode-go/1.2.3" || !receipt.AcceptedOutput ||
		receipt.RunID != "run-1" || receipt.OperationID != "op-1" ||
		!receipt.CapturedAt.Equal(captured) {
		t.Fatalf("expected live evidence applied, got %+v", receipt)
	}
	if receipt.Usage.State != QualificationUsageUnknownReserved || !receipt.Usage.ReservedUnknown {
		t.Fatalf("expected unknown usage preserved as reserved, got %+v", receipt.Usage)
	}
	// Captured evidence must not alias the caller's slices.
	receipt.ProviderAcknowledgment[0] = "mutated"
	receipt.Limitations[0] = "mutated"
	if evidence.ProviderAcknowledgment[0] != "model=opencode-go/deepseek-v4.1-flash" ||
		evidence.Limitations[0] != "exact account allowance unknown" {
		t.Fatal("expected evidence slices to be copied, not aliased")
	}
}

func TestCaptureQualificationReceipt_NilAdmissionReturnsNil(t *testing.T) {
	if got := CaptureQualificationReceipt(nil, QualificationEvidence{Route: "code.flatrate"}); got != nil {
		t.Fatalf("expected nil receipt for nil admission, got %+v", got)
	}
}

func TestQualificationReceipt_MissingIdentitiesOrdersGaps(t *testing.T) {
	got := (&QualificationReceipt{}).MissingIdentities()
	want := []string{"acceptedOutput", "route", "runtimeVersion", "effectiveRunner", "effectiveModel"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("missing identities = %#v, want %#v", got, want)
	}
	if missing := fullyQualifiedReceipt().MissingIdentities(); len(missing) != 0 {
		t.Fatalf("fully qualified receipt must have no missing identities, got %#v", missing)
	}
	var nilReceipt *QualificationReceipt
	if got := nilReceipt.MissingIdentities(); !reflect.DeepEqual(got, []string{"receipt"}) {
		t.Fatalf("nil receipt missing identities = %#v", got)
	}
}

func TestQualificationReceipt_PartialReceiptCannotBeReportedComplete(t *testing.T) {
	partial := fullyQualifiedReceipt()
	partial.ProviderAcknowledgment = nil // allowed to be absent
	partial.RuntimeVersion = ""
	missing := partial.MissingIdentities()
	if !reflect.DeepEqual(missing, []string{"runtimeVersion"}) {
		t.Fatalf("expected only runtimeVersion missing, got %#v", missing)
	}
	req := DependentDelegationRequest{Runner: "opencode", Model: "opencode-go/deepseek-v4.1-flash", Effort: "medium"}
	if err := AdmitDependentDelegation(partial, req); err == nil {
		t.Fatal("expected a receipt with no runtime identity to stay closed")
	}
}

func TestQualificationReceipt_PersistsInRunConfigResolvedSnapshot(t *testing.T) {
	cfg := &RunConfig{
		RunnerType: "opencode",
		Admission: &RunAdmission{
			EffectiveRunner: "opencode",
			EffectiveModel:  "opencode-go/deepseek-v4.1-flash",
			EffectiveEffort: "medium",
			Receipt:         fullyQualifiedReceipt(),
		},
	}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded RunConfig
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Admission == nil || decoded.Admission.Receipt == nil {
		t.Fatal("expected receipt to round-trip through the resolved config snapshot")
	}
	if decoded.Admission.Receipt.Route != "code.flatrate" {
		t.Fatalf("expected route to survive round-trip, got %q", decoded.Admission.Receipt.Route)
	}
	if decoded.Admission.Receipt.Usage.State != QualificationUsageUnknownReserved || !decoded.Admission.Receipt.Usage.ReservedUnknown {
		t.Fatalf("expected unknown usage to persist as reserved, got %+v", decoded.Admission.Receipt.Usage)
	}
	if strings.TrimSpace(decoded.Admission.Receipt.RuntimeVersion) == "" {
		t.Fatal("expected runtime identity to survive round-trip")
	}
}
