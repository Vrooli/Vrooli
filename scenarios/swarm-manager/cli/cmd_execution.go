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
	if err := fs.Parse(args); err != nil {
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
			cliCommand("execution", "create", "<backlog-kind>", "<backlog-name>"),
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
		cliCommand("execution", "get", "<execution-id>"),
		cliCommand("execution", "get", first.ExecutionID),
		cliCommand("execution", "start", first.ExecutionID),
	})
	return nil
}

func (a *App) cmdExecutionGet(args []string) error {
	fs := flag.NewFlagSet("execution get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: execution get <execution-id> [--json]")
	}
	executionID := strings.TrimSpace(fs.Arg(0))
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
		cliCommand("execution", "start", item.ExecutionID),
		cliCommand("execution", "cancel", item.ExecutionID),
		cliCommand("execution", "retry", item.ExecutionID),
	})
	return nil
}

func (a *App) cmdExecutionCreate(args []string) error {
	fs := flag.NewFlagSet("execution create", flag.ContinueOnError)
	mode, delay, operation, startedBy := addExecutionOptionsFlags(fs)
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: execution create <backlog-kind> <backlog-name> [--mode manual|scheduled|yolo] [--delay-seconds N] [--operation generator|improver] [--started-by NAME] [--json]")
	}

	opts, err := parseExecutionOptions(mode, delay, operation, startedBy, false)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"backlog_kind":  strings.TrimSpace(fs.Arg(0)),
		"backlog_name":  strings.TrimSpace(fs.Arg(1)),
		"mode":          opts.mode,
		"delay_seconds": opts.delaySeconds,
		"operation":     opts.operation,
		"started_by":    opts.startedBy,
	}
	if payload["backlog_kind"] == "" || payload["backlog_name"] == "" {
		return fmt.Errorf("backlog-kind and backlog-name are required")
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
		cliCommand("execution", "get", response.Execution.ExecutionID),
		cliCommand("execution", "start", response.Execution.ExecutionID),
	})
	return nil
}

func (a *App) cmdExecutionPolicyGet(args []string) error {
	fs := flag.NewFlagSet("execution policy get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, err := a.getV1("/execution/policy", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ExecutionPolicyResponse](body)
	if err != nil {
		return err
	}
	printSection("Summary")
	fmt.Printf("  Default mode: %s\n", response.Policy.DefaultMode)
	fmt.Printf("  Default delay seconds: %d\n", response.Policy.DefaultDelaySeconds)
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "policy-update", "--mode", response.Policy.DefaultMode, "--delay-seconds", fmt.Sprintf("%d", response.Policy.DefaultDelaySeconds)),
	})
	return nil
}

func (a *App) cmdExecutionPolicyUpdate(args []string) error {
	fs := flag.NewFlagSet("execution policy update", flag.ContinueOnError)
	mode := fs.String("mode", "", "Default mode: manual|scheduled|yolo")
	delay := fs.Int64("delay-seconds", 300, "Default delay seconds for scheduled mode")
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	opts, err := parsePolicyOptions(mode, delay)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"default_mode":          opts.mode,
		"default_delay_seconds": opts.delaySeconds,
	})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}
	body, err := a.requestV1("PUT", "/execution/policy", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ExecutionPolicyResponse](body)
	if err != nil {
		return err
	}
	printSection("Result")
	fmt.Printf("  Updated execution policy\n")
	printSection("What Changed")
	fmt.Printf("  Default mode: %s\n", response.Policy.DefaultMode)
	fmt.Printf("  Default delay seconds: %d\n", response.Policy.DefaultDelaySeconds)
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "policy-get"),
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
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: execution %s <execution-id> [--json]", action)
	}
	executionID := strings.TrimSpace(fs.Arg(0))
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
		cliCommand("execution", "get", response.Execution.ExecutionID),
		cliCommand("execution", "list", "--status", response.Execution.Status),
	})
	return nil
}
