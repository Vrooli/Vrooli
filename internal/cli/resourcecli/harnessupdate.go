package resourcecli

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/packages/harness-lifecycle"
)

// CheckAction names the read-only observation in the update result.
const CheckAction = "check"

// RefreshCatalogAction rewrites the harness's model policy from its own evidence.
const RefreshCatalogAction = "refresh_catalog"

// UpdateRequest selects one harness and what to do to it.
type UpdateRequest struct {
	Name          string
	Action        string
	OnBehalfOf    string
	CleanupPlanID string
}

// ParseUpdateRequest maps the operator-facing flags to one action.
func ParseUpdateRequest(args []string) (UpdateRequest, error) {
	const command = "resource update"
	parsed, err := commandtree.ParseArgs(command, CommandHelpText(CommandUpdate), commandSpec(CommandUpdate).Args, args)
	if err != nil {
		return UpdateRequest{}, err
	}
	if len(parsed.Positionals) != 1 {
		return UpdateRequest{}, fmt.Errorf("%s requires exactly one coding-agent resource name", command)
	}
	req := UpdateRequest{Name: parsed.Positionals[0], Action: agentinstall.ActionUpdate, OnBehalfOf: parsed.FlagValue("--on-behalf-of"), CleanupPlanID: parsed.FlagValue("--cleanup-plan")}
	if parsed.HasFlag("--check") && parsed.HasFlag("--action") {
		return UpdateRequest{}, fmt.Errorf("%s: --check and --action cannot be combined", command)
	}
	if parsed.HasFlag("--check") {
		req.Action = CheckAction
	}
	if parsed.HasFlag("--action") {
		switch value := parsed.FlagValue("--action"); value {
		case "install", "use-managed-copy", "link-managed-copy", "rollback", "refresh-catalog", "doctor", "preview-cleanup", "uninstall":
			req.Action = strings.ReplaceAll(value, "-", "_")
		default:
			return UpdateRequest{}, fmt.Errorf("%s: unknown --action %q (use-managed-copy, link-managed-copy, rollback, refresh-catalog)", command, value)
		}
	}
	return req, nil
}

// ParseUpdateCheckRequest parses the read-only verb. It exists apart from
// `update --check` because Bridge admits a relayed command by its verb's declared
// effect: a read-scoped node can run this one and not `update`.
func ParseUpdateCheckRequest(args []string) (UpdateRequest, error) {
	const command = "resource update-check"
	parsed, err := commandtree.ParseArgs(command, CommandHelpText(CommandUpdateCheck), commandSpec(CommandUpdateCheck).Args, args)
	if err != nil {
		return UpdateRequest{}, err
	}
	if len(parsed.Positionals) != 1 {
		return UpdateRequest{}, fmt.Errorf("%s requires exactly one coding-agent resource name", command)
	}
	return UpdateRequest{Name: parsed.Positionals[0], Action: CheckAction}, nil
}

// WriteHarnessUpdate renders an observation or action outcome. JSON is the
// contract Web Console reads, on a failed action as well as a successful one.
func WriteHarnessUpdate(w io.Writer, format cliout.Format, result agentinstall.HarnessActionResult) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error {
		return cliout.WriteFieldsWithSuccess(w, result.Success, map[string]any{"harness_update": result})
	}, func(w io.Writer) error {
		if result.Action != CheckAction {
			fmt.Fprintln(w, result.Message)
		}
		report := result.Report
		fmt.Fprintf(w, "%-14s state=%-13s running=%-10s latest=%s\n", report.Name, report.State, orUnknown(report.Installed), orUnknown(report.Latest))
		fmt.Fprintf(w, "  %s\n", report.Reason)
		if report.OperatorCommand != "" {
			fmt.Fprintf(w, "  run: %s\n", report.OperatorCommand)
		}
		if report.CommandPath != "" {
			fmt.Fprintf(w, "  runs: %s\n", report.CommandPath)
		}
		if report.ManagedVersion != "" {
			fmt.Fprintf(w, "  managed: %s at %s\n", report.ManagedVersion, report.ManagedPath)
		}
		if report.RollbackVersion != "" {
			fmt.Fprintf(w, "  rollback: %s\n", report.RollbackVersion)
		}
		for _, check := range report.Checks {
			mark := "ok  "
			if !check.Passed {
				mark = "FAIL"
			}
			fmt.Fprintf(w, "  %s %-16s %s\n", mark, check.ID, check.Message)
		}
		return nil
	})
}
