package optimize

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

func validRequest() RoleRequest {
	cases := make([]string, MinFixedCases)
	for i := range cases {
		cases[i] = fmt.Sprintf("case-%02d", i+1)
	}
	return RoleRequest{
		RequestID:           "lm6-chat-default-001",
		Target:              "ollama-role",
		Role:                RoleChatDefault,
		CandidateModels:     []string{"granite4:3b", "qwen3.5:9b", "minicpm:2b"},
		FixedSuiteID:        "chat-default-caller-suite-v1",
		FixedCaseIDs:        cases,
		EvaluationReceiptID: "model-eval/run-123",
		WinnerModel:         "granite4:3b",
	}
}

func TestInspectCarriesCurrentRightsEvidenceForUninstalledCandidates(t *testing.T) {
	contextPath := filepath.Join(t.TempDir(), "rights-context.json")
	writeContext := func(version, revenue string) {
		t.Helper()
		document := map[string]any{"schema_version": "1", "context": sharedpolicy.RightsContext{
			ID: "fixture-operator-context", Version: version, EntityScope: "operator-group", RevenueStatus: revenue, AsOf: "2026-11-14", Source: "test_fixture",
		}}
		raw, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(contextPath, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	artifact := &sharedpolicy.ArtifactRights{
		ArtifactID: "source:model-a", Revision: "v1", SHA256: strings.Repeat("a", 64), LicenseID: "fixture", Classification: sharedpolicy.RightsUnrestricted,
		Evidence: sharedpolicy.RightsEvidence{URL: "https://example.invalid/license", Revision: "v1", SHA256: strings.Repeat("b", 64), ObservedAt: "2026-11-14"},
	}
	models := map[string]policy.Model{}
	for _, name := range validRequest().CandidateModels {
		models[name] = policy.Model{Rights: artifact, Capabilities: []string{"generate", "chat"}, DefaultEligible: true}
	}
	p := policy.Policy{Roles: map[string]policy.Role{RoleChatDefault: {RequiredCapabilities: []string{"generate", "chat"}}}, Models: models}
	getenv := func(key string) string {
		if key == sharedpolicy.RightsContextPathEnv {
			return contextPath
		}
		return ""
	}
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	writeContext("fixture-v1", "pre_revenue")
	first, err := inspectAt(t.Context(), validRequest(), nil, p, getenv, now)
	if err != nil {
		t.Fatal(err)
	}
	writeContext("fixture-v2", "pre_revenue")
	changed, err := inspectAt(t.Context(), validRequest(), nil, p, getenv, now)
	if err != nil {
		t.Fatal(err)
	}
	writeContext("fixture-v1", "pre_revenue")
	restored, err := inspectAt(t.Context(), validRequest(), nil, p, getenv, now)
	if err != nil {
		t.Fatal(err)
	}
	for i := range first.Candidates {
		a, b, c := first.Candidates[i], changed.Candidates[i], restored.Candidates[i]
		if a.Installed || a.Rights != "allowed" || a.RightsEvidence.Status != "allowed" || a.RightsEvidence.DecisionSHA256 == "" || a.RightsEvidence.ContextSHA256 == "" || a.RightsEvidence.ContextID != "fixture-operator-context" || a.RightsEvidence.ContextVersion != "fixture-v1" || a.RightsEvidence.ArtifactID != artifact.ArtifactID || a.RightsEvidence.ArtifactSHA256 != artifact.SHA256 || a.RightsEvidence.EvaluatedAt == "" {
			t.Fatalf("uninstalled candidate lacks source rights receipt: %#v", a)
		}
		if b.RightsEvidence.DecisionSHA256 == a.RightsEvidence.DecisionSHA256 || b.RightsEvidence.ContextSHA256 == a.RightsEvidence.ContextSHA256 {
			t.Fatalf("changed context did not change current rights receipt: before=%#v after=%#v", a.RightsEvidence, b.RightsEvidence)
		}
		if c.RightsEvidence.DecisionSHA256 != a.RightsEvidence.DecisionSHA256 || c.RightsEvidence.ContextSHA256 != a.RightsEvidence.ContextSHA256 {
			t.Fatalf("restored context did not restore receipt: original=%#v restored=%#v", a.RightsEvidence, c.RightsEvidence)
		}
	}
}

func TestRoleRequestValidateAcceptsLM6Shape(t *testing.T) {
	if err := validRequest().Validate(); err != nil {
		t.Fatalf("valid LM6 request rejected: %v", err)
	}
}

func TestRoleRequestValidateAcceptsPolicyOwnedNarrowRoleNames(t *testing.T) {
	r := validRequest()
	r.Role = "classify.routing"
	if err := r.Validate(); err != nil {
		t.Fatalf("valid policy-owned narrow role rejected: %v", err)
	}
}

func TestRoleRequestValidateAllowsOmittingPreEvaluationReceipt(t *testing.T) {
	r := validRequest()
	r.EvaluationReceiptID = ""
	if err := r.Validate(); err != nil {
		t.Fatalf("pre-evaluation inspection rejected without a receipt ID: %v", err)
	}
}

func TestInspectNeverQualifiesAnOmittedEvaluationReceipt(t *testing.T) {
	r := validRequest()
	r.Role = "classify.routing"
	r.EvaluationReceiptID = ""
	p := policy.Policy{Roles: map[string]policy.Role{"classify.routing": {RequiredCapabilities: []string{"classify"}}}}
	receipt, err := Inspect(t.Context(), r, nil, p, func(string) string { return "" })
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if receipt.EvaluationVerified || receipt.EvaluationReceiptID != "" || !strings.Contains(strings.Join(receipt.Refusals, ";"), "evaluation receipt is not verified") {
		t.Fatalf("omitted receipt was treated as evaluation evidence: %#v", receipt)
	}
}

func TestInspectRefusesUnknownPolicyRole(t *testing.T) {
	r := validRequest()
	r.Role = "caller.invented"
	receipt, err := Inspect(t.Context(), r, nil, policy.Policy{Roles: map[string]policy.Role{}}, func(string) string { return "" })
	if err != nil {
		t.Fatalf("Inspect() error = %v, want explicit refusal receipt", err)
	}
	if receipt.Status != "refused" || len(receipt.Candidates) != 0 || !strings.Contains(strings.Join(receipt.Refusals, ";"), "not present in the authoritative Ollama model policy") {
		t.Fatalf("unknown role was not refused by owner policy: %#v", receipt)
	}
}

func TestDecodeRoleRequestRejectsCallerSuppliedRoleCapabilities(t *testing.T) {
	_, err := decodeRoleRequest(bytes.NewBufferString(`{"role":"chat.default","role_capabilities":["generate"]}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("decodeRoleRequest() error = %v, want caller-supplied role metadata rejected", err)
	}
}

func TestRoleRequestValidateFailsClosedOnIncompleteOrAmbiguousWork(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*RoleRequest)
		want   string
	}{
		{"wrong target", func(r *RoleRequest) { r.Target = "desktop-dp1" }, "unsupported"},
		{"missing role", func(r *RoleRequest) { r.Role = " " }, "unsupported"},
		{"candidate count", func(r *RoleRequest) { r.CandidateModels = r.CandidateModels[:2] }, "exactly 3"},
		{"duplicate candidate", func(r *RoleRequest) { r.CandidateModels[2] = r.CandidateModels[1] }, "duplicated"},
		{"case minimum", func(r *RoleRequest) { r.FixedCaseIDs = r.FixedCaseIDs[:19] }, "at least 20"},
		{"duplicate case", func(r *RoleRequest) { r.FixedCaseIDs[1] = r.FixedCaseIDs[0] }, "duplicated"},
		{"winner outside candidates", func(r *RoleRequest) { r.WinnerModel = "unrequested:latest" }, "one of the requested candidates"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := validRequest()
			tt.mutate(&r)
			err := r.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestNewRefusalCarriesClosedOwnerEffectStates(t *testing.T) {
	r := NewRefusal(validRequest(), "promotion owner does not expose a restart-safe policy transaction")
	if r.Status != "refused" || r.EvaluationVerified || r.DiskAdmission != "not-admitted" || r.Promotion != "refused" || r.Cleanup != "refused" {
		t.Fatalf("unsafe receipt state: %#v", r)
	}
	if r.ReceiptID == "" || r.RequestID == "" || len(r.Refusals) != 1 {
		t.Fatalf("receipt is not attributable: %#v", r)
	}
}

func TestMissingCapabilitiesReportsRoleRequirementsNotProvidedByCandidate(t *testing.T) {
	got := missingCapabilities([]string{"generate", "chat", "summarize"}, []string{"generate", "chat"})
	if len(got) != 1 || got[0] != "summarize" {
		t.Fatalf("missingCapabilities() = %v, want [summarize]", got)
	}
}
