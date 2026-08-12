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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"vrooli-bridge/agent/internal/buildinfo"
	"vrooli-bridge/agent/internal/channel"
	"vrooli-bridge/agent/internal/config"
	"vrooli-bridge/agent/internal/cpverify"
	"vrooli-bridge/agent/internal/discovery"
	"vrooli-bridge/agent/internal/nodecred"
	"vrooli-bridge/agent/internal/platform"
	"vrooli-bridge/agent/internal/privsep"
	"vrooli-bridge/agent/internal/service"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("vrooli-bridge-agent: %v", err)
	}
}

func run(args []string) error {
	logger := log.New(os.Stderr, "", log.LstdFlags)

	// `service install|status|uninstall` is the OS-service management surface the
	// bootstrap installer drives (OT-P0-007). It is layered over the same config
	// resolution + service.Definition as --print-service-unit, so the installed
	// unit's argv matches exactly what --print-service-unit renders.
	if len(args) > 0 && args[0] == "service" {
		return runService(args[1:])
	}

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

	if cfg.ProvisionHelper {
		if strings.TrimSpace(cfg.ProvisionSocket) == "" {
			return fmt.Errorf("provision helper requires --provision-socket")
		}
		helperCtx, stopHelper := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stopHelper()
		logger.Printf("provisioning helper socket=%s", cfg.ProvisionSocket)
		if err := privsep.Serve(helperCtx, cfg.ProvisionSocket, cfg.VrooliBin, cfg.WorkDir, cfg.ProvisionClientUID); err != nil {
			return fmt.Errorf("provisioning helper: %w", err)
		}
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

	// Pin the control-plane key BEFORE dialing (SECURITY.md boundary 2): a paired
	// agent verifies every server push against the key `pair redeem` wrote to
	// <state-dir>/control_plane.pub. A missing pin is a hard, actionable failure —
	// there is deliberately no trust-on-first-frame fallback. An unpaired agent
	// skips this and exits cleanly on ErrNotConfigured below.
	var cpVerifier *cpverify.Verifier
	if cfg.Paired() {
		cpVerifier, err = cpverify.Load(cfg.ControlPlaneKeyPath)
		if err != nil {
			if errors.Is(err, cpverify.ErrNoPin) {
				return fmt.Errorf("%w — run `vrooli-bridge pair redeem` (or the bootstrap installer) to pin the control-plane key before starting the agent", err)
			}
			return fmt.Errorf("load pinned control-plane key: %w", err)
		}
	}

	client := channel.NewClient(cfg, channel.WithLogger(logger), channel.WithCredential(cred), channel.WithCPVerifier(cpVerifier))
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
// from the resolved config.
func renderServiceUnit(cfg config.Config) (string, error) {
	def, err := serviceDefinition(cfg)
	if err != nil {
		return "", err
	}
	return service.NewManager().Render(def)
}

// serviceDefinition builds the service.Definition for THIS binary from the
// resolved config. The argv it embeds re-runs this binary in its long-lived dial
// mode (the same control-plane URL / node id / state dir), so the installed
// service reconnects exactly as the foreground process would. Both
// --print-service-unit and the `service` verbs go through it so the rendered and
// installed unit are byte-for-byte the same argv.
func serviceDefinition(cfg config.Config) (service.Definition, error) {
	exe, err := os.Executable()
	if err != nil {
		return service.Definition{}, fmt.Errorf("resolve agent binary path: %w", err)
	}
	var args []string
	name := "vrooli-bridge-agent"
	description := "Vrooli Bridge node agent"
	if cfg.ProvisionHelper {
		name = "vrooli-bridge-provisioner"
		description = "Vrooli Bridge privileged provisioning helper"
		args = []string{"--state-dir", cfg.StateDir, "--provision-helper", "--provision-socket", cfg.ProvisionSocket}
		if cfg.ProvisionClientUID >= 0 {
			args = append(args, "--provision-client-uid", strconv.Itoa(cfg.ProvisionClientUID))
		}
	} else {
		args = []string{"--control-plane-url", cfg.ControlPlaneURL, "--node-id", cfg.NodeID, "--state-dir", cfg.StateDir}
	}
	if cfg.WorkDir != "" {
		args = append(args, "--work-dir", cfg.WorkDir)
	}
	// Persist the runner policy in the native service unit. The bootstrap
	// installer supplies these flags while installing the service; dropping
	// them here would make a reboot silently restore the safe presence-only
	// default and would also make the runner fall back to PATH, which is not
	// reliable for launchd/systemd services.
	if cfg.VrooliBin != "" {
		args = append(args, "--vrooli-bin", cfg.VrooliBin)
	}
	if cfg.ProvisionSocket != "" && !cfg.ProvisionHelper {
		args = append(args, "--provision-socket", cfg.ProvisionSocket)
		if cfg.ProvisionHelperUID >= 0 {
			args = append(args, "--provision-helper-uid", strconv.Itoa(cfg.ProvisionHelperUID))
		}
	}
	if !cfg.ProvisionHelper && len(cfg.Capabilities) > 0 {
		args = append(args, "--capabilities", strings.Join(cfg.Capabilities, ","))
	}
	// Keep the boolean value in one argv token. Go's flag package treats a bare
	// --presence-only as true, so the separated form would silently turn a
	// granted execution node back into presence-only when rendering its service.
	if !cfg.ProvisionHelper {
		args = append(args, "--presence-only="+strconv.FormatBool(cfg.PresenceOnly))
	}
	return service.Definition{
		Name:              name,
		Description:       description,
		ExecPath:          exe,
		Args:              args,
		WorkingDir:        cfg.WorkDir,
		User:              cfg.ServiceUser,
		StandardOutPath:   filepath.Join(cfg.StateDir, "agent.stdout.log"),
		StandardErrorPath: filepath.Join(cfg.StateDir, "agent.stderr.log"),
		System:            cfg.SystemService,
	}, nil
}

// runService dispatches `service install|status|uninstall`. It resolves config
// from the remaining args exactly like the dial path (so the installed unit
// carries the same control-plane URL / node id / state dir), builds this
// binary's service.Definition, and drives the host's Manager. A --json flag on
// any verb prints the machine-readable result (the phase-4 bootstrap script
// parses install/status output); the default is a concise human line.
func runService(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("service: expected a verb (install|status|uninstall)")
	}
	verb := args[0]
	rest, asJSON := extractJSONFlag(args[1:])

	cfg, err := config.Load(rest)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	def, err := serviceDefinition(cfg)
	if err != nil {
		return err
	}
	mgr := service.NewManager()
	ctx := context.Background()

	switch verb {
	case "install":
		res, err := mgr.Install(ctx, def)
		if err != nil {
			return fmt.Errorf("service install: %w", err)
		}
		if asJSON {
			return printJSON(res)
		}
		fmt.Printf("installed %s (%s)\n  unit:    %s\n  enabled: %t\n  running: %t\n",
			res.UnitName, res.Kind, res.UnitPath, res.Enabled, res.Running)
		return nil
	case "status":
		res, err := mgr.Status(ctx, def)
		if err != nil {
			return fmt.Errorf("service status: %w", err)
		}
		if asJSON {
			if jerr := printJSON(res); jerr != nil {
				return jerr
			}
		} else {
			fmt.Printf("%s (%s)\n  unit:      %s\n  installed: %t\n  enabled:   %t\n  running:   %t\n  pid:       %d\n  detail:    %s\n",
				res.UnitName, res.Kind, res.UnitPath, res.Installed, res.Enabled, res.Running, res.PID, res.Detail)
		}
		// Exit non-zero when not running so a caller (bootstrap script) can gate on
		// `service status` without parsing output.
		if !res.Running {
			return errServiceNotRunning
		}
		return nil
	case "uninstall":
		res, err := mgr.Uninstall(ctx, def)
		if err != nil {
			return fmt.Errorf("service uninstall: %w", err)
		}
		if asJSON {
			return printJSON(res)
		}
		fmt.Printf("uninstalled %s (%s)\n  unit:    %s\n  removed: %t\n",
			res.UnitName, res.Kind, res.UnitPath, res.Removed)
		return nil
	default:
		return fmt.Errorf("service: unknown verb %q (want install|status|uninstall)", verb)
	}
}

// errServiceNotRunning is returned by `service status` when the service is not
// active, so the process exits non-zero for scripted gating.
var errServiceNotRunning = errors.New("service is not running")

// extractJSONFlag removes a --json / -json token from args, returning the
// remaining args and whether the flag was present. The service verbs hand the
// remainder to config.Load, which owns the rest of the flag surface.
func extractJSONFlag(args []string) ([]string, bool) {
	out := make([]string, 0, len(args))
	asJSON := false
	for _, a := range args {
		if a == "--json" || a == "-json" {
			asJSON = true
			continue
		}
		out = append(out, a)
	}
	return out, asJSON
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
