package automations

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"home-automation/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "automations",
		Description: "Generate, inspect, validate, and review automation safety",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List automations [--active] [--profile id] [--json]", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Aliases: []string{"generate"}, Description: "Generate a candidate automation from natural language", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "validate", Description: "Validate automation code and constraints", Run: func(args []string) error { return runValidate(core, args) }},
			{Name: "safety", Aliases: []string{"safety-check"}, Description: "Inspect stored safety status for an automation", Run: func(args []string) error { return runSafety(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("automations list")
	profileID := fs.String("profile", "", "Filter by creator/profile ID")
	activeOnly := fs.Bool("active", false, "Only show active automations")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/automations", support.BuildQuery(map[string]string{
		"profile_id": strings.TrimSpace(*profileID),
		"active":     boolQuery(*activeOnly),
	}))
	if err != nil {
		return err
	}

	var response support.AutomationListResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	sort.SliceStable(response.Automations, func(i, j int) bool {
		return response.Automations[i].CreatedAt > response.Automations[j].CreatedAt
	})

	results := make([]string, 0, len(response.Automations))
	for _, automation := range response.Automations {
		results = append(results, fmt.Sprintf("%s | %s | active=%t | risk-source=ai:%t | executions=%d | updated=%s", automation.ID, firstNonEmpty(automation.Name, automation.Description, automation.ID), automation.Active, automation.GeneratedByAI, automation.ExecutionCount, support.HumanTime(automation.UpdatedAt)))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Automations returned: %d", len(response.Automations)),
			fmt.Sprintf("Server total: %d", response.Total),
			fmt.Sprintf("Active filter: %t", *activeOnly),
			fmt.Sprintf("Profile filter: %s", firstNonEmpty(strings.TrimSpace(*profileID), "none")),
		},
		ResultsHeading: "Automations",
		Results:        results,
		RetrievalHints: []string{"home-automation automations safety <automation-id>", "home-automation automations create --description \"Turn on porch lights at sunset\" --device light.porch"},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("automations create")
	description := fs.String("description", "", "Natural-language automation description")
	profileID := fs.String("profile", "", "Profile ID associated with the automation")
	contextName := fs.String("context", "", "Optional context or schedule context")
	jsonOutput := cliutil.JSONFlag(fs)
	var devices cliutil.StringList
	fs.Var(&devices, "device", "Target device ID (repeatable)")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*description) == "" {
		return fmt.Errorf("usage: home-automation automations create --description <text> [--profile id] [--context name] [--device id] [--json]")
	}

	body, err := core.Request("POST", "/automations/generate", nil, map[string]interface{}{
		"description":    *description,
		"profile_id":     strings.TrimSpace(*profileID),
		"context":        strings.TrimSpace(*contextName),
		"target_devices": normalizeList(devices.Values()),
	})
	if err != nil {
		return err
	}

	var response support.AutomationGenerationResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Generated candidate automation: %s", firstNonEmpty(response.AutomationID, "unknown")),
			fmt.Sprintf("Ready to deploy: %t", response.ReadyToDeploy),
			fmt.Sprintf("Estimated energy impact: %s", firstNonEmpty(response.EstimatedEnergyImpact, "unknown")),
		},
		Changes: []string{
			"Explanation: " + firstNonEmpty(response.Explanation, "No explanation returned"),
			fmt.Sprintf("Validation passed: %t", response.Validation.ValidationPassed),
			fmt.Sprintf("Risk level: %s", firstNonEmpty(response.Validation.OverallRiskLevel, "unknown")),
			"Conflicts: " + joinOrNone(response.Conflicts),
			"Recommendations: " + joinOrNone(response.Validation.Recommendations),
			"Generated code:\n" + strings.TrimSpace(response.GeneratedCode),
		},
		NextCommand: []string{
			fmt.Sprintf("home-automation automations safety %s", response.AutomationID),
			"home-automation automations validate --description \"...\" --code-file <path>",
		},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runValidate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("automations validate")
	description := fs.String("description", "", "Automation description")
	code := fs.String("code", "", "Inline automation code")
	codeFile := fs.String("code-file", "", "Path to file containing automation code")
	profileID := fs.String("profile", "", "Profile ID")
	jsonOutput := cliutil.JSONFlag(fs)
	var devices cliutil.StringList
	fs.Var(&devices, "device", "Target device ID (repeatable)")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	codeValue := strings.TrimSpace(*code)
	if strings.TrimSpace(*codeFile) != "" {
		fileValue, err := cliutil.ReadFileString(strings.TrimSpace(*codeFile))
		if err != nil {
			return err
		}
		codeValue = strings.TrimSpace(fileValue)
	}
	if codeValue == "" || strings.TrimSpace(*description) == "" {
		return fmt.Errorf("usage: home-automation automations validate --description <text> (--code <value> | --code-file <path>) [--profile id] [--device id] [--json]")
	}

	body, err := core.Request("POST", "/automations/validate", nil, map[string]interface{}{
		"description":     strings.TrimSpace(*description),
		"profile_id":      strings.TrimSpace(*profileID),
		"automation_code": codeValue,
		"target_devices":  normalizeList(devices.Values()),
	})
	if err != nil {
		return err
	}

	var response support.AutomationValidationResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := validationReport(response)
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runSafety(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("automations safety")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: home-automation automations safety <automation-id> [--json]")
	}

	body, err := core.Get("/automations/"+strings.TrimSpace(fs.Arg(0))+"/safety-check", nil)
	if err != nil {
		return err
	}

	var response support.SafetyStatus
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := validationReport(response.ValidationInfo)
	report.Status = append([]string{
		fmt.Sprintf("Automation: %s", response.AutomationID),
		fmt.Sprintf("Current status: %s", firstNonEmpty(response.CurrentStatus, "unknown")),
		fmt.Sprintf("Risk level: %s", firstNonEmpty(response.RiskLevel, "unknown")),
		fmt.Sprintf("Last validated: %s", support.HumanTime(response.LastValidated)),
	}, report.Status...)
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func validationReport(response support.AutomationValidationResponse) cliapp.OperationalReport {
	return cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Automation ID: %s", firstNonEmpty(response.AutomationID, "unknown")),
			fmt.Sprintf("Validation passed: %t", response.ValidationPassed),
			fmt.Sprintf("Overall risk level: %s", firstNonEmpty(response.OverallRiskLevel, "unknown")),
			fmt.Sprintf("Validated at: %s", support.HumanTime(response.ValidationTimestamp)),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Security", Items: validationRows(response.SecurityValidation.SecurityIssues, response.SecurityValidation.Warnings, fmt.Sprintf("Risk level: %s", firstNonEmpty(response.SecurityValidation.RiskLevel, "unknown")), fmt.Sprintf("Passed: %t", response.SecurityValidation.Passed))},
			{Heading: "Permissions", Items: validationRows(response.PermissionValidation.PermissionIssues, nil, fmt.Sprintf("Can create: %t", response.PermissionValidation.UserPermissions.CanCreate), fmt.Sprintf("Allowed devices: %d", response.PermissionValidation.UserPermissions.AllowedDevices), fmt.Sprintf("Automation count: %d/%d", response.PermissionValidation.UserPermissions.AutomationCount, response.PermissionValidation.UserPermissions.AutomationLimit))},
			{Heading: "Logic", Items: validationRows(response.LogicValidation.LogicIssues, response.LogicValidation.Suggestions, fmt.Sprintf("Device compatibility: %s", firstNonEmpty(response.LogicValidation.DeviceCompatibility, "unknown")), fmt.Sprintf("Schedule safety: %s", firstNonEmpty(response.LogicValidation.ScheduleSafety, "unknown")), fmt.Sprintf("Passed: %t", response.LogicValidation.Passed))},
		},
		NextSteps: []string{
			"home-automation automations create --description \"...\" --device <device-id>",
			"home-automation automations validate --description \"...\" --code-file <path>",
		},
	}
}

func validationRows(issues []support.ValidationMsg, secondary []support.ValidationMsg, prefix ...string) []string {
	rows := make([]string, 0, len(prefix)+len(issues)+len(secondary))
	for _, item := range prefix {
		if strings.TrimSpace(item) != "" {
			rows = append(rows, item)
		}
	}
	for _, issue := range issues {
		rows = append(rows, fmt.Sprintf("%s [%s] %s", firstNonEmpty(issue.Type, "issue"), firstNonEmpty(issue.Severity, "unknown"), firstNonEmpty(issue.Message, "no details")))
	}
	for _, suggestion := range secondary {
		rows = append(rows, fmt.Sprintf("%s [%s] %s", firstNonEmpty(suggestion.Type, "note"), firstNonEmpty(suggestion.Severity, "info"), firstNonEmpty(suggestion.Message, "no details")))
	}
	return rows
}

func boolQuery(enabled bool) string {
	if enabled {
		return "true"
	}
	return ""
}

func normalizeList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, "; ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
