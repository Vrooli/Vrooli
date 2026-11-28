package walletrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RecoveryEvidence is the non-secret join between DBM's selected snapshot,
// the exact Treasury artifacts restored from it, and the post-restart signed
// control proof. It is the handoff record consumed by the CI7 provider-recovery
// drill; raw bundle, witness bytes, operator secret and repository credentials
// are deliberately excluded.
type RecoveryEvidence struct {
	Version           int                   `json:"version"`
	WalletID          string                `json:"wallet_id"`
	Network           string                `json:"network"`
	Address           string                `json:"address"`
	BridgeNodeID      string                `json:"bridge_node_id"`
	BridgeMachineID   string                `json:"bridge_machine_id"`
	BridgeScope       string                `json:"bridge_scope"`
	DBMDestinationID  string                `json:"dbm_destination_id"`
	DBMTargetName     string                `json:"dbm_target_name"`
	SnapshotID        string                `json:"snapshot_id"`
	StagingDir        string                `json:"staging_dir"`
	BundleSHA256      string                `json:"bundle_sha256"`
	WitnessSHA256     string                `json:"witness_sha256"`
	RecoveryStartedAt string                `json:"recovery_started_at"`
	RestoredAt        string                `json:"restored_at"`
	Proof             *RecoveryControlProof `json:"post_restart_proof,omitempty"`
}

type RecoveryControlProof struct {
	WalletID  string `json:"wallet_id"`
	Address   string `json:"address"`
	Nonce     string `json:"nonce"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
	Verified  bool   `json:"verified"`
	ProvedAt  string `json:"proved_at"`
}

func prepareEvidencePath(path, stage string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(abs)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return "", errors.New("provider-recovery evidence parent must be a real private directory with mode 0700 or stricter")
	}
	rel, err := filepath.Rel(stage, abs)
	if err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return "", errors.New("provider-recovery evidence must be stored outside the DBM staging target")
	}
	if _, err := os.Lstat(abs); err == nil {
		return "", errors.New("provider-recovery evidence path already exists; choose a new private evidence file")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect provider-recovery evidence path: %w", err)
	}
	return abs, nil
}

func writeNewEvidence(path string, evidence RecoveryEvidence) error {
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(path)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return closeErr
	}
	return nil
}

func readEvidence(path string) (RecoveryEvidence, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return RecoveryEvidence{}, "", err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return RecoveryEvidence{}, "", fmt.Errorf("read provider-recovery evidence: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || info.Size() <= 0 || info.Size() > 32<<10 {
		return RecoveryEvidence{}, "", errors.New("provider-recovery evidence must be a private regular file no larger than 32 KiB")
	}
	data, err := os.ReadFile(abs)
	if err != nil || len(data) != int(info.Size()) {
		return RecoveryEvidence{}, "", errors.New("provider-recovery evidence changed while it was read")
	}
	var evidence RecoveryEvidence
	if err := json.Unmarshal(data, &evidence); err != nil || evidence.Version != 1 || evidence.Proof != nil {
		return RecoveryEvidence{}, "", errors.New("provider-recovery evidence is malformed or already has a post-restart proof")
	}
	if _, err := time.Parse(time.RFC3339Nano, evidence.RecoveryStartedAt); err != nil {
		return RecoveryEvidence{}, "", errors.New("provider-recovery evidence has an invalid recovery start time")
	}
	return evidence, abs, nil
}

func writeUpdatedEvidence(path string, evidence RecoveryEvidence) error {
	previous, _, err := readEvidence(path)
	if err != nil {
		return err
	}
	if previous.Version != evidence.Version || previous.WalletID != evidence.WalletID || previous.Network != evidence.Network ||
		previous.Address != evidence.Address || previous.DBMDestinationID != evidence.DBMDestinationID ||
		previous.BridgeNodeID != evidence.BridgeNodeID || previous.BridgeMachineID != evidence.BridgeMachineID || previous.BridgeScope != evidence.BridgeScope ||
		previous.DBMTargetName != evidence.DBMTargetName || previous.SnapshotID != evidence.SnapshotID ||
		previous.StagingDir != evidence.StagingDir || previous.BundleSHA256 != evidence.BundleSHA256 ||
		previous.WitnessSHA256 != evidence.WitnessSHA256 || previous.RecoveryStartedAt != evidence.RecoveryStartedAt ||
		previous.RestoredAt != evidence.RestoredAt || previous.Proof != nil || evidence.Proof == nil {
		return errors.New("provider-recovery proof update changed immutable restore evidence")
	}
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".wallet-recovery-evidence-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("provider-recovery evidence path changed before proof receipt update")
	}
	return os.Rename(tempPath, path)
}

// verifyBridgeRecoveryTarget binds a consumer run to Bridge's durable node ID
// and exact approved scope, then requires the control plane's live readiness
// doctor to pass. The node-agent owns heartbeat/channel status; a stale or
// disconnected node therefore fails before DBM import/restore or Treasury.
func verifyBridgeRecoveryTarget(ctx context.Context, runner Runner, nodeID, machineID, scope string) error {
	if strings.TrimSpace(nodeID) == "" || strings.TrimSpace(machineID) == "" || strings.TrimSpace(scope) == "" {
		return errors.New("Bridge node ID, Machine ID and exact approved scope are required")
	}
	if scope != "scenario-to-cloud:read" {
		return errors.New("wallet recovery requires the exact Bridge scope scenario-to-cloud:read")
	}
	out, err := runner(ctx, "vrooli-bridge", []string{"nodes", "get", nodeID, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("read durable Bridge node identity: %w", err)
	}
	var response struct {
		ID     string   `json:"id"`
		Scopes []string `json:"scopes"`
		Node   *struct {
			ID     string   `json:"id"`
			Scopes []string `json:"scopes"`
		} `json:"node"`
	}
	if err := json.Unmarshal(out, &response); err != nil {
		return errors.New("Bridge node response is not valid JSON")
	}
	id, scopes := response.ID, response.Scopes
	if response.Node != nil {
		id, scopes = response.Node.ID, response.Node.Scopes
	}
	if id != nodeID {
		return errors.New("Bridge returned a different durable node identity")
	}
	scopeFound := false
	for _, approved := range scopes {
		if approved == scope {
			scopeFound = true
			break
		}
	}
	if !scopeFound {
		return errors.New("requested recovery scope is not approved for this Bridge node")
	}
	machineOut, err := runner(ctx, "vrooli-bridge", []string{"machines", "get", machineID, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("read durable Bridge machine lineage: %w", err)
	}
	var machineResponse struct {
		Machine *struct {
			ID          string `json:"id"`
			NodeLineage []struct {
				NodeID  string `json:"nodeId"`
				Current bool   `json:"current"`
			} `json:"nodeLineage"`
		} `json:"machine"`
	}
	if err := json.Unmarshal(machineOut, &machineResponse); err != nil || machineResponse.Machine == nil || machineResponse.Machine.ID != machineID {
		return errors.New("Bridge machine response does not match the durable Machine identity")
	}
	currentNode := false
	for _, lineage := range machineResponse.Machine.NodeLineage {
		if lineage.NodeID == nodeID && lineage.Current {
			currentNode = true
			break
		}
	}
	if !currentNode {
		return errors.New("Bridge node is not the current node in the requested Machine lineage")
	}
	if _, err := runner(ctx, "vrooli-bridge", []string{"nodes", "doctor", nodeID, "--json"}, nil); err != nil {
		return errors.New("Bridge node is stale, disconnected or not dispatchable")
	}
	return nil
}

func verifyEvidenceArtifacts(evidence RecoveryEvidence) error {
	if evidence.StagingDir == "" || evidence.BundleSHA256 == "" || evidence.WitnessSHA256 == "" {
		return errors.New("provider-recovery evidence omits its staging identity or artifact digests")
	}
	bundlePath := filepath.Join(evidence.StagingDir, "wallet.bundle")
	witnessPath := filepath.Join(evidence.StagingDir, "witness.json")
	bundleHash, err := sha256RegularFile(bundlePath, maxBundleBytes)
	if err != nil || bundleHash != evidence.BundleSHA256 {
		return errors.New("provider-recovery bundle bytes no longer match the DBM snapshot evidence")
	}
	witnessHash, err := sha256RegularFile(witnessPath, 16<<10)
	if err != nil || witnessHash != evidence.WitnessSHA256 {
		return errors.New("provider-recovery witness bytes no longer match the DBM snapshot evidence")
	}
	return nil
}

func recoveryElapsed(start, end string) (time.Duration, error) {
	startedAt, err := time.Parse(time.RFC3339Nano, start)
	if err != nil {
		return 0, errors.New("invalid recovery start time")
	}
	provedAt, err := time.Parse(time.RFC3339Nano, end)
	if err != nil {
		return 0, errors.New("invalid post-restart proof time")
	}
	if provedAt.Before(startedAt) {
		return 0, errors.New("post-restart proof time precedes recovery start")
	}
	return provedAt.Sub(startedAt), nil
}

func sha256RegularFile(path string, maxBytes int64) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || info.Size() <= 0 || info.Size() > maxBytes {
		return "", errors.New("artifact is not a private regular file within its size limit")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) != int(info.Size()) {
		return "", errors.New("artifact changed while it was hashed")
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
