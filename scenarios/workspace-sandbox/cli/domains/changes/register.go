package changes

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"workspace-sandbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "change",
		Description: "Review, approve, reject, and rebase sandbox changes",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "diff", Description: "Show sandbox changes", Run: func(args []string) error { return runDiff(deps, args) }},
			{Name: "approve", Description: "Apply sandbox changes", Run: func(args []string) error { return runApprove(deps, args) }},
			{Name: "reject", Description: "Discard sandbox changes", Run: func(args []string) error { return runReject(deps, args) }},
			{Name: "conflicts", Description: "Check for repo conflicts", Run: func(args []string) error { return runConflicts(deps, args) }},
			{Name: "rebase", Description: "Refresh a sandbox against the current repo", Run: func(args []string) error { return runRebase(deps, args) }},
		},
	}
}

func runDiff(deps support.Dependencies, args []string) error {
	var sandboxID string
	var raw, jsonOut bool
	for _, arg := range args {
		switch {
		case arg == "--raw":
			raw = true
		case arg == "--json":
			jsonOut = true
		case !strings.HasPrefix(arg, "-") && sandboxID == "":
			sandboxID = arg
		}
	}
	if sandboxID == "" {
		return fmt.Errorf("usage: %s change diff <sandbox-id> [--raw] [--json]", support.CLIName)
	}

	resolvedID, err := support.ResolveSandboxID(deps.ScenarioApp(), sandboxID)
	if err != nil {
		return err
	}

	body, err := deps.ScenarioApp().Get("/sandboxes/"+resolvedID+"/diff", nil)
	if err != nil {
		return err
	}

	var diff support.DiffResponse
	if err := json.Unmarshal(body, &diff); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if raw {
		fmt.Println(diff.UnifiedDiff)
		return nil
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Sandbox ID: " + diff.SandboxID,
			fmt.Sprintf("Added: %d", diff.TotalAdded),
			fmt.Sprintf("Modified: %d", diff.TotalModified),
			fmt.Sprintf("Deleted: %d", diff.TotalDeleted),
		},
		Results:        renderDiffRows(diff.Files),
		RetrievalHints: []string{support.CLIName + " change approve " + diff.SandboxID, support.CLIName + " change reject " + diff.SandboxID},
	}
	if jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	if err := cliapp.RenderListReport(os.Stdout, report); err != nil {
		return err
	}
	if diff.UnifiedDiff != "" {
		fmt.Println()
		fmt.Println("Unified Diff")
		fmt.Println(diff.UnifiedDiff)
	}
	return nil
}

func runApprove(deps support.Dependencies, args []string) error {
	var sandboxID, message string
	var force, createCommit, overrideAcceptance, jsonOut bool

	for i, arg := range args {
		switch {
		case arg == "-m" && i+1 < len(args):
			message = args[i+1]
		case strings.HasPrefix(arg, "-m="):
			message = strings.TrimPrefix(arg, "-m=")
		case strings.HasPrefix(arg, "--message="):
			message = strings.TrimPrefix(arg, "--message=")
		case arg == "--force" || arg == "-f":
			force = true
		case arg == "--commit" || arg == "-c":
			createCommit = true
		case arg == "--override-acceptance":
			overrideAcceptance = true
		case arg == "--json":
			jsonOut = true
		case !strings.HasPrefix(arg, "-") && (i == 0 || args[i-1] != "-m"):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	if sandboxID == "" {
		return fmt.Errorf("usage: %s change approve <sandbox-id> [-m MESSAGE] [--commit] [--force] [--override-acceptance] [--json]", support.CLIName)
	}

	resolvedID, err := support.ResolveSandboxID(deps.ScenarioApp(), sandboxID)
	if err != nil {
		return err
	}

	reqBody := map[string]any{"mode": "all"}
	if message != "" {
		reqBody["commitMessage"] = message
	}
	if force {
		reqBody["force"] = true
	}
	if createCommit {
		reqBody["createCommit"] = true
	}
	if overrideAcceptance {
		reqBody["overrideAcceptance"] = true
	}

	body, err := deps.ScenarioApp().Request("POST", "/sandboxes/"+resolvedID+"/approve", nil, reqBody)
	if err != nil {
		return err
	}

	var resp support.ApprovalResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("approval failed: %s", resp.ErrorMsg)
	}

	report := cliapp.MutationReport{
		Result: []string{
			"Sandbox changes approved",
			fmt.Sprintf("Applied changes: %d", resp.Applied),
		},
		Changes: []string{},
		NextCommand: []string{
			support.CLIName + " provenance pending",
			support.CLIName + " sandbox list",
		},
	}
	if resp.CommitHash != "" {
		report.Changes = append(report.Changes, "Commit: "+resp.CommitHash)
	} else {
		report.Changes = append(report.Changes, "Applied to working tree without creating a commit")
	}
	if force {
		report.Changes = append(report.Changes, "Force approval enabled")
	}
	if overrideAcceptance {
		report.Changes = append(report.Changes, "Acceptance rules overridden")
	}

	if jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runReject(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("change reject", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: %s change reject <sandbox-id> [--json]", support.CLIName)
	}

	sandboxID, err := support.ResolveSandboxID(deps.ScenarioApp(), fs.Arg(0))
	if err != nil {
		return err
	}

	body, err := deps.ScenarioApp().Request("POST", "/sandboxes/"+sandboxID+"/reject", nil, nil)
	if err != nil {
		return err
	}

	var sandbox support.SandboxResponse
	if err := json.Unmarshal(body, &sandbox); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result:      []string{"Sandbox changes rejected", "Sandbox ID: " + sandbox.ID},
		Changes:     []string{"Discarded unapproved changes"},
		NextCommand: []string{support.CLIName + " sandbox inspect " + sandbox.ID},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runConflicts(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("change conflicts", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: %s change conflicts <sandbox-id> [--json]", support.CLIName)
	}

	sandboxID, err := support.ResolveSandboxID(deps.ScenarioApp(), fs.Arg(0))
	if err != nil {
		return err
	}

	body, err := deps.ScenarioApp().Get("/sandboxes/"+sandboxID+"/conflicts", nil)
	if err != nil {
		return err
	}

	var resp support.ConflictCheckResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.OperationalReport{
		Status: []string{
			"Sandbox ID: " + sandboxID,
			fmt.Sprintf("Conflicts detected: %t", resp.HasConflict),
			"Base commit: " + support.TruncateHash(resp.BaseCommitHash),
			"Current commit: " + support.TruncateHash(resp.CurrentHash),
		},
		Triage: []cliapp.TriageGroup{},
		NextSteps: []string{
			support.CLIName + " change rebase " + sandboxID,
			support.CLIName + " change approve " + sandboxID + " --force",
		},
	}
	conflictingItems := append([]string{}, resp.ConflictingFiles...)
	if len(conflictingItems) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Conflicting Files", Items: conflictingItems})
	}
	repoItems := append([]string{}, resp.RepoChangedFiles...)
	if len(repoItems) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Repo Changed Files", Items: repoItems})
	}
	sandboxItems := append([]string{}, resp.SandboxChangedFiles...)
	if len(sandboxItems) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Sandbox Changed Files", Items: sandboxItems})
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runRebase(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("change rebase", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: %s change rebase <sandbox-id> [--json]", support.CLIName)
	}

	sandboxID, err := support.ResolveSandboxID(deps.ScenarioApp(), fs.Arg(0))
	if err != nil {
		return err
	}

	body, err := deps.ScenarioApp().Request("POST", "/sandboxes/"+sandboxID+"/rebase", nil, map[string]any{"strategy": "regenerate"})
	if err != nil {
		return err
	}

	var resp support.RebaseResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("rebase failed: %s", resp.ErrorMsg)
	}

	report := cliapp.MutationReport{
		Result: []string{
			"Sandbox rebased",
			"Previous base: " + support.TruncateHash(resp.PreviousBaseHash),
			"New base: " + support.TruncateHash(resp.NewBaseHash),
		},
		Changes:     []string{},
		NextCommand: []string{support.CLIName + " change diff " + sandboxID, support.CLIName + " change conflicts " + sandboxID},
	}
	for _, path := range resp.RepoChangedFiles {
		report.Changes = append(report.Changes, "Repo changed: "+path)
	}
	for _, path := range resp.ConflictingFiles {
		report.Changes = append(report.Changes, "Potential conflict: "+path)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderDiffRows(files []support.DiffFile) []string {
	if len(files) == 0 {
		return nil
	}
	rows := make([]string, 0, len(files))
	for _, file := range files {
		rows = append(rows, fmt.Sprintf("%s %s", support.ChangeTypeSymbol(file.ChangeType), file.FilePath))
	}
	return rows
}
