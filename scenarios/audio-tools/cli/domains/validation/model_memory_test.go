package validation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
)

func TestBuildModelMemoryEvidenceBindsModelProfileAndMeasuredPeak(t *testing.T) {
	observation := validModelMemoryObservation()
	evidence, err := buildModelMemoryEvidence(observation)
	require.NoError(t, err)
	require.True(t, evidence.GetPassed())
	require.NotEmpty(t, evidence.GetArtifactRef())
	require.Equal(t, modelMemoryPolicy, evidence.GetPolicyProfile())

	var receipt modelMemoryReceipt
	require.NoError(t, json.Unmarshal([]byte(evidence.GetMachineJson()), &receipt))
	require.Equal(t, 1, receipt.SchemaVersion)
	require.Equal(t, "sherpa-streaming", receipt.ProviderCell.EngineID)
	require.Equal(t, "sherpa-zipformer-en-2023-06-26", receipt.ProviderCell.ModelID)
	require.Equal(t, "stt", receipt.ProviderCell.Capability)
	require.Equal(t, "sherpa-onnx", receipt.Model.Runtime)
	require.Equal(t, "cpu", receipt.Execution.Lane)
	require.Equal(t, "none", receipt.Execution.Accelerator)
	require.Equal(t, observation.MemoryBudgetBytes, receipt.Execution.MemoryBudgetBytes)
	require.Equal(t, observation.PeakMemoryBytes, receipt.Execution.PeakMemoryBytes)
	require.Equal(t, true, receipt.Acceptance["memory-budget"].InBand)
	var shape map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(evidence.GetMachineJson()), &shape))
	require.ElementsMatch(t, []string{"schema_version", "provider_cell", "model", "execution", "acceptance"}, jsonMapKeys(shape))
	var executionShape map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(shape["execution"], &executionShape))
	require.ElementsMatch(t, []string{"lane", "os", "arch", "accelerator", "memory_budget_bytes", "peak_memory_bytes", "peak_memory_source", "inference_passed"}, jsonMapKeys(executionShape))
}

func TestBuildModelMemoryEvidenceFailsClosedWhenPeakExceedsBudget(t *testing.T) {
	observation := validModelMemoryObservation()
	observation.PeakMemoryBytes = observation.MemoryBudgetBytes + 1
	evidence, err := buildModelMemoryEvidence(observation)
	require.NoError(t, err)
	require.False(t, evidence.GetPassed())

	var receipt modelMemoryReceipt
	require.NoError(t, json.Unmarshal([]byte(evidence.GetMachineJson()), &receipt))
	require.False(t, receipt.Acceptance["memory-budget"].InBand)
}

func TestBuildModelMemoryEvidenceRejectsMissingOrUntrustworthyMeasurement(t *testing.T) {
	for name, mutate := range map[string]func(*modelMemoryObservation){
		"missing peak":          func(value *modelMemoryObservation) { value.PeakMemoryBytes = 0 },
		"negative peak":         func(value *modelMemoryObservation) { value.PeakMemoryBytes = -1 },
		"missing source":        func(value *modelMemoryObservation) { value.PeakMemorySource = "" },
		"other source":          func(value *modelMemoryObservation) { value.PeakMemorySource = "container-limit" },
		"missing budget":        func(value *modelMemoryObservation) { value.MemoryBudgetBytes = 0 },
		"bad model hash":        func(value *modelMemoryObservation) { value.ArtifactSHA256 = "sha256:" + value.ArtifactSHA256 },
		"bad runtime hash":      func(value *modelMemoryObservation) { value.RuntimeSHA256 = "not-a-digest" },
		"missing runtime model": func(value *modelMemoryObservation) { value.RuntimeModelID = "" },
	} {
		t.Run(name, func(t *testing.T) {
			observation := validModelMemoryObservation()
			mutate(&observation)
			_, err := buildModelMemoryEvidence(observation)
			require.Error(t, err)
		})
	}
}

func TestBuildModelMemoryEvidenceDoesNotPassFailedRunnerOrInvalidIntegrity(t *testing.T) {
	for name, mutate := range map[string]func(*modelMemoryObservation){
		"inference failure":          func(value *modelMemoryObservation) { value.InferencePassed = false },
		"artifact integrity failure": func(value *modelMemoryObservation) { value.MeasurementValid = false },
	} {
		t.Run(name, func(t *testing.T) {
			observation := validModelMemoryObservation()
			mutate(&observation)
			evidence, err := buildModelMemoryEvidence(observation)
			require.NoError(t, err)
			require.False(t, evidence.GetPassed())
		})
	}
}

func TestCanonicalArtifactSetSHA256IsDeterministicAndStrict(t *testing.T) {
	first := map[string]string{"model.bin": testDigest("model"), "tokens.txt": testDigest("tokens")}
	second := map[string]string{"tokens.txt": testDigest("tokens"), "model.bin": testDigest("model")}
	firstDigest, err := canonicalArtifactSetSHA256(first)
	require.NoError(t, err)
	secondDigest, err := canonicalArtifactSetSHA256(second)
	require.NoError(t, err)
	require.Equal(t, firstDigest, secondDigest)
	require.True(t, validSHA256(firstDigest))
	_, err = canonicalArtifactSetSHA256(map[string]string{"model.bin": ""})
	require.Error(t, err)
}

func TestMinRAMBudgetBytesRejectsUnknownAndOverflowValues(t *testing.T) {
	got, err := minRAMBudgetBytes(2)
	require.NoError(t, err)
	require.EqualValues(t, 2*1024*1024*1024, got)
	for _, value := range []float64{0, -1, math.NaN(), math.Inf(1), math.MaxFloat64} {
		_, err := minRAMBudgetBytes(value)
		require.Error(t, err)
	}
}

func TestParseCgroupV2MemoryPeakRequiresExactlyOnePositiveInt64(t *testing.T) {
	got, err := parseCgroupV2MemoryPeak("native test passed\nD1_MEMORY_PEAK_BYTES=123456\n", "D1_MEMORY_PEAK_BYTES=")
	require.NoError(t, err)
	require.EqualValues(t, 123456, got)

	for _, output := range []string{
		"native test passed\n",
		"D1_MEMORY_PEAK_BYTES=0\n",
		"D1_MEMORY_PEAK_BYTES=-1\n",
		"D1_MEMORY_PEAK_BYTES=9223372036854775808\n",
		"D1_MEMORY_PEAK_BYTES=1\nD1_MEMORY_PEAK_BYTES=2\n",
	} {
		_, err := parseCgroupV2MemoryPeak(output, "D1_MEMORY_PEAK_BYTES=")
		require.Error(t, err, "output %q", output)
	}
	_, err = parseCgroupV2MemoryPeak("D1_MEMORY_PEAK_BYTES=1\n", "D1_MEMORY_PEAK_BYTES")
	require.Error(t, err)
}

func TestStripMarkedLineKeepsTranscriptWhenMemoryMarkerIsAppended(t *testing.T) {
	output := "whisper log\nThe quick brown fox jumps.\nD1_MEMORY_PEAK_BYTES=123456\n"
	require.Equal(t, "whisper log\nThe quick brown fox jumps.\n", stripMarkedLine(output, "D1_MEMORY_PEAK_BYTES="))
}

func TestMemoryObservationFromRunRequiresCanonicalModelAndIsolatedMeasuredProfile(t *testing.T) {
	model := &modelsv1.ModelRecord{
		Id: "sherpa-zipformer-en-2023-06-26", EngineIds: []string{"sherpa-streaming"}, Capability: "stt",
		Runtime: "sherpa-onnx", RuntimeModelId: "sherpa-onnx/streaming-zipformer-en-2023-06-26",
		ArtifactResource: "sherpa-onnx", ArtifactName: "streaming-zipformer-en-2023-06-26",
		Hardware: &modelsv1.Hardware{MinRamGb: 2},
	}
	artifacts := map[string]string{}
	for _, name := range d2SherpaInt8ModelFiles {
		artifacts[name] = "sha256:" + testDigest(name) + ",size_bytes:10"
	}
	artifacts["libsherpa-onnx-c-api.so"] = "sha256:" + testDigest("sherpa-runtime") + ",size_bytes:10"
	artifacts["libonnxruntime.so"] = "sha256:" + testDigest("onnx-runtime") + ",size_bytes:10"
	result := map[string]any{
		"state": "available", "inference": "passed", "peak_memory_measured": true,
		"peak_memory_bytes": int64(426_516_480), "artifacts": artifacts,
		"profile": map[string]any{"os": "linux", "arch": "amd64", "accelerator": "none", "network": "none", "container_memory_limit_bytes": int64(4 * 1024 * 1024 * 1024)},
	}
	observation, available, reason := memoryObservationFromRun("sherpa-streaming", "cpu_streaming", model, result)
	require.True(t, available, reason)
	require.EqualValues(t, 2*1024*1024*1024, observation.MemoryBudgetBytes)
	require.True(t, validSHA256(observation.ArtifactSHA256))
	require.True(t, validSHA256(observation.RuntimeSHA256))

	model.EngineIds = []string{"different-engine"}
	_, available, reason = memoryObservationFromRun("sherpa-streaming", "cpu_streaming", model, result)
	require.False(t, available)
	require.Contains(t, reason, "does not declare")
}

func TestMemoryObservationUsesCanonicalModelIDWhenRuntimeModelIDIsOmitted(t *testing.T) {
	model := &modelsv1.ModelRecord{
		Id: "whisper-small", EngineIds: []string{"whisper-local"}, Capability: "stt", Runtime: "whisper.cpp",
		ArtifactResource: "whisper", ArtifactName: "small-model", Hardware: &modelsv1.Hardware{MinRamGb: 2},
	}
	result := map[string]any{
		"state": "available", "inference": "passed", "peak_memory_measured": true,
		"peak_memory_bytes": int64(512_000_000),
		"artifacts":         map[string]string{"ggml-small.bin": "sha256:" + testDigest("whisper-model") + ",size_bytes:10"},
		"profile":           map[string]any{"os": "linux", "arch": "amd64", "accelerator": "none", "network": "none", "container_memory_limit_bytes": int64(8 * 1024 * 1024 * 1024)},
	}
	observation, available, reason := memoryObservationFromRun("whisper-local", "cpu_batch", model, result)
	require.True(t, available, reason)
	require.Equal(t, "whisper-small", observation.RuntimeModelID)
	require.Equal(t, "small-model", observation.ArtifactName)
}

func validModelMemoryObservation() modelMemoryObservation {
	return modelMemoryObservation{
		EngineID: "sherpa-streaming", ModelID: "sherpa-zipformer-en-2023-06-26", Strategy: "vad_segment",
		Runtime: "sherpa-onnx", RuntimeModelID: "sherpa-onnx/streaming-zipformer-en-2023-06-26",
		ArtifactResource: "sherpa-onnx", ArtifactName: "streaming-zipformer-en-2023-06-26",
		ArtifactSHA256: testDigest("model artifacts"), RuntimeSHA256: testDigest("runtime artifacts"),
		OS: "linux", Arch: "amd64", MemoryBudgetBytes: 2 * 1024 * 1024 * 1024,
		PeakMemoryBytes: 426_516_480, PeakMemorySource: "cgroup-v2-memory.peak",
		InferencePassed: true, MeasurementValid: true,
		ObservedAt: time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC),
	}
}

func testDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func jsonMapKeys[V any](value map[string]V) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	return keys
}
