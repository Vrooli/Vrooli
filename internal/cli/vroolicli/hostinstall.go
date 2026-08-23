package vroolicli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	hostruntime "github.com/vrooli/vrooli/internal/runtime"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func hostInstallSpec() commandtree.Spec[string] {
	return commandtree.Spec[string]{
		Name:    "install",
		Summary: "Install a single host tool (url/release fetch into ~/.vrooli/bin, no sudo)",
		Help: commandtree.Help{
			Description: "Ensures one host tool by name through the runtime tool handler: data-declared url/release binaries are fetched + checksum-verified into ~/.vrooli/bin; package/manual tools report how to install them. Honors the hardware-capability gate (a tool whose requirements are unmet on this host is cleanly skipped, not failed).",
			Usage:       "vrooli host install <tool> [--json] [--dry-run] [--sudo-mode ask|skip|error]",
			Options: []commandtree.OptionArg{
				commandtree.JSONOption(),
				{Name: "--dry-run", Description: "Report what would be fetched without downloading"},
				{Name: "--sudo-mode", ValueName: "mode", Description: "Privilege policy for package installs: ask, skip, or error (default: skip)"},
			},
			Examples: []string{
				"vrooli host install realesrgan-ncnn-vulkan",
				"vrooli host install realesrgan-ncnn-vulkan --json",
				"vrooli host install sd --dry-run",
			},
		},
		Args: commandtree.ArgSchema{
			Positionals: []commandtree.PositionalArg{
				{Name: "tool", Required: true, Description: "Host tool name (see internal/tools/<name>)"},
			},
			Options: []commandtree.OptionArg{
				commandtree.JSONOption(),
				{Name: "--dry-run", Description: "Report what would be fetched without downloading"},
				{Name: "--sudo-mode", ValueName: "mode", Description: "Privilege policy for package installs: ask, skip, or error (default: skip)"},
			},
		},
		Handler: "install",
	}
}

func hostSafeguardSpec() commandtree.Spec[string] {
	return commandtree.Spec[string]{
		Name:    "safeguard",
		Summary: "Inspect or apply one declared host safeguard",
		Help: commandtree.Help{
			Description: "Applies one typed host safeguard through Vrooli's requirement runtime. Use this for focused, auditable repairs such as a kernel driver; high-risk safeguards can report a typed reboot-required result instead of pretending the host is ready.",
			Usage:       "vrooli host safeguard <name|list> [--json] [--dry-run] [--maintenance-window] [--sudo-mode ask|skip|error]",
			Options:     []commandtree.OptionArg{{Name: "--dry-run", Description: "Report the managed change without applying it"}, {Name: "--maintenance-window", Description: "Acknowledge graphical/remote-session interruption risk"}, {Name: "--sudo-mode", ValueName: "mode", Description: "Privilege policy: ask, skip, or error (default: skip)"}},
			Examples:    []string{"vrooli host safeguard list --json", "vrooli host safeguard nvidia_driver --dry-run", "vrooli host safeguard nvidia-driver --maintenance-window --sudo-mode ask"},
		},
		Args:    commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "name", Required: true, Description: "Safeguard name in hyphenated or underscored form, or list"}}, Options: []commandtree.OptionArg{commandtree.JSONOption(), {Name: "--dry-run", Description: "Report the managed change without applying it"}, {Name: "--maintenance-window", Description: "Acknowledge graphical/remote-session interruption risk"}, {Name: "--sudo-mode", ValueName: "mode", Description: "Privilege policy: ask, skip, or error"}}},
		Handler: "safeguard",
	}
}

func (app *App) runHostSafeguardCommand(ctx *CommandContext, args []string) error {
	spec := hostSafeguardSpec()
	parsed, err := commandtree.ParseArgs("host safeguard", commandtree.SpecHelpText("", "vrooli host safeguard", spec), spec.Args, args)
	if err != nil {
		if rootcli.HandleHelp(ctx.Stdout, err) {
			return nil
		}
		return rootcli.UsageErrorf("host safeguard", "%s", err.Error())
	}
	name := strings.TrimSpace(parsed.Positionals[0])
	jsonOut := ctx.Globals.JSON || parsed.HasFlag("--json")
	if strings.EqualFold(name, "list") {
		return renderSafeguardList(ctx.Stdout, jsonOut)
	}
	name = strings.ReplaceAll(strings.ToLower(name), "-", "_")
	sudoMode := strings.ToLower(strings.TrimSpace(parsed.FlagValue("--sudo-mode")))
	if sudoMode != "" && sudoMode != "ask" && sudoMode != "skip" && sudoMode != "error" {
		return rootcli.UsageErrorf("host safeguard", "invalid --sudo-mode %q (want ask, skip, or error)", sudoMode)
	}
	status, err := hostruntime.EnsureSafeguard(name, hostruntime.EnsureOptions{AutoInstall: true, DryRun: parsed.HasFlag("--dry-run"), MaintenanceWindow: parsed.HasFlag("--maintenance-window"), SudoMode: sudoMode, Stdout: ctx.Stdout, Stderr: ctx.Stderr})
	if err != nil {
		return fmt.Errorf("host safeguard %q: %w", name, err)
	}
	if jsonOut {
		if err := json.NewEncoder(ctx.Stdout).Encode(status); err != nil {
			return err
		}
	} else {
		renderHostInstallText(ctx.Stdout, status)
	}
	if !hostSafeguardOK(status) {
		return rootcli.ExitCodeError{Code: 1, Silent_: true}
	}
	return nil
}

type safeguardListEntry struct {
	Name           string   `json:"name"`
	Capability     string   `json:"capability"`
	CapabilityRole string   `json:"capability_role"`
	Platforms      []string `json:"platforms"`
	ObservedState  string   `json:"observed_state"`
	SupportClass   string   `json:"support_class"`
	ObservedAt     string   `json:"observed_at"`
	ObservedNotes  []string `json:"observed_notes,omitempty"`
}

func renderSafeguardList(output io.Writer, jsonOut bool) error {
	entries, err := listSafeguards()
	if err != nil {
		return err
	}
	if jsonOut {
		return json.NewEncoder(output).Encode(entries)
	}
	for _, entry := range entries {
		fmt.Fprintf(output, "%s\t%s\t%s\t%s\t%s\n", entry.Name, entry.Capability, entry.CapabilityRole, strings.Join(entry.Platforms, ","), entry.ObservedState)
	}
	return nil
}

func listSafeguards() ([]safeguardListEntry, error) {
	observed, err := hostruntime.ListObservedSafeguardsAt(".", nil)
	if err != nil {
		return nil, err
	}
	entries := make([]safeguardListEntry, 0, len(observed))
	for _, item := range observed {
		entries = append(entries, safeguardListEntry{
			Name: item.Name, Capability: item.Capability, CapabilityRole: item.CapabilityRole,
			Platforms: append([]string(nil), item.Platforms...), ObservedState: string(item.ExecutionState),
			SupportClass: string(item.SupportClass), ObservedAt: item.ObservedAt.Format(time.RFC3339Nano),
			ObservedNotes: append([]string(nil), item.Notes...),
		})
	}
	return entries, nil
}

// hostSafeguardOK reports whether a safeguard run should exit zero. It is the
// safeguard twin of hostInstallOK; the two are kept adjacent because they
// previously drifted. The inline predicate this replaced omitted
// ExecutionApplied, so a safeguard that successfully did its work exited 1
// while the same command with --dry-run exited 0 — an inversion that reads as
// a failure exactly when the repair succeeded.
//
// ExecutionRebootRequired is deliberately NOT success: the safeguard's whole
// point in that case is to refuse to pretend the host is ready. Likewise
// ExecutionManualActionRequired, where the operator still has work to do.
func hostSafeguardOK(status hostreqkit.ItemStatus) bool {
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

func (app *App) runHostInstallCommand(ctx *CommandContext, args []string) error {
	spec := hostInstallSpec()
	parsed, err := commandtree.ParseArgs("host install", commandtree.SpecHelpText("", "vrooli host install", spec), spec.Args, args)
	if err != nil {
		if rootcli.HandleHelp(ctx.Stdout, err) {
			return nil
		}
		return rootcli.UsageErrorf("host install", "%s", err.Error())
	}
	if len(parsed.Positionals) == 0 {
		return rootcli.UsageErrorf("host install", "a tool name is required")
	}
	tool := strings.TrimSpace(parsed.Positionals[0])
	jsonOut := ctx.Globals.JSON || parsed.HasFlag("--json")
	dryRun := parsed.HasFlag("--dry-run")
	sudoMode := strings.ToLower(strings.TrimSpace(parsed.FlagValue("--sudo-mode")))
	if sudoMode != "" && sudoMode != "ask" && sudoMode != "skip" && sudoMode != "error" {
		return rootcli.UsageErrorf("host install", "invalid --sudo-mode %q (want ask, skip, or error)", sudoMode)
	}

	opts := hostruntime.EnsureOptions{
		AutoInstall:     true,
		IncludeOptional: true,
		DryRun:          dryRun,
		SudoMode:        sudoMode,
	}
	// Stream human progress to stderr so --json keeps stdout a clean contract.
	if !jsonOut {
		opts.Stdout = ctx.Stdout
		opts.Stderr = ctx.Stderr
	}

	status, err := hostruntime.EnsureTool(tool, opts)
	if err != nil {
		return fmt.Errorf("host install %q: %w", tool, err)
	}

	if jsonOut {
		if err := writeHostInstallJSON(ctx.Stdout, status); err != nil {
			return err
		}
	} else {
		renderHostInstallText(ctx.Stdout, status)
	}

	if !hostInstallOK(status) {
		return rootcli.ExitCodeError{Code: 1, Silent_: true}
	}
	return nil
}

// hostInstallOK reports whether the operation reached a healthy terminal state.
func hostInstallOK(status hostreqkit.ItemStatus) bool {
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

func renderHostInstallText(w io.Writer, status hostreqkit.ItemStatus) {
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

// hostInstallStatusResponse maps the runtime ItemStatus onto the vrooli.cli.v1
// wire contract. A proto field rename breaks this mapping at compile time.
func hostInstallStatusResponse(status hostreqkit.ItemStatus) *cliv1.CliHostInstallStatus {
	return &cliv1.CliHostInstallStatus{
		Name:           status.Name,
		Command:        status.Command,
		Installed:      status.Installed,
		SupportClass:   string(status.SupportClass),
		ExecutionState: string(status.ExecutionState),
		BlockingReason: string(status.BlockingReason),
		Version:        status.Version,
		Notes:          status.Notes,
		Ok:             hostInstallOK(status),
	}
}

func writeHostInstallJSON(w io.Writer, status hostreqkit.ItemStatus) error {
	return cliout.WriteProtoJSON(w, hostInstallStatusResponse(status))
}
