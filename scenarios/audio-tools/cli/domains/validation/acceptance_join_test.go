package validation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSelectCompositeSourcesRejectsIncompleteHostPortabilityMatrix(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	claim := map[string]any{"in_band": true, "negative_controls_passed": true}
	for name, details := range map[string]map[string]any{
		"missing":                {},
		"d2-unavailable":         {"d1_cpu": map[string]any{"state": "available", "inference": "passed"}, "d1_accelerator": map[string]any{"state": "available", "inference": "passed"}, "d2": map[string]any{"state": "unavailable", "inference": "not_run"}},
		"d1-accelerator-missing": {"d1_cpu": map[string]any{"state": "available", "inference": "passed"}, "d2": map[string]any{"state": "available", "inference": "passed"}},
	} {
		machine, err := json.Marshal(map[string]any{
			"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
			"cohort": "voice-pilot-en-v1", "policy_revision": "policy-v1", "corpus_revision": "voice-pilot-en-v1",
			"acceptance": map[string]any{"host-portability": claim}, "details": details,
		})
		require.NoError(t, err)
		items := []*experimentv1.QualificationEvidence{{
			Passed: true, ArtifactRef: name, MachineJson: string(machine), ObservedAt: timestamppb.New(now),
		}}
		selected, missing := selectCompositeSources(items, now)
		require.NotContains(t, selected, "host-portability", name)
		require.Contains(t, missing, "host-portability", name)
	}
}

func TestSelectCompositeSourcesRejectsPriorTargetRevision(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	machine, err := json.Marshal(map[string]any{
		"target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v2",
		"cohort": "voice-pilot-en-v1", "policy_revision": "policy-v1", "corpus_revision": "voice-pilot-en-v1",
		"acceptance": map[string]any{"dictation-trust": map[string]any{"in_band": true, "negative_controls_passed": true}},
	})
	require.NoError(t, err)
	items := []*experimentv1.QualificationEvidence{{Passed: true, ArtifactRef: "prior-target", MachineJson: string(machine), ObservedAt: timestamppb.New(now)}}
	selected, missing := selectCompositeSources(items, now)
	require.NotContains(t, selected, "dictation-trust")
	require.Contains(t, missing, "dictation-trust")
}

func TestSelectCompositeSourcesAcceptsCompleteHostPortabilityMatrix(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	machine, err := json.Marshal(map[string]any{
		"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
		"cohort": "voice-pilot-en-v1", "policy_revision": "policy-v1", "corpus_revision": "voice-pilot-en-v1",
		"acceptance": map[string]any{"host-portability": map[string]any{"in_band": true, "negative_controls_passed": true}},
		"details":    completeHostMatrixDetails(false),
	})
	require.NoError(t, err)
	items := []*experimentv1.QualificationEvidence{{Passed: true, ArtifactRef: "complete", MachineJson: string(machine), ObservedAt: timestamppb.New(now)}}
	selected, _ := selectCompositeSources(items, now)
	require.Contains(t, selected, "host-portability")
}

func TestHostPortabilityReceiptRequiresMatchingPassedCells(t *testing.T) {
	valid := func() map[string]any {
		return map[string]any{"details": completeHostMatrixDetails(false)}
	}
	require.True(t, hostPortabilityMatrixComplete(valid()))
	for name, mutate := range map[string]func(map[string]any){
		"missing details": func(e map[string]any) { delete(e, "details") },
		"D1 CPU unavailable": func(e map[string]any) {
			e["details"].(map[string]any)["d1_cpu"].(map[string]any)["state"] = "unavailable"
		},
		"D1 CPU not run": func(e map[string]any) {
			e["details"].(map[string]any)["d1_cpu"].(map[string]any)["inference"] = "not_run"
		},
		"D1 CPU wrong model": func(e map[string]any) {
			e["details"].(map[string]any)["d1_cpu"].(map[string]any)["model_id"] = "whisper-large"
		},
		"accelerator missing": func(e map[string]any) { delete(e["details"].(map[string]any), "d1_accelerator") },
		"accelerator wrong engine": func(e map[string]any) {
			e["details"].(map[string]any)["d1_accelerator"].(map[string]any)["engine_id"] = "whisper-local"
		},
		"accelerator missing model": func(e map[string]any) {
			delete(e["details"].(map[string]any)["d1_accelerator"].(map[string]any), "model_id")
		},
		"accelerator wrong model": func(e map[string]any) {
			e["details"].(map[string]any)["d1_accelerator"].(map[string]any)["model_id"] = "kyutai-stt-1b-en-fr"
		},
		"D2 unavailable": func(e map[string]any) { e["details"].(map[string]any)["d2"].(map[string]any)["state"] = "unavailable" },
		"D2 wrong route": func(e map[string]any) {
			e["details"].(map[string]any)["d2"].(map[string]any)["selected_route"] = "other"
		},
		"D2 missing model": func(e map[string]any) { delete(e["details"].(map[string]any)["d2"].(map[string]any), "model_id") },
		"D1 CPU missing product path": func(e map[string]any) {
			delete(e["details"].(map[string]any)["d1_cpu"].(map[string]any), "product_path_receipt")
		},
		"D1 CPU missing browser version": func(e map[string]any) {
			delete(e["details"].(map[string]any)["d1_cpu"].(map[string]any), "browser_version")
		},
		"D1 CPU mismatched artifact": func(e map[string]any) {
			e["details"].(map[string]any)["d1_cpu"].(map[string]any)["product_path_receipt"].(map[string]any)["artifact_ref"] = "other"
		},
		"D2 wrong constrained CPU limit": func(e map[string]any) { e["details"].(map[string]any)["d2"].(map[string]any)["resource_cpu_limit"] = 4 },
		"D2 wrong constrained memory limit": func(e map[string]any) {
			e["details"].(map[string]any)["d2"].(map[string]any)["resource_memory_limit_bytes"] = int64(2 * 1024 * 1024 * 1024)
		},
		"D2 non-specific host": func(e map[string]any) {
			e["details"].(map[string]any)["d2"].(map[string]any)["device_identity"] = "unknown"
		},
	} {
		t.Run(name, func(t *testing.T) {
			envelope := valid()
			mutate(envelope)
			require.False(t, hostPortabilityMatrixComplete(envelope))
		})
	}
}

func TestHostPortabilityAllowsOnlyProvenUnavailableAcceleratorWithProductFallback(t *testing.T) {
	details := completeHostMatrixDetails(true)
	receipt := map[string]any{"details": details}
	require.True(t, hostPortabilityMatrixComplete(receipt))

	for name, mutate := range map[string]func(map[string]any){
		"inventory only":     func(d map[string]any) { delete(d["d1_accelerator"].(map[string]any), "fallback_product_path") },
		"missing reason":     func(d map[string]any) { delete(d["d1_accelerator"].(map[string]any), "reason") },
		"failed inference":   func(d map[string]any) { d["d1_accelerator"].(map[string]any)["inference"] = "failed" },
		"missing provenance": func(d map[string]any) { delete(d["d1_accelerator"].(map[string]any), "browser_version") },
		"artifact mismatch": func(d map[string]any) {
			d["d1_accelerator"].(map[string]any)["fallback_product_path"].(map[string]any)["artifact_ref"] = "other"
		},
		"wrong fallback": func(d map[string]any) {
			d["d1_accelerator"].(map[string]any)["fallback_product_path"].(map[string]any)["engine_id"] = "inventory-only"
		},
		"fallback not visible": func(d map[string]any) {
			d["d1_accelerator"].(map[string]any)["fallback_product_path"].(map[string]any)["visible_fallback"] = false
		},
		"unavailable Kyutai revision missing": func(d map[string]any) {
			delete(d["d1_accelerator"].(map[string]any), "model_revision")
		},
		"unavailable Kyutai engine version missing": func(d map[string]any) {
			delete(d["d1_accelerator"].(map[string]any), "engine_version")
		},
		"fallback missing transcript artifact": func(d map[string]any) {
			delete(d["d1_accelerator"].(map[string]any)["fallback_product_path"].(map[string]any), "visible_transcript_ref")
		},
		"fallback digest mismatch": func(d map[string]any) {
			d["d1_accelerator"].(map[string]any)["fallback_product_path"].(map[string]any)["artifact_sha256"] = strings.Repeat("d", 64)
		},
	} {
		t.Run(name, func(t *testing.T) {
			copy, err := json.Marshal(details)
			require.NoError(t, err)
			var cloned map[string]any
			require.NoError(t, json.Unmarshal(copy, &cloned))
			mutate(cloned)
			require.False(t, hostPortabilityMatrixComplete(map[string]any{"details": cloned}))
		})
	}
}

func completeHostMatrixDetails(unavailableAccelerator bool) map[string]any {
	cell := func(profile, engine, model, route, accelerator, artifact, digest, transcript string, cpuLimit int64, memoryLimit int64) map[string]any {
		productPath := map[string]any{
			"outcome": "passed", "engine_id": engine, "model_id": model, "selected_route": route,
			"artifact_ref": artifact, "artifact_sha256": digest, "visible_transcript_ref": transcript,
		}
		return map[string]any{
			"profile_id": profile, "state": "available", "inference": "passed",
			"engine_id": engine, "model_id": model, "selected_route": route,
			"os": "linux", "os_version": "Linux 6.8.0", "architecture": "amd64",
			"browser": "Chromium", "browser_version": "150.0.0.0", "device_identity": "named-d1-d2-host",
			"cpu_count": 8, "memory_total_bytes": int64(16 * 1024 * 1024 * 1024), "accelerator_identity": accelerator,
			"engine_version": "engine-v1", "model_revision": "sha256:model-revision",
			"artifact_ref": artifact, "artifact_sha256": digest,
			"resource_cpu_limit": cpuLimit, "resource_memory_limit_bytes": memoryLimit,
			"product_path": "Dictation Studio microphone input", "product_path_status": "passed",
			"product_path_receipt": productPath,
		}
	}
	d1CPUArtifact, d1GPUArtifact, d2Artifact := "d1-cpu", "d1-accelerator", "d2-constrained"
	d1CPUDigest, d1GPUDigest, d2Digest := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)
	details := map[string]any{
		"d1_cpu":         cell("D1-linux-amd64-cpu", "whisper-local", "whisper-small", "whisper.cpp-cpu", "none", d1CPUArtifact, d1CPUDigest, "trace:d1-cpu", 0, 0),
		"d1_accelerator": cell("D1-linux-amd64-accelerator", "kyutai", "kyutai/stt-1b-en_fr", "kyutai", "NVIDIA GPU named device", d1GPUArtifact, d1GPUDigest, "trace:d1-accelerator", 0, 0),
		"d2":             cell("D2-linux-amd64-constrained", "sherpa-streaming", "sherpa-onnx-streaming-zipformer-en-2023-06-26", "sherpa-streaming-zipformer-en-2023-06-26", "none", d2Artifact, d2Digest, "trace:d2", 2, 4*1024*1024*1024),
	}
	if unavailableAccelerator {
		fallback := map[string]any{
			"outcome": "passed", "engine_id": "sherpa-streaming",
			"model_id":       "sherpa-onnx-streaming-zipformer-en-2023-06-26",
			"selected_route": "sherpa-streaming-zipformer-en-2023-06-26", "visible_fallback": true,
			"visible_transcript_ref": "trace:d1-fallback", "artifact_ref": d1GPUArtifact,
			"artifact_sha256": d1GPUDigest, "engine_version": "sherpa-v1.13.2", "model_revision": "sha256:sherpa-model",
		}
		accelerator := details["d1_accelerator"].(map[string]any)
		accelerator["state"] = "unavailable"
		accelerator["inference"] = "not_run"
		accelerator["reason"] = "no accelerator device assigned"
		accelerator["accelerator_identity"] = "none"
		accelerator["fallback_product_path"] = fallback
	}
	return details
}

func TestSelectCompositeSourcesPreservesOnlyTargetAwareSimulatedClaims(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	claim := func(row, ref string) map[string]any {
		checks := []string{"not-a-supported-check"}
		if row == "explicit-routing" {
			checks = []string{"route_local_zero_debit", "route_byok_zero_debit", "route_owned_requires_consent"}
		}
		return map[string]any{
			"target_id": row, "target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
			"receipt_reference": ref, "execution_lane": "simulated", "simulated_as_native": false,
			"in_band": true, "negative_controls_passed": true, "checks": checks,
		}
	}
	machine, err := json.Marshal(map[string]any{
		"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2", "cohort": "voice-pilot-en-v1",
		"policy_revision": "policy-v1", "corpus_revision": "voice-pilot-en-v1", "execution_lane": "simulated", "simulated_as_native": false,
		"acceptance": map[string]any{"explicit-routing": claim("explicit-routing", "sim-ref"), "dictation-trust": claim("dictation-trust", "sim-ref")},
	})
	require.NoError(t, err)
	items := []*experimentv1.QualificationEvidence{{Passed: true, ArtifactRef: "sim-ref", MachineJson: string(machine), ObservedAt: timestamppb.New(now)}}
	selected, missing := selectCompositeSources(items, now)
	require.Contains(t, selected, "explicit-routing")
	require.NotContains(t, selected, "dictation-trust")
	require.Contains(t, missing, "dictation-trust")
}

func TestCompositeRequiredRowsMatchCurrentSetpointInventory(t *testing.T) {
	wantRows := []string{
		"dictation-trust", "provider-parity", "speaker-policy", "provider-evidence", "mobile-recovery",
		"interactive-latency", "corpus-quality", "owned-settlement", "explicit-routing", "host-portability",
		"private-lifecycle", "acceptance-integrity", "desktop-dictation-parity", "adapter-maintainability",
		"efficient-model-selection", "advanced-voice-workflows", "standalone-offer-upgrade", "offline-install-lifecycle",
	}
	require.ElementsMatch(t, wantRows, compositeRequiredRows)
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	selected, missing := selectCompositeSources(nil, now)
	require.Empty(t, selected)
	require.ElementsMatch(t, wantRows, missing)

	claims := make(map[string]any, len(wantRows))
	for _, row := range wantRows {
		claims[row] = map[string]any{"in_band": true, "negative_controls_passed": true}
	}
	machine, err := json.Marshal(map[string]any{
		"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2", "cohort": "voice-pilot-en-v1",
		"build": "complete", "device_profile": "D1", "policy_revision": "policy-v1", "corpus_revision": "voice-pilot-en-v1",
		"acceptance": claims,
		"details":    completeHostMatrixDetails(false),
	})
	require.NoError(t, err)
	items := []*experimentv1.QualificationEvidence{{Passed: true, ArtifactRef: "complete", MachineJson: string(machine), ObservedAt: timestamppb.New(now)}}
	selected, missing = selectCompositeSources(items, now)
	require.ElementsMatch(t, wantRows, keys(selected))
	require.Empty(t, missing)
}

func TestAcceptanceJoinReceiptBindsProviderCellToCompositeDeviceProfile(t *testing.T) {
	source := compositeSource{
		ArtifactRef: "simulated-source",
		Lane:        "simulated",
		Claim: map[string]any{
			"in_band": true, "negative_controls_passed": true,
			"target_id": "explicit-routing", "target_revision": "portable-voice-v2",
			"method": "audio-tools-acceptance-v2", "receipt_reference": "simulated-source",
			"execution_lane": "simulated", "simulated_as_native": false,
			"checks": []string{"route_local_zero_debit", "route_byok_zero_debit", "route_owned_requires_consent"},
		},
	}
	payload, build, digest, err := acceptanceJoinMachineJSON(map[string]compositeSource{"explicit-routing": source}, []string{"host-portability"})
	require.NoError(t, err)
	require.NotEmpty(t, build)
	require.NotZero(t, digest)
	var receipt map[string]any
	require.NoError(t, json.Unmarshal(payload, &receipt))
	cell, ok := receipt["provider_cell"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, receipt["device_profile"], cell["device_profile"])
	require.Equal(t, "D1+SIMULATED", cell["device_profile"])
	require.Equal(t, "acceptance-join", cell["engine_id"])
	require.Equal(t, "portable-voice-v2", cell["model_id"])
	require.Equal(t, "composite", cell["strategy"])
}

func keys(values map[string]compositeSource) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	return result
}

func TestHostPortabilityDetailsDoNotPromoteAvailableInventoryToInference(t *testing.T) {
	details := hostPortabilityDetails(map[string]*sttv1.EngineInfo{
		"whisper-local": {Id: "whisper-local", Available: true, Verdict: "selected"},
		"kyutai":        {Id: "kyutai", Available: true, Verdict: "selected"},
	}, d1CPUUnavailable("test not run"), d2Unavailable("test unavailable"))
	require.False(t, hostPortabilityMatrixComplete(map[string]any{"details": details}))
	require.Equal(t, true, details["engine_inventory"].(map[string]any)["whisper-local"].(map[string]any)["available"])
	require.Equal(t, "not_run", details["d1_cpu"].(map[string]any)["inference"])
	require.Equal(t, "unavailable", details["d1_cpu"].(map[string]any)["state"])
}

func TestHostPortabilityDetailsStillRecordReceiptWhenEnginesAreAbsent(t *testing.T) {
	details := hostPortabilityDetails(map[string]*sttv1.EngineInfo{}, d1CPUUnavailable("test not run"), d2Unavailable("test unavailable"))
	require.Empty(t, details["engine_inventory"])
	require.Equal(t, "not_run", details["d1_cpu"].(map[string]any)["inference"])
	require.Equal(t, "not_run", details["d1_accelerator"].(map[string]any)["inference"])
	require.Equal(t, "Kyutai engine is absent from the live engine inventory", details["d1_accelerator"].(map[string]any)["reason"])
	require.False(t, hostPortabilityMatrixComplete(map[string]any{"details": details}))
}

func TestUnavailableAcceleratorDiagnosticsNameObservedFallbackWithoutPromotion(t *testing.T) {
	details := hostPortabilityDetails(map[string]*sttv1.EngineInfo{
		"kyutai":           {Id: "kyutai", Available: false, Verdict: "rejected", Reason: "not_serviceable: provider health reports unavailable"},
		"sherpa-streaming": {Id: "sherpa-streaming", Available: true, Verdict: "selected"},
	}, d1CPUUnavailable("not run"), d2Unavailable("not run"))
	accelerator := details["d1_accelerator"].(map[string]any)
	require.Equal(t, "not_serviceable: provider health reports unavailable", accelerator["reason"])
	require.Equal(t, "sherpa-streaming", accelerator["fallback_candidate"])
	require.Equal(t, true, accelerator["fallback_available"])
	require.Equal(t, "not_run", accelerator["inference"])
	require.False(t, hostPortabilityMatrixComplete(map[string]any{"details": details}))
}

func TestD2InferenceDoesNotPromoteWholeMatrixWithoutD1Cells(t *testing.T) {
	for _, result := range []map[string]any{
		d2Unavailable("cached image unavailable"),
		{"state": "available", "inference": "passed", "selected_route": "sherpa-streaming-zipformer-en-2023-06-26"},
	} {
		details := hostPortabilityDetails(map[string]*sttv1.EngineInfo{"sherpa-streaming": {Id: "sherpa-streaming", Available: true}}, d1CPUUnavailable("test not run"), result)
		d2 := details["d2"].(map[string]any)
		require.Equal(t, "sherpa-streaming-zipformer-en-2023-06-26", d2["selected_route"])
		require.False(t, hostPortabilityMatrixComplete(map[string]any{"details": details}))
	}
}

func TestD1CPUInferencePassDoesNotCompleteMatrixWithoutAccelerator(t *testing.T) {
	d1CPU := map[string]any{"engine_id": "whisper-local", "model_id": "whisper-small", "state": "available", "inference": "passed"}
	details := hostPortabilityDetails(map[string]*sttv1.EngineInfo{}, d1CPU, d2Unavailable("test unavailable"))
	require.Equal(t, "whisper-local", details["d1_cpu"].(map[string]any)["engine_id"])
	require.False(t, hostPortabilityMatrixComplete(map[string]any{"details": details}))
}

func TestD2PeakMemoryReceiptRequiresMeasuredCgroupValue(t *testing.T) {
	require.EqualValues(t, 123456, parseD2PeakMemory("PASS\nD2_MEMORY_PEAK_BYTES=123456\n"))
	require.Zero(t, parseD2PeakMemory("PASS\nD2_MEMORY_PEAK_BYTES=unavailable\n"))
	require.Zero(t, parseD2PeakMemory("PASS\n"))
}

func TestD2SherpaUsesAndReportsInt8ModelArtifacts(t *testing.T) {
	require.Equal(t, []string{
		"encoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx",
		"decoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx",
		"joiner-epoch-99-avg-1-chunk-16-left-128.int8.onnx",
		"tokens.txt",
	}, d2SherpaInt8ModelFiles)
	require.Equal(t, "/models/asr/"+d2SherpaInt8ModelFiles[0], d2SherpaModelPath(d2SherpaInt8ModelFiles[0]))
	require.Equal(t, "/models/asr/"+d2SherpaInt8ModelFiles[3], d2SherpaModelPath(d2SherpaInt8ModelFiles[3]))
}
