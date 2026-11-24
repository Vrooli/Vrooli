//go:build audio_lab

package soak

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"audio-tools/internal/acceptance"
	"audio-tools/internal/conformance"
	intexp "audio-tools/internal/experiment"
	"audio-tools/internal/soak"

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

func TestPersistEvidenceAssemblyKeepsMobileClaimAndSurfaceSelectionNarrow(t *testing.T) {
	result := soak.Result{Run: conformance.Run{RunID: "run-1", Lane: conformance.LaneRealtime, Code: conformance.Code{Server: "server-1"}, Cell: conformance.Cell{EngineID: "engine", ModelID: "model", Strategy: "strategy"}}}
	opt := soak.Options{Surface: " Swarm-Manager ", MobileRecovery: true}
	machine := baseMachineEvidence(opt, result)
	require.Equal(t, "swarm-manager-quick-capture", machine["product"])
	claims := addMobileRecoveryClaim(nil, opt, soak.Result{MobileRecoveryPassed: true}, nil)
	require.True(t, claimInBand(claims["mobile-recovery"]))
	require.NoError(t, addAcceptanceEvidence(machine, result.Run, "D1", claims))
	require.Equal(t, acceptance.Cohort, machine["cohort"])
	addMobileEmulationEvidence(machine, opt, soak.Result{MobileRecoveryPassed: true})
	require.Equal(t, map[string]any{"viewport_width": 390, "viewport_height": 844, "native": false, "diagnostic_export": true}, machine["mobile_emulation"])

	failed := addMobileRecoveryClaim(nil, opt, soak.Result{MobileRecoveryPassed: true}, requireError{})
	require.Nil(t, failed)
	defaultProduct := baseMachineEvidence(soak.Options{Surface: "unknown"}, result)
	require.Equal(t, "dictation-studio", defaultProduct["product"])
}

func TestDeviceEvidenceRequiresQualifiedRunAndEnumeratedQualificationDevice(t *testing.T) {
	run := completeAcceptanceRun()
	device := &soak.AudioDeviceEvidence{Enumerated: true, SelectedLabel: "Vrooli_Qualification_Microphone", SampleRate: 16000, ChannelCount: 1}
	require.True(t, deviceEvidencePassed(run, nil, device))
	require.False(t, deviceEvidencePassed(run, requireError{}, device))
	require.False(t, deviceEvidencePassed(run, nil, nil))
	device.SelectedLabel = "default microphone"
	require.False(t, deviceEvidencePassed(run, nil, device))

	device.SelectedLabel = "Vrooli_Qualification_Microphone"
	machine := map[string]any{"acceptance": map[string]any{"dictation-trust": true}, "device_profile": "D1", "run_id": run.RunID}
	deviceMachine := deviceMachineEvidence(machine, device)
	require.NotContains(t, deviceMachine, "acceptance")
	require.NotContains(t, deviceMachine, "device_profile")
	require.Equal(t, "pipewire-host-device", deviceMachine["device_path"])
	require.NotContains(t, deviceEvidenceNotes(device), "BAS returned no device evidence")
	require.Contains(t, deviceEvidenceNotes(nil), "BAS returned no device evidence")
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
		{EngineID: "kyutai", ModelID: "model-1", Strategy: "passthrough", Passed: true, MachineJSON: acceptanceEnvelope(t, map[string]any{
			"speaker-policy": map[string]any{"in_band": true, "negative_controls_passed": true},
		})},
		{EngineID: "kyutai", ModelID: "model-1", Strategy: "passthrough", Passed: true, MachineJSON: acceptanceEnvelope(t, map[string]any{
			"corpus-quality": map[string]any{"in_band": true, "negative_controls_passed": true},
		})},
	}
	require.True(t, providerParityEligible(run, "D1", history))
	history[1].Passed = false
	require.False(t, providerParityEligible(run, "D1", history))
	history[1].Passed = true

	for _, mutate := range []func(map[string]any){
		func(e map[string]any) { e["build"] = "other-build" },
		func(e map[string]any) { e["device_profile"] = "D2" },
		func(e map[string]any) { e["provider_cell"].(map[string]any)["strategy"] = "other-strategy" },
	} {
		mismatched := append([]intexp.QualificationEvidence(nil), history...)
		for i := range mismatched {
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(mismatched[i].MachineJSON, &envelope))
			mutate(envelope)
			mismatched[i].MachineJSON, _ = json.Marshal(envelope)
		}
		require.False(t, providerParityEligible(run, "D1", mismatched))
	}
}

func acceptanceEnvelope(t *testing.T, claims map[string]any) []byte {
	t.Helper()
	payload, err := acceptance.MachineJSONForCell("server-1", "D1", acceptance.Cohort, acceptance.Policy, "kyutai", "model-1", "stt", "passthrough", claims)
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
