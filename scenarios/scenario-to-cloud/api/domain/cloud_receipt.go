package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CloudDeploymentReceipt is the owner-produced proof that a VPS deployment
// reached the requested destination and passed the deployer's health checks.
// It contains coordinates and artifact identity, never credentials.
type CloudDeploymentReceipt struct {
	SchemaVersion      int    `json:"schema_version"`
	DeploymentID       string `json:"deployment_id"`
	ScenarioID         string `json:"scenario_id"`
	ScenarioVersion    string `json:"scenario_version"`
	TargetKind         string `json:"target_kind"`
	DestinationID      string `json:"destination_id"`
	DestinationHost    string `json:"destination_host"`
	DestinationWorkdir string `json:"destination_workdir"`
	DestinationDomain  string `json:"destination_domain"`
	BundleSHA256       string `json:"bundle_sha256"`
	Outcome            string `json:"outcome"`
	Health             string `json:"health"`
	ExternalReceipt    string `json:"external_receipt"`
	ObservedAt         string `json:"observed_at"`
}

// CloudRecoveryReceipt is the owner-produced proof of a recovery effect.
// A preview is deliberately represented as an unavailable effect so callers
// cannot mistake planning for a remote mutation.
type CloudRecoveryReceipt struct {
	SchemaVersion   int    `json:"schema_version"`
	DeploymentID    string `json:"deployment_id"`
	OperationID     string `json:"operation_id,omitempty"`
	Action          string `json:"action"`
	Outcome         string `json:"outcome"`
	Health          string `json:"health"`
	BundleSHA256    string `json:"bundle_sha256,omitempty"`
	ExternalReceipt string `json:"external_receipt"`
	ObservedAt      string `json:"observed_at"`
	DryRun          bool   `json:"dry_run"`
	Error           string `json:"error,omitempty"`
}

func (r CloudRecoveryReceipt) Valid() bool {
	validOutcome := r.Outcome == "halted" || r.Outcome == "rolled_back" || r.Outcome == "forward_repaired"
	validHealth := r.Health == "stopped" || r.Health == "healthy"
	if r.Action != "halt" && (r.OperationID == "" || r.BundleSHA256 == "") {
		return false
	}
	return r.SchemaVersion == 1 && r.DeploymentID != "" && r.Action != "" &&
		validOutcome && validHealth && r.ExternalReceipt != "" &&
		r.ObservedAt != "" && !r.DryRun
}

// RecoveryOperation records the owner-side durable lifecycle for a recovery
// request. The operation row is the source of truth while a repair runs; the
// receipt is emitted only after the deployment record and persisted deploy
// result agree on the repaired bundle.
type RecoveryOperation struct {
	ID                string                `json:"id"`
	DeploymentID      string                `json:"deployment_id"`
	IdempotencyKey    string                `json:"idempotency_key"`
	Action            string                `json:"action"`
	ExpectedBundleSHA string                `json:"expected_bundle_sha256,omitempty"`
	RepairBundleSHA   string                `json:"repair_bundle_sha256,omitempty"`
	RepairBundlePath  string                `json:"repair_bundle_path,omitempty"`
	DataCompatibility string                `json:"data_compatibility,omitempty"`
	Status            string                `json:"status"`
	Receipt           *CloudRecoveryReceipt `json:"receipt,omitempty"`
	ErrorMessage      string                `json:"error_message,omitempty"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
	StartedAt         *time.Time            `json:"started_at,omitempty"`
	CompletedAt       *time.Time            `json:"completed_at,omitempty"`
}

// BuildCloudDeploymentReceipt constructs a receipt only after the persisted
// deployment record and persisted deploy result agree on a successful effect.
func BuildCloudDeploymentReceipt(deployment *Deployment, observedAt time.Time) (*CloudDeploymentReceipt, error) {
	if deployment == nil {
		return nil, fmt.Errorf("deployment is required")
	}
	if deployment.ID == "" || deployment.ScenarioID == "" {
		return nil, fmt.Errorf("deployment identity is incomplete")
	}
	if deployment.Status != StatusDeployed {
		return nil, fmt.Errorf("deployment %q is not deployed", deployment.ID)
	}
	if !deployment.DeployResult.Valid || len(deployment.DeployResult.Data) == 0 {
		return nil, fmt.Errorf("deployment %q has no persisted deploy result", deployment.ID)
	}
	var result VPSDeployResult
	if err := json.Unmarshal(deployment.DeployResult.Data, &result); err != nil {
		return nil, fmt.Errorf("decode deploy result: %w", err)
	}
	if !result.OK {
		return nil, fmt.Errorf("deployment %q deploy result is not successful", deployment.ID)
	}
	if len(deployment.Manifest) == 0 {
		return nil, fmt.Errorf("deployment %q has no manifest", deployment.ID)
	}
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		return nil, fmt.Errorf("decode deployment manifest: %w", err)
	}
	if manifest.Target.VPS == nil {
		return nil, fmt.Errorf("deployment %q has no VPS target", deployment.ID)
	}
	host := strings.TrimSpace(manifest.Target.VPS.Host)
	workdir := strings.TrimSpace(manifest.Target.VPS.Workdir)
	domain := strings.TrimSpace(manifest.Edge.Domain)
	bundleSHA := ""
	if deployment.BundleSHA256 != nil {
		bundleSHA = strings.TrimSpace(*deployment.BundleSHA256)
	}
	if host == "" || workdir == "" || domain == "" || bundleSHA == "" {
		return nil, fmt.Errorf("deployment %q is missing exact destination or bundle identity", deployment.ID)
	}
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	destinationID := destinationIdentity(deployment.ScenarioID, host, workdir, domain)
	return &CloudDeploymentReceipt{
		SchemaVersion:      1,
		DeploymentID:       deployment.ID,
		ScenarioID:         deployment.ScenarioID,
		ScenarioVersion:    strings.TrimSpace(manifest.Scenario.Ref),
		TargetKind:         "vps",
		DestinationID:      destinationID,
		DestinationHost:    host,
		DestinationWorkdir: workdir,
		DestinationDomain:  domain,
		BundleSHA256:       bundleSHA,
		Outcome:            "deployed",
		Health:             "healthy",
		ExternalReceipt:    "scenario-to-cloud:" + deployment.ID,
		ObservedAt:         observedAt.UTC().Format(time.RFC3339Nano),
	}, nil
}

func destinationIdentity(scenarioID, host, workdir, domain string) string {
	canonical := strings.Join([]string{scenarioID, host, workdir, domain}, "\x00")
	sum := sha256.Sum256([]byte(canonical))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Valid reports whether the receipt has enough owner evidence to be consumed
// as a successful cloud effect.
func (r CloudDeploymentReceipt) Valid() bool {
	return r.SchemaVersion == 1 && r.DeploymentID != "" && r.ScenarioID != "" &&
		r.TargetKind == "vps" && r.DestinationID != "" && r.DestinationHost != "" &&
		r.DestinationWorkdir != "" && r.DestinationDomain != "" && r.BundleSHA256 != "" &&
		r.Outcome == "deployed" && r.Health == "healthy" && r.ExternalReceipt != "" && r.ObservedAt != ""
}
