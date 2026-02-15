package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

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
		fmt.Println("No execution runs found.")
		return nil
	}

	fmt.Printf("Found %d execution run(s):\n\n", len(response.Items))
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
	return nil
}

func (a *App) cmdExecutionGet(args []string) error {
	if err := requireArgs(args, 1, "execution get <execution-id>"); err != nil {
		return err
	}
	executionID := strings.TrimSpace(args[0])
	body, err := a.getV1("/execution/"+executionID, nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdExecutionPolicyGet(_ []string) error {
	body, err := a.getV1("/execution/policy", nil)
	if err != nil {
		return err
	}
	var response ExecutionPolicyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	fmt.Printf("Default mode: %s\n", response.Policy.DefaultMode)
	fmt.Printf("Default delay seconds: %d\n", response.Policy.DefaultDelaySeconds)
	return nil
}

func (a *App) cmdExecutionPolicyUpdate(args []string) error {
	fs := flag.NewFlagSet("execution policy update", flag.ContinueOnError)
	mode := fs.String("mode", "", "Default mode: manual|scheduled|yolo")
	delay := fs.Int64("delay-seconds", 300, "Default delay seconds for scheduled mode")
	if err := fs.Parse(args); err != nil {
		return err
	}
	modeValue := strings.ToLower(strings.TrimSpace(*mode))
	if modeValue != "manual" && modeValue != "scheduled" && modeValue != "yolo" {
		return fmt.Errorf("mode is required and must be manual, scheduled, or yolo")
	}
	if *delay < 0 {
		return fmt.Errorf("delay-seconds must be >= 0")
	}
	payload, err := json.Marshal(map[string]any{
		"default_mode":          modeValue,
		"default_delay_seconds": *delay,
	})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}
	body, err := a.requestV1("PUT", "/execution/policy", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	var response ExecutionPolicyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	fmt.Printf("Updated execution policy:\n")
	fmt.Printf("  Default mode: %s\n", response.Policy.DefaultMode)
	fmt.Printf("  Default delay seconds: %d\n", response.Policy.DefaultDelaySeconds)
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
	if err := requireArgs(args, 1, "execution "+action+" <execution-id>"); err != nil {
		return err
	}
	executionID := strings.TrimSpace(args[0])
	body, err := a.requestV1("POST", "/execution/"+executionID+"/"+action, nil, nil)
	if err != nil {
		return err
	}

	var response ExecutionItemResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Execution %s: %s\n", action, response.Execution.ExecutionID)
	fmt.Printf("  Status: %s\n", response.Execution.Status)
	fmt.Printf("  Backlog: %s/%s\n", response.Execution.BacklogKind, response.Execution.BacklogName)
	return nil
}
