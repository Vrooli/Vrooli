package vroolicli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/cliout"
)

const uninstallHelpText = `vrooli uninstall - Remove only recorded Vrooli installation artifacts

Usage:
  vrooli uninstall --plan --scope agent|runtime|all --confirm-target <hostname>
	  vrooli uninstall --apply <plan-id> --confirm-target <hostname> --break-glass-token-stdin < token

Safety:
  Planning freezes the recorded inventory and disk fingerprints. Applying a plan
  never discovers new paths. The target hostname and a signed, time-boxed
  break-glass credential are required for apply.
`

func uninstallArgSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{Options: []commandtree.OptionArg{
		{Name: "--plan", Description: "Freeze a removal inventory without removing anything"},
		{Name: "--apply", ValueName: "plan-id", Description: "Apply one previously frozen plan"},
		{Name: "--verify", ValueName: "plan-id", Description: "Read-only verification of a frozen plan"},
		{Name: "--plan-id", ValueName: "plan-id", Description: "Use a caller-supplied safe plan id (Bridge helper only)"},
		{Name: "--scope", ValueName: "scope", Description: "Removal scope: agent, runtime, or all"},
		{Name: "--confirm-target", ValueName: "hostname", Description: "Confirm the live hostname"},
		{Name: "--break-glass-token", ValueName: "token", Description: "Signed destructive-operation credential (apply only)"},
		{Name: "--break-glass-token-stdin", Description: "Read the signed destructive credential from stdin (apply only)"},
		{Name: "--machine-id", ValueName: "id", Description: "Bridge cleanup machine binding"},
		{Name: "--node-id", ValueName: "id", Description: "Bridge cleanup node binding"},
		{Name: "--operation-id", ValueName: "id", Description: "Bridge cleanup operation binding"},
		{Name: "--plan-hash", ValueName: "hash", Description: "Bridge cleanup plan binding"},
		{Name: "--operator-id", ValueName: "id", Description: "Bridge cleanup operator binding"},
	}}
}

//nolint:gocyclo // uninstall coordinates confirmation, plan execution, reporting, and cleanup failure policies.
func (app *App) runUninstallCommand(ctx *CommandContext, args []string) error {
	parsed, err := commandtree.ParseArgs("uninstall", uninstallHelpText, uninstallArgSchema(), args)
	if err != nil {
		if rootcli.HandleHelp(ctx.Stdout, err) {
			return nil
		}
		return rootcli.UsageErrorf("uninstall", "%s", err.Error())
	}
	request := cliinstall.UninstallRequest{
		Scope:           cliinstall.InstallScope(strings.TrimSpace(parsed.FlagValue("--scope"))),
		PlanID:          parsed.FlagValue("--plan-id"),
		ConfirmTarget:   strings.TrimSpace(parsed.FlagValue("--confirm-target")),
		BreakGlass:      parsed.FlagValue("--break-glass-token"),
		MachineID:       parsed.FlagValue("--machine-id"),
		NodeID:          parsed.FlagValue("--node-id"),
		OperationID:     parsed.FlagValue("--operation-id"),
		PlanHash:        parsed.FlagValue("--plan-hash"),
		AuthorizingUser: parsed.FlagValue("--operator-id"),
	}
	if parsed.HasFlag("--break-glass-token-stdin") {
		if strings.TrimSpace(request.BreakGlass) != "" {
			return rootcli.UsageErrorf("uninstall", "choose exactly one of --break-glass-token or --break-glass-token-stdin")
		}
		input := ctx.Stdin
		if input == nil {
			input = os.Stdin
		}
		var secret bytes.Buffer
		if _, err := io.CopyN(&secret, input, 64*1024+1); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("read break-glass token from stdin: %w", err)
		}
		if secret.Len() > 64*1024 {
			return fmt.Errorf("break-glass token from stdin exceeds 64 KiB")
		}
		request.BreakGlass = strings.TrimSpace(secret.String())
		secret.Reset()
		if request.BreakGlass == "" {
			return fmt.Errorf("break-glass token from stdin is required")
		}
	}
	if request.Scope == "" {
		request.Scope = cliinstall.ScopeAll
	}
	planFlag := parsed.HasFlag("--plan")
	applyID := strings.TrimSpace(parsed.FlagValue("--apply"))
	verifyID := strings.TrimSpace(parsed.FlagValue("--verify"))
	selected := 0
	if planFlag {
		selected++
	}
	if applyID != "" {
		selected++
	}
	if verifyID != "" {
		selected++
	}
	if selected != 1 {
		return rootcli.UsageErrorf("uninstall", "choose exactly one of --plan, --apply <plan-id>, or --verify <plan-id>")
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
		if verifyID != "" {
			verifier, ok := uninstaller.(interface {
				Verify(cliinstall.UninstallRequest) (cliinstall.UninstallVerification, error)
			})
			if !ok {
				return fmt.Errorf("uninstall: verification is not configured")
			}
			request.Mode = cliinstall.UninstallVerifyMode
			request.PlanID = verifyID
			verification, verifyErr := verifier.Verify(request)
			if verifyErr != nil {
				return verifyErr
			}
			output = verification
		} else {
			request.Mode = cliinstall.UninstallApplyMode
			request.PlanID = applyID
			receipt, applyErr := uninstaller.Apply(request)
			if applyErr != nil {
				return applyErr
			}
			output = receipt
		}
	}
	if ctx.Globals.JSON {
		return cliout.WriteJSONValue(ctx.Stdout, output)
	}
	return writeUninstallHuman(ctx.Stdout, output)
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
	case cliinstall.UninstallVerification:
		_, _ = fmt.Fprintf(w, "Verified plan %s (target=%s scope=%s complete=%t)\n", result.PlanID, result.Target, result.Scope, result.Complete)
		writeVerificationSection(w, "Remaining", result.Remaining)
		writeVerificationSection(w, "Not checked", result.NotChecked)
	default:
		return fmt.Errorf("uninstall: unsupported output %T", value)
	}
	return nil
}

func writeVerificationSection(w io.Writer, title string, entries []cliinstall.RemovalReceiptEntry) {
	_, _ = fmt.Fprintf(w, "%s:\n", title)
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(w, "- none")
		return
	}
	for _, entry := range entries {
		_, _ = fmt.Fprintf(w, "- [%s] %s %s\n", entry.Scope, entry.Kind, entry.Path)
	}
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
