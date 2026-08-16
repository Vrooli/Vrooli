package vroolicli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliinstall"
)

const uninstallHelpText = `vrooli uninstall - Remove only recorded Vrooli installation artifacts

Usage:
  vrooli uninstall --plan --scope agent|runtime|all --confirm-target <hostname>
  vrooli uninstall --apply <plan-id> --confirm-target <hostname> --break-glass-token <token>

Safety:
  Planning freezes the recorded inventory and disk fingerprints. Applying a plan
  never discovers new paths. The target hostname and a signed, time-boxed
  break-glass credential are required for apply.
`

func uninstallArgSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{Options: []commandtree.OptionArg{
		{Name: "--plan", Description: "Freeze a removal inventory without removing anything"},
		{Name: "--apply", ValueName: "plan-id", Description: "Apply one previously frozen plan"},
		{Name: "--scope", ValueName: "scope", Description: "Removal scope: agent, runtime, or all"},
		{Name: "--confirm-target", ValueName: "hostname", Description: "Confirm the live hostname"},
		{Name: "--break-glass-token", ValueName: "token", Description: "Signed destructive-operation credential (apply only)"},
	}}
}

func (app *App) runUninstallCommand(ctx *CommandContext, args []string) error {
	parsed, err := commandtree.ParseArgs("uninstall", uninstallHelpText, uninstallArgSchema(), args)
	if err != nil {
		if rootcli.HandleHelp(ctx.Stdout, err) {
			return nil
		}
		return rootcli.UsageErrorf("uninstall", "%s", err.Error())
	}
	request := cliinstall.UninstallRequest{
		Scope:         cliinstall.InstallScope(strings.TrimSpace(parsed.FlagValue("--scope"))),
		ConfirmTarget: strings.TrimSpace(parsed.FlagValue("--confirm-target")),
		BreakGlass:    parsed.FlagValue("--break-glass-token"),
	}
	if request.Scope == "" {
		request.Scope = cliinstall.ScopeAll
	}
	planFlag := parsed.HasFlag("--plan")
	applyID := strings.TrimSpace(parsed.FlagValue("--apply"))
	if planFlag == (applyID != "") {
		return rootcli.UsageErrorf("uninstall", "choose exactly one of --plan or --apply <plan-id>")
	}
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	if app.NewUninstallerFn == nil {
		return fmt.Errorf("uninstall: production remover is not configured")
	}
	uninstaller, err := app.NewUninstallerFn(ctx.Root, home)
	if err != nil {
		return err
	}
	var output any
	if planFlag {
		request.Mode = cliinstall.UninstallPlanMode
		plan, planErr := uninstaller.Plan(request)
		if planErr != nil {
			return planErr
		}
		output = plan
	} else {
		request.Mode = cliinstall.UninstallApplyMode
		request.PlanID = applyID
		receipt, applyErr := uninstaller.Apply(request)
		if applyErr != nil {
			return applyErr
		}
		output = receipt
	}
	if ctx.Globals.JSON {
		return writeUninstallJSON(ctx.Stdout, output)
	}
	return writeUninstallHuman(ctx.Stdout, output)
}

func writeUninstallJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeUninstallHuman(w io.Writer, value any) error {
	switch result := value.(type) {
	case cliinstall.UninstallPlan:
		_, _ = fmt.Fprintf(w, "Uninstall plan %s (target=%s scope=%s)\n", result.ID, result.Target, result.Scope)
		writePlanSection(w, "Remove", result.RemoveOrEntries())
		writeDecisionSection(w, "Keep", result.Keep)
		writeDecisionSection(w, "Cannot attribute", result.CannotAttribute)
	case cliinstall.RemovalReceipt:
		_, _ = fmt.Fprintf(w, "Removed plan %s (target=%s scope=%s)\n", result.PlanID, result.Target, result.Scope)
		for _, entry := range result.Removed {
			_, _ = fmt.Fprintf(w, "- [%s] %s %s\n", entry.Scope, entry.Kind, entry.Path)
		}
	default:
		return fmt.Errorf("uninstall: unsupported output %T", value)
	}
	return nil
}

func writePlanSection(w io.Writer, title string, entries []cliinstall.InstallEntry) {
	_, _ = fmt.Fprintf(w, "%s:\n", title)
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(w, "- none")
		return
	}
	for _, entry := range entries {
		_, _ = fmt.Fprintf(w, "- [%s] %s %s\n", entry.Scope, entry.Kind, entry.Path)
	}
}

func writeDecisionSection(w io.Writer, title string, entries []cliinstall.UninstallDecision) {
	_, _ = fmt.Fprintf(w, "%s:\n", title)
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(w, "- none")
		return
	}
	for _, entry := range entries {
		_, _ = fmt.Fprintf(w, "- [%s] %s %s — %s\n", entry.Scope, entry.Kind, entry.Path, entry.Reason)
	}
}
