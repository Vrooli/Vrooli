package acceptance

import (
	"encoding/json"
	"testing"

	"audio-tools/internal/conformance"
)

func completeRun() conformance.Run {
	run := conformance.Run{SchemaVersion: conformance.SchemaVersion, RunID: "run-1", Lane: conformance.LaneRealtime, DurationMs: 60 * 60 * 1000,
		Cell: conformance.Cell{EngineID: "whisper-local", ModelID: "model-1", Strategy: "vad", Policy: "default"}, Code: conformance.Code{CapturePackage: "capture-1", Server: "server-1"}}
	for _, name := range conformance.RequiredAssertions(conformance.LaneRealtime) {
		run.Assertions = append(run.Assertions, conformance.Assertion{Name: name, Outcome: conformance.OutcomePassed, Detail: "observed"})
	}
	return run
}

func TestClaimsForRealtimeRunRequiresRealDurationAndCompleteAssertions(t *testing.T) {
	run := completeRun()
	claims := ClaimsForRealtimeRun(run)
	if claims["dictation-trust"] == nil || claims["provider-evidence"] == nil {
		t.Fatalf("claims = %#v", claims)
	}
	run.Lane = conformance.LaneAccelerated
	if got := ClaimsForRealtimeRun(run); got != nil {
		t.Fatalf("accelerated run claims = %#v", got)
	}
}

func TestClaimsForRealtimeRunRequiresThirtyLatencyAttempts(t *testing.T) {
	run := completeRun()
	if _, ok := ClaimsForRealtimeRun(run)["interactive-latency"]; ok {
		t.Fatal("single-turn realtime run must not claim interactive latency")
	}
	run.Turns = 30
	claim, ok := ClaimsForRealtimeRun(run)["interactive-latency"].(map[string]any)
	if !ok || claim["in_band"] != true || claim["attempts"] != 30 {
		t.Fatalf("latency claim = %#v", ClaimsForRealtimeRun(run)["interactive-latency"])
	}
}

func TestClaimsForRealtimeRunSeparatesTransportFromQuality(t *testing.T) {
	run := completeRun()
	for i := range run.Assertions {
		if run.Assertions[i].Name == "word_error_rate_stable" {
			run.Assertions[i].Outcome = conformance.OutcomeFailed
			run.Assertions[i].Detail = "recognition error; transport remained complete"
		}
	}
	claims := ClaimsForRealtimeRun(run)
	if claims["dictation-trust"] == nil || claims["provider-evidence"] == nil {
		t.Fatalf("independent transport claims were suppressed by WER: %#v", claims)
	}
	if claims["interactive-latency"] != nil {
		t.Fatal("single-turn run must not claim interactive latency")
	}
}

func TestClaimsForRealtimeRunDoesNotClaimTransportAfterInvariantFailure(t *testing.T) {
	run := completeRun()
	for i := range run.Assertions {
		if run.Assertions[i].Name == "zero_duplicate_committed_segments" {
			run.Assertions[i].Outcome = conformance.OutcomeFailed
		}
	}
	if claims := ClaimsForRealtimeRun(run); claims != nil {
		t.Fatalf("transport failure must suppress claims, got %#v", claims)
	}
}

func TestMachineJSONCarriesExactIdentity(t *testing.T) {
	run := completeRun()
	payload, err := MachineJSON(run, "D1", Cohort, Policy, ClaimsForRealtimeRun(run))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"target_revision": TargetRevision, "method": Method, "cohort": Cohort, "build": "server-1", "device_profile": "D1", "policy_revision": Policy, "corpus_revision": Cohort} {
		if got[key] != want {
			t.Fatalf("%s = %v, want %s", key, got[key], want)
		}
	}
}

func TestMachineJSONRejectsMissingBuildIdentity(t *testing.T) {
	run := completeRun()
	run.Code.Server = ""
	if _, err := MachineJSON(run, "D1", Cohort, Policy, nil); err == nil {
		t.Fatal("missing build identity must be rejected")
	}
}

func TestMachineJSONForCellCarriesProviderIdentityAndRejectsIncompleteCells(t *testing.T) {
	payload, err := MachineJSONForCell("build-1", "D1", Cohort, Policy, "whisper-local", "whisper-medium", "vad_segment", map[string]any{
		"corpus-quality": map[string]any{"in_band": false},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	if got["target_revision"] != TargetRevision || got["method"] != Method {
		t.Fatalf("envelope identity = %#v", got)
	}
	cell := got["provider_cell"].(map[string]any)
	if cell["engine_id"] != "whisper-local" || cell["model_id"] != "whisper-medium" {
		t.Fatalf("provider cell = %#v", cell)
	}
	if _, err := MachineJSONForCell("build-1", "D1", Cohort, Policy, "", "model", "strategy", nil); err == nil {
		t.Fatal("missing provider identity must be rejected")
	}
}
