// Package config parses the node-agent's runtime configuration from flags and
// environment, layered over per-OS defaults. Flags win over env; env wins over
// defaults. The agent is intentionally configuration-light: a control-plane
// URL, the node's durable id, and where its credential material lives — the
// pairing bootstrap (Phase 2) writes the rest.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vrooli-bridge/agent/internal/platform"
)

// Defaults that are not derived from the OS state directory.
const (
	defaultHeartbeatInterval = 15 * time.Second
	// credentialFileName is the on-disk *file name* the Ed25519 private key is
	// written to — not a secret value itself.
	credentialFileName      = "node_credential.key" // #nosec G101 -- this is a filename constant, not an embedded credential.
	controlPlaneKeyFileName = "control_plane.pub"   // the pinned control-plane public key
)

// Config is the fully-resolved agent configuration.
type Config struct {
	// ControlPlaneURL is the base URL the agent dials out to. Required before
	// the agent can hold a channel; empty is valid for the Phase 0 skeleton
	// (the agent reports "no control plane configured" and exits).
	ControlPlaneURL string

	// NodeID is the durable identity assigned at registration. Empty until the
	// node is paired.
	NodeID string

	// StateDir is the resolved per-OS directory holding credential material.
	StateDir string

	// CredentialPath is where the node's Ed25519 private key lives.
	CredentialPath string

	// ControlPlaneKeyPath is where the pinned control-plane public key lives
	// (written out-of-band at bootstrap; the node verifies every push against
	// it, SECURITY.md boundary 2).
	ControlPlaneKeyPath string

	// HeartbeatInterval is how often the agent sends a Heartbeat once
	// connected.
	HeartbeatInterval time.Duration

	// Capabilities is the verb-namespace allowlist the node advertises (the
	// authoritative scopes live on the node record server-side).
	Capabilities []string

	// WorkDir is the directory the runner executes `vrooli` jobs in (typically
	// the node's Vrooli checkout). Empty means the agent's current working
	// directory.
	WorkDir string

	// VrooliBin is the path/name of the local vrooli CLI the runner shells to as
	// an argv (never via a shell). Defaults to "vrooli" (resolved on PATH).
	VrooliBin string

	// PrintPublicKey, when true, makes the agent load-or-generate its Ed25519
	// keypair, print the base64 public key (for the `pair redeem --public-key`
	// bootstrap step), and exit without dialing.
	PrintPublicKey bool

	// PrintServiceUnit, when true, makes the agent render the platform-native
	// background-service unit (systemd unit / launchd plist / Windows `sc.exe`
	// argv) for THIS binary + config and exit. The bootstrap installer pipes it
	// into the unit file (OT-P0-007). It dials nothing.
	PrintServiceUnit bool

	// ServiceUser is the OS principal the rendered service runs as (the
	// dedicated unprivileged service user). Empty installs under the installing
	// user. Used only with PrintServiceUnit.
	ServiceUser string

	// Discover, when true and no ControlPlaneURL is configured, lets the agent
	// try mDNS LAN auto-discovery (internal/discovery, OT-P1-006) to locate the
	// control plane on a trusted LAN. The manual URL+code path remains the
	// cross-network default and the fallback when discovery finds nothing; an
	// explicit ControlPlaneURL always wins over discovery.
	Discover bool

	// PresenceOnly denies all pushed job and provisioning frames after pairing.
	// It defaults true so a newly enrolled agent can heartbeat without gaining
	// remote execution authority. An explicit policy-approved upgrade must opt
	// in to control actions.
	PresenceOnly bool

	// ProvisionHelper makes the binary run only the privileged provisioning IPC
	// server. It never dials the control plane and is installed as a distinct
	// service under the provisioning principal.
	ProvisionHelper bool

	// ProvisionSocket is the local IPC endpoint shared by the runner and the
	// provisioning helper. An installer can place it in a root-owned,
	// group-readable runtime directory.
	ProvisionSocket string

	// ProvisionClientUID restricts the Unix IPC server to the runner's OS uid.
	// A negative value disables the optional peer check on platforms without a
	// portable peer-credential API.
	ProvisionClientUID int

	// ProvisionHelperUID lets the runner verify that it connected to the
	// provisioner principal rather than an arbitrary process that replaced the
	// socket. A negative value leaves that check to the socket's native ACL.
	ProvisionHelperUID int

	// ProvisionClientHome is the runner principal's home directory. The
	// privileged helper runs as a separate OS user, so it must carry this
	// identity explicitly when invoking user-scoped Vrooli commands.
	ProvisionClientHome string

	// SystemService asks the native manager for a machine-wide service unit. It
	// is reserved for the privileged helper; ordinary agents remain user-scoped.
	SystemService bool

	// CleanupStdin runs one typed cleanup command received as protobuf JSON on
	// stdin and exits. It is the fixed SSH transport entrypoint; it never opens
	// a shell and never accepts caller-supplied argv.
	CleanupStdin bool
}

// Paired reports whether the agent has the minimum configuration to hold a
// channel: a control-plane URL and a node id. The Phase 0 skeleton uses this
// to decide between attempting a (stubbed) dial and exiting cleanly.
func (c Config) Paired() bool {
	return c.ControlPlaneURL != "" && c.NodeID != ""
}

// Load resolves configuration from the given args (typically os.Args[1:]) and
// the process environment. It is pure with respect to its inputs except for
// resolving the OS state directory, which it creates if missing.
func Load(args []string) (Config, error) {
	fs := flag.NewFlagSet("vrooli-bridge-agent", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		controlPlaneURL  = fs.String("control-plane-url", envOr("BRIDGE_CONTROL_PLANE_URL", ""), "Base URL of the control plane to dial out to")
		nodeID           = fs.String("node-id", envOr("BRIDGE_NODE_ID", ""), "Durable node identity assigned at registration")
		stateDirFlag     = fs.String("state-dir", envOr("BRIDGE_AGENT_STATE_DIR", ""), "Directory for agent credential/state material")
		heartbeat        = fs.Duration("heartbeat-interval", envDurationOr("BRIDGE_HEARTBEAT_INTERVAL", defaultHeartbeatInterval), "Interval between heartbeats once connected")
		capabilities     = fs.String("capabilities", envOr("BRIDGE_CAPABILITIES", ""), "Comma-separated verb-namespace allowlist the node advertises")
		workDir          = fs.String("work-dir", envOr("BRIDGE_WORK_DIR", ""), "Directory the runner executes jobs in (default: current working directory)")
		vrooliBin        = fs.String("vrooli-bin", envOr("BRIDGE_VROOLI_BIN", "vrooli"), "Path/name of the local vrooli CLI the runner shells to as an argv")
		printPublicKey   = fs.Bool("print-public-key", false, "Load-or-generate the node keypair, print its base64 public key, and exit (bootstrap helper)")
		printServiceUnit = fs.Bool("print-service-unit", false, "Render this binary's platform-native background-service unit and exit (bootstrap helper)")
		serviceUser      = fs.String("service-user", envOr("BRIDGE_SERVICE_USER", ""), "OS principal the rendered service runs as (with --print-service-unit)")
		discover         = fs.Bool("discover", envBoolOr("BRIDGE_DISCOVER", false), "Try mDNS LAN auto-discovery of the control plane when no --control-plane-url is set (manual URL stays the cross-network default)")
		presenceOnly     = fs.Bool("presence-only", envBoolOr("BRIDGE_PRESENCE_ONLY", true), "Hold presence only; reject pushed jobs and provisioning commands (default true)")
		provisionHelper  = fs.Bool("provision-helper", envBoolOr("BRIDGE_PROVISION_HELPER", false), "Run the privileged provisioning IPC helper instead of the node agent")
		provisionSocket  = fs.String("provision-socket", envOr("BRIDGE_PROVISION_SOCKET", ""), "Local IPC socket shared by the runner and provisioning helper")
		clientUID        = fs.Int("provision-client-uid", envIntOr("BRIDGE_PROVISION_CLIENT_UID", -1), "Unix uid permitted to call the provisioning helper")
		helperUID        = fs.Int("provision-helper-uid", envIntOr("BRIDGE_PROVISION_HELPER_UID", -1), "Unix uid expected for the provisioning helper")
		clientHome       = fs.String("provision-client-home", envOr("BRIDGE_PROVISION_CLIENT_HOME", ""), "Home directory of the runner principal for privileged child commands")
		systemService    = fs.Bool("system-service", envBoolOr("BRIDGE_SYSTEM_SERVICE", false), "Install a machine-wide native service (privileged helper only)")
		cleanupStdin     = fs.Bool("cleanup-stdin", false, "Run one typed cleanup command from protobuf JSON on stdin")
	)

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	stateDir := strings.TrimSpace(*stateDirFlag)
	if stateDir == "" {
		resolved, err := platform.StateDir()
		if err != nil {
			return Config{}, fmt.Errorf("resolve agent state dir: %w", err)
		}
		stateDir = resolved
	} else if !*provisionHelper {
		if err := os.MkdirAll(stateDir, 0o700); err != nil {
			return Config{}, fmt.Errorf("create agent state dir %q: %w", stateDir, err)
		}
	}

	if *heartbeat <= 0 {
		return Config{}, errors.New("heartbeat-interval must be positive")
	}
	if *provisionHelper && *clientUID < 0 {
		return Config{}, errors.New("provision helper requires --provision-client-uid")
	}
	if *systemService && !*provisionHelper {
		return Config{}, errors.New("--system-service is reserved for --provision-helper")
	}

	return Config{
		ControlPlaneURL:     strings.TrimSpace(*controlPlaneURL),
		NodeID:              strings.TrimSpace(*nodeID),
		StateDir:            stateDir,
		CredentialPath:      filepath.Join(stateDir, credentialFileName),
		ControlPlaneKeyPath: filepath.Join(stateDir, controlPlaneKeyFileName),
		HeartbeatInterval:   *heartbeat,
		Capabilities:        splitCapabilities(*capabilities),
		WorkDir:             strings.TrimSpace(*workDir),
		VrooliBin:           strings.TrimSpace(*vrooliBin),
		PrintPublicKey:      *printPublicKey,
		PrintServiceUnit:    *printServiceUnit,
		ServiceUser:         strings.TrimSpace(*serviceUser),
		Discover:            *discover,
		PresenceOnly:        *presenceOnly,
		ProvisionHelper:     *provisionHelper,
		ProvisionSocket:     strings.TrimSpace(*provisionSocket),
		ProvisionClientUID:  *clientUID,
		ProvisionHelperUID:  *helperUID,
		ProvisionClientHome: strings.TrimSpace(*clientHome),
		SystemService:       *systemService,
		CleanupStdin:        *cleanupStdin,
	}, nil
}

func splitCapabilities(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// envBoolOr reads a boolean env var (1/t/T/TRUE/true/0/f/false/…) falling back
// to fallback when unset, empty, or unparseable. It mirrors flag's own bool
// parsing so --discover and BRIDGE_DISCOVER accept the same spellings.
func envBoolOr(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return b
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envIntOr(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return n
}
