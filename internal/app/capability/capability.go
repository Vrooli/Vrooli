package capabilityapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/deployability"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/setup"
	"github.com/vrooli/vrooli/internal/tuning"
	"github.com/vrooli/vrooli/internal/values"
	portabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability"
	"google.golang.org/protobuf/proto"
)

const (
	capabilityBlocked  = "blocked"
	capabilityDocker   = "docker"
	capabilityPeerless = "peerless"
	capabilityDesktop  = "desktop"
	capabilityUpgrades = "upgrades"
)

// Ledger runs a typed capability ledger readout.
func (app *Service) Ledger(ctx context.Context, out io.Writer, opts LedgerOptions) error {
	if ctx == nil {
		return fmt.Errorf("capability operation context is nil")
	}
	jsonOutput := opts.JSON
	query := strings.TrimSpace(opts.Query)
	// Both readouts are owned by the infrastructure-manager instrument. The
	// control plane delegates and renders; it does not keep a second
	// aggregation that could disagree with the owner's.
	requestCtx, cancel := context.WithTimeout(ctx, tuning.CapabilityRequestTimeout())
	defer cancel()

	if opts.Fleet {
		readout, err := fetchCapabilityFleet(requestCtx)
		if err != nil {
			return reportCapabilityDegraded(out, jsonOutput, err)
		}
		return renderCapabilityFleet(out, jsonOutput, query, readout)
	}
	grid, err := fetchCapabilityGrid(requestCtx)
	if err != nil {
		return reportCapabilityDegraded(out, jsonOutput, err)
	}
	return renderCapabilityGrid(out, jsonOutput, grid)
}

// Conformance runs the typed capability declaration check.
func (app *Service) Conformance(ctx context.Context, root string, out io.Writer, opts ConformanceOptions) error {
	if ctx == nil {
		return fmt.Errorf("capability operation context is nil")
	}
	jsonOutput := opts.JSON
	var report deployability.ConformanceReport
	var err error
	if opts.DeclarationsOnly {
		findings, checkErr := deployability.CheckResourceDeclarations(root)
		report = deployability.ConformanceReport{Findings: findings}
		err = checkErr
	} else {
		report, err = deployability.CheckRepository(ctx, root)
	}
	if err != nil {
		return fmt.Errorf("capability conformance: %w", err)
	}
	if jsonOutput {
		if err := cliout.WriteJSONValue(out, report); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(out, "checked_targets=%d findings=%d\n", len(report.Targets), len(report.Findings))
		for _, finding := range report.Findings {
			fmt.Fprintf(out, "%s [%s/%s] %s\n", finding.ManifestPath, finding.OS, finding.Architecture, finding.Message)
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
func reportCapabilityDegraded(out io.Writer, jsonOutput bool, err error) error {
	var degraded capabilityDegradedError
	if !errors.As(err, &degraded) {
		return err
	}
	if jsonOutput {
		if encodeErr := cliout.WriteJSONValue(out, newCapabilityDegradedReadout(degraded)); encodeErr != nil {
			return encodeErr
		}
	}
	return degraded
}

func renderCapabilityGrid(out io.Writer, jsonOutput bool, grid *portabilityv1.Grid) error {
	if jsonOutput {
		return cliout.WriteProtoJSON(out, grid)
	}
	fmt.Fprintf(out, "manifest_root=%s manifests_read=%d\n", grid.GetManifestRoot(), grid.GetManifestsRead())
	for _, entry := range grid.GetCapabilities() {
		fmt.Fprintln(out, entry.GetCapability())
		for _, platform := range entry.GetPlatforms() {
			fmt.Fprintf(out, "  %-7s %-7s %-12s %-14s %s\n",
				enumToken(platform.GetHostOs().String(), "HOST_OS_"),
				platform.GetArchitecture(),
				enumToken(platform.GetStatus().String(), "RESOLUTION_STATUS_"),
				enumToken(platform.GetQualification().String(), "QUALIFICATION_"),
				values.FirstNonEmpty(platform.GetImplementer(), platform.GetMechanism(), platform.GetReason()))
			if len(platform.GetControls()) > 0 {
				fmt.Fprintf(out, "    controls: %s\n", strings.Join(platform.GetControls(), ", "))
			}
			if len(platform.GetAbsent()) > 0 {
				fmt.Fprintf(out, "    absent: %s\n", strings.Join(platform.GetAbsent(), ", "))
			}
			for _, declarer := range platform.GetDeclarers() {
				if !declarer.GetResolved() {
					fmt.Fprintf(out, "    missing %s (%s): %s\n", declarer.GetName(), declarer.GetRole(), declarer.GetReason())
				}
			}
		}
	}
	return nil
}

func renderCapabilityFleet(out io.Writer, jsonOutput bool, query string, readout *portabilityv1.FleetReadout) error {
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
		return cliout.WriteProtoJSON(out, value)
	}
	switch query {
	case capabilityBlocked:
		fmt.Fprintf(out, "blocked_by_os=%d\n", len(readout.GetBlockedByOs()))
		return nil
	case capabilityDocker:
		fmt.Fprintf(out, "docker_blocked=%d\n", len(readout.GetDockerBlocked()))
		return nil
	case "peerless":
		fmt.Fprintf(out, "peerless=%d\n", len(readout.GetPeerless()))
		return nil
	case "upgrades":
		fmt.Fprintf(out, "tier_upgrades=%d\n", len(readout.GetTierUpgrades()))
		return nil
	case "desktop":
		fmt.Fprintln(out, readout.GetDesktopBundling().GetReason())
		return nil
	}
	fmt.Fprintf(out, "blocked_by_os=%d docker_blocked=%d peerless=%d tier_upgrades=%d\n", len(readout.GetBlockedByOs()), len(readout.GetDockerBlocked()), len(readout.GetPeerless()), len(readout.GetTierUpgrades()))
	fmt.Fprintf(out, "desktop_bundling: %s\n", readout.GetDesktopBundling().GetReason())
	return nil
}

func enumToken(full, prefix string) string {
	return strings.ToLower(strings.TrimPrefix(full, prefix))
}

// Workflow executes a typed capability catalog/status/preview/apply request.
func (app *Service) Workflow(ctx context.Context, root string, out io.Writer, opts WorkflowOptions) error {
	if ctx == nil {
		return fmt.Errorf("capability operation context is nil")
	}
	action := strings.TrimSpace(opts.Action)
	jsonOutput := opts.JSON
	home, err := config.VrooliHome()
	if err != nil {
		return err
	}
	registry, err := setup.NewCapabilityRegistry(root, home)
	if err != nil {
		return err
	}
	if action == "catalog" || action == "status" {
		statuses, err := registry.Discover(ctx)
		if err != nil {
			return err
		}
		if jsonOutput {
			return cliout.WriteJSONValue(out, statuses)
		}
		rows := make([][]string, 0, len(statuses))
		for _, status := range statuses {
			rows = append(rows, []string{status.Descriptor.ID, string(status.State), values.FirstNonEmpty(status.Remediation, status.Descriptor.Remediation)})
		}
		return cliout.WriteSection(out, cliout.Section{Rows: rows})
	}
	request := opts.Request
	if request.CapabilityID == "" {
		return fmt.Errorf("capability action JSON requires capability_id")
	}
	if request.IdempotencyKey == "" {
		request.IdempotencyKey = operatorcapability.StableIdempotencyKey(request.CapabilityID, request.Inputs)
	}
	var output any
	if action == "preview" {
		output, err = registry.Preview(ctx, request)
	} else {
		output, err = registry.Apply(ctx, request)
	}
	if jsonOutput {
		if encodeErr := cliout.WriteJSONValue(out, output); encodeErr != nil {
			return encodeErr
		}
	} else {
		fmt.Fprintf(out, "%s\n", workflowOutcome(output))
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
