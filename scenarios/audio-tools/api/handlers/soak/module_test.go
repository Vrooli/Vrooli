package soak

import (
	"os"
	"testing"

	"audio-tools/internal/acceptance"
	"audio-tools/internal/conformance"
	intexp "audio-tools/internal/experiment"

	"github.com/stretchr/testify/require"
)

func TestQualificationRunPassedRequiresEveryAssertion(t *testing.T) {
	run := conformance.Run{
		SchemaVersion: conformance.SchemaVersion,
		RunID:         "browser-soak-test",
		Lane:          conformance.LaneRealtime,
		Cell:          conformance.Cell{EngineID: "kyutai", ModelID: "test", Strategy: "passthrough", Policy: "default"},
		Code:          conformance.Code{CapturePackage: "sha256:capture", Server: "sha256:server"},
	}
	for _, name := range conformance.RequiredAssertions(conformance.LaneRealtime) {
		run.Assertions = append(run.Assertions, conformance.Measured(name, true, "focused qualification test"))
	}
	require.True(t, qualificationRunPassed(run, nil))

	run.Assertions[0].Outcome = conformance.OutcomeFailed
	require.False(t, qualificationRunPassed(run, nil))
	require.False(t, qualificationRunPassed(run, requireError{}))
}

func TestAcceptanceClaimsPassedScopesBrowserEvidenceToIndependentClaims(t *testing.T) {
	run := completeAcceptanceRun()
	for i := range run.Assertions {
		if run.Assertions[i].Name == "word_error_rate_stable" {
			run.Assertions[i].Outcome = conformance.OutcomeFailed
		}
	}
	claims := acceptance.ClaimsForRealtimeRun(run)
	if !acceptanceClaimsPassed(claims, nil) {
		t.Fatal("transport/provider claims should remain passable when only WER fails")
	}
	for i := range run.Assertions {
		if run.Assertions[i].Name == "zero_duplicate_committed_segments" {
			run.Assertions[i].Outcome = conformance.OutcomeFailed
		}
	}
	if acceptanceClaimsPassed(acceptance.ClaimsForRealtimeRun(run), nil) {
		t.Fatal("transport invariant failure must fail target-aware browser evidence")
	}
}

func TestClaimsAfterTrustCohortWithholdsTrustUntilThreeRuns(t *testing.T) {
	run := completeAcceptanceRun()
	claims := acceptance.ClaimsForRealtimeRun(run)
	for prior, wantTrust := range map[int]bool{0: false, 1: false, 2: true} {
		got := claimsAfterTrustCohort(claims, prior)
		_, hasTrust := got["dictation-trust"]
		_, hasProvider := got["provider-evidence"]
		if hasTrust != wantTrust || hasProvider != wantTrust {
			t.Fatalf("prior=%d claims=%#v", prior, got)
		}
	}
}

func TestProviderParityRequiresProductAndIndependentProviderReceipts(t *testing.T) {
	run := completeAcceptanceRun()
	run.Turns = 30
	history := []intexp.QualificationEvidence{
		{EngineID: "kyutai", ModelID: "model-1", Passed: true, MachineJSON: acceptanceEnvelope(t, map[string]any{
			"speaker-policy": map[string]any{"in_band": true, "negative_controls_passed": true},
		})},
		{EngineID: "kyutai", ModelID: "model-1", Passed: true, MachineJSON: acceptanceEnvelope(t, map[string]any{
			"corpus-quality": map[string]any{"in_band": true, "negative_controls_passed": true},
		})},
	}
	require.True(t, providerParityEligible(run, history))
	history[1].Passed = false
	require.False(t, providerParityEligible(run, history))
}

func acceptanceEnvelope(t *testing.T, claims map[string]any) []byte {
	t.Helper()
	payload, err := acceptance.MachineJSONForCell("build", "D1", acceptance.Cohort, acceptance.Policy, "kyutai", "model-1", "passthrough", claims)
	require.NoError(t, err)
	return payload
}

func TestRealtimeTransportArtifactPassedAllowsTrustCohortBootstrap(t *testing.T) {
	run := completeAcceptanceRun()
	path := t.TempDir() + "/run.json"
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, run.WriteJSON(file))
	require.NoError(t, file.Close())
	require.True(t, realtimeTransportArtifactPassed(path))

	for i := range run.Assertions {
		if run.Assertions[i].Name == "server_retention_bounded" {
			run.Assertions[i].Outcome = conformance.OutcomeFailed
		}
	}
	file, err = os.Create(path)
	require.NoError(t, err)
	require.NoError(t, run.WriteJSON(file))
	require.NoError(t, file.Close())
	require.False(t, realtimeTransportArtifactPassed(path))
}

func completeAcceptanceRun() conformance.Run {
	run := conformance.Run{
		SchemaVersion: conformance.SchemaVersion,
		RunID:         "browser-acceptance-test",
		Lane:          conformance.LaneRealtime,
		DurationMs:    60 * 60 * 1000,
		Cell:          conformance.Cell{EngineID: "kyutai", ModelID: "model-1", Strategy: "passthrough", Policy: "default"},
		Code:          conformance.Code{CapturePackage: "capture-1", Server: "server-1"},
	}
	for _, name := range conformance.RequiredAssertions(conformance.LaneRealtime) {
		run.Assertions = append(run.Assertions, conformance.Assertion{Name: name, Outcome: conformance.OutcomePassed, Detail: "observed"})
	}
	return run
}

type requireError struct{}

func (requireError) Error() string { return "run failed" }
