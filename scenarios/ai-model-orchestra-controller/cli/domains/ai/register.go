package ai

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"ai-model-orchestra-controller/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `ai` subcommand group covering model selection, request
// routing, model inventory, and resource metrics. Every command is a thin
// wrapper around a single API endpoint; the orchestrator API is the source of
// truth for routing decisions.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "ai",
		Description: "Select models, route requests, and inspect orchestrator state",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "select-model",
				Aliases:     []string{"select"},
				Description: "Select the optimal model for a task type",
				Run:         func(args []string) error { return runSelectModel(core, args) },
			},
			{
				Name:        "route-request",
				Aliases:     []string{"route"},
				Description: "Route a complete AI request through the orchestrator",
				Run:         func(args []string) error { return runRouteRequest(core, args) },
			},
			{
				Name:        "models",
				Description: "List available models with status and metrics",
				Run:         func(args []string) error { return runModels(core, args) },
			},
			{
				Name:        "resources",
				Description: "Show system resource usage and pressure history",
				Run:         func(args []string) error { return runResources(core, args) },
			},
		},
	}
}

func runSelectModel(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai select-model")
	complexity := fs.String("complexity", "", "Task complexity: simple|moderate|complex")
	priority := fs.String("priority", "", "Request priority: low|normal|high|critical")
	costLimit := fs.String("cost-limit", "", "Maximum cost per request")
	maxTokens := fs.String("max-tokens", "", "Maximum tokens to generate")
	quality := fs.String("quality", "", "Quality requirement tier")
	bodyFile := fs.String("body-file", "", "Optional JSON file overriding the built request payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: ai select-model <task-type> [--complexity ...] [--priority ...] [--cost-limit N] [--max-tokens N] [--quality ...]")
	}
	taskType := fs.Arg(0)

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		requirements := map[string]interface{}{}
		if v := strings.TrimSpace(*complexity); v != "" {
			requirements["complexity"] = v
		}
		if v := strings.TrimSpace(*priority); v != "" {
			requirements["priority"] = v
		}
		if v := strings.TrimSpace(*costLimit); v != "" {
			num, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("--cost-limit must be numeric: %w", err)
			}
			requirements["costLimit"] = num
		}
		if v := strings.TrimSpace(*maxTokens); v != "" {
			num, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("--max-tokens must be an integer: %w", err)
			}
			requirements["maxTokens"] = num
		}
		if v := strings.TrimSpace(*quality); v != "" {
			requirements["qualityRequirement"] = v
		}
		payload = map[string]interface{}{
			"taskType":     taskType,
			"requirements": requirements,
		}
	}

	body, err := core.Request("POST", "/ai/select-model", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ModelSelectResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Selected model: %s", resp.SelectedModel),
		fmt.Sprintf("Task type: %s", resp.TaskType),
		fmt.Sprintf("Request ID: %s", resp.RequestID),
		fmt.Sprintf("Fallback used: %t", resp.FallbackUsed),
	}
	if len(resp.Alternatives) > 0 {
		changes = append(changes, fmt.Sprintf("Alternatives: %s", strings.Join(resp.Alternatives, ", ")))
	}
	if len(resp.SystemMetrics) > 0 {
		changes = append(changes, "System metrics:")
		for _, row := range support.MapRows(resp.SystemMetrics) {
			changes = append(changes, "  "+row)
		}
	}
	if len(resp.ModelInfo) > 0 {
		changes = append(changes, "Model info:")
		for _, row := range support.MapRows(resp.ModelInfo) {
			changes = append(changes, "  "+row)
		}
	}

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Model selected for task %q", resp.TaskType)},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s ai route-request %s <prompt>", support.CLIName, resp.TaskType),
			fmt.Sprintf("%s ai models", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runRouteRequest(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai route-request")
	maxTokens := fs.String("max-tokens", "", "Maximum tokens to generate")
	temperature := fs.String("temperature", "", "Sampling temperature (0.0-1.0)")
	priority := fs.String("priority", "", "Request priority: low|normal|high|critical")
	retryAttempts := fs.String("retry-attempts", "", "Number of retry attempts")
	bodyFile := fs.String("body-file", "", "Optional JSON file overriding the built request payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: ai route-request <task-type> <prompt> [--max-tokens N] [--temperature N] [--priority ...] [--retry-attempts N]")
	}
	taskType := fs.Arg(0)
	prompt := fs.Arg(1)

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		requirements := map[string]interface{}{}
		if v := strings.TrimSpace(*maxTokens); v != "" {
			num, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("--max-tokens must be an integer: %w", err)
			}
			requirements["maxTokens"] = num
		}
		if v := strings.TrimSpace(*temperature); v != "" {
			num, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("--temperature must be numeric: %w", err)
			}
			requirements["temperature"] = num
		}
		if v := strings.TrimSpace(*priority); v != "" {
			requirements["priority"] = v
		}
		body := map[string]interface{}{
			"taskType":     taskType,
			"prompt":       prompt,
			"requirements": requirements,
		}
		if v := strings.TrimSpace(*retryAttempts); v != "" {
			num, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("--retry-attempts must be an integer: %w", err)
			}
			body["retryAttempts"] = num
		}
		payload = body
	}

	respBody, err := core.Request("POST", "/ai/route-request", nil, payload)
	if err != nil {
		return err
	}
	var resp support.RouteResponse
	if err := support.Decode(respBody, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Request ID: %s", resp.RequestID),
		fmt.Sprintf("Model used: %s", resp.SelectedModel),
		fmt.Sprintf("Fallback used: %t", resp.FallbackUsed),
	}
	if resp.Response != "" {
		changes = append(changes, "Response:", resp.Response)
	}
	if len(resp.Metrics) > 0 {
		changes = append(changes, "Metrics:")
		for _, row := range support.MapRows(resp.Metrics) {
			changes = append(changes, "  "+row)
		}
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Routed %q request through %s", resp.SelectedModel, support.CLIName)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s ai models", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runModels(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai models")
	includeMetrics := fs.Bool("include-metrics", false, "Include performance metrics in the response")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"includeMetrics": boolToQuery(*includeMetrics),
	})
	body, err := core.Get("/ai/models/status", query)
	if err != nil {
		return err
	}
	var resp support.ModelsStatusResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Models))
	for _, m := range resp.Models {
		successRate := 100.0
		if m.RequestCount > 0 {
			successRate = float64(m.SuccessCount) / float64(m.RequestCount) * 100
		}
		lastUsed := "never"
		if m.LastUsed != nil {
			lastUsed = support.FormatTimeValue(*m.LastUsed)
		}
		results = append(results, fmt.Sprintf(
			"%s | healthy=%t | reqs=%d | success=%.1f%% | avg=%.0fms | last=%s",
			m.ModelName, m.Healthy, m.RequestCount, successRate, m.AvgResponseTimeMs, lastUsed,
		))
	}
	if len(results) == 0 {
		results = []string{"(no models reported)"}
	}

	summary := []string{
		fmt.Sprintf("Total models: %d", resp.TotalModels),
		fmt.Sprintf("Healthy models: %d", resp.HealthyModels),
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Models",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s ai models --include-metrics", support.CLIName),
			fmt.Sprintf("%s ai resources", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runResources(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai resources")
	hours := fs.String("hours", "", "Hours of history to include (default server-side: 1)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"hours": *hours,
	})
	body, err := core.Get("/ai/resources/metrics", query)
	if err != nil {
		return err
	}
	var resp support.ResourceMetricsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	current := support.MapRows(resp.Current)
	triage := []cliapp.TriageGroup{
		{Heading: "Current", Items: current},
		{Heading: "Memory pressure", Items: []string{fmt.Sprintf("%.2f", resp.MemoryPressure)}},
	}
	if len(resp.History) > 0 {
		triage = append(triage, cliapp.TriageGroup{
			Heading: fmt.Sprintf("History (%d samples)", len(resp.History)),
			Items:   historyRows(resp.History),
		})
	}

	report := cliapp.OperationalReport{
		Status:    []string{"System resource metrics"},
		Triage:    triage,
		NextSteps: []string{fmt.Sprintf("%s ai resources --hours 24", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func boolToQuery(v bool) string {
	if v {
		return "true"
	}
	return ""
}

func historyRows(entries []map[string]interface{}) []string {
	if len(entries) == 0 {
		return []string{"(no history samples)"}
	}
	rows := make([]string, 0, len(entries))
	for _, e := range entries {
		ts := support.RenderValue(e["timestamp"])
		pressure := support.RenderValue(e["memoryPressure"])
		cpu := support.RenderValue(e["cpuUsage"])
		avail := support.RenderValue(e["availableMemoryGb"])
		rows = append(rows, fmt.Sprintf("%s | pressure=%s | cpu=%s | avail=%sGB", ts, pressure, cpu, avail))
	}
	return rows
}
