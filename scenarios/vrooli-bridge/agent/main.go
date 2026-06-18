// Command vrooli-bridge-agent is the cross-compiled node-agent that runs on
// each trusted Vrooli node (OT-P0-007). It holds a persistent dial-out channel
// to the control plane, validates typed jobs against the local CLI manifest,
// and shells to the node's own `vrooli` CLI as a non-privileged runner — a
// separate privileged helper performs provisioning (DECISIONS.md two trust
// tiers).
//
// The agent parses configuration, reports the build fingerprint and the channel
// handshake it presents, then holds the live dial-out channel (SSE stream +
// heartbeat loop with reconnect/backoff) until it is signalled to stop. An
// unpaired agent reports that and exits cleanly. The binary cross-compiles
// CGO_ENABLED=0 for linux/darwin/windows × amd64/arm64 (see Makefile and
// build/crosscompile_test.sh). Mutual auth (Phase 2), the non-privileged job
// runner (Phase 3, internal/exec), and the structurally separate privileged
// provisioning helper (Phase 4, internal/privsep) are all wired; --print-public-key
// and --print-service-unit are the two bootstrap helpers the installer drives.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"vrooli-bridge/agent/internal/buildinfo"
	"vrooli-bridge/agent/internal/channel"
	"vrooli-bridge/agent/internal/config"
	"vrooli-bridge/agent/internal/discovery"
	"vrooli-bridge/agent/internal/nodecred"
	"vrooli-bridge/agent/internal/platform"
	"vrooli-bridge/agent/internal/service"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("vrooli-bridge-agent: %v", err)
	}
}

func run(args []string) error {
	logger := log.New(os.Stderr, "", log.LstdFlags)

	cfg, err := config.Load(args)
	if err != nil {
		// flag.ContinueOnError already printed usage for parse errors; treat a
		// help request as a clean exit.
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	logger.Printf("vrooli-bridge-agent %s (%s service manager, state dir %s)",
		buildinfo.Fingerprint(), platform.NativeServiceManager(), cfg.StateDir)

	if cfg.PrintServiceUnit {
		// Bootstrap helper: render this binary's platform-native service unit and
		// exit. The installer writes it to the OS's unit location (OT-P0-007).
		unit, err := renderServiceUnit(cfg)
		if err != nil {
			return fmt.Errorf("render service unit: %w", err)
		}
		fmt.Println(unit)
		return nil
	}

	// The node's Ed25519 keypair is generated once (at first run / bootstrap)
	// and held for the node's lifetime; the public key is registered with the
	// control plane at pairing, and every dial/heartbeat is signed with it.
	cred, err := nodecred.LoadOrCreate(cfg.CredentialPath)
	if err != nil {
		return fmt.Errorf("load node credential: %w", err)
	}
	if cfg.PrintPublicKey {
		// Bootstrap helper: print the key the installer feeds to `pair redeem`.
		fmt.Println(cred.PublicKeyBase64())
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// LAN auto-discovery (OT-P1-006): when no control-plane URL is configured and
	// --discover is set, browse the trusted LAN (mDNS) for the advertised control
	// plane. The manual URL always wins (so off-LAN bootstrap never depends on
	// mDNS); a browse that finds nothing or errors falls back cleanly to the
	// manual path.
	if cfg.ControlPlaneURL == "" && cfg.Discover {
		res, derr := discovery.Resolve(ctx, cfg.ControlPlaneURL, &discovery.MDNSBrowser{}, 0)
		if derr != nil {
			logger.Printf("mDNS discovery error (falling back to a manual control-plane URL): %v", derr)
		}
		if res.Found() {
			cfg.ControlPlaneURL = res.URL
			logger.Printf("discovered control plane via %s: %s", res.Source, cfg.ControlPlaneURL)
		}
	}

	client := channel.NewClient(cfg, channel.WithLogger(logger), channel.WithCredential(cred))
	hs := client.Handshake()
	logger.Printf("channel handshake: node_id=%q protocol_version=%d os=%s arch=%s capabilities=%v",
		hs.GetNodeId(), hs.GetProtocolVersion(), hs.GetOs(), hs.GetArch(), hs.GetCapabilities())

	logger.Printf("holding dial-out channel to %s (heartbeat every %s); send SIGINT/SIGTERM to stop",
		cfg.ControlPlaneURL, cfg.HeartbeatInterval)

	if err := client.Dial(ctx); err != nil {
		if errors.Is(err, channel.ErrNotConfigured) {
			logger.Printf("not paired yet: %v — run the bootstrap installer to pair this node", err)
			return nil
		}
		return fmt.Errorf("dial control plane: %w", err)
	}

	logger.Printf("channel closed (shutdown signal received)")
	return nil
}

// renderServiceUnit builds this binary's platform-native background-service unit
// from the resolved config. The argv it embeds re-runs THIS binary in its
// long-lived dial mode (the same control-plane URL / node id / state dir), so
// the installed service reconnects exactly as the foreground process would.
func renderServiceUnit(cfg config.Config) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve agent binary path: %w", err)
	}
	args := []string{
		"--control-plane-url", cfg.ControlPlaneURL,
		"--node-id", cfg.NodeID,
		"--state-dir", cfg.StateDir,
	}
	if cfg.WorkDir != "" {
		args = append(args, "--work-dir", cfg.WorkDir)
	}
	def := service.Definition{
		Name:        "vrooli-bridge-agent",
		Description: "Vrooli Bridge node agent",
		ExecPath:    exe,
		Args:        args,
		WorkingDir:  cfg.WorkDir,
		User:        cfg.ServiceUser,
	}
	return service.NewManager().Render(def)
}
