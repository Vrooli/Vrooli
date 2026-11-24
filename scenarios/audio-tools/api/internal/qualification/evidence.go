package trustfloor

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// QualificationEvidence is a transcript-free proof record for a declared
// provider cell. It is shared by product promotion reads and the lab-owned
// persistence adapter; the product contract does not depend on experiment.
type QualificationEvidence struct {
	ID            string
	EngineID      string
	ModelID       string
	Strategy      string
	ProviderTier  string
	PolicyProfile string
	Kind          string
	FaultProfile  string
	Passed        bool
	ArtifactRef   string
	Notes         string
	MachineJSON   []byte
	ObservedAt    time.Time
}

// QualificationEvidenceFilter selects persisted proof for a provider cell.
type QualificationEvidenceFilter struct {
	EngineID      string
	ModelID       string
	Strategy      string
	PolicyProfile string
}

// ModelMemoryReceipt is the versioned machine contract emitted by the
// Audio Tools model-memory qualification producer. Hashes are lowercase
// SHA-256 hex without a prefix. artifact_sha256 is the producer's canonical
// digest over the exact model input-file hashes. runtime_sha256 pins the exact
// runtime executable or canonical runtime-artifact set used by the runner
// (for example, an OCI image digest or the Sherpa/ONNX runtime libraries).
// The isolated owner producer attests these digests; the promotion resolver
// checks their encoding and joins the declared model/runtime identities.
type ModelMemoryReceipt struct {
	SchemaVersion int `json:"schema_version"`
	ProviderCell  struct {
		EngineID   string `json:"engine_id"`
		ModelID    string `json:"model_id"`
		Capability string `json:"capability"`
		Strategy   string `json:"strategy"`
	} `json:"provider_cell"`
	Model struct {
		Runtime          string `json:"runtime"`
		RuntimeModelID   string `json:"runtime_model_id"`
		ArtifactResource string `json:"artifact_resource"`
		ArtifactName     string `json:"artifact_name"`
		ArtifactSHA256   string `json:"artifact_sha256"`
		RuntimeSHA256    string `json:"runtime_sha256"`
	} `json:"model"`
	Execution struct {
		Lane              string `json:"lane"`
		OS                string `json:"os"`
		Arch              string `json:"arch"`
		Accelerator       string `json:"accelerator"`
		MemoryBudgetBytes int64  `json:"memory_budget_bytes"`
		PeakMemoryBytes   int64  `json:"peak_memory_bytes"`
		PeakMemorySource  string `json:"peak_memory_source"`
		InferencePassed   bool   `json:"inference_passed"`
	} `json:"execution"`
	Acceptance map[string]struct {
		InBand                   bool `json:"in_band"`
		Measured                 bool `json:"measured"`
		MeasurementIntegrityPass bool `json:"measurement_integrity_passed"`
	} `json:"acceptance"`
}

// ParseModelMemoryReceipt rejects incomplete, ambiguous, or unversioned
// evidence before it enters the durable qualification store. Catalog-specific
// joins and exact declared-budget checks belong to the promotion resolver.
func ParseModelMemoryReceipt(raw []byte, passed bool) (ModelMemoryReceipt, error) {
	var receipt ModelMemoryReceipt
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return ModelMemoryReceipt{}, fmt.Errorf("parse model-memory receipt: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			return ModelMemoryReceipt{}, errors.New("model-memory receipt contains trailing JSON")
		}
		return ModelMemoryReceipt{}, fmt.Errorf("decode model-memory receipt trailer: %w", err)
	}
	if receipt.SchemaVersion != 1 {
		return ModelMemoryReceipt{}, fmt.Errorf("model-memory schema_version must be 1, got %d", receipt.SchemaVersion)
	}
	cell := receipt.ProviderCell
	if cell.EngineID == "" || cell.ModelID == "" || cell.Capability == "" || cell.Strategy == "" {
		return ModelMemoryReceipt{}, errors.New("model-memory provider_cell requires engine_id, model_id, capability, and strategy")
	}
	model := receipt.Model
	if model.Runtime == "" || model.RuntimeModelID == "" || model.ArtifactResource == "" || model.ArtifactName == "" {
		return ModelMemoryReceipt{}, errors.New("model-memory model requires runtime, runtime_model_id, artifact_resource, and artifact_name")
	}
	if !lowercaseSHA256(model.ArtifactSHA256) || !lowercaseSHA256(model.RuntimeSHA256) {
		return ModelMemoryReceipt{}, errors.New("model-memory artifact_sha256 and runtime_sha256 must be lowercase 64-character SHA-256 hex")
	}
	execution := receipt.Execution
	if execution.Lane != "cpu" || execution.OS == "" || execution.Arch == "" || execution.Accelerator != "none" {
		return ModelMemoryReceipt{}, errors.New("model-memory execution requires lane=cpu, os, arch, and accelerator=none")
	}
	if execution.MemoryBudgetBytes <= 0 || execution.PeakMemoryBytes <= 0 || execution.PeakMemorySource != "cgroup-v2-memory.peak" {
		return ModelMemoryReceipt{}, errors.New("model-memory execution requires positive budget and measured peak from cgroup-v2-memory.peak")
	}
	claim, ok := receipt.Acceptance["memory-budget"]
	if !ok {
		return ModelMemoryReceipt{}, errors.New("model-memory acceptance requires memory-budget claim")
	}
	if passed && (!execution.InferencePassed || execution.PeakMemoryBytes > execution.MemoryBudgetBytes || !claim.InBand || !claim.Measured || !claim.MeasurementIntegrityPass) {
		return ModelMemoryReceipt{}, errors.New("passing model-memory evidence requires completed inference, an in-budget measured peak, and passing integrity acceptance")
	}
	if !passed && claim.InBand && claim.Measured && claim.MeasurementIntegrityPass && execution.InferencePassed && execution.PeakMemoryBytes <= execution.MemoryBudgetBytes {
		return ModelMemoryReceipt{}, errors.New("model-memory evidence outcome cannot fail when every declared acceptance condition passes")
	}
	return receipt, nil
}

func lowercaseSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
