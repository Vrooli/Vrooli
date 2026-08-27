package capabilityapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/capabilitycatalog"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/deployability"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	portabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability"
	"google.golang.org/protobuf/proto"
)

const (
	capabilityDocker = "docker"
)

const (
	capabilityBlocked  = "blocked"
	capabilityFleet    = "fleet"
	capabilityHelp     = "--help"
	capabilityJson     = "--json"
	capabilityPeerless = "peerless"
	capabilityDesktop  = "desktop"
	capabilityUpgrades = "upgrades"
)

// Run dispatches `vrooli capability`.
func (app *App) Run(ctx *CommandContext, args []string) error {
	return app.runCapabilityCommand(ctx, args)
}

func (app *App) runCapabilityCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 || args[0] == capabilityHelp || args[0] == "-h" {
		fmt.Fprintln(ctx.Stdout, "Usage: vrooli capability ledger|fleet [query] [--json]\n  vrooli capability conformance [--json|--declarations-only]\n  vrooli capability catalog|status [--json]\n  vrooli capability preview|apply [--json] < action JSON")
		return nil
	}
	if args[0] == "conformance" {
		return app.runCapabilityConformanceCommand(ctx, args[1:])
	}
	if args[0] == "catalog" || args[0] == "status" || args[0] == "preview" || args[0] == "apply" {
		return app.runCapabilityWorkflow(ctx, args)
	}
	if args[0] != "ledger" && args[0] != capabilityFleet {
		return fmt.Errorf("unknown capability command %q", args[0])
	}
	jsonOutput := ctx.Globals.JSON
	query := ""
	for _, arg := range args[1:] {
		switch strings.TrimSpace(arg) {
		case capabilityJson:
			jsonOutput = true
		case capabilityHelp, "-h":
			fmt.Fprintln(ctx.Stdout, "Usage: vrooli capability ledger|fleet [query] [--json]\n  vrooli capability catalog|status [--json]\n  vrooli capability preview|apply [--json] < action JSON")
			return nil
		case capabilityBlocked, capabilityDocker, capabilityPeerless, capabilityUpgrades, capabilityDesktop:
			if args[0] != capabilityFleet {
				return fmt.Errorf("query %q is only valid for capability fleet", arg)
			}
			query = strings.TrimSpace(arg)
		default:
			return fmt.Errorf("unknown capability ledger option %q", arg)
		}
	}
	// Both readouts are owned by the infrastructure-manager instrument. The
	// control plane delegates and renders; it does not keep a second
	// aggregation that could disagree with the owner's.
	requestCtx, cancel := context.WithTimeout(context.Background(), capabilityRequestTimeout)
	defer cancel()

	if args[0] == capabilityFleet {
		readout, err := fetchCapabilityFleet(requestCtx)
		if err != nil {
			return reportCapabilityDegraded(ctx, jsonOutput, err)
		}
		return renderCapabilityFleet(ctx, jsonOutput, query, readout)
	}
	grid, err := fetchCapabilityGrid(requestCtx)
	if err != nil {
		return reportCapabilityDegraded(ctx, jsonOutput, err)
	}
	return renderCapabilityGrid(ctx, jsonOutput, grid)
}

func (app *App) runCapabilityConformanceCommand(ctx *CommandContext, args []string) error {
	jsonOutput := ctx.Globals.JSON
	declarationsOnly := false
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case capabilityJson:
			jsonOutput = true
		case "--declarations-only":
			declarationsOnly = true
		case capabilityHelp, "-h":
			fmt.Fprintln(ctx.Stdout, "Usage: vrooli capability conformance [--json|--declarations-only]")
			return nil
		default:
			return fmt.Errorf("unknown capability conformance option %q", arg)
		}
	}
	var report deployability.ConformanceReport
	var err error
	if declarationsOnly {
		findings, checkErr := deployability.CheckResourceDeclarations(ctx.Root)
		report = deployability.ConformanceReport{Findings: findings}
		err = checkErr
	} else {
		report, err = deployability.CheckRepository(context.Background(), ctx.Root)
	}
	if err != nil {
		return fmt.Errorf("capability conformance: %w", err)
	}
	if jsonOutput {
		if err := cliout.WriteJSONValue(ctx.Stdout, report); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(ctx.Stdout, "checked_targets=%d findings=%d\n", len(report.Targets), len(report.Findings))
		for _, finding := range report.Findings {
			fmt.Fprintf(ctx.Stdout, "%s [%s/%s] %s\n", finding.ManifestPath, finding.OS, finding.Architecture, finding.Message)
		}
	}
	if len(report.Findings) > 0 {
		return fmt.Errorf("capability conformance failed with %d finding(s)", len(report.Findings))
	}
	return nil
}

// reportCapabilityDegraded renders the degraded state and still fails. A
// machine consumer gets an envelope whose `state` is `degraded`, so it can
// never mistake the response for a grid; a human gets the error naming the
// owner and the command that starts it.
func reportCapabilityDegraded(ctx *CommandContext, jsonOutput bool, err error) error {
	var degraded capabilityDegradedError
	if !errors.As(err, &degraded) {
		return err
	}
	if jsonOutput {
		if encodeErr := cliout.WriteJSONValue(ctx.Stdout, newCapabilityDegradedReadout(degraded)); encodeErr != nil {
			return encodeErr
		}
	}
	return degraded
}

func renderCapabilityGrid(ctx *CommandContext, jsonOutput bool, grid *portabilityv1.Grid) error {
	if jsonOutput {
		return cliout.WriteProtoJSON(ctx.Stdout, grid)
	}
	fmt.Fprintf(ctx.Stdout, "manifest_root=%s manifests_read=%d\n", grid.GetManifestRoot(), grid.GetManifestsRead())
	for _, entry := range grid.GetCapabilities() {
		fmt.Fprintln(ctx.Stdout, entry.GetCapability())
		for _, platform := range entry.GetPlatforms() {
			fmt.Fprintf(ctx.Stdout, "  %-7s %-7s %-12s %-14s %s\n",
				enumToken(platform.GetHostOs().String(), "HOST_OS_"),
				platform.GetArchitecture(),
				enumToken(platform.GetStatus().String(), "RESOLUTION_STATUS_"),
				enumToken(platform.GetQualification().String(), "QUALIFICATION_"),
				firstNonEmpty(platform.GetImplementer(), platform.GetMechanism(), platform.GetReason()))
			if len(platform.GetControls()) > 0 {
				fmt.Fprintf(ctx.Stdout, "    controls: %s\n", strings.Join(platform.GetControls(), ", "))
			}
			if len(platform.GetAbsent()) > 0 {
				fmt.Fprintf(ctx.Stdout, "    absent: %s\n", strings.Join(platform.GetAbsent(), ", "))
			}
			for _, declarer := range platform.GetDeclarers() {
				if !declarer.GetResolved() {
					fmt.Fprintf(ctx.Stdout, "    missing %s (%s): %s\n", declarer.GetName(), declarer.GetRole(), declarer.GetReason())
				}
			}
		}
	}
	return nil
}

func renderCapabilityFleet(ctx *CommandContext, jsonOutput bool, query string, readout *portabilityv1.FleetReadout) error {
	if jsonOutput {
		var value proto.Message = readout
		switch query {
		case capabilityBlocked:
			value = &portabilityv1.FleetReadout{BlockedByOs: readout.GetBlockedByOs()}
		case capabilityDocker:
			value = &portabilityv1.FleetReadout{DockerBlocked: readout.GetDockerBlocked()}
		case "peerless":
			value = &portabilityv1.FleetReadout{Peerless: readout.GetPeerless()}
		case "upgrades":
			value = &portabilityv1.FleetReadout{TierUpgrades: readout.GetTierUpgrades()}
		case "desktop":
			value = &portabilityv1.FleetReadout{DesktopBundling: readout.GetDesktopBundling()}
		}
		return cliout.WriteProtoJSON(ctx.Stdout, value)
	}
	switch query {
	case capabilityBlocked:
		fmt.Fprintf(ctx.Stdout, "blocked_by_os=%d\n", len(readout.GetBlockedByOs()))
		return nil
	case capabilityDocker:
		fmt.Fprintf(ctx.Stdout, "docker_blocked=%d\n", len(readout.GetDockerBlocked()))
		return nil
	case "peerless":
		fmt.Fprintf(ctx.Stdout, "peerless=%d\n", len(readout.GetPeerless()))
		return nil
	case "upgrades":
		fmt.Fprintf(ctx.Stdout, "tier_upgrades=%d\n", len(readout.GetTierUpgrades()))
		return nil
	case "desktop":
		fmt.Fprintln(ctx.Stdout, readout.GetDesktopBundling().GetReason())
		return nil
	}
	fmt.Fprintf(ctx.Stdout, "blocked_by_os=%d docker_blocked=%d peerless=%d tier_upgrades=%d\n", len(readout.GetBlockedByOs()), len(readout.GetDockerBlocked()), len(readout.GetPeerless()), len(readout.GetTierUpgrades()))
	fmt.Fprintf(ctx.Stdout, "desktop_bundling: %s\n", readout.GetDesktopBundling().GetReason())
	return nil
}

func enumToken(full, prefix string) string {
	return strings.ToLower(strings.TrimPrefix(full, prefix))
}

func (app *App) runCapabilityWorkflow(ctx *CommandContext, args []string) error {
	action := strings.TrimSpace(args[0])
	jsonOutput := ctx.Globals.JSON
	for _, arg := range args[1:] {
		switch strings.TrimSpace(arg) {
		case capabilityJson:
			jsonOutput = true
		case capabilityHelp, "-h":
			fmt.Fprintln(ctx.Stdout, "Usage: vrooli capability catalog|status [--json]\n  vrooli capability preview|apply [--json] < action JSON")
			return nil
		default:
			return fmt.Errorf("unknown capability workflow option %q", arg)
		}
	}
	home, err := config.VrooliHome()
	if err != nil {
		return err
	}
	registry, err := capabilitycatalog.New(ctx.Root, home)
	if err != nil {
		return err
	}
	if action == "catalog" || action == "status" {
		statuses, err := registry.Discover(context.Background())
		if err != nil {
			return err
		}
		if jsonOutput {
			return cliout.WriteJSONValue(ctx.Stdout, statuses)
		}
		rows := make([][]string, 0, len(statuses))
		for _, status := range statuses {
			rows = append(rows, []string{status.Descriptor.ID, string(status.State), firstNonEmpty(status.Remediation, status.Descriptor.Remediation)})
		}
		return cliout.WriteSection(ctx.Stdout, cliout.Section{Rows: rows})
	}
	input := ctx.Stdin
	if input == nil {
		input = os.Stdin
	}
	var request operatorcapability.ActionRequest
	if err := json.NewDecoder(input).Decode(&request); err != nil {
		return fmt.Errorf("capability action JSON is required on standard input: %w", err)
	}
	if request.CapabilityID == "" {
		return fmt.Errorf("capability action JSON requires capability_id")
	}
	if request.IdempotencyKey == "" {
		request.IdempotencyKey = operatorcapability.StableIdempotencyKey(request.CapabilityID, request.Inputs)
	}
	var output any
	if action == "preview" {
		output, err = registry.Preview(context.Background(), request)
	} else {
		output, err = registry.Apply(context.Background(), request)
	}
	if jsonOutput {
		if encodeErr := cliout.WriteJSONValue(ctx.Stdout, output); encodeErr != nil {
			return encodeErr
		}
	} else {
		fmt.Fprintf(ctx.Stdout, "%s\n", workflowOutcome(output))
	}
	return err
}

func workflowOutcome(value any) string {
	switch result := value.(type) {
	case operatorcapability.Preview:
		return fmt.Sprintf("%s: %s", result.State, result.Remediation)
	case operatorcapability.Result:
		return fmt.Sprintf("%s: %s", result.State, result.Outcome)
	default:
		return "capability action completed"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
