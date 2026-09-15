package hostapp

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	hostruntime "github.com/vrooli/vrooli/internal/runtime"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// HostSafeguardOK reports whether a safeguard reached a successful terminal
// state. It is exported for the CLI handler and remains the single predicate
// for safeguard exit classification.
func HostSafeguardOK(status hostreqkit.ItemStatus) bool {
	switch status.ExecutionState {
	case hostreqkit.ExecutionApplied,
		hostreqkit.ExecutionAlreadyPresent,
		hostreqkit.ExecutionWouldApply,
		hostreqkit.ExecutionNotApplicable:
		return true
	default:
		return false
	}
}

// HostInstallOK reports whether a host-tool operation reached a successful
// terminal state.
func HostInstallOK(status hostreqkit.ItemStatus) bool {
	switch status.ExecutionState {
	case hostreqkit.ExecutionInstalled,
		hostreqkit.ExecutionAlreadyPresent,
		hostreqkit.ExecutionWouldInstall,
		hostreqkit.ExecutionNotApplicable:
		return true
	default:
		return false
	}
}

// Compatibility names retained for package-local predicate tests.
func hostSafeguardOK(status hostreqkit.ItemStatus) bool { return HostSafeguardOK(status) }
func hostInstallOK(status hostreqkit.ItemStatus) bool   { return HostInstallOK(status) }

// HostInstallStatusResponse maps a host-tool result to the CLI wire contract.
func HostInstallStatusResponse(status hostreqkit.ItemStatus) *cliv1.CliHostInstallStatus {
	return &cliv1.CliHostInstallStatus{
		Name: status.Name, Command: status.Command, Installed: status.Installed,
		SupportClass: string(status.SupportClass), ExecutionState: string(status.ExecutionState),
		BlockingReason: string(status.BlockingReason), Version: status.Version,
		Notes: status.Notes, Ok: HostInstallOK(status),
	}
}

func hostInstallStatusResponse(status hostreqkit.ItemStatus) *cliv1.CliHostInstallStatus {
	return HostInstallStatusResponse(status)
}

// RenderHostInstallText preserves the human host-tool output contract.
func RenderHostInstallText(w io.Writer, status hostreqkit.ItemStatus) {
	_, _ = fmt.Fprintf(w, "%s: %s\n", status.Name, status.ExecutionState)
	if status.Command != "" {
		_, _ = fmt.Fprintf(w, "  command: %s\n", status.Command)
	}
	if status.Version != "" {
		_, _ = fmt.Fprintf(w, "  version: %s\n", status.Version)
	}
	for _, note := range status.Notes {
		_, _ = fmt.Fprintf(w, "  - %s\n", note)
	}
	if len(status.Config) > 0 {
		if data, err := json.Marshal(status.Config); err == nil {
			_, _ = fmt.Fprintf(w, "  config: %s\n", data)
		}
	}
}

// RenderSafeguardList renders the list report returned by Service.Safeguard.
func RenderSafeguardList(output io.Writer, jsonOut bool) error {
	entries, err := listSafeguards()
	if err != nil {
		return err
	}
	if jsonOut {
		return cliout.WriteJSONValue(output, entries)
	}
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, []string{entry.Name, entry.Capability, entry.CapabilityRole, strings.Join(entry.Platforms, ","), entry.ObservedState})
	}
	return cliout.WriteSection(output, cliout.Section{Rows: rows})
}

// RenderInvariantCoverage renders the invariant coverage report.
func RenderInvariantCoverage(output io.Writer, jsonOut bool) error {
	report, err := hostruntime.SafeguardInvariantCoverage()
	if err != nil {
		return err
	}
	if jsonOut {
		return cliout.WriteJSONValue(output, report)
	}
	_, err = fmt.Fprintf(output, "sites walked: %d; invariants declared: %d; invariants evaluated: %d\n", report.SitesWalked, report.InvariantsDeclared, report.InvariantsEvaluated)
	for _, gap := range report.Gaps {
		if _, writeErr := fmt.Fprintf(output, "gap: %s\n", gap); writeErr != nil {
			return writeErr
		}
	}
	return err
}

// RenderHostStateAudit renders the host-state audit report.
func RenderHostStateAudit(output io.Writer, jsonOut bool) error {
	report, err := hostruntime.SafeguardHostStateAudit()
	if err != nil {
		return err
	}
	if jsonOut {
		return cliout.WriteJSONValue(output, report)
	}
	rows := make([][]string, 0, len(report.Findings))
	for _, finding := range report.Findings {
		rows = append(rows, []string{string(finding.Kind), finding.Path, finding.Reason})
	}
	return cliout.WriteSection(output, cliout.Section{Rows: rows})
}

// RenderPortabilityBacklog renders the portability backlog report.
func RenderPortabilityBacklog(output io.Writer, jsonOut bool) error {
	report, err := hostruntime.SafeguardPortabilityBacklog()
	if err != nil {
		return err
	}
	if jsonOut {
		return cliout.WriteJSONValue(output, report)
	}
	_, err = fmt.Fprintf(output, "not_implemented safeguards: %d/%d\n", report.NotImplemented, report.Total)
	for _, entry := range report.Entries {
		if _, writeErr := fmt.Fprintf(output, "- %s\n", entry); writeErr != nil {
			return writeErr
		}
	}
	return err
}
