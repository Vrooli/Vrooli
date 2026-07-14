package onboard

import (
	"context"
	"time"
)

// This file declares the onboard domain's outside-world seams over proto-free
// DTOs. The domain imports no proto and no sibling handler; main.go / the
// handler module bind these to the concrete SSH capability, the pairing service,
// and the presence read. Service unit tests bind fakes.

// Conn is a resolved, key-authenticated SSH target the orchestrator drives after
// first touch (host/port/user + the bridge onboarding key that now authorizes).
type Conn struct {
	Host    string
	Port    int
	User    string
	KeyPath string
}

// FirstTouchParams is the owner-supplied SSH target for the first-touch phase.
// Password is the transient owner credential: the SSHDriver uses it once to
// install the bridge key and zeroes it, and it is NEVER persisted or logged.
type FirstTouchParams struct {
	Host     string
	Port     int
	User     string
	Password []byte
}

// RunParams drives the remote bootstrap. PairingCode is the single-use,
// server-issued code injected into the remote script over stdin (never argv,
// never logged); the orchestrator zeroes it after RunBootstrap returns. Args are
// the bootstrap flags (control-plane URL, node name, revision, …) — the pairing
// code is deliberately NOT among them.
type RunParams struct {
	Conn        Conn
	RemotePath  string
	Args        []string
	PairingCode []byte
}

// BootstrapResult is the terminal outcome of a remote bootstrap run.
type BootstrapResult struct {
	// ExitCode is the bootstrap script's process exit code (0 success; 2 usage, 3
	// unsupported platform, 4 pairing, 1 other — bootstrap/README.md).
	ExitCode int
}

// SSHDriver is the SSH-capability seam: establish passwordless SSH, copy the
// bootstrap script to the node, and run it while streaming its VBOOTSTRAP
// markers back. Production wraps internal/onboard/ssh; unit tests fake it.
type SSHDriver interface {
	// FirstTouch establishes working passwordless SSH to the host (generate key,
	// copy it with the password, retest) and returns the key-authenticated Conn.
	// The password is zeroed by the driver.
	FirstTouch(ctx context.Context, p FirstTouchParams) (Conn, error)

	// PushScript copies the bootstrap script to the node and returns the remote
	// path it landed at.
	PushScript(ctx context.Context, conn Conn) (remotePath string, err error)

	// RunBootstrap runs the bootstrap script remotely, injecting the pairing code
	// over stdin, and invokes onMarker for every parsed VBOOTSTRAP stdout marker
	// as it streams. It returns once the script exits.
	RunBootstrap(ctx context.Context, p RunParams, onMarker func(Marker)) (BootstrapResult, error)
}

// RevisionResolver defaults, validates, and preflights the onboarding target
// revision before an op is dispatched. An empty or "@cp" request resolves to the
// control plane's current commit; any other value passes through after a
// metacharacter validation and a remote push-check (so a commit the node could
// never fetch is refused loudly here). Production wires api/internal/cprev; unit
// tests fake it or leave it unset (legacy: default via WithDefaultRevision and
// require a non-empty revision).
type RevisionResolver interface {
	Resolve(ctx context.Context, requested string) (string, error)
}

// IssueParams is the server-side pairing-code request.
type IssueParams struct {
	NodeName string
	Scopes   []string
}

// CodeIssuer is the pairing seam: mint a single-use pairing code server-side so
// the operator never handles one. Production wraps internal/pairing.Service.
// The returned code is an owned []byte the orchestrator zeroes after injecting
// it; it is never persisted or logged.
type CodeIssuer interface {
	Issue(ctx context.Context, p IssueParams) (code []byte, err error)
}

// OnlineConfirmer is the presence read seam: confirm a freshly-onboarded node is
// ONLINE in the fleet (holds a live dial-out channel) with its control-plane key
// pinned. Production wraps the presence hub + the credential store. It blocks up
// to timeout, returning online=false (no error) if the node never appears.
type OnlineConfirmer interface {
	ConfirmOnline(ctx context.Context, nodeID string, timeout time.Duration) (online bool, err error)
}
