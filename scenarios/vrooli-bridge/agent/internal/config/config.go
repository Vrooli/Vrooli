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
	"strings"
	"time"

	"vrooli-bridge/agent/internal/platform"
)

// Defaults that are not derived from the OS state directory.
const (
	defaultHeartbeatInterval = 15 * time.Second
	// credentialFileName is the on-disk *file name* the Ed25519 private key is
	// written to — not a secret value itself.
	credentialFileName      = "node_credential.key" //nolint:gosec // G101: filename, not an embedded credential
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

	// PrintPublicKey, when true, makes the agent load-or-generate its Ed25519
	// keypair, print the base64 public key (for the `pair redeem --public-key`
	// bootstrap step), and exit without dialing.
	PrintPublicKey bool
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
		controlPlaneURL = fs.String("control-plane-url", envOr("BRIDGE_CONTROL_PLANE_URL", ""), "Base URL of the control plane to dial out to")
		nodeID          = fs.String("node-id", envOr("BRIDGE_NODE_ID", ""), "Durable node identity assigned at registration")
		stateDirFlag    = fs.String("state-dir", envOr("BRIDGE_AGENT_STATE_DIR", ""), "Directory for agent credential/state material")
		heartbeat       = fs.Duration("heartbeat-interval", envDurationOr("BRIDGE_HEARTBEAT_INTERVAL", defaultHeartbeatInterval), "Interval between heartbeats once connected")
		capabilities    = fs.String("capabilities", envOr("BRIDGE_CAPABILITIES", ""), "Comma-separated verb-namespace allowlist the node advertises")
		printPublicKey  = fs.Bool("print-public-key", false, "Load-or-generate the node keypair, print its base64 public key, and exit (bootstrap helper)")
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
	} else {
		if err := os.MkdirAll(stateDir, 0o700); err != nil {
			return Config{}, fmt.Errorf("create agent state dir %q: %w", stateDir, err)
		}
	}

	if *heartbeat <= 0 {
		return Config{}, errors.New("heartbeat-interval must be positive")
	}

	return Config{
		ControlPlaneURL:     strings.TrimSpace(*controlPlaneURL),
		NodeID:              strings.TrimSpace(*nodeID),
		StateDir:            stateDir,
		CredentialPath:      filepath.Join(stateDir, credentialFileName),
		ControlPlaneKeyPath: filepath.Join(stateDir, controlPlaneKeyFileName),
		HeartbeatInterval:   *heartbeat,
		Capabilities:        splitCapabilities(*capabilities),
		PrintPublicKey:      *printPublicKey,
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
