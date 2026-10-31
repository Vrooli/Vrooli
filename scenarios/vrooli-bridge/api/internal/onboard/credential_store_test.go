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
	pending map[string][]byte
	saved   []string
	journal *[]string
	loadErr error
	saveErr error
	// failSave, failSavePending, and failClear inject one interruption each,
	// then clear themselves, so a rotation can be stopped at a chosen step.
	failSave        error
	failSavePending error
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
	if e.failSave != nil {
		err := e.failSave
		e.failSave = nil
		return err
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

// nodeScript is a node's store as its vrooli CLI reports it. passphrase is
// the value the store's passphrase wrap opens with; agentServes is what the
// node agent would hand a fresh shell through the unlock socket.
type nodeScript struct {
	mu          sync.Mutex
	journal     []string
	initialized bool
	wraps       string
	providers   []string
	parsed      bool
	passphrase  string
	agentServes string
	unattended  bool
	initExit    int
	initStderr  string
	rewrapOut   string
	rewrapExit  int
	// oldCLI answers the 2026-09-15 verbs the way an older node CLI does.
	oldCLI bool
	// failChange and failVerifyTransport inject one interruption each.
	failChange          bool
	failVerifyTransport bool
}

func (n *nodeScript) providerList() []string {
	if !n.parsed {
		n.parsed = true
		var wraps []struct {
			Provider string `json:"provider"`
		}
		if n.wraps != "" {
			_ = json.Unmarshal([]byte(n.wraps), &wraps)
		}
		for _, wrap := range wraps {
			n.providers = append(n.providers, wrap.Provider)
		}
	}
	return n.providers
}

func (n *nodeScript) hasPassphraseWrap() bool {
	for _, provider := range n.providerList() {
		if provider == "passphrase" {
			return true
		}
	}
	return false
}

var unknownSubcommand = onboard.NodeCommandResult{ExitCode: 1, Stderr: "Runtime error: Unknown subcommand: store\nRun 'vrooli credentials store help' for available subcommands"}

func (n *nodeScript) run(args []string, stdin []byte) (onboard.NodeCommandResult, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	line := strings.TrimSpace(string(stdin))
	switch strings.Join(args[:3], " ") {
	case "credentials store status":
		n.journal = append(n.journal, "status")
		wraps := make([]string, 0, len(n.providerList()))
		for _, provider := range n.providerList() {
			wraps = append(wraps, `{"provider":"`+provider+`"}`)
		}
		unlocked := n.unattended || (n.hasPassphraseWrap() && n.agentServes != "" && n.agentServes == n.passphrase)
		return onboard.NodeCommandResult{Stdout: `{"initialized":` + boolText(n.initialized) + `,"unlocked":` + boolText(unlocked) +
			`,"wraps":[` + strings.Join(wraps, ",") + `],"entries":0,"unattended":{"enabled":` + boolText(n.unattended) + `}}`}, nil
	case "credentials store init":
		n.journal = append(n.journal, "init")
		if n.initExit != 0 {
			return onboard.NodeCommandResult{ExitCode: n.initExit, Stderr: n.initStderr}, nil
		}
		n.initialized = true
		n.providerList()
		n.providers = append(n.providers, "passphrase")
		n.passphrase = line
		return onboard.NodeCommandResult{Stdout: `{"initialized":true}`}, nil
	case "credentials store rewrap":
		n.journal = append(n.journal, "rewrap")
		return onboard.NodeCommandResult{ExitCode: n.rewrapExit, Stdout: n.rewrapOut}, nil
	case "credentials store add-passphrase":
		n.journal = append(n.journal, "add")
		if n.oldCLI {
			return unknownSubcommand, nil
		}
		if n.hasPassphraseWrap() {
			return onboard.NodeCommandResult{ExitCode: 1, Stderr: "credential store already has a passphrase wrap"}, nil
		}
		n.providers = append(n.providers, "passphrase")
		n.passphrase = line
		return onboard.NodeCommandResult{Stdout: `{"added":true,"generation":1}`}, nil
	case "credentials store verify-passphrase":
		n.journal = append(n.journal, "verify")
		if n.oldCLI {
			return unknownSubcommand, nil
		}
		if n.failVerifyTransport {
			n.failVerifyTransport = false
			return onboard.NodeCommandResult{}, errors.New("ssh: connection reset")
		}
		if !n.hasPassphraseWrap() {
			return onboard.NodeCommandResult{ExitCode: 1, Stderr: "credential store has no passphrase wrap"}, nil
		}
		if line == n.passphrase {
			return onboard.NodeCommandResult{Stdout: `{"valid":true,"generation":1}`}, nil
		}
		return onboard.NodeCommandResult{ExitCode: 1, Stdout: `{"valid":false,"generation":1}`}, nil
	case "credentials store change-passphrase":
		n.journal = append(n.journal, "change")
		if n.failChange {
			n.failChange = false
			return onboard.NodeCommandResult{ExitCode: 1, Stderr: "interrupted"}, nil
		}
		current, next, _ := strings.Cut(string(stdin), "\n")
		if strings.TrimSpace(current) != n.passphrase {
			return onboard.NodeCommandResult{ExitCode: 1, Stderr: "credential store passphrase did not open the data key"}, nil
		}
		n.passphrase = strings.TrimSpace(next)
		return onboard.NodeCommandResult{Stdout: "Credential store passphrase changed."}, nil
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

// A store with a passphrase someone else chose is never re-initialized,
// re-wrapped, or given a second passphrase.
func TestCredentialStore_ExistingPassphraseStoreWithoutEscrowIsLeftUnchanged(t *testing.T) {
	node := &nodeScript{initialized: true, wraps: `[{"provider":"host-bound"},{"provider":"passphrase"}]`, passphrase: "the operator's own"}
	escrow := &fakeEscrow{}
	_, op, events := runStoreOnboarding(t, node, escrow)

	require.Equal(t, onboard.StatePaired, op.State)
	require.Equal(t, []string{"status"}, node.journal, "no init, add, or rewrap against a store Bridge cannot open")
	require.Empty(t, escrow.saved, "no passphrase is invented for a store that already has one")
	final := storeEvents(events)
	last := final[len(final)-1]
	require.Equal(t, onboard.StepStatusSkipped, last.Status)
	require.Contains(t, last.Detail, "does not hold")
	require.Contains(t, last.Detail, "host-bound, passphrase")
}

// A TPM-only store `vrooli setup` created gains an escrowed recovery
// passphrase wrap beside its TPM wrap, escrowed before the node sees it, and
// a second connect changes nothing.
func TestCredentialStore_UnattendedOnlyStoreGainsAnEscrowedRecoveryWrapOnce(t *testing.T) {
	node := &nodeScript{initialized: true, wraps: `[{"provider":"host-bound"}]`, unattended: true, rewrapOut: `{"enabled":true,"provider":"host-bound"}`}
	escrow := &fakeEscrow{}
	escrow.journal = &node.journal
	driver, op, events := runStoreOnboarding(t, node, escrow)

	key := "node-" + testNodeID
	secret := string(escrow.values[key])
	require.NotEmpty(t, secret)
	require.Equal(t, []string{"status", "escrow-save", "add", "rewrap"}, node.journal, "escrow must be durable before the node gains the wrap")
	require.Equal(t, secret, node.passphrase, "the node's new passphrase wrap opens with the escrowed value")
	require.Equal(t, []string{"host-bound", "passphrase"}, node.providers, "the TPM wrap is kept")
	last := storeEvents(events)[len(storeEvents(events))-1]
	require.Equal(t, onboard.StepStatusOK, last.Status)
	require.Contains(t, last.Detail, "recovery passphrase wrap")
	require.Contains(t, last.Detail, "host-bound")
	assertSecretNowhere(t, secret, driver, op, events)

	node.journal = nil
	runStoreOnboarding(t, node, escrow)
	require.Equal(t, []string{"status", "verify", "rewrap"}, node.journal, "a converged store is only verified")
	require.Equal(t, []string{key}, escrow.saved, "no second passphrase is escrowed")
	require.Equal(t, secret, node.passphrase)
}

// A crash between escrow and the node change leaves an escrowed passphrase the
// node does not have yet; the next connect gives it to the node.
func TestCredentialStore_EscrowedPassphraseMissingOnTheNodeIsAdded(t *testing.T) {
	node := &nodeScript{initialized: true, wraps: `[{"provider":"host-bound"}]`, unattended: true, rewrapOut: `{"enabled":true,"provider":"host-bound"}`}
	escrow := &fakeEscrow{values: map[string][]byte{"node-" + testNodeID: []byte("escrowed-before-a-crash")}}
	runStoreOnboarding(t, node, escrow)
	require.Equal(t, []string{"status", "add", "rewrap"}, node.journal)
	require.Equal(t, "escrowed-before-a-crash", node.passphrase)
	require.Empty(t, escrow.saved)
}

// An older node CLI cannot add the wrap; the step says to update the node
// instead of claiming a recovery path exists.
func TestCredentialStore_OldNodeCLIIsReportedNotTrusted(t *testing.T) {
	node := &nodeScript{initialized: true, wraps: `[{"provider":"host-bound"}]`, unattended: true, oldCLI: true}
	_, _, events := runStoreOnboarding(t, node, &fakeEscrow{})
	last := storeEvents(events)[len(storeEvents(events))-1]
	require.Equal(t, onboard.StepStatusSkipped, last.Status)
	require.Contains(t, last.Detail, "update the node")
	require.Empty(t, node.passphrase)
}

// A pending value left by an interrupted rotation that the node already uses
// is promoted on the next connect, and the node is granted the value that
// opens it.
func TestCredentialStore_ConnectFinishesAnInterruptedRotation(t *testing.T) {
	key := "node-" + testNodeID
	node := &nodeScript{initialized: true, wraps: `[{"provider":"passphrase"}]`, passphrase: "rotated-on-node", rewrapOut: `{"enabled":false,"blocked":"no tpm"}`}
	escrow := &fakeEscrow{values: map[string][]byte{key: []byte("before-rotation")}, pending: map[string][]byte{key: []byte("rotated-on-node")}}
	_, _, events := runStoreOnboarding(t, node, escrow)
	require.Equal(t, "rotated-on-node", string(escrow.values[key]))
	require.Empty(t, escrow.pending)
	last := storeEvents(events)[len(storeEvents(events))-1]
	require.Equal(t, onboard.StepStatusOK, last.Status)
	require.Contains(t, last.Detail, "finished an interrupted passphrase rotation")
}

// A store whose passphrase is escrowed is verified and converged, not recreated.
func TestCredentialStore_EscrowedStoreIsConvergedNotRecreated(t *testing.T) {
	node := &nodeScript{initialized: true, wraps: `[{"provider":"passphrase"}]`, passphrase: "escrowed-passphrase-value", rewrapOut: `{"enabled":true,"provider":"native-wrap"}`}
	escrow := &fakeEscrow{values: map[string][]byte{"node-" + testNodeID: []byte("escrowed-passphrase-value")}}
	driver, op, events := runStoreOnboarding(t, node, escrow)

	require.Equal(t, []string{"status", "verify", "rewrap"}, node.journal)
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
	runStoreOnboardingWithGrant(t, &nodeScript{initialized: true, wraps: `[{"provider":"passphrase"}]`, passphrase: "not ours"}, &fakeEscrow{}, unheld)
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

func (e *fakeEscrow) LoadPending(_ context.Context, key string) ([]byte, bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	value, ok := e.pending[key]
	return append([]byte(nil), value...), ok, nil
}

func (e *fakeEscrow) SavePending(_ context.Context, key string, secret []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.failSavePending != nil {
		err := e.failSavePending
		e.failSavePending = nil
		return err
	}
	if e.pending == nil {
		e.pending = map[string][]byte{}
	}
	e.pending[key] = append([]byte(nil), secret...)
	if e.journal != nil {
		*e.journal = append(*e.journal, "pending-save")
	}
	return nil
}

func (e *fakeEscrow) ClearPending(_ context.Context, key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.pending, key)
	if e.journal != nil {
		*e.journal = append(*e.journal, "pending-clear")
	}
	return nil
}
