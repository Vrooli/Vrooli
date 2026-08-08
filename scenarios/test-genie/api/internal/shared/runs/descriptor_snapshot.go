package runs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// DescriptorSnapshotSchemaVersion 2 adds the comparison contract. Version 1
// snapshots remain readable as explicitly legacy evidence; they do not acquire
// invented comparison semantics on read.
const DescriptorSnapshotSchemaVersion = 2

var (
	ErrDescriptorSnapshotNotFound           = errors.New("run descriptor snapshot not found")
	ErrInvalidDescriptorSnapshot            = errors.New("invalid run descriptor snapshot")
	ErrUnsupportedDescriptorSnapshotVersion = errors.New("unsupported run descriptor snapshot version")
)

// DescriptorPolicySnapshot preserves the provider-owned policy vocabulary
// without coupling durable shared storage to the orchestrator policy package.
type DescriptorPolicySnapshot struct {
	Selection         string `json:"selection,omitempty"`
	ProviderReadiness string `json:"provider_readiness,omitempty"`
	ProviderLifecycle string `json:"provider_lifecycle,omitempty"`
	Freshness         string `json:"freshness,omitempty"`
	ResultGating      string `json:"result_gating,omitempty"`
	Unavailable       string `json:"unavailable,omitempty"`
}

type ApplicabilityReasonSnapshot struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ApplicabilityDecisionSnapshot struct {
	Status  string                        `json:"status"`
	Reasons []ApplicabilityReasonSnapshot `json:"reasons,omitempty"`
	Planned bool                          `json:"planned"`
}

type DeterminismSnapshot struct {
	Default      string                         `json:"default,omitempty"`
	Inputs       []string                       `json:"inputs,omitempty"`
	Reason       string                         `json:"reason,omitempty"`
	Capabilities map[string]DeterminismOverride `json:"capabilities,omitempty"`
}

type DeterminismOverride struct {
	Mode   string   `json:"mode,omitempty"`
	Inputs []string `json:"inputs,omitempty"`
	Reason string   `json:"reason,omitempty"`
}

// PhaseDescriptorSnapshot is the historical presentation and policy contract
// for one immutable phase key.
type PhaseDescriptorSnapshot struct {
	Phase                string                   `json:"phase"`
	DisplayName          string                   `json:"display_name,omitempty"`
	Description          string                   `json:"description,omitempty"`
	Provider             string                   `json:"provider,omitempty"`
	OrderHint            int                      `json:"order_hint,omitempty"`
	PhaseClass           string                   `json:"phase_class,omitempty"`
	RuntimeClass         string                   `json:"runtime_class,omitempty"`
	Dimensions           []string                 `json:"dimensions,omitempty"`
	FindingSource        string                   `json:"finding_source,omitempty"`
	Policy               DescriptorPolicySnapshot `json:"policy"`
	DocsPath             string                   `json:"docs_path,omitempty"`
	MaturityReference    string                   `json:"maturity_reference,omitempty"`
	ApplicabilityDefault string                   `json:"applicability_default,omitempty"`
	EvidenceKinds        []string                 `json:"evidence_kinds,omitempty"`
	Aliases              []string                 `json:"aliases,omitempty"`
	Supersedes           []string                 `json:"supersedes,omitempty"`
	// ComparisonFingerprint is derived only from validation semantics and
	// execution policy, never from presentation metadata. It lets comparison
	// distinguish an oracle change from a change in observed behavior.
	ComparisonFingerprint string `json:"comparison_fingerprint,omitempty"`
	// ComparisonMode declares how a changed fingerprint is handled:
	// compatible | changed-unreviewed | invalidated | superseded.
	ComparisonMode         string                        `json:"comparison_mode,omitempty"`
	ValidationContract     string                        `json:"validation_contract,omitempty"`
	ValidationDeliveryMode string                        `json:"validation_delivery_mode,omitempty"`
	ValidationExecution    bool                          `json:"validation_execution,omitempty"`
	ValidationRunService   string                        `json:"validation_run_service,omitempty"`
	Determinism            DeterminismSnapshot           `json:"determinism,omitempty"`
	Applicability          ApplicabilityDecisionSnapshot `json:"applicability"`
}

// DescriptorSnapshot is written once before execution. Digest covers the
// canonical schema_version+phases payload and is verified on every read.
type DescriptorSnapshot struct {
	SchemaVersion int                       `json:"schema_version"`
	Digest        string                    `json:"digest"`
	Phases        []PhaseDescriptorSnapshot `json:"phases"`
}

func NewDescriptorSnapshot(phases []PhaseDescriptorSnapshot) (DescriptorSnapshot, error) {
	for i := range phases {
		if phases[i].ComparisonFingerprint == "" {
			fingerprint, err := PhaseComparisonFingerprint(phases[i])
			if err != nil {
				return DescriptorSnapshot{}, err
			}
			phases[i].ComparisonFingerprint = fingerprint
		}
		if phases[i].ComparisonMode == "" {
			phases[i].ComparisonMode = "changed-unreviewed"
		}
	}
	snapshot := DescriptorSnapshot{
		SchemaVersion: DescriptorSnapshotSchemaVersion,
		Phases:        append([]PhaseDescriptorSnapshot(nil), phases...),
	}
	digest, err := descriptorSnapshotDigest(snapshot)
	if err != nil {
		return DescriptorSnapshot{}, err
	}
	snapshot.Digest = digest
	return snapshot, nil
}

func WriteDescriptorSnapshot(scenarioDir, runID string, snapshot DescriptorSnapshot) error {
	if snapshot.SchemaVersion != DescriptorSnapshotSchemaVersion {
		return fmt.Errorf("%w: got %d", ErrUnsupportedDescriptorSnapshotVersion, snapshot.SchemaVersion)
	}
	expected, err := descriptorSnapshotDigest(snapshot)
	if err != nil {
		return err
	}
	if snapshot.Digest == "" {
		snapshot.Digest = expected
	}
	if snapshot.Digest != expected {
		return fmt.Errorf("%w: digest mismatch", ErrInvalidDescriptorSnapshot)
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal descriptor snapshot: %w", err)
	}
	return writeAtomicDescriptorSnapshot(sharedartifacts.RunDescriptorSnapshotPath(scenarioDir, runID), data)
}

func ReadDescriptorSnapshot(scenarioDir, runID string) (DescriptorSnapshot, error) {
	data, err := os.ReadFile(sharedartifacts.RunDescriptorSnapshotPath(scenarioDir, runID))
	if err != nil {
		if os.IsNotExist(err) {
			return DescriptorSnapshot{}, ErrDescriptorSnapshotNotFound
		}
		return DescriptorSnapshot{}, fmt.Errorf("read descriptor snapshot: %w", err)
	}
	var snapshot DescriptorSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return DescriptorSnapshot{}, fmt.Errorf("%w: decode: %v", ErrInvalidDescriptorSnapshot, err)
	}
	if snapshot.SchemaVersion != 1 && snapshot.SchemaVersion != DescriptorSnapshotSchemaVersion {
		return DescriptorSnapshot{}, fmt.Errorf("%w: got %d", ErrUnsupportedDescriptorSnapshotVersion, snapshot.SchemaVersion)
	}
	if len(snapshot.Phases) == 0 {
		return DescriptorSnapshot{}, fmt.Errorf("%w: phase catalog is empty", ErrInvalidDescriptorSnapshot)
	}
	expected, err := descriptorSnapshotDigest(snapshot)
	if err != nil {
		return DescriptorSnapshot{}, err
	}
	if snapshot.Digest == "" || snapshot.Digest != expected {
		return DescriptorSnapshot{}, fmt.Errorf("%w: digest mismatch", ErrInvalidDescriptorSnapshot)
	}
	return snapshot, nil
}

// PhaseComparisonFingerprint is the stable, descriptor-owned identity of a
// phase's oracle. Display copy, docs, aliases, ordering, and maturity labels
// intentionally do not participate, because changing them must not invalidate
// behavioral comparison.
func PhaseComparisonFingerprint(phase PhaseDescriptorSnapshot) (string, error) {
	payload := struct {
		Phase                  string                   `json:"phase"`
		Provider               string                   `json:"provider"`
		Policy                 DescriptorPolicySnapshot `json:"policy"`
		ApplicabilityDefault   string                   `json:"applicability_default"`
		EvidenceKinds          []string                 `json:"evidence_kinds"`
		PhaseClass             string                   `json:"phase_class"`
		RuntimeClass           string                   `json:"runtime_class"`
		ValidationContract     string                   `json:"validation_contract"`
		ValidationDeliveryMode string                   `json:"validation_delivery_mode"`
		ValidationExecution    bool                     `json:"validation_execution"`
		ValidationRunService   string                   `json:"validation_run_service"`
	}{
		Phase: phase.Phase, Provider: phase.Provider, Policy: phase.Policy,
		ApplicabilityDefault: phase.ApplicabilityDefault,
		EvidenceKinds:        append([]string(nil), phase.EvidenceKinds...),
		PhaseClass:           phase.PhaseClass, RuntimeClass: phase.RuntimeClass,
		ValidationContract: phase.ValidationContract, ValidationDeliveryMode: phase.ValidationDeliveryMode,
		ValidationExecution: phase.ValidationExecution, ValidationRunService: phase.ValidationRunService,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal phase comparison fingerprint: %w", err)
	}
	sum := sha256.Sum256(data)
	return "pcf:sha256:" + hex.EncodeToString(sum[:]), nil
}

func descriptorSnapshotDigest(snapshot DescriptorSnapshot) (string, error) {
	payload := struct {
		SchemaVersion int                       `json:"schema_version"`
		Phases        []PhaseDescriptorSnapshot `json:"phases"`
	}{SchemaVersion: snapshot.SchemaVersion, Phases: snapshot.Phases}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal descriptor snapshot digest: %w", err)
	}
	sum := sha256.Sum256(data)
	return "ds:sha256:" + hex.EncodeToString(sum[:]), nil
}

func writeAtomicDescriptorSnapshot(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create descriptor snapshot dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".descriptor-snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("create descriptor snapshot tmp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write descriptor snapshot tmp: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod descriptor snapshot tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync descriptor snapshot tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close descriptor snapshot tmp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace descriptor snapshot: %w", err)
	}
	if handle, err := os.Open(dir); err == nil {
		_ = handle.Sync()
		_ = handle.Close()
	}
	return nil
}
