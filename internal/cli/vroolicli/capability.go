package vroolicli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/capabilityledger"
)

func (app *App) runCapabilityCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(ctx.Stdout, "Usage: vrooli capability ledger|fleet [query] [--json]")
		return nil
	}
	if args[0] != "ledger" && args[0] != "fleet" {
		return fmt.Errorf("unknown capability command %q", args[0])
	}
	jsonOutput := ctx.Globals.JSON
	query := ""
	for _, arg := range args[1:] {
		switch strings.TrimSpace(arg) {
		case "--json":
			jsonOutput = true
		case "--help", "-h":
			fmt.Fprintln(ctx.Stdout, "Usage: vrooli capability ledger|fleet [query] [--json]")
			return nil
		case "blocked", "docker", "peerless", "upgrades", "desktop":
			if args[0] != "fleet" {
				return fmt.Errorf("query %q is only valid for capability fleet", arg)
			}
			query = strings.TrimSpace(arg)
		default:
			return fmt.Errorf("unknown capability ledger option %q", arg)
		}
	}
	if args[0] == "fleet" {
		readout, err := capabilityledger.GenerateFleet(ctx.Root)
		if err != nil {
			return err
		}
		if jsonOutput {
			var value interface{} = readout
			switch query {
			case "blocked":
				value = readout.BlockedByOS
			case "docker":
				value = readout.DockerBlocked
			case "peerless":
				value = readout.Peerless
			case "upgrades":
				value = readout.TierUpgrades
			case "desktop":
				value = readout.DesktopBundling
			}
			return json.NewEncoder(ctx.Stdout).Encode(value)
		}
		switch query {
		case "blocked":
			fmt.Fprintf(ctx.Stdout, "blocked_by_os=%d\n", len(readout.BlockedByOS))
			return nil
		case "docker":
			fmt.Fprintf(ctx.Stdout, "docker_blocked=%d\n", len(readout.DockerBlocked))
			return nil
		case "peerless":
			fmt.Fprintf(ctx.Stdout, "peerless=%d\n", len(readout.Peerless))
			return nil
		case "upgrades":
			fmt.Fprintf(ctx.Stdout, "tier_upgrades=%d\n", len(readout.TierUpgrades))
			return nil
		case "desktop":
			fmt.Fprintln(ctx.Stdout, readout.DesktopBundling.Reason)
			return nil
		}
		fmt.Fprintf(ctx.Stdout, "blocked_by_os=%d docker_blocked=%d peerless=%d tier_upgrades=%d\n", len(readout.BlockedByOS), len(readout.DockerBlocked), len(readout.Peerless), len(readout.TierUpgrades))
		fmt.Fprintf(ctx.Stdout, "desktop_bundling: %s\n", readout.DesktopBundling.Reason)
		return nil
	}
	ledger, err := capabilityledger.Generate(ctx.Root)
	if err != nil {
		return err
	}
	if jsonOutput {
		return json.NewEncoder(ctx.Stdout).Encode(ledger)
	}
	for _, entry := range ledger.Capabilities {
		fmt.Fprintln(ctx.Stdout, entry.Capability)
		for _, hostOS := range []string{"linux", "macos", "windows"} {
			platform := entry.Platforms[hostOS]
			fmt.Fprintf(ctx.Stdout, "  %-7s %-10s %s\n", hostOS, platform.Status, firstNonEmpty(platform.Implementer, platform.Mechanism, platform.Reason))
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
