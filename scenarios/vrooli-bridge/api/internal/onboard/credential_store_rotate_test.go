package onboard_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/schedule"
	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboard/mocks"
)

const rotateMachineID = "machine-rotate"

// agentGrant plays Bridge's grant delivery: the agent ends up serving
// whatever the escrow holds.
type agentGrant struct {
	node   *nodeScript
	escrow *fakeEscrow
	calls  int
}

func (g *agentGrant) EnsureNodeStoreGrant(_ context.Context, _, logicalID, field string) error {
	g.calls++
	key := strings.TrimPrefix(logicalID, onboard.CredentialStoreEscrowNamespace)
	g.escrow.mu.Lock()
	value := string(g.escrow.values[key])
	g.escrow.mu.Unlock()
	g.node.mu.Lock()
	g.node.agentServes = value
	g.node.mu.Unlock()
	if field != onboard.CredentialStoreEscrowField {
		return errors.New("the unlock grant must name the escrow field")
	}
	return nil
}

func newRotator(t *testing.T, node *nodeScript, escrow *fakeEscrow) (onboard.CredentialStoreRotator, *mocks.FakeSSHDriver, *agentGrant) {
	t.Helper()
	t.Cleanup(onboard.SetAgentUnlockPollForTest(time.Millisecond, 3))
	driver := &mocks.FakeSSHDriver{NodeCLI: node.run}
	grant := &agentGrant{node: node, escrow: escrow}
	svc := onboard.NewService(mocks.NewFakeRepository(), driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true}, schedule.System(),
		onboard.WithCredentialStoreEscrow(escrow), onboard.WithNodeStoreGrant(grant))
	rotator, ok := svc.(onboard.CredentialStoreRotator)
	require.True(t, ok, "the onboarding service owns rotation")
	return rotator, driver, grant
}

func rotateInput() onboard.RotateCredentialStoreInput {
	return onboard.RotateCredentialStoreInput{
		MachineID: rotateMachineID, NodeID: testNodeID,
		Conn:     onboard.Conn{Host: "mini.local", Port: 22, User: "owner", KeyPath: "/keys/machine"},
		Platform: onboard.NodePlatform{OS: "darwin", Arch: "amd64"},
	}
}

func passphraseNode(passphrase string) *nodeScript {
	return &nodeScript{initialized: true, wraps: `[{"provider":"passphrase"}]`, passphrase: passphrase, agentServes: passphrase}
}

func escrowHolding(value string) *fakeEscrow {
	return &fakeEscrow{values: map[string][]byte{rotateMachineID: []byte(value)}}
}

// someEscrowOpensTheNode is the rotation's crash-safety invariant: at every
// point, the escrow or the pending slot holds the passphrase the node's wrap
// opens with.
func someEscrowOpensTheNode(t *testing.T, node *nodeScript, escrow *fakeEscrow) {
	t.Helper()
	escrow.mu.Lock()
	defer escrow.mu.Unlock()
	opens := string(escrow.values[rotateMachineID]) == node.passphrase || string(escrow.pending[rotateMachineID]) == node.passphrase
	require.True(t, opens, "no escrowed value opens the node: escrow=%q pending=%q node=%q",
		escrow.values[rotateMachineID], escrow.pending[rotateMachineID], node.passphrase)
}

func TestRotation_SwapsNodeAndEscrowAndTheAgentServesTheNewValue(t *testing.T) {
	node := passphraseNode("the old passphrase")
	escrow := escrowHolding("the old passphrase")
	escrow.journal = &node.journal
	rotator, driver, grant := newRotator(t, node, escrow)

	result, err := rotator.RotateCredentialStore(context.Background(), rotateInput())
	require.NoError(t, err)

	next := string(escrow.values[rotateMachineID])
	require.NotEqual(t, "the old passphrase", next)
	require.GreaterOrEqual(t, len(next), 43)
	require.Equal(t, next, node.passphrase)
	require.Empty(t, escrow.pending, "the pending slot is cleared once promoted")
	require.Equal(t, []string{"status", "verify", "pending-save", "change", "verify", "escrow-save", "pending-clear", "status"}, node.journal,
		"the new value is escrowed before the node changes and promoted only after it verifies")
	require.Equal(t, 1, grant.calls)
	require.Equal(t, onboard.AgentUnlockVerified, result.AgentUnlock)
	require.False(t, result.Resumed)
	for _, call := range driver.NodeCLIRecorded() {
		args := strings.Join(call.Args, " ")
		require.NotContains(t, args, next)
		require.NotContains(t, args, "the old passphrase")
	}
	require.NotContains(t, result.Detail, next)
}

// Stop the rotation at each step, check the invariant, then re-run: the re-run
// always converges on one escrowed value that opens the node.
func TestRotation_AnInterruptionAnywhereLeavesAnEscrowThatOpensTheNode(t *testing.T) {
	cases := map[string]struct {
		interrupt  func(node *nodeScript, escrow *fakeEscrow)
		wantCode   string
		wantResume bool
	}{
		"escrowing the new passphrase fails": {
			interrupt: func(_ *nodeScript, escrow *fakeEscrow) { escrow.failSavePending = errors.New("authority locked") },
			wantCode:  onboard.RotationEscrowUnavailable,
		},
		"the node change fails": {
			interrupt: func(node *nodeScript, _ *fakeEscrow) { node.failChange = true },
			wantCode:  onboard.RotationNodeCommandFailed,
		},
		"promoting the new passphrase fails": {
			interrupt:  func(_ *nodeScript, escrow *fakeEscrow) { escrow.failSave = errors.New("authority went away") },
			wantCode:   onboard.RotationEscrowUnavailable,
			wantResume: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			node := passphraseNode("original")
			escrow := escrowHolding("original")
			rotator, _, _ := newRotator(t, node, escrow)
			tc.interrupt(node, escrow)

			_, err := rotator.RotateCredentialStore(context.Background(), rotateInput())
			var rotationErr onboard.ErrCredentialStoreRotation
			require.ErrorAs(t, err, &rotationErr)
			require.Equal(t, tc.wantCode, rotationErr.Code)
			require.True(t, rotationErr.Retryable())
			someEscrowOpensTheNode(t, node, escrow)

			result, err := rotator.RotateCredentialStore(context.Background(), rotateInput())
			require.NoError(t, err)
			require.Equal(t, tc.wantResume, result.Resumed)
			require.Equal(t, node.passphrase, string(escrow.values[rotateMachineID]))
			require.NotEqual(t, "original", node.passphrase)
			require.Empty(t, escrow.pending)
			require.Equal(t, onboard.AgentUnlockVerified, result.AgentUnlock)
		})
	}
}

// Losing the connection after the node changed but before it verified leaves
// both values escrowed; the re-run keeps the one the node opens with.
func TestRotation_AnUnverifiedNodeChangeIsResolvedByTheNextRun(t *testing.T) {
	node := passphraseNode("original")
	escrow := escrowHolding("original")
	realRun := node.run
	changed := false
	driverScript := func(args []string, stdin []byte) (onboard.NodeCommandResult, error) {
		if args[2] == "change-passphrase" {
			changed = true
		} else if args[2] == "verify-passphrase" && changed {
			changed = false
			return onboard.NodeCommandResult{}, errors.New("ssh: connection reset")
		}
		return realRun(args, stdin)
	}
	rotator, _, _ := newRotatorWithScript(t, node, escrow, driverScript)

	_, err := rotator.RotateCredentialStore(context.Background(), rotateInput())
	var rotationErr onboard.ErrCredentialStoreRotation
	require.ErrorAs(t, err, &rotationErr)
	require.Equal(t, onboard.RotationNodeCommandFailed, rotationErr.Code)
	require.Equal(t, "original", string(escrow.values[rotateMachineID]), "the escrow is not promoted on an unverified change")
	someEscrowOpensTheNode(t, node, escrow)

	result, err := rotator.RotateCredentialStore(context.Background(), rotateInput())
	require.NoError(t, err)
	require.True(t, result.Resumed)
	require.Equal(t, node.passphrase, string(escrow.values[rotateMachineID]))
}

func newRotatorWithScript(t *testing.T, node *nodeScript, escrow *fakeEscrow, script func([]string, []byte) (onboard.NodeCommandResult, error)) (onboard.CredentialStoreRotator, *mocks.FakeSSHDriver, *agentGrant) {
	t.Helper()
	rotator, driver, grant := newRotator(t, node, escrow)
	driver.NodeCLI = script
	return rotator, driver, grant
}

func TestRotation_RefusesWhenTheEscrowDoesNotOpenTheNode(t *testing.T) {
	node := passphraseNode("what the node really uses")
	escrow := escrowHolding("a stale escrow")
	rotator, _, grant := newRotator(t, node, escrow)

	_, err := rotator.RotateCredentialStore(context.Background(), rotateInput())
	var rotationErr onboard.ErrCredentialStoreRotation
	require.ErrorAs(t, err, &rotationErr)
	require.Equal(t, onboard.RotationEscrowMismatch, rotationErr.Code)
	require.Equal(t, "what the node really uses", node.passphrase)
	require.Empty(t, escrow.pending)
	require.Zero(t, grant.calls)
}

func TestRotation_AnOlderNodeCLIIsRefusedWithoutChanges(t *testing.T) {
	node := passphraseNode("original")
	node.oldCLI = true
	escrow := escrowHolding("original")
	rotator, _, _ := newRotator(t, node, escrow)

	_, err := rotator.RotateCredentialStore(context.Background(), rotateInput())
	var rotationErr onboard.ErrCredentialStoreRotation
	require.ErrorAs(t, err, &rotationErr)
	require.Equal(t, onboard.RotationNodeOutdated, rotationErr.Code)
	require.Equal(t, "original", node.passphrase)
	require.Empty(t, escrow.pending)
}

func TestRotation_RefusesAStoreThisControlPlaneDoesNotHold(t *testing.T) {
	node := passphraseNode("the operator's own")
	rotator, _, _ := newRotator(t, node, &fakeEscrow{})
	_, err := rotator.RotateCredentialStore(context.Background(), rotateInput())
	var rotationErr onboard.ErrCredentialStoreRotation
	require.ErrorAs(t, err, &rotationErr)
	require.Equal(t, onboard.RotationNotEscrowed, rotationErr.Code)
	require.False(t, rotationErr.Retryable())
}

func TestRotation_WindowsNodesAreRefusedWithTheDocumentedReason(t *testing.T) {
	node := passphraseNode("original")
	rotator, driver, _ := newRotator(t, node, escrowHolding("original"))
	input := rotateInput()
	input.Platform.OS = "windows"
	_, err := rotator.RotateCredentialStore(context.Background(), input)
	var rotationErr onboard.ErrCredentialStoreRotation
	require.ErrorAs(t, err, &rotationErr)
	require.Equal(t, onboard.RotationUnsupportedPlatform, rotationErr.Code)
	require.Contains(t, rotationErr.Detail, "DPAPI")
	require.Empty(t, driver.NodeCLIRecorded())
}

// On a store that opens unattended at boot, the agent's copy is recovery-only.
func TestRotation_AnUnattendedStoreReportsTheAgentCopyAsNotRequired(t *testing.T) {
	node := passphraseNode("original")
	node.wraps = `[{"provider":"host-bound"},{"provider":"passphrase"}]`
	node.unattended = true
	escrow := escrowHolding("original")
	rotator, _, _ := newRotator(t, node, escrow)
	result, err := rotator.RotateCredentialStore(context.Background(), rotateInput())
	require.NoError(t, err)
	require.Equal(t, onboard.AgentUnlockNotRequired, result.AgentUnlock)
	require.Equal(t, node.passphrase, string(escrow.values[rotateMachineID]))
}
