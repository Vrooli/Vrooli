package soak

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"audio-tools/internal/acceptance"
	"audio-tools/internal/conformance"
	intexp "audio-tools/internal/experiment"

	"github.com/stretchr/testify/require"
)

func TestManagedSoakRejectsCallerOwnedBrowserEndpointsAndFixtures(t *testing.T) {
	for _, body := range []string{
		`{"driver_url":"http://caller.invalid"}`,
		`{"ui_url":"http://caller.invalid"}`,
		`{"fixture":"/caller/audio.wav"}`,
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/validation/soak", strings.NewReader(body))
		serve(recorder, request, Deps{})
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("body %s returned %d, want 400: %s", body, recorder.Code, recorder.Body.String())
		}
	}
}

func TestManagedSoakRequiresRequestScopedRunGrantAndLease(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/validation/soak", strings.NewReader(`{"lane":"realtime"}`))
	serve(recorder, request, Deps{})
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("missing trusted receipts returned %d, want 401", recorder.Code)
	}
}

func TestAudioToolsDemandLeaseVerifierUsesOpaqueReceiptAndExactTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/runs/demand-lease-receipts/verify" || r.URL.Query().Get("target") != "audio-tools" {
			t.Errorf("unexpected verification request: %s %s", r.Method, r.URL.String())
		}
		if r.Header.Get("X-Test-Genie-Lease-Receipt") != "tglease_opaque" {
			t.Errorf("receipt header = %q", r.Header.Get("X-Test-Genie-Lease-Receipt"))
		}
		if r.ContentLength > 0 {
			t.Errorf("request unexpectedly carried an authority body")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"verified":true,"receipt_id":"tglease_opaque","owner":"test-genie","target":"audio-tools"}`))
	}))
	defer server.Close()
	require.NoError(t, VerifyAudioToolsDemandLease(context.Background(), server.URL, "tglease_opaque", server.Client()))
}

func TestAudioToolsDemandLeaseVerifierRejectsReceiptOwnerMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"verified":true,"receipt_id":"other","owner":"test-genie","target":"audio-tools"}`))
	}))
	defer server.Close()
	require.Error(t, VerifyAudioToolsDemandLease(context.Background(), server.URL, "tglease_opaque", server.Client()))
}

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
