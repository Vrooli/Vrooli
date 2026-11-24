package validation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const modelMemoryPolicy = "model-memory-v1"

type modelMemoryObservation struct {
	EngineID          string
	ModelID           string
	Strategy          string
	Runtime           string
	RuntimeModelID    string
	ArtifactResource  string
	ArtifactName      string
	ArtifactSHA256    string
	RuntimeSHA256     string
	OS                string
	Arch              string
	MemoryBudgetBytes int64
	PeakMemoryBytes   int64
	PeakMemorySource  string
	InferencePassed   bool
	MeasurementValid  bool
	ObservedAt        time.Time
}

type modelMemoryProviderCell struct {
	EngineID   string `json:"engine_id"`
	ModelID    string `json:"model_id"`
	Capability string `json:"capability"`
	Strategy   string `json:"strategy"`
}

type modelMemoryModelIdentity struct {
	Runtime          string `json:"runtime"`
	RuntimeModelID   string `json:"runtime_model_id"`
	ArtifactResource string `json:"artifact_resource"`
	ArtifactName     string `json:"artifact_name"`
	ArtifactSHA256   string `json:"artifact_sha256"`
	RuntimeSHA256    string `json:"runtime_sha256"`
}

type modelMemoryExecution struct {
	Lane              string `json:"lane"`
	OS                string `json:"os"`
	Arch              string `json:"arch"`
	Accelerator       string `json:"accelerator"`
	MemoryBudgetBytes int64  `json:"memory_budget_bytes"`
	PeakMemoryBytes   int64  `json:"peak_memory_bytes"`
	PeakMemorySource  string `json:"peak_memory_source"`
	InferencePassed   bool   `json:"inference_passed"`
}

type modelMemoryAcceptance struct {
	InBand                     bool `json:"in_band"`
	Measured                   bool `json:"measured"`
	MeasurementIntegrityPassed bool `json:"measurement_integrity_passed"`
}

type modelMemoryReceipt struct {
	SchemaVersion int                              `json:"schema_version"`
	ProviderCell  modelMemoryProviderCell          `json:"provider_cell"`
	Model         modelMemoryModelIdentity         `json:"model"`
	Execution     modelMemoryExecution             `json:"execution"`
	Acceptance    map[string]modelMemoryAcceptance `json:"acceptance"`
}

// buildModelMemoryEvidence creates one model-scoped, CPU-memory observation.
// The budget must come from the canonical model declaration; a container limit
// is recorded separately by the runner and is never mistaken for a measurement.
func buildModelMemoryEvidence(observation modelMemoryObservation) (*experimentv1.QualificationEvidence, error) {
	if strings.TrimSpace(observation.EngineID) == "" || strings.TrimSpace(observation.ModelID) == "" || strings.TrimSpace(observation.Strategy) == "" ||
		strings.TrimSpace(observation.Runtime) == "" || strings.TrimSpace(observation.RuntimeModelID) == "" ||
		strings.TrimSpace(observation.ArtifactResource) == "" || strings.TrimSpace(observation.ArtifactName) == "" ||
		!validSHA256(observation.ArtifactSHA256) || !validSHA256(observation.RuntimeSHA256) ||
		strings.TrimSpace(observation.OS) == "" || strings.TrimSpace(observation.Arch) == "" || observation.ObservedAt.IsZero() ||
		observation.MemoryBudgetBytes <= 0 || observation.PeakMemoryBytes <= 0 || observation.PeakMemorySource != "cgroup-v2-memory.peak" {
		return nil, fmt.Errorf("model memory receipt is missing a valid identity, digest, profile, budget, or measured peak")
	}

	passed := observation.InferencePassed && observation.MeasurementValid && observation.PeakMemoryBytes <= observation.MemoryBudgetBytes
	payload := modelMemoryReceipt{
		SchemaVersion: 1,
		ProviderCell: modelMemoryProviderCell{
			EngineID: observation.EngineID, ModelID: observation.ModelID,
			Capability: "stt", Strategy: observation.Strategy,
		},
		Model: modelMemoryModelIdentity{
			Runtime: observation.Runtime, RuntimeModelID: observation.RuntimeModelID,
			ArtifactResource: observation.ArtifactResource, ArtifactName: observation.ArtifactName,
			ArtifactSHA256: observation.ArtifactSHA256, RuntimeSHA256: observation.RuntimeSHA256,
		},
		Execution: modelMemoryExecution{
			Lane: "cpu", OS: observation.OS, Arch: observation.Arch, Accelerator: "none",
			MemoryBudgetBytes: observation.MemoryBudgetBytes, PeakMemoryBytes: observation.PeakMemoryBytes,
			PeakMemorySource: observation.PeakMemorySource, InferencePassed: observation.InferencePassed,
		},
		Acceptance: map[string]modelMemoryAcceptance{
			"memory-budget": {InBand: passed, Measured: true, MeasurementIntegrityPassed: observation.MeasurementValid},
		},
	}
	machine, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode model memory receipt: %w", err)
	}
	return &experimentv1.QualificationEvidence{
		EngineId:      observation.EngineID,
		ModelId:       observation.ModelID,
		Strategy:      observation.Strategy,
		PolicyProfile: modelMemoryPolicy,
		Kind:          experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_MODEL_MEMORY,
		Passed:        passed,
		ArtifactRef:   fmt.Sprintf("audio-tools:model-memory:%s:%s:%s", observation.ModelID, observation.ArtifactSHA256, observation.ObservedAt.UTC().Format(time.RFC3339Nano)),
		Notes:         "Isolated CPU inference memory peak compared with the canonical model RAM budget.",
		MachineJson:   string(machine),
		ObservedAt:    timestamppb.New(observation.ObservedAt.UTC()),
	}, nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func canonicalArtifactSetSHA256(hashes map[string]string) (string, error) {
	if len(hashes) == 0 {
		return "", fmt.Errorf("artifact hash set is empty")
	}
	for name, digest := range hashes {
		if strings.TrimSpace(name) == "" || !validSHA256(digest) {
			return "", fmt.Errorf("artifact hash set contains an invalid name or SHA-256")
		}
	}
	canonical, err := json.Marshal(hashes)
	if err != nil {
		return "", fmt.Errorf("encode artifact hash set: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func minRAMBudgetBytes(minRAMGiB float64) (int64, error) {
	if minRAMGiB <= 0 || math.IsNaN(minRAMGiB) || math.IsInf(minRAMGiB, 0) || minRAMGiB > float64(math.MaxInt64)/(1024*1024*1024) {
		return 0, fmt.Errorf("canonical model minimum RAM is not a positive finite value")
	}
	return int64(math.Round(minRAMGiB * 1024 * 1024 * 1024)), nil
}

func parseCgroupV2MemoryPeak(output, marker string) (int64, error) {
	if strings.TrimSpace(marker) == "" || !strings.HasSuffix(marker, "=") {
		return 0, fmt.Errorf("memory peak marker must end with '='")
	}
	var value int64
	found := false
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, marker) {
			continue
		}
		if found {
			return 0, fmt.Errorf("memory peak marker occurred more than once")
		}
		found = true
		raw := strings.TrimSpace(strings.TrimPrefix(line, marker))
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			return 0, fmt.Errorf("memory peak value is not a positive int64")
		}
		value = parsed
	}
	if !found {
		return 0, fmt.Errorf("memory peak marker %q is missing", marker)
	}
	return value, nil
}

func recordHostMemoryEvidence(
	ctx context.Context,
	client experimentconnect.ExperimentServiceClient,
	models map[string]*modelsv1.ModelRecord,
	d1CPU, d2 map[string]any,
	observedAt time.Time,
) (map[string]any, error) {
	if client == nil {
		return nil, fmt.Errorf("qualification evidence client is unavailable")
	}
	profiles := []struct {
		name, modelID, engineID, strategy string
		result                            map[string]any
	}{
		{"D1 CPU Whisper", "whisper-small", "whisper-local", "cpu_batch", d1CPU},
		{"D2 CPU Sherpa", "sherpa-zipformer-en-2023-06-26", "sherpa-streaming", "cpu_streaming", d2},
	}
	recorded := make(map[string]any, len(profiles))
	for _, profile := range profiles {
		model := models[profile.modelID]
		if model == nil {
			return nil, fmt.Errorf("canonical model %q is missing from the model registry response", profile.modelID)
		}
		observation, available, reason := memoryObservationFromRun(profile.engineID, profile.strategy, model, profile.result)
		if !available {
			recorded[profile.modelID] = map[string]any{"state": "unavailable", "reason": reason}
			continue
		}
		observation.ObservedAt = observedAt
		evidence, err := buildModelMemoryEvidence(observation)
		if err != nil {
			return nil, fmt.Errorf("%s memory receipt: %w", profile.name, err)
		}
		resp, err := client.RecordQualificationEvidence(ctx, connect.NewRequest(&experimentv1.RecordQualificationEvidenceRequest{Evidence: evidence}))
		if err != nil {
			return nil, fmt.Errorf("persist %s memory receipt: %w", profile.name, err)
		}
		if resp == nil || resp.Msg == nil || resp.Msg.GetEvidence() == nil {
			return nil, fmt.Errorf("persist %s memory receipt returned no stored evidence", profile.name)
		}
		stored := resp.Msg.GetEvidence()
		recorded[profile.modelID] = map[string]any{
			"state": "recorded", "receipt_id": stored.GetId(), "passed": stored.GetPassed(),
			"peak_memory_bytes": observation.PeakMemoryBytes, "memory_budget_bytes": observation.MemoryBudgetBytes,
		}
	}
	return recorded, nil
}

func memoryObservationFromRun(engineID, strategy string, model *modelsv1.ModelRecord, result map[string]any) (modelMemoryObservation, bool, string) {
	if result == nil || result["inference"] != "passed" || result["state"] != "available" {
		return modelMemoryObservation{}, false, "isolated inference did not pass"
	}
	if model.GetCapability() != "stt" || !modelDeclaresEngine(model, engineID) {
		return modelMemoryObservation{}, false, "canonical model does not declare the observed STT engine"
	}
	if result["peak_memory_measured"] != true {
		if reason, ok := result["memory_measurement_error"].(string); ok && reason != "" {
			return modelMemoryObservation{}, false, reason
		}
		return modelMemoryObservation{}, false, "runner did not measure cgroup-v2 memory.peak"
	}
	profile, _ := result["profile"].(map[string]any)
	containerMemoryLimit := int64(0)
	if profile != nil {
		containerMemoryLimit = memoryInt64(profile["container_memory_limit_bytes"])
		if containerMemoryLimit == 0 {
			containerMemoryLimit = memoryInt64(profile["memory_limit_bytes"])
		}
	}
	if profile == nil || profile["network"] != "none" || profile["accelerator"] != "none" || containerMemoryLimit <= 0 {
		return modelMemoryObservation{}, false, "runner did not prove a bounded, networkless CPU profile"
	}
	artifacts, _ := result["artifacts"].(map[string]string)
	if artifacts == nil {
		return modelMemoryObservation{}, false, "runner artifact hashes are unavailable"
	}
	modelHashes, runtimeHashes, err := memoryArtifactHashes(model.GetId(), artifacts)
	if err != nil {
		return modelMemoryObservation{}, false, err.Error()
	}
	artifactHash, err := canonicalArtifactSetSHA256(modelHashes)
	if err != nil {
		return modelMemoryObservation{}, false, err.Error()
	}
	runtimeHash, err := canonicalArtifactSetSHA256(runtimeHashes)
	if err != nil {
		return modelMemoryObservation{}, false, err.Error()
	}
	hardware := model.GetHardware()
	if hardware == nil {
		return modelMemoryObservation{}, false, "canonical model has no hardware budget"
	}
	runtimeModelID := strings.TrimSpace(model.GetRuntimeModelId())
	if runtimeModelID == "" {
		// The registry deliberately omits runtime_model_id when the canonical
		// model ID is itself the runtime identity.
		runtimeModelID = model.GetId()
	}
	budget, err := minRAMBudgetBytes(hardware.GetMinRamGb())
	if err != nil {
		return modelMemoryObservation{}, false, err.Error()
	}
	peak := memoryInt64(result["peak_memory_bytes"])
	if peak <= 0 {
		return modelMemoryObservation{}, false, "runner returned a nonpositive memory peak"
	}
	observation := modelMemoryObservation{
		EngineID: engineID, ModelID: model.GetId(), Strategy: strategy,
		Runtime: model.GetRuntime(), RuntimeModelID: runtimeModelID,
		ArtifactResource: model.GetArtifactResource(), ArtifactName: model.GetArtifactName(),
		ArtifactSHA256: artifactHash, RuntimeSHA256: runtimeHash,
		OS: memoryString(profile["os"]), Arch: memoryString(profile["arch"]),
		MemoryBudgetBytes: budget, PeakMemoryBytes: peak, PeakMemorySource: "cgroup-v2-memory.peak",
		InferencePassed: true, MeasurementValid: true,
	}
	return observation, true, ""
}

func modelDeclaresEngine(model *modelsv1.ModelRecord, engineID string) bool {
	if model == nil {
		return false
	}
	for _, declared := range model.GetEngineIds() {
		if declared == engineID {
			return true
		}
	}
	return false
}

func memoryArtifactHashes(modelID string, artifacts map[string]string) (map[string]string, map[string]string, error) {
	modelHashes := map[string]string{}
	runtimeHashes := map[string]string{}
	for name, description := range artifacts {
		digest := strings.TrimPrefix(strings.SplitN(description, ",", 2)[0], "sha256:")
		if !validSHA256(digest) {
			return nil, nil, fmt.Errorf("runner artifact %q lacks an exact SHA-256 digest", name)
		}
		switch modelID {
		case "whisper-small":
			if name == "ggml-small.bin" {
				modelHashes[name] = digest
			}
		case "sherpa-zipformer-en-2023-06-26":
			for _, modelFile := range d2SherpaInt8ModelFiles {
				if name == modelFile {
					modelHashes[name] = digest
				}
			}
		}
		if strings.Contains(name, "sherpa-onnx") || strings.Contains(name, "onnxruntime") {
			runtimeHashes[name] = digest
		}
	}
	if modelID == "whisper-small" {
		imageDigest := digestFromPinnedImage(whisperCppImage)
		if !validSHA256(imageDigest) {
			return nil, nil, fmt.Errorf("pinned whisper.cpp image digest is invalid")
		}
		runtimeHashes["container-image"] = imageDigest
	}
	if len(modelHashes) == 0 || len(runtimeHashes) == 0 {
		return nil, nil, fmt.Errorf("runner did not provide complete model/runtime artifact hashes")
	}
	return modelHashes, runtimeHashes, nil
}

func digestFromPinnedImage(image string) string {
	index := strings.LastIndex(image, "@sha256:")
	if index < 0 {
		return ""
	}
	return image[index+len("@sha256:"):]
}

func memoryInt64(value any) int64 {
	switch number := value.(type) {
	case int64:
		return number
	case int:
		return int64(number)
	case float64:
		if math.IsNaN(number) || math.IsInf(number, 0) || number > math.MaxInt64 || number < math.MinInt64 {
			return 0
		}
		return int64(number)
	default:
		return 0
	}
}

func memoryString(value any) string {
	valueString, _ := value.(string)
	return valueString
}
