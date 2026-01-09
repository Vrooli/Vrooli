package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// RequirementsGenerateRequest matches the API request structure
type RequirementsGenerateRequest struct {
	EntityType string `json:"entity_type"`
	EntityName string `json:"entity_name"`
	Context    string `json:"context,omitempty"`
	Model      string `json:"model,omitempty"`
}

// RequirementsGenerateResponse matches the API response structure
type RequirementsGenerateResponse struct {
	EntityType       string   `json:"entity_type"`
	EntityName       string   `json:"entity_name"`
	Success          bool     `json:"success"`
	Message          string   `json:"message,omitempty"`
	RequirementCount int      `json:"requirement_count"`
	P0Count          int      `json:"p0_count"`
	P1Count          int      `json:"p1_count"`
	P2Count          int      `json:"p2_count"`
	FilesCreated     []string `json:"files_created"`
	Model            string   `json:"model,omitempty"`
	GeneratedAt      string   `json:"generated_at"`
}

// RequirementsValidateResponse matches the API response structure
type RequirementsValidateResponse struct {
	EntityType       string               `json:"entity_type"`
	EntityName       string               `json:"entity_name"`
	Status           string               `json:"status"`
	Message          string               `json:"message,omitempty"`
	RequirementCount int                  `json:"requirement_count"`
	TargetCount      int                  `json:"target_count"`
	LinkedCount      int                  `json:"linked_count"`
	UnlinkedTargets  []string             `json:"unlinked_targets,omitempty"`
	Violations       []StandardsViolation `json:"violations"`
	GeneratedAt      string               `json:"generated_at"`
}

// RequirementsFixRequest matches the API request structure
type RequirementsFixRequest struct {
	EntityType string `json:"entity_type"`
	EntityName string `json:"entity_name"`
	Context    string `json:"context,omitempty"`
	Model      string `json:"model,omitempty"`
}

// RequirementsFixResponse matches the API response structure
type RequirementsFixResponse struct {
	EntityType          string   `json:"entity_type"`
	EntityName          string   `json:"entity_name"`
	Success             bool     `json:"success"`
	Message             string   `json:"message,omitempty"`
	TargetsFixed        int      `json:"targets_fixed"`
	RequirementsAdded   int      `json:"requirements_added"`
	TotalRequirements   int      `json:"total_requirements"`
	RemainingViolations int      `json:"remaining_violations"`
	FilesModified       []string `json:"files_modified"`
	Model               string   `json:"model,omitempty"`
	FixedAt             string   `json:"fixed_at"`
}

func (a *App) cmdRequirements(args []string) error {
	if len(args) == 0 {
		return a.requirementsHelp()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "generate":
		return a.requirementsGenerate(subArgs)
	case "validate":
		return a.requirementsValidate(subArgs)
	case "fix":
		return a.requirementsFix(subArgs)
	case "help", "--help", "-h":
		return a.requirementsHelp()
	default:
		return fmt.Errorf("unknown requirements subcommand: %s\n\nRun 'prd-control-tower requirements help' for usage", subcommand)
	}
}

func (a *App) requirementsHelp() error {
	help := `Requirements management commands

Usage:
  prd-control-tower requirements <subcommand> [options]

Subcommands:
  generate    Generate requirements from PRD operational targets
  validate    Validate requirements against PRD
  fix         Fix missing requirements for uncovered P0/P1 targets

Examples:
  prd-control-tower requirements generate my-scenario --context-file /tmp/context.md
  prd-control-tower requirements validate my-scenario
  prd-control-tower requirements fix my-scenario

Run 'prd-control-tower requirements <subcommand> --help' for subcommand details.
`
	fmt.Print(help)
	return nil
}

// requirementsGenerate implements 'requirements generate' - generate requirements from PRD
func (a *App) requirementsGenerate(args []string) error {
	fs := flag.NewFlagSet("requirements generate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	entityType := fs.String("type", "scenario", "Entity type: scenario or resource")
	context := fs.String("context", "", "Additional context for AI generation")
	contextFile := fs.String("context-file", "", "Path to a file containing context for AI generation")
	model := fs.String("model", "", "Override OpenRouter model")

	remaining, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return fmt.Errorf("usage: requirements generate <name> [--type scenario|resource] [--context ...|--context-file ...] [--json]")
	}

	name := strings.TrimSpace(remaining[0])
	if name == "" {
		return fmt.Errorf("usage: requirements generate <name> [--type scenario|resource] [--context ...|--context-file ...] [--json]")
	}

	resolvedType := strings.ToLower(strings.TrimSpace(*entityType))
	if resolvedType != "scenario" && resolvedType != "resource" {
		return fmt.Errorf("invalid --type %q (must be scenario or resource)", *entityType)
	}

	// Build context from flags
	finalContext := strings.TrimSpace(*context)
	if *contextFile != "" {
		data, err := os.ReadFile(*contextFile)
		if err != nil {
			return fmt.Errorf("read --context-file: %w", err)
		}
		fileContext := strings.TrimSpace(string(data))
		if finalContext == "" {
			finalContext = fileContext
		} else if fileContext != "" {
			finalContext = finalContext + "\n\n" + fileContext
		}
	}

	// Call API
	req := RequirementsGenerateRequest{
		EntityType: resolvedType,
		EntityName: name,
		Context:    finalContext,
		Model:      strings.TrimSpace(*model),
	}

	body, err := a.services.Requirements.Generate(req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Parse response for pretty output
	var resp RequirementsGenerateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if !resp.Success {
		return fmt.Errorf("requirements generation failed: %s", resp.Message)
	}

	fmt.Printf("✅ Generated %d requirements for %s/%s\n", resp.RequirementCount, resp.EntityType, resp.EntityName)
	fmt.Printf("   P0: %d | P1: %d | P2: %d\n", resp.P0Count, resp.P1Count, resp.P2Count)
	fmt.Printf("   Model: %s\n", resp.Model)
	fmt.Println()
	fmt.Println("Files created:")
	for _, f := range resp.FilesCreated {
		fmt.Printf("   • %s\n", f)
	}

	return nil
}

// requirementsFix implements 'requirements fix' - fix missing requirements for uncovered targets
func (a *App) requirementsFix(args []string) error {
	fs := flag.NewFlagSet("requirements fix", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	entityType := fs.String("type", "scenario", "Entity type: scenario or resource")
	context := fs.String("context", "", "Additional context for AI generation")
	contextFile := fs.String("context-file", "", "Path to a file containing context for AI generation")
	model := fs.String("model", "", "Override OpenRouter model")

	remaining, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return fmt.Errorf("usage: requirements fix <name> [--type scenario|resource] [--context ...] [--json]")
	}

	name := strings.TrimSpace(remaining[0])
	if name == "" {
		return fmt.Errorf("usage: requirements fix <name> [--type scenario|resource] [--context ...] [--json]")
	}

	resolvedType := strings.ToLower(strings.TrimSpace(*entityType))
	if resolvedType != "scenario" && resolvedType != "resource" {
		return fmt.Errorf("invalid --type %q (must be scenario or resource)", *entityType)
	}

	// Build context from flags
	finalContext := strings.TrimSpace(*context)
	if *contextFile != "" {
		data, err := os.ReadFile(*contextFile)
		if err != nil {
			return fmt.Errorf("read --context-file: %w", err)
		}
		fileContext := strings.TrimSpace(string(data))
		if finalContext == "" {
			finalContext = fileContext
		} else if fileContext != "" {
			finalContext = finalContext + "\n\n" + fileContext
		}
	}

	// Call API
	req := RequirementsFixRequest{
		EntityType: resolvedType,
		EntityName: name,
		Context:    finalContext,
		Model:      strings.TrimSpace(*model),
	}

	body, err := a.services.Requirements.Fix(req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Parse response for pretty output
	var resp RequirementsFixResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if !resp.Success {
		return fmt.Errorf("requirements fix failed: %s", resp.Message)
	}

	fmt.Printf("✅ %s\n", resp.Message)
	fmt.Printf("   Targets fixed: %d\n", resp.TargetsFixed)
	fmt.Printf("   Requirements added: %d\n", resp.RequirementsAdded)
	fmt.Printf("   Total requirements: %d\n", resp.TotalRequirements)
	if resp.RemainingViolations > 0 {
		fmt.Printf("   ⚠️  Remaining violations: %d\n", resp.RemainingViolations)
	}
	if resp.Model != "" {
		fmt.Printf("   Model: %s\n", resp.Model)
	}

	return nil
}

// requirementsValidate implements 'requirements validate' - validate requirements against PRD
func (a *App) requirementsValidate(args []string) error {
	fs := flag.NewFlagSet("requirements validate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	entityType := fs.String("type", "", "Entity type: scenario or resource (default: auto-detect)")
	noCache := fs.Bool("no-cache", false, "Bypass validation cache")

	remaining, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return fmt.Errorf("usage: requirements validate <name> [--type scenario|resource] [--no-cache] [--json]")
	}

	name := strings.TrimSpace(remaining[0])
	if name == "" {
		return fmt.Errorf("usage: requirements validate <name> [--type scenario|resource] [--no-cache] [--json]")
	}

	resolvedType := strings.ToLower(strings.TrimSpace(*entityType))
	if resolvedType == "" {
		resolvedType = detectEntityTypeFromRepo(name)
	}
	if resolvedType != "scenario" && resolvedType != "resource" {
		return fmt.Errorf("invalid --type %q (must be scenario or resource)", *entityType)
	}

	// Call API
	body, err := a.services.Requirements.Validate(resolvedType, name, !*noCache)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Parse response for pretty output
	var resp RequirementsValidateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Requirements Validation: %s/%s\n", resp.EntityType, resp.EntityName)
	fmt.Printf("Status: %s\n", resp.Status)
	if resp.Message != "" {
		fmt.Printf("Message: %s\n", resp.Message)
	}
	fmt.Println()

	fmt.Printf("Requirements: %d | Targets: %d | Linked: %d\n", resp.RequirementCount, resp.TargetCount, resp.LinkedCount)
	fmt.Println()

	if len(resp.UnlinkedTargets) > 0 {
		fmt.Printf("⚠️  Unlinked Targets (%d):\n", len(resp.UnlinkedTargets))
		for _, target := range resp.UnlinkedTargets {
			fmt.Printf("   • %s\n", target)
		}
		fmt.Println()
	}

	if len(resp.Violations) == 0 {
		fmt.Println("✅ No violations found")
		return nil
	}

	// Group violations by severity
	critical := []StandardsViolation{}
	high := []StandardsViolation{}
	medium := []StandardsViolation{}
	low := []StandardsViolation{}

	for _, v := range resp.Violations {
		switch strings.ToLower(v.Severity) {
		case "critical":
			critical = append(critical, v)
		case "high":
			high = append(high, v)
		case "medium":
			medium = append(medium, v)
		default:
			low = append(low, v)
		}
	}

	printViolationGroup := func(label string, violations []StandardsViolation) {
		if len(violations) == 0 {
			return
		}
		fmt.Printf("%s (%d):\n", label, len(violations))
		for _, v := range violations {
			fmt.Printf("  • %s\n", v.Title)
			if v.Description != "" {
				fmt.Printf("    %s\n", v.Description)
			}
			if v.Recommendation != "" {
				fmt.Printf("    → %s\n", v.Recommendation)
			}
		}
		fmt.Println()
	}

	printViolationGroup("🔴 Critical", critical)
	printViolationGroup("🟠 High", high)
	printViolationGroup("🟡 Medium", medium)
	printViolationGroup("🟢 Low", low)

	fmt.Printf("Total: %d violation(s)\n", len(resp.Violations))
	return nil
}
