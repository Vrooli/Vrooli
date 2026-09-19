package onboard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CredentialStoreRotator replaces a managed node's store passphrase from the
// control plane. The onboarding service implements it because it already owns
// the node-CLI path, the escrow, and the unlock grant the rotation needs.
type CredentialStoreRotator interface {
	RotateCredentialStore(ctx context.Context, in RotateCredentialStoreInput) (RotateCredentialStoreResult, error)
}

// RotateCredentialStoreInput names the machine, its current node, and how to
// reach it. The machines handler resolves these from Bridge's own records.
type RotateCredentialStoreInput struct {
	MachineID string
	NodeID    string
	Conn      Conn
	Platform  NodePlatform
}

// Agent-unlock outcomes after a rotation.
const (
	// AgentUnlockVerified — a fresh node shell opened the store with the
	// passphrase the agent now serves, and the store has no other wrap that
	// could have opened it.
	AgentUnlockVerified = "verified"
	// AgentUnlockNotRequired — the store opens through an unattended wrap at
	// boot, so the agent's copy is recovery-only; it was re-pushed regardless.
	AgentUnlockNotRequired = "not_required"
	// AgentUnlockUnverified — the new passphrase was pushed but a node shell
	// could not open the store with it within the wait.
	AgentUnlockUnverified = "unverified"
)

// RotateCredentialStoreResult reports what changed. It never carries a
// passphrase.
type RotateCredentialStoreResult struct {
	MachineID   string
	NodeID      string
	Resumed     bool
	AgentUnlock string
	Detail      string
}

// Stable rotation refusal codes. The API maps them to status codes and
// callers branch on them, not on the prose.
const (
	RotationUnavailable         = "rotation_unavailable"
	RotationUnsupportedPlatform = "unsupported_platform"
	RotationNodeUnreachable     = "node_unreachable"
	RotationStoreAbsent         = "store_absent"
	RotationNotEscrowed         = "not_escrowed"
	RotationEscrowMismatch      = "escrow_mismatch"
	RotationNodeOutdated        = "node_cli_outdated"
	RotationEscrowUnavailable   = "escrow_unavailable"
	RotationNodeCommandFailed   = "node_command_failed"
)

// ErrCredentialStoreRotation is a typed rotation refusal or failure. Every
// variant leaves at least one escrowed passphrase that opens the node; Detail
// says which and what re-running does.
type ErrCredentialStoreRotation struct {
	Code   string
	Detail string
}

func (e ErrCredentialStoreRotation) Error() string {
	return fmt.Sprintf("credential store rotation %s: %s", e.Code, e.Detail)
}

// Retryable reports whether re-running the rotation is the remedy (it
// reconciles whatever an interrupted run left).
func (e ErrCredentialStoreRotation) Retryable() bool {
	switch e.Code {
	case RotationNodeUnreachable, RotationEscrowUnavailable, RotationNodeCommandFailed:
		return true
	default:
		return false
	}
}

// Agent-unlock polling after the grant is re-pushed. Package variables so
// tests do not wait.
var (
	agentUnlockPollInterval = 2 * time.Second
	agentUnlockPollAttempts = 15
)

var _ CredentialStoreRotator = (*service)(nil)

// RotateCredentialStore generates a new passphrase for the node's store and
// swaps it in with this ordering, so an interruption at any point leaves an
// escrowed value that opens the node:
//
//  1. finish or discard any earlier interrupted rotation (pending slot);
//  2. prove the escrowed passphrase opens the node's passphrase wrap;
//  3. write the new passphrase to the pending escrow slot;
//  4. change the node's passphrase wrap (current → new, one atomic file write);
//  5. prove the new passphrase opens it;
//  6. promote pending to the escrow and clear pending;
//  7. re-push the unlock grant and prove the agent serves the new value.
//
// Before 4 the escrow opens the node; between 4 and 6 the pending slot does;
// after 6 the escrow does again. A re-run starts at 1, so it completes an
// interrupted rotation instead of starting over.
func (s *service) RotateCredentialStore(ctx context.Context, in RotateCredentialStoreInput) (RotateCredentialStoreResult, error) {
	result := RotateCredentialStoreResult{MachineID: in.MachineID, NodeID: in.NodeID}
	if s.storeEscrow == nil {
		return result, ErrCredentialStoreRotation{Code: RotationUnavailable, Detail: "this control plane does not escrow node credential-store passphrases"}
	}
	pendingEscrow, ok := s.storeEscrow.(CredentialStorePendingEscrow)
	if !ok {
		return result, ErrCredentialStoreRotation{Code: RotationUnavailable, Detail: "this control plane's escrow cannot hold a pending passphrase, so a rotation could not survive an interruption"}
	}
	if strings.EqualFold(in.Platform.OS, "windows") {
		return result, ErrCredentialStoreRotation{Code: RotationUnsupportedPlatform, Detail: windowsCredentialStoreSkip}
	}
	status, err := s.readNodeStoreStatus(ctx, in.Conn, in.Platform)
	if err != nil {
		return result, ErrCredentialStoreRotation{Code: RotationNodeUnreachable, Detail: "could not read the node's credential store: " + err.Error()}
	}
	if !status.Initialized {
		return result, ErrCredentialStoreRotation{Code: RotationStoreAbsent, Detail: "the node has no credential store; repair the machine so Bridge creates and escrows one"}
	}
	key := credentialStoreEscrowKey(in.MachineID, in.NodeID)
	current, found, err := s.storeEscrow.Load(ctx, key)
	defer func() { zeroBytes(current) }()
	if err != nil {
		return result, ErrCredentialStoreRotation{Code: RotationEscrowUnavailable, Detail: "this control plane could not read its escrowed passphrase: " + redact(err.Error(), current)}
	}
	if !found {
		return result, ErrCredentialStoreRotation{Code: RotationNotEscrowed, Detail: "this control plane holds no passphrase for the node's store (wraps: " + wrapProviders(status.Wraps) + "); repair the machine so Bridge escrows it, or restore the store from a recovery bundle"}
	}

	// 1–2. Converge on a single escrowed value that provably opens the node.
	if !status.hasWrap(nodePassphraseWrapProvider) {
		if err := s.addNodePassphraseWrap(ctx, in.Conn, in.Platform, current); err != nil {
			return result, rotationNodeError(err, current)
		}
	} else {
		effective, resumed, reconcileErr := s.reconcilePendingEscrow(ctx, in.Conn, in.Platform, key, current)
		current = effective
		if reconcileErr != nil {
			return result, rotationNodeError(reconcileErr, current)
		}
		if resumed {
			// The interrupted rotation already produced a passphrase no
			// earlier grant carried; finishing it is the rotation.
			result.Resumed = true
			result.AgentUnlock, result.Detail = s.pushAndProveUnlock(ctx, in, key, current, status)
			result.Detail = "finished an interrupted rotation: the node already used the new passphrase, which is now the escrowed one; " + result.Detail
			return result, nil
		}
	}
	opens, err := s.verifyNodePassphrase(ctx, in.Conn, in.Platform, current)
	if err != nil {
		return result, rotationNodeError(err, current)
	}
	if !opens {
		return result, ErrCredentialStoreRotation{Code: RotationEscrowMismatch, Detail: "the escrowed passphrase does not open the node's passphrase wrap; nothing was changed"}
	}

	// 3. The new value is durable before any node change.
	next, err := generateStorePassphrase()
	if err != nil {
		return result, ErrCredentialStoreRotation{Code: RotationUnavailable, Detail: err.Error()}
	}
	defer zeroBytes(next)
	if err := pendingEscrow.SavePending(ctx, key, next); err != nil {
		return result, ErrCredentialStoreRotation{Code: RotationEscrowUnavailable, Detail: "the new passphrase could not be escrowed, so the node was not changed: " + redact(err.Error(), next)}
	}

	// 4. One atomic wrap replacement on the node.
	stdin := make([]byte, 0, len(current)+len(next)+2)
	stdin = append(append(append(append(stdin, current...), '\n'), next...), '\n')
	res, runErr := s.driver.RunNodeCLI(ctx, in.Conn, in.Platform, storeChangePassphraseArgs, stdin)
	zeroBytes(stdin)
	if runErr != nil || res.ExitCode != 0 {
		detail := redact(redact(commandFailure(res, runErr), current), next)
		return result, ErrCredentialStoreRotation{Code: RotationNodeCommandFailed, Detail: "changing the node's passphrase failed (" + detail + "); the escrowed passphrase still opens it and the unused new one is discarded by the next run"}
	}

	// 5. Prove the node now opens with the new value before trusting it.
	opens, err = s.verifyNodePassphrase(ctx, in.Conn, in.Platform, next)
	if err != nil || !opens {
		reason := "the new passphrase did not verify"
		if err != nil {
			reason = err.Error()
		}
		return result, ErrCredentialStoreRotation{Code: RotationNodeCommandFailed, Detail: "could not confirm the node's new passphrase (" + redact(reason, next) + "); both the old and new passphrases stay escrowed, and re-running the rotation keeps whichever opens the node"}
	}

	// 6. Promote. Until this succeeds the pending slot is what opens the node.
	if err := s.storeEscrow.Save(ctx, key, next); err != nil {
		return result, ErrCredentialStoreRotation{Code: RotationEscrowUnavailable, Detail: "the node now uses the new passphrase, which is held as the pending escrow; re-run the rotation to promote it (" + redact(err.Error(), next) + ")"}
	}
	clearNote := ""
	if err := pendingEscrow.ClearPending(ctx, key); err != nil {
		clearNote = "; the pending slot could not be cleared and will be reconciled by the next run"
	}

	// 7. The agent must serve the new value, or the node cannot reopen the
	// store after its next reboot.
	result.AgentUnlock, result.Detail = s.pushAndProveUnlock(ctx, in, key, next, status)
	result.Detail = "rotated the node's credential-store passphrase; the new one is escrowed on this control plane; " + result.Detail + clearNote
	return result, nil
}

// pushAndProveUnlock re-pushes the node's unlock grant and checks that a fresh
// node shell opens the store with what the agent serves. Its answer is only a
// proof when the passphrase wrap is the store's sole way in.
func (s *service) pushAndProveUnlock(ctx context.Context, in RotateCredentialStoreInput, key string, secret []byte, status nodeStoreStatus) (string, string) {
	grantNote := s.ensureStoreGrant(ctx, in.NodeID, key, secret)
	if status.Unattended.Enabled {
		return AgentUnlockNotRequired, "the store opens at boot through its " + nonEmpty(status.Unattended.Provider, "unattended") + " wrap; the agent's copy is recovery-only" + grantNote
	}
	for attempt := 0; attempt < agentUnlockPollAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return AgentUnlockUnverified, "the wait for the agent was cancelled" + grantNote
			case <-time.After(agentUnlockPollInterval):
			}
		}
		now, err := s.readNodeStoreStatus(ctx, in.Conn, in.Platform)
		if err == nil && now.Unlocked && !now.Unattended.Enabled {
			return AgentUnlockVerified, "a fresh node shell opened the store with the passphrase the agent now serves" + grantNote
		}
	}
	return AgentUnlockUnverified, "a fresh node shell could not open the store with the agent's passphrase yet; the agent receives it on its next reconnect" + grantNote
}

func rotationNodeError(err error, secret []byte) error {
	if errors.Is(err, errNodeCLIPredates) {
		return ErrCredentialStoreRotation{Code: RotationNodeOutdated, Detail: err.Error()}
	}
	return ErrCredentialStoreRotation{Code: RotationNodeCommandFailed, Detail: redact(err.Error(), secret)}
}
