package onboard_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/schedule"
	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboard/mocks"
)

// fakeEscrow records every save and the order of events against the node CLI,
// so a test can prove the passphrase is escrowed before it is ever used.
type fakeEscrow struct {
	mu      sync.Mutex
	values  map[string][]byte
	saved   []string
	journal *[]string
	loadErr error
	saveErr error
}

func (e *fakeEscrow) Load(_ context.Context, key string) ([]byte, bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.loadErr != nil {
		return nil, false, e.loadErr
	}
	value, ok := e.values[key]
	return append([]byte(nil), value...), ok, nil
}

func (e *fakeEscrow) Save(_ context.Context, key string, secret []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.saveErr != nil {
		return e.saveErr
	}
	if e.values == nil {
		e.values = map[string][]byte{}
	}
	e.values[key] = append([]byte(nil), secret...)
	e.saved = append(e.saved, key)
	if e.journal != nil {
		*e.journal = append(*e.journal, "escrow-save")
	}
	return nil
}

type nodeScript struct {
	mu          sync.Mutex
	journal     []string
	initialized bool
	wraps       string
	initExit    int
	initStderr  string
	rewrapOut   string
	rewrapExit  int
}

func (n *nodeScript) run(args []string, stdin []byte) (onboard.NodeCommandResult, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	switch strings.Join(args[:3], " ") {
	case "credentials store status":
		n.journal = append(n.journal, "status")
		wraps := n.wraps
		if wraps == "" {
			wraps = "[]"
		}
		return onboard.NodeCommandResult{Stdout: `{"initialized":` + boolText(n.initialized) + `,"wraps":` + wraps + `,"entries":0}`}, nil
	case "credentials store init":
		n.journal = append(n.journal, "init")
		if n.initExit != 0 {
			return onboard.NodeCommandResult{ExitCode: n.initExit, Stderr: n.initStderr}, nil
		}
		n.initialized = true
		return onboard.NodeCommandResult{Stdout: `{"initialized":true}`}, nil
	case "credentials store rewrap":
		n.journal = append(n.journal, "rewrap")
		return onboard.NodeCommandResult{ExitCode: n.rewrapExit, Stdout: n.rewrapOut}, nil
	}
	return onboard.NodeCommandResult{}, errors.New("unexpected node command " + strings.Join(args, " "))
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func runStoreOnboarding(t *testing.T, node *nodeScript, escrow *fakeEscrow) (*mocks.FakeSSHDriver, onboard.Op, []onboard.StepEvent) {
	t.Helper()
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID), NodeCLI: node.run}
	svc := onboard.NewService(repo, driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true}, schedule.System(),
		onboard.WithEnrollmentResolver(fixedEnrollmentResolver{nodeID: testNodeID, paired: true}),
		onboard.WithCredentialStoreEscrow(escrow),
	)
	dec, err := svc.Start(context.Background(), validInput())
	require.NoError(t, err)
	op := waitTerminal(t, svc, dec.OpID)
	_, events, err := svc.GetOp(context.Background(), dec.OpID)
	require.NoError(t, err)
	return driver, op, events
}

func storeEvents(events []onboard.StepEvent) []onboard.StepEvent {
	var out []onboard.StepEvent
	for _, event := range events {
		if event.StepID == onboard.StepCredentialStore {
			out = append(out, event)
		}
	}
	return out
}

// assertSecretNowhere proves the passphrase never reached argv, step events,
// or the durable op record.
func assertSecretNowhere(t *testing.T, secret string, driver *mocks.FakeSSHDriver, op onboard.Op, events []onboard.StepEvent) {
	t.Helper()
	require.NotEmpty(t, secret)
	for _, call := range driver.NodeCLIRecorded() {
		require.NotContains(t, strings.Join(call.Args, " "), secret, "the passphrase must never be an argument")
	}
	for _, event := range events {
		require.NotContains(t, event.Detail, secret)
	}
	record, err := json.Marshal(op)
	require.NoError(t, err)
	require.NotContains(t, string(record), secret)
}

// A node with no credential store gets one, with a passphrase this control
// plane generated and escrowed before the node ever saw it.
func TestCredentialStore_NewNodeGetsAStoreWhosePassphraseIsEscrowedFirst(t *testing.T) {
	node := &nodeScript{rewrapExit: 1, rewrapOut: `{"enabled":false,"blocked":"the login keychain is locked"}`}
	escrow := &fakeEscrow{}
	escrow.journal = &node.journal
	driver, op, events := runStoreOnboarding(t, node, escrow)

	require.Equal(t, onboard.StatePaired, op.State)
	key := "node-" + testNodeID
	require.Equal(t, []string{key}, escrow.saved)
	secret := string(escrow.values[key])
	require.GreaterOrEqual(t, len(secret), 43, "a 32-byte base64url passphrase")
	require.Equal(t, []string{"status", "escrow-save", "init", "rewrap"}, node.journal, "escrow must be durable before the store is created")

	var initStdin, rewrapStdin string
	for _, call := range driver.NodeCLIRecorded() {
		switch call.Args[2] {
		case "init":
			initStdin = string(call.Stdin)
		case "rewrap":
			rewrapStdin = string(call.Stdin)
		}
	}
	require.Equal(t, secret+"\n", initStdin, "the passphrase rides stdin")
	require.Equal(t, secret+"\n", rewrapStdin)

	final := storeEvents(events)
	require.NotEmpty(t, final)
	last := final[len(final)-1]
	require.Equal(t, onboard.StepStatusOK, last.Status)
	require.Contains(t, last.Detail, "escrowed on this control plane")
	require.Contains(t, last.Detail, "login keychain is locked", "a blocked unattended wrap is reported, not fatal")
	assertSecretNowhere(t, secret, driver, op, events)
}

// A store someone else created is never re-initialized or overwritten.
func TestCredentialStore_ExistingStoreWithoutEscrowIsLeftUnchanged(t *testing.T) {
	node := &nodeScript{initialized: true, wraps: `[{"provider":"host-bound"}]`}
	escrow := &fakeEscrow{}
	_, op, events := runStoreOnboarding(t, node, escrow)

	require.Equal(t, onboard.StatePaired, op.State)
	require.Equal(t, []string{"status"}, node.journal, "no init or rewrap against a store Bridge cannot open")
	require.Empty(t, escrow.saved, "no passphrase is invented for a store that already has one")
	final := storeEvents(events)
	last := final[len(final)-1]
	require.Equal(t, onboard.StepStatusSkipped, last.Status)
	require.Contains(t, last.Detail, "does not hold")
	require.Contains(t, last.Detail, "host-bound")
}

// A store whose passphrase is escrowed is verified and converged, not recreated.
func TestCredentialStore_EscrowedStoreIsConvergedNotRecreated(t *testing.T) {
	node := &nodeScript{initialized: true, wraps: `[{"provider":"passphrase"}]`, rewrapOut: `{"enabled":true,"provider":"native-wrap"}`}
	escrow := &fakeEscrow{values: map[string][]byte{"node-" + testNodeID: []byte("escrowed-passphrase-value")}}
	driver, op, events := runStoreOnboarding(t, node, escrow)

	require.Equal(t, []string{"status", "rewrap"}, node.journal)
	final := storeEvents(events)
	last := final[len(final)-1]
	require.Equal(t, onboard.StepStatusOK, last.Status)
	require.Contains(t, last.Detail, "opens unattended at boot via native-wrap")
	assertSecretNowhere(t, "escrowed-passphrase-value", driver, op, events)
}

// A failed store creation is degraded: the node is paired and online, and the
// next connect converges it.
func TestCredentialStore_InitFailureIsDegradedNotFatal(t *testing.T) {
	node := &nodeScript{initExit: 1, initStderr: "no key wrap provider is available"}
	escrow := &fakeEscrow{}
	driver, op, events := runStoreOnboarding(t, node, escrow)

	require.Equal(t, onboard.StatePaired, op.State)
	final := storeEvents(events)
	last := final[len(final)-1]
	require.Equal(t, onboard.StepStatusSkipped, last.Status)
	require.Contains(t, last.Detail, "degraded")
	require.Contains(t, last.Detail, "no key wrap provider is available")
	assertSecretNowhere(t, string(escrow.values["node-"+testNodeID]), driver, op, events)
}

// An escrow failure leaves the node's store uninitialized rather than create a
// store whose passphrase nobody holds.
func TestCredentialStore_EscrowFailureNeverCreatesAnUnheldStore(t *testing.T) {
	node := &nodeScript{}
	escrow := &fakeEscrow{saveErr: errors.New("control-plane store locked")}
	_, op, events := runStoreOnboarding(t, node, escrow)

	require.Equal(t, onboard.StatePaired, op.State)
	require.Equal(t, []string{"status"}, node.journal, "init must not run when escrow failed")
	final := storeEvents(events)
	require.Contains(t, final[len(final)-1].Detail, "left uninitialized")
}

type recordingStoreGrant struct {
	mu    sync.Mutex
	calls []string
}

func (g *recordingStoreGrant) EnsureNodeStoreGrant(_ context.Context, nodeID, logicalID, field string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls = append(g.calls, nodeID+"|"+logicalID+"|"+field)
	return nil
}

func runStoreOnboardingWithGrant(t *testing.T, node *nodeScript, escrow *fakeEscrow, grant *recordingStoreGrant) []onboard.StepEvent {
	t.Helper()
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID), NodeCLI: node.run}
	svc := onboard.NewService(repo, driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true}, schedule.System(),
		onboard.WithEnrollmentResolver(fixedEnrollmentResolver{nodeID: testNodeID, paired: true}),
		onboard.WithCredentialStoreEscrow(escrow),
		onboard.WithNodeStoreGrant(grant),
	)
	dec, err := svc.Start(context.Background(), validInput())
	require.NoError(t, err)
	waitTerminal(t, svc, dec.OpID)
	_, events, err := svc.GetOp(context.Background(), dec.OpID)
	require.NoError(t, err)
	return events
}

// Once the passphrase is escrowed the node is granted it, at the escrow
// address, so Bridge re-pushes it after a reboot; a store Bridge does not hold
// the passphrase for is never granted.
func TestCredentialStore_EscrowedStoreGrantsTheNodeItsUnlock(t *testing.T) {
	grant := &recordingStoreGrant{}
	events := runStoreOnboardingWithGrant(t, &nodeScript{rewrapOut: `{"enabled":true,"provider":"native-wrap"}`}, &fakeEscrow{}, grant)
	want := testNodeID + "|vrooli-bridge/node-credential-store/node-" + testNodeID + "|passphrase"
	require.Equal(t, []string{want}, grant.calls)
	final := storeEvents(events)
	require.Contains(t, final[len(final)-1].Detail, "unlock grant")

	unheld := &recordingStoreGrant{}
	runStoreOnboardingWithGrant(t, &nodeScript{initialized: true, wraps: `[{"provider":"host-bound"}]`}, &fakeEscrow{}, unheld)
	require.Empty(t, unheld.calls)
}

// The step runs after the node is confirmed online.
func TestCredentialStore_RunsAfterOnlineConfirmation(t *testing.T) {
	node := &nodeScript{initialized: true}
	_, _, events := runStoreOnboarding(t, node, &fakeEscrow{})
	var onlineSeq, storeSeq uint64
	for _, event := range events {
		if event.StepID == onboard.StepVerifyOnline && event.Status == onboard.StepStatusOK {
			onlineSeq = event.Sequence
		}
		if event.StepID == onboard.StepCredentialStore && storeSeq == 0 {
			storeSeq = event.Sequence
		}
	}
	require.NotZero(t, onlineSeq)
	require.Greater(t, storeSeq, onlineSeq)
}
