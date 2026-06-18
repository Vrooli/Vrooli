// Command vrooli-bridge-agent is the cross-compiled node-agent that runs on
// each trusted Vrooli node (OT-P0-007). It holds a persistent dial-out channel
// to the control plane, validates typed jobs against the local CLI manifest,
// and shells to the node's own `vrooli` CLI as a non-privileged runner — a
// separate privileged helper performs provisioning (DECISIONS.md two trust
// tiers).
//
// Phase 0 ships the skeleton: parse configuration, report the build
// fingerprint and the channel handshake it would present, attempt a (stubbed)
// dial, and exit cleanly. The binary cross-compiles CGO_ENABLED=0 for
// linux/darwin/windows × amd64/arm64 (see Makefile). The live channel,
// heartbeat loop, mutual auth, runner, and provisioning helper land in
// Phases 1–4.
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
	"vrooli-bridge/agent/internal/platform"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := channel.NewClient(cfg)
	hs := client.Handshake()
	logger.Printf("channel handshake: node_id=%q protocol_version=%d os=%s arch=%s capabilities=%v",
		hs.GetNodeId(), hs.GetProtocolVersion(), hs.GetOs(), hs.GetArch(), hs.GetCapabilities())

	if err := client.Dial(ctx); err != nil {
		if errors.Is(err, channel.ErrNotConfigured) {
			logger.Printf("not paired yet: %v — run the bootstrap installer to pair this node", err)
			return nil
		}
		return fmt.Errorf("dial control plane: %w", err)
	}

	logger.Printf("dial complete (Phase 0 stub: no persistent channel held yet)")
	return nil
}
