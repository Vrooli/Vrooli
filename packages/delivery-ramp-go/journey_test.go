package deliveryramp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestJourneyContractUsesOracleFieldNames(t *testing.T) {
	value := JourneyResult{
		SchemaVersion:   JourneySchemaVersion,
		EvidenceVersion: JourneyEvidenceVersion,
		SmokeTestID:     "smoke-1",
		ScenarioName:    "hello-desktop",
		Capability:      "hello-desktop",
		PlanID:          "hello-desktop.baseline.v2",
		Profile:         "normal-review",
		Platform:        "linux-amd64",
		Disposition:     DispositionPass,
		Steps:           []JourneyStep{{ID: "launch", Name: "launch", Action: "launch", Disposition: StepPassed, Evidence: []EvidenceReference{{ID: "capture-1", Kind: "screenshot"}}}},
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, field := range []string{`"schema_version"`, `"evidence_version"`, `"smoke_test_id"`, `"scenario_name"`, `"capability"`, `"plan_id"`, `"profile"`, `"platform"`, `"disposition"`, `"steps"`, `"evidence"`} {
		if !strings.Contains(text, field) {
			t.Fatalf("marshaled journey omitted oracle field %s: %s", field, text)
		}
	}
}

func TestReadinessAndSettlePoliciesRoundTripMilliseconds(t *testing.T) {
	input := struct {
		Readiness ReadinessPolicy `json:"readiness"`
		Settle    SettlePolicy    `json:"settle"`
	}{
		Readiness: ReadinessPolicy{ID: "ready", Timeout: 12 * time.Second, PollInterval: 100 * time.Millisecond, StabilityCount: 2, Cancellation: "context_or_timeout"},
		Settle:    SettlePolicy{ID: "settle", Minimum: 500 * time.Millisecond, Maximum: 2 * time.Second, PollInterval: 100 * time.Millisecond, Cancellation: "context_or_timeout"},
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Readiness ReadinessPolicy `json:"readiness"`
		Settle    SettlePolicy    `json:"settle"`
	}
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatal(err)
	}
	if output.Readiness.Timeout != input.Readiness.Timeout || output.Settle.Maximum != input.Settle.Maximum || output.Readiness.StabilityCount != 2 {
		t.Fatalf("policy round trip changed durations or stability: %#v", output)
	}
}

func TestUnavailableRequiresRecoveryInformation(t *testing.T) {
	if err := (DispositionResult{Disposition: DispositionUnavailable}).Validate(); err == nil {
		t.Fatal("unavailable result without recovery information was accepted")
	}
	result := DispositionResult{Disposition: DispositionUnavailable, MissingCapability: "android.adb", NextAction: "install Android SDK platform tools"}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestUnavailableAndUnsupportedCannotBecomePass(t *testing.T) {
	for _, current := range []Disposition{DispositionUnavailable, DispositionUnsupported} {
		if err := ValidatePromotion(current, DispositionPass); err == nil {
			t.Fatalf("%s was promoted to pass", current)
		}
	}
	if err := ValidatePromotion(DispositionFailed, DispositionPass); err != nil {
		t.Fatalf("failed-to-pass promotion should remain available for a verified rerun: %v", err)
	}
}
