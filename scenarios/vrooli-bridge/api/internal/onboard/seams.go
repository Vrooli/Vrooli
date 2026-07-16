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

	// SudoState is the passwordless-sudo provisioning outcome from first touch
	// (empty when not requested). Informational: the orchestrator surfaces it in
	// the ssh-setup step detail. It carries no credential material.
	SudoState string
}

// FirstTouchParams is the owner-supplied SSH target for the first-touch phase.
// Password is the transient owner credential: the SSHDriver uses it once to
// install the bridge key and zeroes it, and it is NEVER persisted or logged.
type FirstTouchParams struct {
	Host     string
	Port     int
	User     string
	Password []byte

	// ProvisionSudo asks the driver to install a scoped passwordless-sudo drop-in
	// for User while the password is still held, so later privileged steps work
	// over non-interactive SSH. Declining never fails the first touch.
	ProvisionSudo bool
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

	// Diagnostics is a bounded tail of the node-side diagnostic stream (the
	// bootstrap script routes ALL human/build/setup output to stderr, markers to
	// stdout) — on a failed run its tail is exactly the "output above" the script's
	// failure message points at (e.g. the `make setup` error). Empty on success or
	// when the node produced no output. It never carries secret material: the
	// pairing code rides stdin and the SSH password never reaches the node.
	Diagnostics string
}

// SyncParams drives the working-tree ship: tar the RepoDir-relative Files and
// unpack them into DestDir on the node over the established SSH channel. Files is
// the exact list the WorkingTreeSource enumerated (tracked + modified +
// untracked-non-ignored); the transport must preserve names with spaces/newlines
// (it uses the tar archive format, not shell word-splitting).
type SyncParams struct {
	Conn    Conn
	RepoDir string
	Files   []string
	DestDir string
}

// SyncResult is the outcome of a working-tree ship.
type SyncResult struct {
	// BytesTransferred is the size of the tar stream sent to the node, for the
	// step detail (so the operator sees how much was shipped).
	BytesTransferred int64

	// ResolvedDestDir is the concrete node-side directory the tree landed in — the
	// operator's DestDir when explicit, or the node-resolved default ($HOME/vrooli)
	// otherwise. The orchestrator threads it into the bootstrap `--source-dir` and
	// `--checkout-dir` so the script verifies exactly where the tree was unpacked.
	ResolvedDestDir string
}

// NodePlatform is the cross-compile target reported by the node over the
// established SSH connection. OS and Arch use Go's canonical GOOS/GOARCH names.
type NodePlatform struct {
	OS   string
	Arch string
}

// ArtifactBuildParams identifies the exact local tree and one node target the
// control plane must build. RepoDir is the same snapshotted tree SyncTree ships.
type ArtifactBuildParams struct {
	RepoDir string
	Target  NodePlatform
}

// PrebuiltArtifacts are the three control-plane-built executables and their
// freshness sidecars. Directory is temporary and owned by the caller.
type PrebuiltArtifacts struct {
	Directory     string
	Vrooli        string
	VrooliSidecar string
	BridgeCLI     string
	BridgeSidecar string
	Agent         string
	AgentSidecar  string
	Fingerprint   string
	Target        NodePlatform
}

// ArtifactBuilder cross-compiles the exact live tree for one detected node
// platform. Production invokes the project-level vrooli-dist command for the
// Vrooli executable, keeping the release and bridge consumers on one primitive.
type ArtifactBuilder interface {
	Build(ctx context.Context, p ArtifactBuildParams) (PrebuiltArtifacts, error)
}

// ArtifactPushParams transfers one completed bundle over the established SSH
// channel. The driver chooses a private, unique remote staging directory.
type ArtifactPushParams struct {
	Conn      Conn
	Artifacts PrebuiltArtifacts
}

// RemoteArtifacts are the node-side paths bootstrap receives as its prebuilt
// intake flags.
type RemoteArtifacts struct {
	Vrooli    string
	BridgeCLI string
	Agent     string
}

// SSHDriver is the SSH-capability seam: establish passwordless SSH, copy the
// bootstrap script to the node, ship the control plane's working tree (working-
// tree mode), and run the bootstrap while streaming its VBOOTSTRAP markers back.
// Production wraps internal/onboard/ssh; unit tests fake it.
type SSHDriver interface {
	// FirstTouch establishes working passwordless SSH to the host (generate key,
	// copy it with the password, retest) and returns the key-authenticated Conn.
	// The password is zeroed by the driver.
	FirstTouch(ctx context.Context, p FirstTouchParams) (Conn, error)

	// PushScript copies the bootstrap script to the node and returns the remote
	// path it landed at.
	PushScript(ctx context.Context, conn Conn) (remotePath string, err error)

	// SyncTree ships the control plane's working tree to the node's DestDir over
	// the established SSH channel (tar-over-ssh). Only called in working-tree
	// source mode. It returns the number of bytes transferred.
	SyncTree(ctx context.Context, p SyncParams) (SyncResult, error)

	// DetectPlatform resolves the one GOOS/GOARCH pair to cross-compile. It is
	// called only after passwordless SSH is established.
	DetectPlatform(ctx context.Context, conn Conn) (NodePlatform, error)

	// PushArtifacts stages the three executables and their .fp sidecars in a
	// private node-side directory and returns their executable paths.
	PushArtifacts(ctx context.Context, p ArtifactPushParams) (RemoteArtifacts, error)

	// RunBootstrap runs the bootstrap script remotely, injecting the pairing code
	// over stdin, and invokes onMarker for every parsed VBOOTSTRAP stdout marker
	// as it streams. It returns once the script exits.
	RunBootstrap(ctx context.Context, p RunParams, onMarker func(Marker)) (BootstrapResult, error)
}

// WorkingTreeSnapshot is the control plane's local working tree at ship time: the
// base commit it sits on, a deterministic content digest, the repo root, and the
// exact file list to ship. Empty Files with a valid BaseHEAD is a clean checkout
// at that commit (nothing uncommitted); the ship still happens so the node builds
// from exactly what the operator has locally.
type WorkingTreeSnapshot struct {
	// BaseHEAD is the control plane's HEAD commit the tree sits on.
	BaseHEAD string
	// Digest is a deterministic hash over the file list + per-file content, so
	// re-shipping changed work is detectable (it re-keys the node's setup sentinel).
	Digest string
	// RepoDir is the control-plane checkout root the Files are relative to.
	RepoDir string
	// Files are the repo-relative paths to ship (tracked + modified +
	// untracked-non-ignored), from `git ls-files -z -c -o --exclude-standard`.
	Files []string
}

// WorkingTreeSource snapshots the control plane's local working tree for a
// working-tree-mode ship. Production (worktree.go) drives git + the filesystem;
// unit tests fake it.
type WorkingTreeSource interface {
	// Snapshot enumerates the control plane's working tree and computes its base
	// HEAD and content digest. It reads only; it never mutates the checkout.
	Snapshot(ctx context.Context) (WorkingTreeSnapshot, error)
}

// NodeRevisionRecorder writes a node's provenance revision after onboarding
// verifies it ONLINE, so `nodes list` / node detail / the fleet UI render what
// the node was actually brought to — a pinned commit, or a "<base>+dirty" marker
// for a working-tree node. Production wraps the registry service's Update; unit
// tests fake it. A recording failure never fails onboarding (the node is already
// paired and online) — it is logged and the op still succeeds.
type NodeRevisionRecorder interface {
	RecordRevision(ctx context.Context, nodeID, revision string) error
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

	// ResolveWorkingTree is the working-tree-mode variant: it defaults (empty/"@cp"
	// → the control plane's commit) and metacharacter-validates the base revision
	// but SKIPS the pushed preflight — the tree is shipped over SSH, not fetched, so
	// an unpushed base commit is expected and must not hard-fail. Pinned mode keeps
	// Resolve (ErrNotPushed stays a hard failure); only working-tree mode bypasses.
	ResolveWorkingTree(ctx context.Context, requested string) (string, error)
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
