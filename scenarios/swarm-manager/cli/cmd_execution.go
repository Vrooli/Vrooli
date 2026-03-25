package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type executionOptions struct {
	mode         string
	delaySeconds int64
	operation    string
	startedBy    string
}

func addExecutionOptionsFlags(fs *flag.FlagSet) (mode *string, delay *int64, operation *string, startedBy *string) {
	mode = fs.String("mode", "", "Execution mode: manual|scheduled|yolo")
	delay = fs.Int64("delay-seconds", 0, "Schedule delay in seconds (scheduled mode)")
	operation = fs.String("operation", "generator", "Operation hint: generator|improver")
	startedBy = fs.String("started-by", "swarm-manager", "Started-by attribution label")
	return mode, delay, operation, startedBy
}

func parseExecutionOptions(mode *string, delay *int64, operation *string, startedBy *string, requireMode bool) (executionOptions, error) {
	parsedMode := strings.ToLower(strings.TrimSpace(*mode))
	if requireMode && parsedMode == "" {
		return executionOptions{}, fmt.Errorf("mode is required and must be manual, scheduled, or yolo")
	}
	if parsedMode != "" && parsedMode != "manual" && parsedMode != "scheduled" && parsedMode != "yolo" {
		return executionOptions{}, fmt.Errorf("invalid mode %q (expected manual, scheduled, or yolo)", parsedMode)
	}
	parsedOperation := strings.ToLower(strings.TrimSpace(*operation))
	if parsedOperation != "generator" && parsedOperation != "improver" {
		return executionOptions{}, fmt.Errorf("invalid operation %q (expected generator or improver)", parsedOperation)
	}
	if *delay < 0 {
		return executionOptions{}, fmt.Errorf("delay-seconds must be >= 0")
	}

	return executionOptions{
		mode:         parsedMode,
		delaySeconds: *delay,
		operation:    parsedOperation,
		startedBy:    strings.TrimSpace(*startedBy),
	}, nil
}

func parsePolicyOptions(mode *string, delay *int64) (executionOptions, error) {
	defaultOperation := "generator"
	defaultStartedBy := "swarm-manager"
	opts, err := parseExecutionOptions(mode, delay, &defaultOperation, &defaultStartedBy, true)
	if err != nil {
		return executionOptions{}, err
	}
	return opts, nil
}

func (a *App) resolveExecutionMode(mode string) (string, error) {
	if strings.TrimSpace(mode) != "" {
		return mode, nil
	}

	body, err := a.getV1("/settings", nil)
	if err != nil {
		return "", fmt.Errorf("resolve execution mode from settings: %w", err)
	}
	var response struct {
		Settings struct {
			DefaultMode string `json:"default_mode"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("decode settings: %w", err)
	}
	resolved := strings.ToLower(strings.TrimSpace(response.Settings.DefaultMode))
	if resolved == "" {
		return "", fmt.Errorf("settings default_mode is empty; provide --mode manual|scheduled|yolo")
	}
	if resolved != "manual" && resolved != "scheduled" && resolved != "yolo" {
		return "", fmt.Errorf("settings returned invalid default_mode %q", resolved)
	}
	return resolved, nil
}

func (a *App) cmdExecutionList(args []string) error {
	fs := flag.NewFlagSet("execution list", flag.ContinueOnError)
	status := fs.String("status", "", "Filter by status")
	mode := fs.String("mode", "", "Filter by mode")
	backlogKind := fs.String("backlog-kind", "", "Filter by backlog kind")
	backlogName := fs.String("backlog-name", "", "Filter by backlog name")
	startedBy := fs.String("started-by", "", "Filter by started_by/source team")
	createdFrom := fs.String("created-from", "", "Filter by created_at lower bound (RFC3339)")
	createdTo := fs.String("created-to", "", "Filter by created_at upper bound (RFC3339)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*status) != "" {
		query.Set("status", strings.TrimSpace(*status))
	}
	if strings.TrimSpace(*mode) != "" {
		query.Set("mode", strings.TrimSpace(*mode))
	}
	if strings.TrimSpace(*backlogKind) != "" {
		query.Set("backlog_kind", strings.TrimSpace(*backlogKind))
	}
	if strings.TrimSpace(*backlogName) != "" {
		query.Set("backlog_name", strings.TrimSpace(*backlogName))
	}
	if strings.TrimSpace(*startedBy) != "" {
		query.Set("started_by", strings.TrimSpace(*startedBy))
	}
	if strings.TrimSpace(*createdFrom) != "" {
		query.Set("created_from", strings.TrimSpace(*createdFrom))
	}
	if strings.TrimSpace(*createdTo) != "" {
		query.Set("created_to", strings.TrimSpace(*createdTo))
	}

	body, err := a.getV1("/execution", query)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var response ExecutionListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if len(response.Items) == 0 {
		printSection("Summary")
		fmt.Println("  No execution runs found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "list"),
			cliCommand("execution", "create", "--kind", "<backlog-kind>", "--name", "<backlog-name>"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d execution run(s)\n", len(response.Items))
	printSection("Results")
	for _, item := range response.Items {
		fmt.Printf("  %s (%s)\n", item.ExecutionID, item.Status)
		fmt.Printf("    Backlog: %s/%s\n", item.BacklogKind, item.BacklogName)
		fmt.Printf("    Mode: %s\n", item.Mode)
		if item.RunID != "" {
			fmt.Printf("    Run ID: %s\n", item.RunID)
		}
		if item.TaskID != "" {
			fmt.Printf("    Task ID: %s\n", item.TaskID)
		}
		if item.FailureReason != "" {
			fmt.Printf("    Failure: %s\n", item.FailureReason)
		}
		fmt.Println()
	}
	first := response.Items[0]
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("execution", "get", "--id", "<execution-id>"),
		cliCommand("execution", "get", "--id", first.ExecutionID),
		cliCommand("execution", "start", "--id", first.ExecutionID),
	})
	return nil
}

func (a *App) cmdExecutionGet(args []string) error {
	fs := flag.NewFlagSet("execution get", flag.ContinueOnError)
	id := fs.String("id", "", "Execution ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
		return fmt.Errorf("usage: execution get --id ID [--json]\n\n%s", err)
	}
	executionID := strings.TrimSpace(*id)
	body, err := a.getV1("/execution/"+executionID, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ExecutionItemResponse](body)
	if err != nil {
		return err
	}
	item := response.Execution
	printSection("Summary")
	fmt.Printf("  Execution %s (%s)\n", item.ExecutionID, item.Status)
	printSection("Details")
	fmt.Printf("  Backlog: %s/%s\n", item.BacklogKind, item.BacklogName)
	fmt.Printf("  Mode: %s\n", item.Mode)
	if item.RunID != "" {
		fmt.Printf("  Run ID: %s\n", item.RunID)
	}
	if item.TaskID != "" {
		fmt.Printf("  Task ID: %s\n", item.TaskID)
	}
	if item.FailureReason != "" {
		fmt.Printf("  Failure: %s\n", item.FailureReason)
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "start", "--id", item.ExecutionID),
		cliCommand("execution", "cancel", "--id", item.ExecutionID),
		cliCommand("execution", "retry", "--id", item.ExecutionID),
	})
	return nil
}

func (a *App) cmdExecutionCreate(args []string) error {
	fs := flag.NewFlagSet("execution create", flag.ContinueOnError)
	kind := fs.String("kind", "", "Backlog kind")
	name := fs.String("name", "", "Backlog name")
	mode, delay, operation, startedBy := addExecutionOptionsFlags(fs)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kind, "name", *name); err != nil {
		return fmt.Errorf("usage: execution create --kind KIND --name NAME [--mode manual|scheduled|yolo] [--delay-seconds N] [--operation generator|improver] [--started-by NAME] [--json]\n\n%s", err)
	}

	opts, err := parseExecutionOptions(mode, delay, operation, startedBy, false)
	if err != nil {
		return err
	}
	resolvedMode, err := a.resolveExecutionMode(opts.mode)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"backlog_kind":  strings.TrimSpace(*kind),
		"backlog_name":  strings.TrimSpace(*name),
		"mode":          resolvedMode,
		"delay_seconds": opts.delaySeconds,
		"operation":     opts.operation,
		"started_by":    opts.startedBy,
	}

	body, err := a.requestV1("POST", "/execution", nil, payload)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var response ExecutionItemResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	printSection("Result")
	fmt.Printf("  Execution created: %s\n", response.Execution.ExecutionID)
	printSection("What Changed")
	fmt.Printf("  Backlog: %s/%s\n", response.Execution.BacklogKind, response.Execution.BacklogName)
	fmt.Printf("  Status: %s\n", response.Execution.Status)
	fmt.Printf("  Mode: %s\n", response.Execution.Mode)
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "get", "--id", response.Execution.ExecutionID),
		cliCommand("execution", "start", "--id", response.Execution.ExecutionID),
	})
	return nil
}

func (a *App) cmdExecutionPolicyGet(args []string) error {
	fs := flag.NewFlagSet("execution policy get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.getV1("/settings", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var response struct {
		Settings struct {
			DefaultMode         string `json:"default_mode"`
			DefaultDelaySeconds int64  `json:"default_delay_seconds"`
			AutoFixup           bool   `json:"auto_fixup"`
			MaxFixupAttempts    int    `json:"max_fixup_attempts"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	s := response.Settings
	printSection("Summary")
	fmt.Printf("  Default mode: %s\n", s.DefaultMode)
	fmt.Printf("  Default delay seconds: %d\n", s.DefaultDelaySeconds)
	fmt.Printf("  Auto fixup: %t\n", s.AutoFixup)
	fmt.Printf("  Max fixup attempts: %d\n", s.MaxFixupAttempts)
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "policy-update", "--mode", s.DefaultMode, "--delay-seconds", fmt.Sprintf("%d", s.DefaultDelaySeconds)),
		cliCommand("settings", "get"),
	})
	return nil
}

func (a *App) cmdExecutionPolicyUpdate(args []string) error {
	fs := flag.NewFlagSet("execution policy update", flag.ContinueOnError)
	mode := fs.String("mode", "", "Default mode: manual|scheduled|yolo")
	delay := fs.Int64("delay-seconds", 300, "Default delay seconds for scheduled mode")
	autoFixup := fs.Bool("auto-fixup", false, "Enable auto fixup")
	maxFixup := fs.Int("max-fixup-attempts", -1, "Max fixup attempts (0-5)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	opts, err := parsePolicyOptions(mode, delay)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"default_mode":          opts.mode,
		"default_delay_seconds": opts.delaySeconds,
	}
	if *autoFixup {
		payload["auto_fixup"] = true
	}
	if *maxFixup >= 0 {
		payload["max_fixup_attempts"] = *maxFixup
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}
	body, err := a.requestV1("PUT", "/settings", nil, json.RawMessage(payloadBytes))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var response struct {
		Settings struct {
			DefaultMode         string `json:"default_mode"`
			DefaultDelaySeconds int64  `json:"default_delay_seconds"`
			AutoFixup           bool   `json:"auto_fixup"`
			MaxFixupAttempts    int    `json:"max_fixup_attempts"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	s := response.Settings
	printSection("Result")
	fmt.Printf("  Updated execution policy settings\n")
	printSection("What Changed")
	fmt.Printf("  Default mode: %s\n", s.DefaultMode)
	fmt.Printf("  Default delay seconds: %d\n", s.DefaultDelaySeconds)
	fmt.Printf("  Auto fixup: %t\n", s.AutoFixup)
	fmt.Printf("  Max fixup attempts: %d\n", s.MaxFixupAttempts)
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "policy-get"),
		cliCommand("settings", "get"),
	})
	return nil
}

func (a *App) cmdExecutionPromptTrace(args []string) error {
	fs := flag.NewFlagSet("execution prompt-trace", flag.ContinueOnError)
	id := fs.String("id", "", "Execution ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
		return fmt.Errorf("usage: execution prompt-trace --id ID [--json]\n\n%s", err)
	}
	executionID := strings.TrimSpace(*id)

	body, err := a.getV1("/execution/"+executionID+"/prompt-trace", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptTraceResponse](body)
	if err != nil {
		return err
	}
	printPromptTraceSummary(
		"Summary",
		fmt.Sprintf("Prompt trace for execution %s", executionID),
		response.Trace,
	)
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "get", "--id", executionID),
		cliCommand("execution", "retry", "--id", executionID),
	})
	return nil
}

func (a *App) cmdExecutionStart(args []string) error {
	return a.runExecutionMutation(args, "start")
}

func (a *App) cmdExecutionCancel(args []string) error {
	return a.runExecutionMutation(args, "cancel")
}

func (a *App) cmdExecutionRetry(args []string) error {
	return a.runExecutionMutation(args, "retry")
}

func (a *App) runExecutionMutation(args []string, action string) error {
	fs := flag.NewFlagSet("execution "+action, flag.ContinueOnError)
	id := fs.String("id", "", "Execution ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
		return fmt.Errorf("usage: execution %s --id ID [--json]\n\n%s", action, err)
	}
	executionID := strings.TrimSpace(*id)
	body, err := a.requestV1("POST", "/execution/"+executionID+"/"+action, nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ExecutionItemResponse](body)
	if err != nil {
		return err
	}

	printSection("Result")
	fmt.Printf("  Execution %s: %s\n", action, response.Execution.ExecutionID)
	printSection("What Changed")
	fmt.Printf("  Status: %s\n", response.Execution.Status)
	fmt.Printf("  Backlog: %s/%s\n", response.Execution.BacklogKind, response.Execution.BacklogName)
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "get", "--id", response.Execution.ExecutionID),
		cliCommand("execution", "list", "--status", response.Execution.Status),
	})
	return nil
}
