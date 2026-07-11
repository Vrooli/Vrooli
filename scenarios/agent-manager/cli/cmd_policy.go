package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

func (a *App) cmdPolicy(args []string) error {
	if len(args) == 0 {
		return a.policyHelp()
	}
	switch args[0] {
	case "status":
		return a.policyStatus(args[1:])
	case "catalog":
		return a.policyCatalog(args[1:])
	case "validate":
		return a.policyValidate(args[1:])
	case "reload":
		return a.policyReload(args[1:])
	case "explain":
		return a.policyExplain(args[1:])
	case "help", "-h", "--help":
		return a.policyHelp()
	default:
		return fmt.Errorf("unknown role-policy subcommand: %s\n\nRun 'agent-manager role-policy help' for usage", args[0])
	}
}

func (a *App) policyHelp() error {
	fmt.Println(`Usage: agent-manager role-policy <subcommand> [options]

Subcommands:
  status                 Show active revision, path, readiness, and latest diagnostic
  catalog                Inspect the active portable role catalog
  validate               Validate configured declared state without activation
  reload                 Validate and atomically activate configured declared state
  explain profile <id>   Resolve a profile against the active catalog
  explain run <id>       Show the immutable snapshot stored with a run

Options:
  --json                 Output the generated API response as JSON`)
	return nil
}

func parsePolicyFlags(name string, args []string) (*bool, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return nil, err
	}
	return jsonOutput, nil
}

func (a *App) policyStatus(args []string) error {
	jsonOutput, err := parsePolicyFlags("policy status", args)
	if err != nil {
		return err
	}
	body, response, err := a.services.Policy.Status()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	printPolicyStatus(response.Status)
	return nil
}

func (a *App) policyCatalog(args []string) error {
	jsonOutput, err := parsePolicyFlags("policy catalog", args)
	if err != nil {
		return err
	}
	body, response, err := a.services.Policy.Catalog()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	printPolicyStatus(response.Status)
	if response.Catalog == nil {
		fmt.Println("Catalog: no validated revision is active")
		return nil
	}
	fmt.Printf("Catalog: %s (schema %d, default role %s)\n", response.Catalog.Metadata.GetCatalogId(), response.Catalog.SchemaVersion, response.Catalog.DefaultRole)
	for _, role := range response.Catalog.Roles {
		fmt.Printf("  Role %s (%s): %s\n", role.RoleRef, role.Intent, role.Description)
		for index, candidate := range role.Candidates {
			fmt.Printf("    %d. %s / %s\n", index, policyRunnerLabel(candidate.RunnerType.String()), candidate.ResourceRole)
		}
	}
	return nil
}

func (a *App) policyValidate(args []string) error {
	jsonOutput, err := parsePolicyFlags("policy validate", args)
	if err != nil {
		return err
	}
	body, response, err := a.services.Policy.Validate()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else if response.Valid {
		fmt.Printf("Status: valid\nCandidate revision: %s\nActive revision: %s\n", response.CandidateDigest, response.ActiveDigest)
	} else {
		printPolicyDiagnostic(response.Diagnostic)
	}
	if !response.Valid {
		return fmt.Errorf("role policy catalog is invalid")
	}
	return nil
}

func (a *App) policyReload(args []string) error {
	jsonOutput, err := parsePolicyFlags("policy reload", args)
	if err != nil {
		return err
	}
	body, response, err := a.services.Policy.Reload()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		printPolicyStatus(response.Status)
		if response.Diagnostic != nil {
			printPolicyDiagnostic(response.Diagnostic)
		}
	}
	if !response.Activated {
		return fmt.Errorf("role policy catalog was not activated; the previous revision remains active")
	}
	return nil
}

func (a *App) policyExplain(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: agent-manager role-policy explain <profile|run> <id> [--json]")
	}
	targetType, targetID := strings.ToLower(args[0]), args[1]
	jsonOutput, err := parsePolicyFlags("policy explain", args[2:])
	if err != nil {
		return err
	}
	request := &apipb.ExplainRolePolicyRequest{}
	switch targetType {
	case "profile":
		request.Target = &apipb.ExplainRolePolicyRequest_ProfileId{ProfileId: targetID}
	case "run":
		request.Target = &apipb.ExplainRolePolicyRequest_RunId{RunId: targetID}
	default:
		return fmt.Errorf("explain target must be profile or run")
	}
	body, response, err := a.services.Policy.Explain(request)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Target: %s %s\n", response.TargetType, response.TargetId)
	fmt.Printf("Summary: %s\n", response.Summary)
	if response.HistoricalWithoutSnapshot || response.Snapshot == nil {
		fmt.Println("Snapshot: unavailable")
		return nil
	}
	fmt.Printf("Catalog revision: %s\nRole: %s\nSelected index: %d\n", response.Snapshot.CatalogDigest, response.Snapshot.RoleRef, response.Snapshot.SelectedIndex)
	for index, candidate := range response.Snapshot.Candidates {
		fmt.Printf("  %d. %s / %s", index, policyRunnerLabel(candidate.RunnerType.String()), policySelectionLabel(candidate.SelectionType.String()))
		if candidate.Model != "" {
			fmt.Printf(" / %s", candidate.Model)
		}
		if int32(index) == response.Snapshot.SelectedIndex {
			fmt.Print(" [selected]")
		}
		fmt.Println()
	}
	return nil
}

func printPolicyStatus(status *apipb.RolePolicyStatus) {
	if status == nil {
		fmt.Println("Status: unavailable")
		return
	}
	fmt.Printf("Status: %s\nPath: %s\nActive revision: %s\n", map[bool]string{true: "ready", false: "not ready"}[status.Ready], status.Path, status.ActiveDigest)
	if status.Requirement != nil && status.Requirement.Required {
		fmt.Printf("Required: yes (%s)\n", status.Requirement.Reason)
	} else {
		fmt.Println("Required: no")
	}
	if attempt := status.LastReloadAttempt; attempt != nil && attempt.Diagnostic != nil {
		printPolicyDiagnostic(attempt.Diagnostic)
	}
}

func printPolicyDiagnostic(diagnostic *apipb.RolePolicyDiagnostic) {
	if diagnostic == nil {
		return
	}
	fmt.Printf("Diagnostic: %s: %s\n", diagnostic.Code, diagnostic.Message)
	if diagnostic.Cause != "" && diagnostic.Cause != diagnostic.Message {
		fmt.Printf("Cause: %s\n", diagnostic.Cause)
	}
}

func policyRunnerLabel(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(value, "RUNNER_TYPE_"), "_", "-"))
}

func policySelectionLabel(value string) string {
	return strings.ToLower(strings.TrimPrefix(value, "MODEL_SELECTION_TYPE_"))
}
