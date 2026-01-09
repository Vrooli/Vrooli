package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// PRDGenerateOutput is the structured output for prd generate.
type PRDGenerateOutput struct {
	Generation AIGenerateDraftResponse `json:"generation"`
	Publish    *PublishResponse        `json:"publish,omitempty"`
}

// PRDValidateOutput is the structured output for prd validate.
type PRDValidateOutput struct {
	EntityType  string               `json:"entity_type"`
	EntityName  string               `json:"entity_name"`
	Status      string               `json:"status"`
	Message     string               `json:"message,omitempty"`
	Violations  []StandardsViolation `json:"violations"`
	IssueCounts QualityIssueCounts   `json:"issue_counts"`
}

// StandardsViolation matches the API response structure.
type StandardsViolation struct {
	RuleID         string         `json:"rule_id"`
	Severity       string         `json:"severity"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	FilePath       string         `json:"file_path,omitempty"`
	Recommendation string         `json:"recommendation,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// QualityIssueCounts matches the API response structure.
type QualityIssueCounts struct {
	MissingPRD              int `json:"missing_prd"`
	MissingTemplateSections int `json:"missing_template_sections"`
	TargetCoverage          int `json:"target_coverage"`
	RequirementCoverage     int `json:"requirement_coverage"`
	PRDRef                  int `json:"prd_ref"`
	Documentation           int `json:"documentation"`
	Total                   int `json:"total"`
	Blocking                int `json:"blocking"`
}

// PRDFixOutput is the structured output for prd fix.
type PRDFixOutput struct {
	EntityType    string   `json:"entity_type"`
	EntityName    string   `json:"entity_name"`
	Success       bool     `json:"success"`
	Message       string   `json:"message,omitempty"`
	FixedIssues   []string `json:"fixed_issues,omitempty"`
	PRDPath       string   `json:"prd_path,omitempty"`
	BackupPath    string   `json:"backup_path,omitempty"`
	Model         string   `json:"model,omitempty"`
	GeneratedText string   `json:"generated_text,omitempty"`
}

func (a *App) cmdPRD(args []string) error {
	if len(args) == 0 {
		return a.prdHelp()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "generate":
		return a.prdGenerate(subArgs)
	case "validate":
		return a.prdValidate(subArgs)
	case "fix":
		return a.prdFix(subArgs)
	case "help", "--help", "-h":
		return a.prdHelp()
	default:
		return fmt.Errorf("unknown prd subcommand: %s\n\nRun 'prd-control-tower prd help' for usage", subcommand)
	}
}

func (a *App) prdHelp() error {
	help := `PRD management commands

Usage:
  prd-control-tower prd <subcommand> [options]

Subcommands:
  generate    AI-generate a full PRD (optionally publish)
  validate    Validate a PRD against template and quality standards
  fix         Use AI to fix PRD issues

Examples:
  prd-control-tower prd generate my-scenario --context-file /tmp/context.md --publish
  prd-control-tower prd validate my-scenario
  prd-control-tower prd fix my-scenario --auto

Run 'prd-control-tower prd <subcommand> --help' for subcommand details.
`
	fmt.Print(help)
	return nil
}

// prdGenerate implements 'prd generate' - AI-generate a full PRD.
func (a *App) prdGenerate(args []string) error {
	fs := flag.NewFlagSet("prd generate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	entityType := fs.String("type", "scenario", "Entity type: scenario or resource")
	context := fs.String("context", "", "Context for AI generation")
	contextFile := fs.String("context-file", "", "Path to a file containing context for AI generation")
	model := fs.String("model", "", "Override OpenRouter model (e.g. openrouter/x-ai/grok-code-fast-1)")
	owner := fs.String("owner", "", "Owner metadata for the created/updated draft")

	templateName := fs.String("template", "", "If set, publish the generated PRD into a scenario created from this template")
	publish := fs.Bool("publish", false, "Publish the generated PRD to PRD.md for an existing scenario/resource")
	noBackup := fs.Bool("no-backup", false, "Do not create a backup when publishing")
	force := fs.Bool("force", false, "Allow overwriting an existing scenario when publishing with --template")
	runHooks := fs.Bool("run-hooks", false, "Run template hooks when publishing with --template")
	noSaveDraft := fs.Bool("no-save-draft", false, "Do not persist generated content into the draft (default: persist)")

	remaining, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return fmt.Errorf("usage: prd generate <name> [--type scenario|resource] [--context ...|--context-file ...] [--publish] [--json]")
	}

	name := strings.TrimSpace(remaining[0])
	if name == "" {
		return fmt.Errorf("usage: prd generate <name> [--type scenario|resource] [--context ...|--context-file ...] [--publish] [--json]")
	}

	resolvedType := strings.ToLower(strings.TrimSpace(*entityType))
	if resolvedType != "scenario" && resolvedType != "resource" {
		return fmt.Errorf("invalid --type %q (must be scenario or resource)", *entityType)
	}

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

	save := true
	if *noSaveDraft {
		save = false
	}

	req := AIGenerateDraftRequest{
		EntityType: resolvedType,
		EntityName: name,
		Owner:      strings.TrimSpace(*owner),
		Section:    "🎯 Full PRD",
		Context:    finalContext,
		Action:     "",
		Model:      strings.TrimSpace(*model),
	}
	req.SaveGeneratedToDraft = &save

	genBody, genResp, err := a.services.AI.GenerateDraft(req)
	if err != nil {
		return err
	}
	if !genResp.Success {
		if *jsonOutput {
			cliutil.PrintJSON(genBody)
			return nil
		}
		if strings.TrimSpace(genResp.Message) != "" {
			return fmt.Errorf("AI generation failed: %s", genResp.Message)
		}
		return fmt.Errorf("AI generation failed")
	}

	var publishParsed *PublishResponse
	if strings.TrimSpace(*templateName) != "" {
		pubReq := PublishRequest{
			CreateBackup: true,
			DeleteDraft:  true,
			Template: &PublishTemplateRequest{
				Name: strings.TrimSpace(*templateName),
				Variables: map[string]string{
					"SCENARIO_ID": name,
				},
				Force:    *force,
				RunHooks: *runHooks,
			},
		}
		pubBody, pubResp, err := a.services.Drafts.Publish(genResp.DraftID, pubReq)
		if err != nil {
			return err
		}
		if !pubResp.Success {
			if *jsonOutput {
				out := PRDGenerateOutput{Generation: genResp, Publish: &pubResp}
				data, _ := json.MarshalIndent(out, "", "  ")
				fmt.Println(string(data))
				return nil
			}
			if strings.TrimSpace(pubResp.Message) != "" {
				return fmt.Errorf("publish failed: %s", pubResp.Message)
			}
			return fmt.Errorf("publish failed")
		}
		_ = pubBody
		publishParsed = &pubResp
	} else if *publish {
		pubReq := PublishRequest{
			CreateBackup: !*noBackup,
			DeleteDraft:  true,
			Template:     nil,
		}
		_, pubResp, err := a.services.Drafts.Publish(genResp.DraftID, pubReq)
		if err != nil {
			return err
		}
		if !pubResp.Success {
			if *jsonOutput {
				out := PRDGenerateOutput{Generation: genResp, Publish: &pubResp}
				data, _ := json.MarshalIndent(out, "", "  ")
				fmt.Println(string(data))
				return nil
			}
			if strings.TrimSpace(pubResp.Message) != "" {
				return fmt.Errorf("publish failed: %s", pubResp.Message)
			}
			return fmt.Errorf("publish failed")
		}
		publishParsed = &pubResp
	}

	if *jsonOutput {
		out := PRDGenerateOutput{Generation: genResp, Publish: publishParsed}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("encode output: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Generated draft %s for %s/%s\n", genResp.DraftID, genResp.EntityType, genResp.EntityName)
	if genResp.DraftFilePath != "" {
		fmt.Printf("Draft file: %s\n", genResp.DraftFilePath)
	}
	if publishParsed != nil {
		fmt.Printf("Published to: %s\n", publishParsed.PublishedTo)
		if publishParsed.CreatedScenario && publishParsed.ScenarioPath != "" {
			fmt.Printf("Scenario created: %s\n", publishParsed.ScenarioPath)
		}
	}
	return nil
}

// prdValidate implements 'prd validate' - validate a PRD against quality standards.
func (a *App) prdValidate(args []string) error {
	fs := flag.NewFlagSet("prd validate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	entityType := fs.String("type", "", "Entity type: scenario or resource (default: auto-detect)")
	noCache := fs.Bool("no-cache", false, "Bypass validation cache")

	remaining, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return fmt.Errorf("usage: prd validate <name> [--type scenario|resource] [--no-cache] [--json]")
	}

	name := strings.TrimSpace(remaining[0])
	if name == "" {
		return fmt.Errorf("usage: prd validate <name> [--type scenario|resource] [--no-cache] [--json]")
	}

	resolvedType := strings.ToLower(strings.TrimSpace(*entityType))
	if resolvedType == "" {
		resolvedType = detectEntityTypeFromRepo(name)
	}
	if resolvedType != "scenario" && resolvedType != "resource" {
		return fmt.Errorf("invalid --type %q (must be scenario or resource)", *entityType)
	}

	// Call the quality standards API which provides detailed violations
	body, err := a.services.PRD.ValidateStandards(resolvedType, name, !*noCache)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Parse response for pretty output
	var resp struct {
		EntityType  string               `json:"entity_type"`
		EntityName  string               `json:"entity_name"`
		Status      string               `json:"status"`
		Message     string               `json:"message"`
		Violations  []StandardsViolation `json:"violations"`
		GeneratedAt string               `json:"generated_at"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		// Fall back to raw output
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Printf("PRD Validation: %s/%s\n", resp.EntityType, resp.EntityName)
	fmt.Printf("Status: %s\n", resp.Status)
	if resp.Message != "" {
		fmt.Printf("Message: %s\n", resp.Message)
	}
	fmt.Println()

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

// prdFix implements 'prd fix' - use AI to fix PRD issues.
func (a *App) prdFix(args []string) error {
	fs := flag.NewFlagSet("prd fix", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	entityType := fs.String("type", "", "Entity type: scenario or resource (default: auto-detect)")
	model := fs.String("model", "", "Override OpenRouter model")
	auto := fs.Bool("auto", false, "Automatically apply fixes without confirmation")
	noBackup := fs.Bool("no-backup", false, "Do not create a backup before fixing")

	remaining, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return fmt.Errorf("usage: prd fix <name> [--type scenario|resource] [--auto] [--no-backup] [--json]")
	}

	name := strings.TrimSpace(remaining[0])
	if name == "" {
		return fmt.Errorf("usage: prd fix <name> [--type scenario|resource] [--auto] [--no-backup] [--json]")
	}

	resolvedType := strings.ToLower(strings.TrimSpace(*entityType))
	if resolvedType == "" {
		resolvedType = detectEntityTypeFromRepo(name)
	}
	if resolvedType != "scenario" && resolvedType != "resource" {
		return fmt.Errorf("invalid --type %q (must be scenario or resource)", *entityType)
	}

	// Step 1: Get current violations
	validationBody, err := a.services.PRD.ValidateStandards(resolvedType, name, false)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	var validationResp struct {
		Status     string               `json:"status"`
		Violations []StandardsViolation `json:"violations"`
	}
	if err := json.Unmarshal(validationBody, &validationResp); err != nil {
		return fmt.Errorf("parse validation response: %w", err)
	}

	if len(validationResp.Violations) == 0 {
		output := PRDFixOutput{
			EntityType: resolvedType,
			EntityName: name,
			Success:    true,
			Message:    "No violations to fix",
		}
		if *jsonOutput {
			data, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("✅ %s/%s has no violations to fix\n", resolvedType, name)
		}
		return nil
	}

	// Step 2: Build fix context from violations
	var fixContext strings.Builder
	fixContext.WriteString("Fix the following PRD validation issues:\n\n")
	for i, v := range validationResp.Violations {
		fixContext.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, v.Severity, v.Title))
		if v.Description != "" {
			fixContext.WriteString(fmt.Sprintf("   Description: %s\n", v.Description))
		}
		if v.Recommendation != "" {
			fixContext.WriteString(fmt.Sprintf("   Recommendation: %s\n", v.Recommendation))
		}
		fixContext.WriteString("\n")
	}

	if !*auto && !*jsonOutput {
		fmt.Printf("Found %d violation(s) to fix:\n", len(validationResp.Violations))
		for _, v := range validationResp.Violations {
			fmt.Printf("  • [%s] %s\n", v.Severity, v.Title)
		}
		fmt.Println()
		fmt.Print("Proceed with AI fix? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Println("Aborted")
			return nil
		}
	}

	// Step 3: Generate fixed PRD
	save := true
	req := AIGenerateDraftRequest{
		EntityType:           resolvedType,
		EntityName:           name,
		Section:              "🎯 Full PRD",
		Context:              fixContext.String(),
		Model:                strings.TrimSpace(*model),
		SaveGeneratedToDraft: &save,
	}
	// Include existing content so AI can fix it
	includeExisting := true
	req.IncludeExistingContent = &includeExisting

	_, genResp, err := a.services.AI.GenerateDraft(req)
	if err != nil {
		return fmt.Errorf("AI generation failed: %w", err)
	}
	if !genResp.Success {
		return fmt.Errorf("AI generation failed: %s", genResp.Message)
	}

	// Step 4: Publish the fix
	pubReq := PublishRequest{
		CreateBackup: !*noBackup,
		DeleteDraft:  true,
	}
	_, pubResp, err := a.services.Drafts.Publish(genResp.DraftID, pubReq)
	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}
	if !pubResp.Success {
		return fmt.Errorf("publish failed: %s", pubResp.Message)
	}

	// Collect fixed issue titles
	fixedIssues := make([]string, len(validationResp.Violations))
	for i, v := range validationResp.Violations {
		fixedIssues[i] = v.Title
	}

	output := PRDFixOutput{
		EntityType:  resolvedType,
		EntityName:  name,
		Success:     true,
		Message:     fmt.Sprintf("Fixed %d violation(s)", len(validationResp.Violations)),
		FixedIssues: fixedIssues,
		PRDPath:     pubResp.PublishedTo,
		BackupPath:  pubResp.BackupPath,
		Model:       genResp.Model,
	}

	if *jsonOutput {
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("✅ Fixed %d violation(s) in %s/%s\n", len(validationResp.Violations), resolvedType, name)
	fmt.Printf("Published to: %s\n", pubResp.PublishedTo)
	if pubResp.BackupPath != "" {
		fmt.Printf("Backup: %s\n", pubResp.BackupPath)
	}
	fmt.Printf("Model: %s\n", genResp.Model)

	return nil
}

// detectEntityTypeFromRepo detects entity type based on repo structure.
func detectEntityTypeFromRepo(name string) string {
	root := findRepoRoot()
	if root == "" {
		return "scenario"
	}
	if statDir(filepath.Join(root, "resources", name)) {
		return "resource"
	}
	if statDir(filepath.Join(root, "scenarios", name)) {
		return "scenario"
	}
	return "scenario"
}

func findRepoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for i := 0; i < 25; i++ {
		if statDir(filepath.Join(dir, "scenarios")) && statDir(filepath.Join(dir, "resources")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func statDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
