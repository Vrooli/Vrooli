// Package walletrecovery owns Scenario-to-Cloud's consumer path for Treasury's
// portable, key-only wallet recovery artifact. DBM transports opaque bytes;
// Treasury alone decrypts and restores them.
package walletrecovery

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const maxBundleBytes = 1 << 20

var walletIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

// Runner executes an owner CLI with exact argv. Stdin is reserved for the
// operator-held Treasury recovery secret; callers must not log or persist it.
type Runner func(context.Context, string, []string, []byte) ([]byte, error)

// Commands implements the portable wallet restore and post-restart proof
// workflow through the production DBM and Treasury command surfaces.
type Commands struct {
	In     io.Reader
	Out    io.Writer
	Runner Runner
}

func (c Commands) runner() Runner {
	if c.Runner != nil {
		return c.Runner
	}
	return func(ctx context.Context, name string, args []string, stdin []byte) ([]byte, error) {
		cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // fixed owner executable and parsed argv; no shell
		cmd.Stdin = bytes.NewReader(stdin)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			// Owner CLIs can echo target metadata or request details. Keep
			// their stderr out of this boundary's diagnostics; the named
			// step-specific error below is the actionable operator surface.
			return nil, fmt.Errorf("%s failed: %w", name, err)
		}
		return stdout.Bytes(), nil
	}
}

// Run exposes `scenario-to-cloud wallet-recovery <rotate|restore|prove-after-restart>`.
func (c Commands) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: scenario-to-cloud wallet-recovery <rotate|restore|prove-after-restart> [flags]")
	}
	switch args[0] {
	case "rotate":
		return c.rotate(args[1:])
	case "restore":
		return c.restore(args[1:])
	case "prove-after-restart":
		return c.proveAfterRestart(args[1:])
	default:
		return fmt.Errorf("unknown wallet-recovery action %q; use rotate, restore or prove-after-restart", args[0])
	}
}

// Restore runs the DBM-to-Treasury restore subcommand.
func (c Commands) Restore(args []string) error { return c.restore(args) }

func (c Commands) Rotate(args []string) error { return c.rotate(args) }

// ProveAfterRestart runs the separate Treasury control-proof subcommand.
func (c Commands) ProveAfterRestart(args []string) error { return c.proveAfterRestart(args) }

func (c Commands) restore(args []string) error {
	fs := flag.NewFlagSet("wallet-recovery restore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("wallet-id", "", "Treasury wallet id")
	network := fs.String("network", "", "Treasury wallet network")
	bridgeNodeID := fs.String("bridge-node-id", "", "durable Bridge Node ID for this recovery host")
	bridgeMachineID := fs.String("bridge-machine-id", "", "durable Bridge Machine ID for this recovery host")
	bridgeScope := fs.String("bridge-scope", "", "exact registry-approved Bridge scope authorizing recovery on this node")
	destinationID := fs.String("destination-id", "", "existing DBM destination id")
	destinationName := fs.String("destination-name", "", "slug-safe DBM repository name")
	backend := fs.String("backend", "", "existing DBM backend: s3 or sftp")
	location := fs.String("location", "", "non-secret existing repository location")
	createdAt := fs.String("created-at", "", "original destination creation time, RFC3339")
	stage := fs.String("staging-dir", "", "new private empty directory for the encrypted bundle")
	evidenceFile := fs.String("evidence-file", "", "new private path for non-secret provider-recovery evidence; consumed by prove-after-restart")
	secretStdin := fs.Bool("secret-stdin", false, "read the operator-held recovery secret from stdin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*secretStdin {
		return errors.New("the recovery secret must be piped on stdin with --secret-stdin")
	}
	for name, value := range map[string]string{"wallet-id": *id, "network": *network, "bridge-node-id": *bridgeNodeID, "bridge-machine-id": *bridgeMachineID, "bridge-scope": *bridgeScope, "destination-id": *destinationID, "destination-name": *destinationName, "location": *location, "created-at": *createdAt, "staging-dir": *stage, "evidence-file": *evidenceFile} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("--%s is required", name)
		}
	}
	if !walletIDPattern.MatchString(*id) {
		return errors.New("--wallet-id must use Treasury's letters, digits, underscore or hyphen form (maximum 80 characters)")
	}
	if *backend != "s3" && *backend != "sftp" {
		return errors.New("--backend must be s3 or sftp for an off-host repository")
	}
	if _, err := time.Parse(time.RFC3339Nano, *createdAt); err != nil {
		return fmt.Errorf("--created-at must be RFC3339: %w", err)
	}
	stageDir, err := prepareStage(*stage)
	if err != nil {
		return err
	}
	evidencePath, err := prepareEvidencePath(*evidenceFile, stageDir)
	if err != nil {
		return err
	}
	recoveryStartedAt := time.Now().UTC()
	bundlePath := filepath.Join(stageDir, "wallet.bundle")
	witnessPath := filepath.Join(stageDir, "witness.json")
	runner := c.runner()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err = verifyBridgeRecoveryTarget(ctx, runner, *bridgeNodeID, *bridgeMachineID, *bridgeScope); err != nil {
		return fmt.Errorf("Bridge recovery target preflight: %w", err)
	}
	if _, err = runner(ctx, "data-backup-manager", []string{"destinations", "import-existing", "--id", *destinationID, "--name", *destinationName, "--backend", *backend, "--location", *location, "--created-at", *createdAt}, nil); err != nil {
		return fmt.Errorf("attach existing off-host DBM repository: %w", err)
	}
	targetName := "treasury-wallet-recovery-" + *id
	targetsOut, err := runner(ctx, "data-backup-manager", []string{"targets", "list", "--owner", "treasury", "--json"}, nil)
	if err != nil {
		return fmt.Errorf("inspect existing DBM recovery target identity: %w", err)
	}
	if err = verifyTargetRegistrationPath(targetsOut, targetName, stageDir); err != nil {
		return err
	}
	snapshotOut, err := runner(ctx, "data-backup-manager", []string{"destinations", "snapshots", "--id", *destinationID, "--owner", "treasury", "--name", targetName, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("list authenticated recovery target history: %w", err)
	}
	snapshotID, err := latestSnapshotID(snapshotOut, targetName)
	if err != nil {
		return fmt.Errorf("select newest recovery snapshot: %w", err)
	}
	readLease, err := acquireSnapshotReadLease(ctx, runner, targetName, *destinationID, snapshotID)
	if err != nil {
		return fmt.Errorf("reserve exact newest DBM snapshot for Treasury restore: %w", err)
	}
	leaseHeld := true
	defer func() {
		if leaseHeld {
			_ = releaseSnapshotReadLease(context.Background(), runner, readLease)
		}
	}()
	targetOut, err := runner(ctx, "data-backup-manager", []string{"targets", "register", "--owner", "treasury", "--name", targetName, "--kind", "filesystem", "--locator", stageDir, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("register private staging target with DBM: %w", err)
	}
	targetID, err := parseID(targetOut)
	if err != nil {
		return fmt.Errorf("DBM target registration: %w", err)
	}
	verifyArgs := []string{"restores", "verify", "--target", targetID, "--destination", *destinationID, "--snapshot", snapshotID}
	verifyOut, verifyErr := runner(ctx, "data-backup-manager", verifyArgs, nil)
	if verifyErr != nil {
		return fmt.Errorf("verify selected off-host snapshot: %w", verifyErr)
	}
	if err = waitForRestore(ctx, runner, verifyOut, targetID, *destinationID, snapshotID, "RESTORE_MODE_VERIFY", "RESTORE_STATUS_VERIFIED"); err != nil {
		return fmt.Errorf("DBM verification did not reach VERIFIED for the selected snapshot: %w", err)
	}
	restoreArgs := []string{"restores", "restore", "--target", targetID, "--destination", *destinationID, "--snapshot", snapshotID, "--location", stageDir}
	restoreOut, restoreErr := runner(ctx, "data-backup-manager", restoreArgs, nil)
	if restoreErr != nil {
		return fmt.Errorf("restore selected DBM snapshot to private staging: %w", restoreErr)
	}
	if err = waitForRestore(ctx, runner, restoreOut, targetID, *destinationID, snapshotID, "RESTORE_MODE_RESTORE", "RESTORE_STATUS_RESTORED"); err != nil {
		return fmt.Errorf("DBM restore did not reach RESTORED for the selected snapshot: %w", err)
	}
	// The catalog may advance while verify/restore runs. Rebind the restored
	// bytes to the latest snapshot before inspecting or passing them to Treasury.
	finalSnapshotOut, err := runner(ctx, "data-backup-manager", []string{"destinations", "snapshots", "--id", *destinationID, "--owner", "treasury", "--name", targetName, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("recheck newest recovery snapshot after restore: %w", err)
	}
	finalSnapshotID, err := latestSnapshotID(finalSnapshotOut, targetName)
	if err != nil {
		return fmt.Errorf("recheck newest recovery snapshot after restore: %w", err)
	}
	if finalSnapshotID != snapshotID {
		return fmt.Errorf("DBM recovery snapshot changed during restore (selected %s, newest %s); refusing to pass artifacts to Treasury", snapshotID, finalSnapshotID)
	}
	if _, err = renewSnapshotReadLease(ctx, runner, readLease); err != nil {
		return fmt.Errorf("renew DBM recovery read lease before Treasury restore: %w", err)
	}
	revocationPath := filepath.Join(stageDir, "revocation.json")
	if _, statErr := os.Lstat(revocationPath); statErr == nil {
		return errors.New("latest DBM snapshot contains a wallet revocation tombstone; refusing recovery and never falling back to an older snapshot")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect latest recovery revocation state: %w", statErr)
	}
	if err = checkBundle(bundlePath); err != nil {
		return err
	}
	if err = checkWitness(witnessPath); err != nil {
		return err
	}
	secret, err := readSecret(c.input())
	if err != nil {
		return err
	}
	defer clear(secret)
	treasuryArgs := []string{"wallet", "restore-portable-recovery", "--id", *id, "--network", *network, "--bundle-file", bundlePath, "--witness-file", witnessPath, "--secret-stdin", "--json"}
	treasuryOut, err := runner(ctx, "treasury", treasuryArgs, secret)
	if err != nil {
		return errors.New("Treasury rejected the portable recovery bundle; inspect Treasury's private operation receipt")
	}
	wallet, err := restoredWallet(treasuryOut)
	if err != nil || wallet.ID != *id || wallet.Network != *network || wallet.Address == "" {
		return errors.New("Treasury restore response did not confirm the requested wallet identity and address")
	}
	bundleSHA, err := sha256RegularFile(bundlePath, maxBundleBytes)
	if err != nil {
		return fmt.Errorf("hash restored encrypted bundle for provider-recovery evidence: %w", err)
	}
	witnessSHA, err := sha256RegularFile(witnessPath, 16<<10)
	if err != nil {
		return fmt.Errorf("hash restored public witness for provider-recovery evidence: %w", err)
	}
	evidence := RecoveryEvidence{
		Version: 1, WalletID: wallet.ID, Network: wallet.Network, Address: wallet.Address,
		BridgeNodeID: *bridgeNodeID, BridgeMachineID: *bridgeMachineID, BridgeScope: *bridgeScope,
		DBMDestinationID: *destinationID, DBMTargetName: targetName, SnapshotID: snapshotID,
		StagingDir: stageDir, BundleSHA256: bundleSHA, WitnessSHA256: witnessSHA,
		RecoveryStartedAt: recoveryStartedAt.Format(time.RFC3339Nano),
		RestoredAt:        time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err = writeNewEvidence(evidencePath, evidence); err != nil {
		return fmt.Errorf("persist non-secret provider-recovery evidence: %w", err)
	}
	if err = releaseSnapshotReadLease(ctx, runner, readLease); err != nil {
		return fmt.Errorf("Treasury restored the bundle but DBM read lease release failed; it will expire safely: %w", err)
	}
	leaseHeld = false
	return c.write("Wallet recovery bundle fetched by DBM and restored by Treasury. Restart Treasury on this node, then run wallet-recovery prove-after-restart with the expected address, a fresh nonce, and the same --evidence-file.")
}

func (c Commands) proveAfterRestart(args []string) error {
	fs := flag.NewFlagSet("wallet-recovery prove-after-restart", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("wallet-id", "", "Treasury wallet id")
	network := fs.String("network", "", "Treasury wallet network")
	bridgeNodeID := fs.String("bridge-node-id", "", "durable Bridge Node ID recorded by restore")
	bridgeMachineID := fs.String("bridge-machine-id", "", "durable Bridge Machine ID recorded by restore")
	bridgeScope := fs.String("bridge-scope", "", "exact registry-approved Bridge scope recorded by restore")
	address := fs.String("expected-address", "", "address witnessed before the host loss")
	nonce := fs.String("nonce", "", "fresh, unique control-proof nonce")
	evidenceFile := fs.String("evidence-file", "", "provider-recovery evidence created by wallet-recovery restore")
	if err := fs.Parse(args); err != nil {
		return err
	}
	for name, value := range map[string]string{"wallet-id": *id, "network": *network, "bridge-node-id": *bridgeNodeID, "bridge-machine-id": *bridgeMachineID, "bridge-scope": *bridgeScope, "expected-address": *address, "nonce": *nonce, "evidence-file": *evidenceFile} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("--%s is required", name)
		}
	}
	evidence, evidencePath, err := readEvidence(*evidenceFile)
	if err != nil {
		return err
	}
	if evidence.WalletID != *id || evidence.Network != *network || !strings.EqualFold(evidence.Address, *address) || evidence.BridgeNodeID != *bridgeNodeID || evidence.BridgeMachineID != *bridgeMachineID || evidence.BridgeScope != *bridgeScope {
		return errors.New("provider-recovery evidence does not match the requested wallet, network, address and Bridge target")
	}
	if err := verifyBridgeRecoveryTarget(context.Background(), c.runner(), *bridgeNodeID, *bridgeMachineID, *bridgeScope); err != nil {
		return fmt.Errorf("Bridge recovery target preflight: %w", err)
	}
	if err := verifyEvidenceArtifacts(evidence); err != nil {
		return err
	}
	runner := c.runner()
	readLease, err := acquireSnapshotReadLease(context.Background(), runner, evidence.DBMTargetName, evidence.DBMDestinationID, evidence.SnapshotID)
	if err != nil {
		return fmt.Errorf("reserve exact newest DBM snapshot for post-restart proof: %w", err)
	}
	// The lease is capped at 15 minutes. Bound this DBM recheck and Treasury
	// proof span below that cap so an unexpectedly hung owner command cannot
	// outlive the reservation and race a later snapshot/tombstone mutation.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	leaseHeld := true
	defer func() {
		if leaseHeld {
			_ = releaseSnapshotReadLease(context.Background(), runner, readLease)
		}
	}()
	snapshotOut, err := runner(ctx, "data-backup-manager", []string{"destinations", "snapshots", "--id", evidence.DBMDestinationID, "--owner", "treasury", "--name", evidence.DBMTargetName, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("recheck newest DBM recovery snapshot before post-restart proof: %w", err)
	}
	newestSnapshotID, err := latestSnapshotID(snapshotOut, evidence.DBMTargetName)
	if err != nil {
		return fmt.Errorf("recheck newest DBM recovery snapshot before post-restart proof: %w", err)
	}
	if newestSnapshotID != evidence.SnapshotID {
		return fmt.Errorf("DBM recovery snapshot changed after restore (proved %s, newest %s); refusing post-restart Treasury proof", evidence.SnapshotID, newestSnapshotID)
	}
	if _, err = renewSnapshotReadLease(ctx, runner, readLease); err != nil {
		return fmt.Errorf("renew DBM recovery read lease before Treasury proof: %w", err)
	}
	out, err := runner(ctx, "treasury", []string{"wallet", "prove", "--id", *id, "--network", *network, "--nonce", *nonce, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("Treasury post-restart control proof failed: %w", err)
	}
	var response struct {
		Proof struct {
			Wallet struct {
				ID      string `json:"id"`
				Address string `json:"address"`
			} `json:"wallet"`
			Message   string `json:"message"`
			Signature string `json:"signature"`
			Verified  bool   `json:"verified"`
		} `json:"proof"`
	}
	if err := json.Unmarshal(out, &response); err != nil {
		return fmt.Errorf("decode Treasury control proof: %w", err)
	}
	addressMatches := strings.EqualFold(response.Proof.Wallet.Address, *address)
	walletMatches := response.Proof.Wallet.ID == *id
	message := "\n" + response.Proof.Message + "\n"
	messageMatches := strings.Contains(message, "\nwallet:"+*id+"\n") &&
		strings.Contains(message, "\nnetwork:"+*network+"\n") &&
		strings.Contains(message, "\naddress:"+response.Proof.Wallet.Address+"\n") &&
		strings.Contains(message, "\nnonce:"+*nonce+"\n")
	if !response.Proof.Verified || response.Proof.Signature == "" || !addressMatches || !walletMatches || !messageMatches {
		return fmt.Errorf("post-restart control proof mismatch: verified=%t wallet_matches=%t address_matches=%t message_matches=%t signature_present=%t", response.Proof.Verified, walletMatches, addressMatches, messageMatches, response.Proof.Signature != "")
	}
	provedAt := time.Now().UTC()
	evidence.Proof = &RecoveryControlProof{
		WalletID: response.Proof.Wallet.ID, Address: response.Proof.Wallet.Address,
		Nonce: *nonce, Message: response.Proof.Message, Signature: response.Proof.Signature,
		Verified: response.Proof.Verified, ProvedAt: provedAt.Format(time.RFC3339Nano),
	}
	elapsed, err := recoveryElapsed(evidence.RecoveryStartedAt, evidence.Proof.ProvedAt)
	if err != nil {
		return fmt.Errorf("compute wallet recovery elapsed time: %w", err)
	}
	if err := writeUpdatedEvidence(evidencePath, evidence); err != nil {
		return fmt.Errorf("persist verified provider-recovery proof: %w", err)
	}
	if err := releaseSnapshotReadLease(context.Background(), runner, readLease); err != nil {
		return fmt.Errorf("Treasury proved control but DBM read lease release failed; it will expire safely: %w", err)
	}
	leaseHeld = false
	return c.write(fmt.Sprintf("Treasury proved control of the witnessed wallet address after restart. Recovery elapsed: %s from wallet-recovery restore start through signed proof.", elapsed.Round(time.Millisecond)))
}

type recoveredWallet struct {
	ID      string `json:"id"`
	Network string `json:"network"`
	Address string `json:"address"`
}

type dbmReadLease struct {
	LeaseID       string    `json:"lease_id"`
	TargetID      string    `json:"target_id"`
	DestinationID string    `json:"destination_id"`
	SnapshotID    string    `json:"snapshot_id"`
	Generation    int64     `json:"generation"`
	ExpiresAt     time.Time `json:"expires_at"`
}

func acquireSnapshotReadLease(ctx context.Context, runner Runner, targetName, destinationID, snapshotID string) (dbmReadLease, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return dbmReadLease{}, fmt.Errorf("generate recovery lease identity: %w", err)
	}
	leaseID := "wallet-recovery-read-" + hex.EncodeToString(nonce[:])
	out, err := runner(ctx, "data-backup-manager", []string{"wallet-recovery", "read-lease-acquire", "--lease-id", leaseID, "--target-name", targetName, "--destination-id", destinationID, "--snapshot-id", snapshotID, "--json"}, nil)
	if err != nil {
		return dbmReadLease{}, fmt.Errorf("DBM read lease acquire failed: %w", err)
	}
	var lease dbmReadLease
	if err := json.Unmarshal(out, &lease); err != nil || lease.LeaseID != leaseID || lease.TargetID == "" || lease.DestinationID != destinationID || lease.SnapshotID != snapshotID || !lease.ExpiresAt.After(time.Now().UTC()) {
		return dbmReadLease{}, errors.New("DBM read lease response did not bind the requested newest snapshot and target")
	}
	return lease, nil
}

func renewSnapshotReadLease(ctx context.Context, runner Runner, lease dbmReadLease) (dbmReadLease, error) {
	out, err := runner(ctx, "data-backup-manager", []string{"wallet-recovery", "read-lease-renew", "--lease-id", lease.LeaseID, "--target-id", lease.TargetID, "--json"}, nil)
	if err != nil {
		return dbmReadLease{}, err
	}
	var renewed dbmReadLease
	if err := json.Unmarshal(out, &renewed); err != nil || renewed.LeaseID != lease.LeaseID || renewed.TargetID != lease.TargetID || !renewed.ExpiresAt.After(time.Now().UTC()) {
		return dbmReadLease{}, errors.New("DBM read lease renewal did not confirm the same active lease")
	}
	lease.ExpiresAt = renewed.ExpiresAt
	return lease, nil
}

func releaseSnapshotReadLease(ctx context.Context, runner Runner, lease dbmReadLease) error {
	_, err := runner(ctx, "data-backup-manager", []string{"wallet-recovery", "read-lease-release", "--lease-id", lease.LeaseID, "--target-id", lease.TargetID, "--json"}, nil)
	return err
}

func restoredWallet(data []byte) (recoveredWallet, error) {
	var response struct {
		Wallet recoveredWallet `json:"wallet"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return recoveredWallet{}, err
	}
	return response.Wallet, nil
}

func (c Commands) input() io.Reader {
	if c.In != nil {
		return c.In
	}
	return os.Stdin
}
func (c Commands) write(message string) error {
	if c.Out == nil {
		return nil
	}
	_, err := fmt.Fprintln(c.Out, message)
	return err
}

func readSecret(in io.Reader) ([]byte, error) {
	if in == nil {
		return nil, errors.New("stdin is unavailable for the operator-held recovery secret")
	}
	secret, err := io.ReadAll(io.LimitReader(in, 4097))
	if err != nil {
		return nil, fmt.Errorf("read operator-held recovery secret from stdin: %w", err)
	}
	secret = bytes.TrimRight(secret, "\r\n")
	if len(secret) < 32 || len(secret) > 4096 {
		clear(secret)
		return nil, errors.New("operator-held recovery secret must contain 32 to 4096 bytes")
	}
	return secret, nil
}

func prepareStage(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return "", fmt.Errorf("staging directory must already exist and be private: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return "", errors.New("staging directory must be a real directory with mode 0700 or stricter")
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return "", err
	}
	if len(entries) != 0 {
		return "", errors.New("staging directory must be empty")
	}
	return abs, nil
}

func checkBundle(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("DBM restore did not produce the portable bundle: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || info.Size() <= 0 || info.Size() > maxBundleBytes {
		return errors.New("DBM restored an unsafe or oversized portable bundle")
	}
	return nil
}

func checkWitness(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("DBM did not restore the public wallet witness: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 || info.Size() == 0 || info.Size() > 16<<10 {
		return errors.New("DBM restored an unsafe public wallet witness")
	}
	return nil
}

func parseID(data []byte) (string, error) {
	var value struct {
		ID     string `json:"id"`
		Target struct {
			ID string `json:"id"`
		} `json:"target"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return "", err
	}
	if value.Target.ID != "" {
		return value.Target.ID, nil
	}
	if value.ID != "" {
		return value.ID, nil
	}
	return "", errors.New("DBM reply omitted target id")
}

func latestSnapshotID(data []byte, targetName string) (string, error) {
	var response struct {
		Snapshots []struct {
			ID        string `json:"id"`
			StartedAt string `json:"started_at"`
			Source    string `json:"source"`
		} `json:"snapshots"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("decode DBM snapshot list: %w", err)
	}
	if len(response.Snapshots) == 0 {
		return "", errors.New("DBM returned no recovery snapshots")
	}
	var selected string
	var newest time.Time
	for _, snapshot := range response.Snapshots {
		if strings.TrimSpace(snapshot.ID) == "" || snapshot.Source != "dbm://treasury/"+targetName {
			return "", errors.New("DBM returned a snapshot with an unexpected stable source identity")
		}
		started, err := time.Parse(time.RFC3339Nano, snapshot.StartedAt)
		if err != nil {
			return "", errors.New("DBM returned a snapshot with an invalid timestamp")
		}
		if selected == "" || started.After(newest) {
			selected, newest = snapshot.ID, started
		} else if started.Equal(newest) && snapshot.ID != selected {
			return "", errors.New("DBM returned snapshots with an ambiguous newest timestamp")
		}
	}
	return selected, nil
}

func verifyTargetRegistrationPath(data []byte, name, stage string) error {
	var response struct {
		Targets []struct {
			Owner   string `json:"owner"`
			Name    string `json:"name"`
			Locator string `json:"locator"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(data, &response); err != nil || response.Targets == nil {
		return errors.New("DBM target list is missing or malformed; refusing to mutate a stable target identity")
	}
	for _, target := range response.Targets {
		if target.Owner != "treasury" || target.Name != name {
			continue
		}
		locator, err := filepath.Abs(target.Locator)
		if err != nil || locator != stage {
			return errors.New("existing DBM recovery target resolves to a different locator; refusing to change its stable source identity")
		}
	}
	return nil
}
