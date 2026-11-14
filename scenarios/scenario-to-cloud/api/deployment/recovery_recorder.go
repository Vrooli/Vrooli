package deployment

import (
	"context"
	"encoding/json"
	"fmt"

	"scenario-to-cloud/backup"
	"scenario-to-cloud/vps"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// RecoveryPointRecorder stores the recovery point the target owner captured
// during data.backup on the cloud side, so rollback admission, retention and
// restore can reference it by the same id the target holds.
type RecoveryPointRecorder struct {
	Repo backup.Repository
}

var _ vps.RecoveryPointRecorder = (*RecoveryPointRecorder)(nil)

// Record implements vps.RecoveryPointRecorder. An existing record with the
// same id is left unchanged (the target replayed its receipt).
func (r *RecoveryPointRecorder) Record(ctx context.Context, deploymentID string, identity vps.Identity, raw json.RawMessage, releaseDigest string) (string, error) {
	if r == nil || r.Repo == nil {
		return "", fmt.Errorf("recovery point recorder has no repository")
	}
	var manifest recoverypoint.Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", fmt.Errorf("recovery point manifest is not decodable: %w", err)
	}
	if manifest.ID == "" {
		return "", fmt.Errorf("recovery point manifest carries no id")
	}
	if existing, err := r.Repo.GetRecoveryPoint(ctx, manifest.ID); err != nil {
		return "", err
	} else if existing != nil {
		return existing.ID, nil
	}
	rp := backup.FromManifest(manifest, "target:"+deploymentID+"/"+manifest.ID)
	rp.DeploymentID = deploymentID
	rp.OperationID = identity.OperationID
	rp.ReleaseDigest = releaseDigest
	rp.ProtectedBy = []string{"operation:" + identity.OperationID}
	rp.Protected = true
	stored, err := r.Repo.CreateRecoveryPoint(ctx, rp)
	if err != nil {
		return "", err
	}
	return stored.ID, nil
}
