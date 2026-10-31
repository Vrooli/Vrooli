package checks

import "testing"

// stubAutoHealConfig lets a test pin the policy for one check id.
type stubAutoHealConfig struct {
	enabled bool
	policy  string
}

func (s stubAutoHealConfig) IsCheckEnabled(string) bool    { return true }
func (s stubAutoHealConfig) IsAutoHealEnabled(string) bool { return s.enabled }
func (s stubAutoHealConfig) GetAutoHealOn(string) string   { return s.policy }

func signatureWarning() Result {
	return Result{
		CheckID: "scenario-web-console",
		Status:  StatusWarning,
		Details: map[string]interface{}{
			"healableDegradation": true,
			"recommendedAction":   "restart",
			"rootCause":           "ui-unreachable",
		},
	}
}

func bareWarning() Result {
	return Result{
		CheckID: "scenario-web-console",
		Status:  StatusWarning,
		Details: map[string]interface{}{},
	}
}

func TestHasHealableDegradationRequiresBothMarkerAndAction(t *testing.T) {
	if !hasHealableDegradation(signatureWarning()) {
		t.Error("marker plus action must qualify")
	}
	if hasHealableDegradation(bareWarning()) {
		t.Error("a bare warning must not qualify")
	}

	// The marker alone is not enough: a check must also name what to run.
	missingAction := Result{
		CheckID: "x",
		Status:  StatusWarning,
		Details: map[string]interface{}{"healableDegradation": true},
	}
	if hasHealableDegradation(missingAction) {
		t.Error("marker without a recommended action must not qualify")
	}

	missingMarker := Result{
		CheckID: "x",
		Status:  StatusWarning,
		Details: map[string]interface{}{"recommendedAction": "restart"},
	}
	if hasHealableDegradation(missingMarker) {
		t.Error("action without the explicit marker must not qualify")
	}

	if hasHealableDegradation(Result{Status: StatusWarning}) {
		t.Error("nil details must not panic or qualify")
	}
}

func TestShouldTriggerAutoHealPolicies(t *testing.T) {
	critical := Result{CheckID: "scenario-web-console", Status: StatusCritical}

	tests := []struct {
		name   string
		policy string
		result Result
		want   bool
	}{
		{"critical-only heals critical", AutoHealOnCritical, critical, true},
		{"critical-only ignores signature warning", AutoHealOnCritical, signatureWarning(), false},
		{"critical-only ignores bare warning", AutoHealOnCritical, bareWarning(), false},

		{"default heals critical", AutoHealOnCriticalSignature, critical, true},
		{"default heals signature warning", AutoHealOnCriticalSignature, signatureWarning(), true},
		{"default ignores bare warning", AutoHealOnCriticalSignature, bareWarning(), false},

		{"widest heals bare warning", AutoHealOnWarningCritical, bareWarning(), true},
		{"widest heals critical", AutoHealOnWarningCritical, critical, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &Registry{config: stubAutoHealConfig{enabled: true, policy: tt.policy}}
			if got := registry.shouldTriggerAutoHeal(tt.result); got != tt.want {
				t.Errorf("policy %q: got %v, want %v", tt.policy, got, tt.want)
			}
		})
	}
}

// The explicit veto must still outrank every widening: fallback and
// low-confidence results set autoHealEligible=false and must never heal.
func TestAutoHealEligibleVetoOutranksPolicy(t *testing.T) {
	vetoed := signatureWarning()
	vetoed.Details["autoHealEligible"] = false

	for _, policy := range []string{AutoHealOnCritical, AutoHealOnCriticalSignature, AutoHealOnWarningCritical} {
		registry := &Registry{config: stubAutoHealConfig{enabled: true, policy: policy}}
		if registry.shouldTriggerAutoHeal(vetoed) {
			t.Errorf("policy %q must respect the autoHealEligible veto", policy)
		}
	}
}
