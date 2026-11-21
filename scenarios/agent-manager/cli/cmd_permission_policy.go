package main

import (
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

func (a *App) cmdPermissionPolicy(args []string) error {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "status":
		return a.permissionPolicyStatus(args[1:])
	case "catalog":
		return a.permissionPolicyCatalog(args[1:])
	case "validate":
		return a.permissionPolicyValidate(args[1:])
	case "reload":
		return a.permissionPolicyReload(args[1:])
	case "plan":
		return a.permissionPolicyPlan(args[1:])
	case "reconcile":
		return a.permissionPolicyReconcile(args[1:])
	case "doctor":
		return a.permissionPolicyDoctor(args[1:])
	case "help", "-h", "--help":
		return nil
	default:
		return fmt.Errorf("unknown permission-policy subcommand: %s\n\nRun 'agent-manager permission-policy help' for usage", args[0])
	}
}

func parsePermissionPolicyFlags(name string, args []string, reconciliation bool) (*bool, *bool, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	authorized := fs.Bool("i-was-explicitly-authorized", false, "Required for mutation after explicit human authorization")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return nil, nil, err
	}
	if reconciliation && !*authorized {
		return nil, nil, fmt.Errorf("reconcile requires %s", agentcatalog.OverrideFlag)
	}
	return jsonOutput, authorized, nil
}

func (a *App) permissionPolicyStatus(args []string) error {
	jsonOutput, _, err := parsePermissionPolicyFlags("permission-policy status", args, false)
	if err != nil {
		return err
	}
	body, response, err := a.services.PermissionPolicy.Status()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	printPermissionPolicyStatus(response.Status)
	if response.LastReconcile != nil {
		fmt.Printf("Last reconcile: %s (success: %t)\n", formatTimestamp(response.LastReconcile.FinishedAt), response.LastReconcile.Success)
	}
	return nil
}

func (a *App) permissionPolicyCatalog(args []string) error {
	jsonOutput, _, err := parsePermissionPolicyFlags("permission-policy catalog", args, false)
	if err != nil {
		return err
	}
	body, response, err := a.services.PermissionPolicy.Catalog()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	printPermissionPolicyStatus(response.Status)
	if response.Catalog == nil {
		fmt.Println("Catalog: no validated revision is active")
		return nil
	}
	fmt.Printf("Catalog: %s (schema %d)\n", response.Catalog.Metadata.GetCatalogId(), response.Catalog.SchemaVersion)
	for _, rule := range response.Catalog.Rules {
		fmt.Printf("  %s: %s %s %q (scope %s, hard enforcement: %t)\n", rule.Id, rule.Action, rule.Matcher.GetKind(), rule.Matcher.GetPattern(), rule.TargetScope, rule.RequiresHardEnforcement)
	}
	return nil
}

func (a *App) permissionPolicyValidate(args []string) error {
	jsonOutput, _, err := parsePermissionPolicyFlags("permission-policy validate", args, false)
	if err != nil {
		return err
	}
	body, response, err := a.services.PermissionPolicy.Validate()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else if response.Valid {
		fmt.Printf("Status: valid\nCandidate revision: %s\nActive revision: %s\n", response.CandidateDigest, response.ActiveDigest)
	} else {
		printPermissionPolicyDiagnostic(response.Diagnostic)
	}
	if !response.Valid {
		return fmt.Errorf("permission policy catalog is invalid")
	}
	return nil
}

func (a *App) permissionPolicyReload(args []string) error {
	jsonOutput, _, err := parsePermissionPolicyFlags("permission-policy reload", args, false)
	if err != nil {
		return err
	}
	body, response, err := a.services.PermissionPolicy.Reload()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		printPermissionPolicyStatus(response.Status)
		printPermissionPolicyDiagnostic(response.Diagnostic)
	}
	if !response.Activated {
		return fmt.Errorf("permission policy catalog was not activated; the previous revision remains active")
	}
	return nil
}

func (a *App) permissionPolicyPlan(args []string) error {
	jsonOutput, _, err := parsePermissionPolicyFlags("permission-policy plan", args, false)
	if err != nil {
		return err
	}
	body, response, err := a.services.PermissionPolicy.Plan()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	printPermissionPolicyPlan(response.Plan)
	return nil
}

func (a *App) permissionPolicyReconcile(args []string) error {
	jsonOutput, authorized, err := parsePermissionPolicyFlags("permission-policy reconcile", args, true)
	if err != nil {
		return err
	}
	body, response, err := a.services.PermissionPolicy.Reconcile(&apipb.ReconcilePermissionPolicyRequest{ExplicitlyAuthorized: *authorized})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	printPermissionPolicyReconcile(response.Result)
	if response.Result == nil || !response.Result.Success {
		return fmt.Errorf("permission policy reconcile did not complete globally; inspect per-resource outcomes")
	}
	return nil
}

func (a *App) permissionPolicyDoctor(args []string) error {
	jsonOutput, _, err := parsePermissionPolicyFlags("permission-policy doctor", args, false)
	if err != nil {
		return err
	}
	body, response, err := a.services.PermissionPolicy.Doctor()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Status: %s\n", response.Summary)
	printPermissionPolicyStatus(response.Status)
	printPermissionPolicyPlan(response.Plan)
	if !response.Healthy {
		return fmt.Errorf("permission policy doctor found a readiness or enforcement gap")
	}
	return nil
}

func printPermissionPolicyStatus(status *apipb.PermissionPolicyStatus) {
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
		printPermissionPolicyDiagnostic(attempt.Diagnostic)
	}
}

func printPermissionPolicyDiagnostic(diagnostic *apipb.PermissionPolicyDiagnostic) {
	if diagnostic == nil {
		return
	}
	fmt.Printf("Diagnostic: %s: %s\n", diagnostic.Code, diagnostic.Message)
}

func printPermissionPolicyPlan(plan *apipb.PermissionPolicyPlan) {
	if plan == nil {
		fmt.Println("Plan: unavailable")
		return
	}
	fmt.Printf("Plan revision: %s\nHard enforcement satisfied: %t\n", plan.CatalogDigest, plan.HardEnforcementSatisfied)
	if len(plan.MissingHardEnforcementRuleIds) > 0 {
		fmt.Printf("Missing hard enforcement: %v\n", plan.MissingHardEnforcementRuleIds)
	}
	for _, resource := range plan.Resources {
		fmt.Printf("  %s / %s: %s", policyRunnerLabel(resource.RunnerType.String()), resource.Scope, resource.Status)
		if resource.Drift {
			fmt.Print(" (drift)")
		}
		if resource.Error != "" {
			fmt.Printf(" — %s", resource.Error)
		}
		fmt.Println()
	}
}

func printPermissionPolicyReconcile(result *apipb.PermissionPolicyReconcileResult) {
	if result == nil {
		fmt.Println("Reconcile: unavailable")
		return
	}
	plan := &apipb.PermissionPolicyPlan{
		CatalogDigest:                 result.CatalogDigest,
		Resources:                     result.Resources,
		HardEnforcementSatisfied:      result.HardEnforcementSatisfied,
		MissingHardEnforcementRuleIds: result.MissingHardEnforcementRuleIds,
	}
	fmt.Printf("Reconcile success: %t\n", result.Success)
	printPermissionPolicyPlan(plan)
}
