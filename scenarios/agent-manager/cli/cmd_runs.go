package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/proto"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// =============================================================================
// Run Command Dispatcher
// =============================================================================

func (a *App) cmdRun(args []string) error {
	if len(args) == 0 {
		return a.runHelp()
	}

	switch args[0] {
	case "list":
		return a.runList(args[1:])
	case "get":
		return a.runGet(args[1:])
	case "get-by-tag":
		return a.runGetByTag(args[1:])
	case "create":
		return a.runCreate(args[1:])
	case "delete":
		return a.runDelete(args[1:])
	case "stop":
		return a.runStop(args[1:])
	case "stop-by-tag":
		return a.runStopByTag(args[1:])
	case "stop-all":
		return a.runStopAll(args[1:])
	case "quiesce":
		return a.runQuiesce(args[1:])
	case "continue":
		return a.runContinue(args[1:])
	case "park":
		return a.runPark(args[1:])
	case "wake":
		return a.runWake(args[1:])
	case "await-result":
		return a.runAwaitResult(args[1:])
	case "recover":
		return a.runRecover(args[1:])
	case "investigate":
		return a.runInvestigate(args[1:])
	case "apply-investigation":
		return a.runApplyInvestigation(args[1:])
	case "sandbox-sync":
		return a.runSandboxSync(args[1:])
	case "approve":
		return a.runApprove(args[1:])
	case "reject":
		return a.runReject(args[1:])
	case "diff":
		return a.runDiff(args[1:])
	case "events":
		return a.runEvents(args[1:])
	case "help", "-h", "--help":
		return a.runHelp()
	default:
		return fmt.Errorf("unknown run subcommand: %s\n\nRun 'agent-manager run help' for usage", args[0])
	}
}

func (a *App) runHelp() error {
	fmt.Println(`Usage: agent-manager run <subcommand> [options]

Subcommands:
  list                        List runs (with optional filters)
  get <id>                    Get run details by UUID
  get-by-tag <tag>            Get run details by custom tag
  create                      Create and start a new run
  delete <id>                 Delete a run
  stop <id>                   Stop a run by UUID
  stop-by-tag <tag>           Stop a run by custom tag
  stop-all                    Stop all running runs
  quiesce --scenario <s>      Drain in-flight runs targeting a scenario (promote)
  continue <id>               Continue a run with a follow-up message
  park <id>                   Park a run on externally-owned async work (durable park/resume)
  wake <id>                   Wake a parked run with a result (ops/manual recovery)
  await-result <id>           Re-fetch a run's last awaited result (non-blocking; no re-run)
  recover <id>                Drain transcript and reconcile a run
  investigate                 Create an investigation run from existing runs
  apply-investigation <id>    Apply investigation recommendations
  sandbox-sync <id>           Sync run state from sandbox
  approve <id>                Approve run changes
  reject <id>                 Reject run changes
  diff <id>                   Show sandbox diff
  events <id>                 Get run events (--after-sequence for gap-fill, --follow for streaming)

Filters (for 'list'):
  --task-id         Filter by task ID
  --profile-id      Filter by profile ID
  --status          Filter by status (running, pending, complete, etc.)
  --tag-prefix      Filter by tag prefix (e.g., "ecosystem-")

Options:
  --json            Output raw JSON
  --quiet           Output only IDs (for piping)

Examples:
  agent-manager run list
  agent-manager run list --status running
  agent-manager run create --task-id abc123 --profile-id def456
  agent-manager run create --task-id abc123 --profile-id def456 --run-mode in_place --execution-mode interactive
  agent-manager run delete abc123 --force
  agent-manager run continue abc123 --message "Also update tests"
  agent-manager run recover abc123
  agent-manager run investigate --run-ids id1,id2 --depth standard
  agent-manager run apply-investigation abc123
  agent-manager run events xyz789 --after-sequence 42 --limit 100
  agent-manager run events xyz789 --follow`)
	return nil
}

// =============================================================================
// Run List
// =============================================================================

func (a *App) runList(args []string) error {
	fs := flag.NewFlagSet("run list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	quiet := fs.Bool("quiet", false, "Output only IDs")
	limit := fs.Int("limit", 0, "Maximum number of runs to return")
	offset := fs.Int("offset", 0, "Number of runs to skip")
	taskID := fs.String("task-id", "", "Filter by task ID")
	profileID := fs.String("profile-id", "", "Filter by profile ID")
	status := fs.String("status", "", "Filter by status")
	tagPrefix := fs.String("tag-prefix", "", "Filter by tag prefix")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, runs, err := a.services.Runs.List(*limit, *offset, *taskID, *profileID, *status, *tagPrefix)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if runs == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if *quiet {
		for _, r := range runs {
			fmt.Println(r.Id)
		}
		return nil
	}

	if len(runs) == 0 {
		fmt.Println("No runs found")
		return nil
	}

	fmt.Printf("%-36s  %-12s  %-18s  %-11s  %-4s  %-20s\n", "ID", "STATUS", "PHASE", "EXEC", "PROG", "UPDATED")
	fmt.Printf("%-36s  %-12s  %-18s  %-11s  %-4s  %-20s\n", strings.Repeat("-", 36), strings.Repeat("-", 12), strings.Repeat("-", 18), strings.Repeat("-", 11), strings.Repeat("-", 4), strings.Repeat("-", 20))
	for _, r := range runs {
		phase := formatEnumValue(r.Phase, "RUN_PHASE_", "_")
		if len(phase) > 18 {
			phase = phase[:15] + "..."
		}
		updated := formatTimestamp(r.UpdatedAt)
		if len(updated) > 20 {
			updated = updated[:19]
		}
		progress := fmt.Sprintf("%d%%", r.ProgressPercent)
		status := formatEnumValue(r.Status, "RUN_STATUS_", "_")
		exec := formatEnumValue(r.ExecutionMode, "EXECUTION_MODE_", "_")
		if exec == "" || exec == "unspecified" {
			exec = "codec_pipe"
		}
		fmt.Printf("%-36s  %-12s  %-18s  %-11s  %-4s  %-20s\n", r.Id, status, phase, exec, progress, updated)
	}

	return nil
}

// =============================================================================
// Run Get
// =============================================================================

func (a *App) runGet(args []string) error {
	fs := flag.NewFlagSet("run get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		remaining := fs.Args()
		if len(remaining) == 0 {
			return fmt.Errorf("usage: agent-manager run get <id>")
		}
		id = remaining[0]
	}

	body, run, err := a.services.Runs.Get(id)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("ID:              %s\n", run.Id)
	fmt.Printf("Task ID:         %s\n", run.TaskId)
	if run.AgentProfileId != nil {
		fmt.Printf("Profile ID:      %s\n", run.GetAgentProfileId())
	}
	fmt.Printf("Status:          %s\n", formatEnumValue(run.Status, "RUN_STATUS_", "_"))
	fmt.Printf("Phase:           %s\n", formatEnumValue(run.Phase, "RUN_PHASE_", "_"))
	fmt.Printf("Progress:        %d%%\n", run.ProgressPercent)
	fmt.Printf("Run Mode:        %s\n", formatEnumValue(run.RunMode, "RUN_MODE_", "_"))
	fmt.Printf("Execution Mode:  %s\n", formatEnumValue(run.ExecutionMode, "EXECUTION_MODE_", "_"))
	if run.WebConsoleSessionId != "" {
		fmt.Printf("Live Session:    %s\n", run.WebConsoleSessionId)
		if run.WebConsoleSessionUrl != "" {
			fmt.Printf("Live Session URL: %s\n", run.WebConsoleSessionUrl)
		}
	}
	if run.SandboxId != nil && run.GetSandboxId() != "" {
		fmt.Printf("Sandbox ID:      %s\n", run.GetSandboxId())
	}
	if started := formatTimestamp(run.StartedAt); started != "" {
		fmt.Printf("Started:         %s\n", started)
	}
	if ended := formatTimestamp(run.EndedAt); ended != "" {
		fmt.Printf("Ended:           %s\n", ended)
	}
	if approval := formatEnumValue(run.ApprovalState, "APPROVAL_STATE_", "_"); approval != "" && approval != "none" {
		fmt.Printf("Approval State:  %s\n", approval)
		if run.ApprovedBy != "" {
			fmt.Printf("Approved By:     %s\n", run.ApprovedBy)
		}
	}
	if run.Summary != nil {
		fmt.Println("Summary:")
		if run.Summary.Description != "" {
			fmt.Printf("  Description:   %s\n", run.Summary.Description)
		}
		if run.Summary.TurnsUsed > 0 {
			fmt.Printf("  Turns Used:    %d\n", run.Summary.TurnsUsed)
		}
		if run.Summary.TokensUsed > 0 {
			fmt.Printf("  Tokens Used:   %d\n", run.Summary.TokensUsed)
		}
		if run.Summary.ContextTokens > 0 {
			fmt.Printf("  Context:       %d tokens\n", run.Summary.ContextTokens)
		}
		if run.Summary.CostEstimate > 0 {
			fmt.Printf("  Cost Estimate: $%.4f\n", run.Summary.CostEstimate)
		}
	}
	if run.Result != nil {
		if run.Result.Selection != nil {
			fmt.Printf("Result Selection: %s\n", formatEnumValue(run.Result.Selection.Status, "FINAL_OUTPUT_SELECTION_STATUS_", "_"))
		} else {
			fmt.Println("Result Selection: unavailable")
		}
		if run.Result.Structured != nil {
			fmt.Printf("Structured:      %s", formatEnumValue(run.Result.Structured.Status, "STRUCTURED_RESULT_STATUS_", "_"))
			if run.Result.Structured.Method != "" {
				fmt.Printf(" (%s)", run.Result.Structured.Method)
			}
			fmt.Println()
			if len(run.Result.Structured.Value) > 0 {
				fmt.Printf("Structured Value: %s\n", string(run.Result.Structured.Value))
			}
		}
	}
	if run.ErrorMsg != "" {
		fmt.Printf("Error:           %s\n", run.ErrorMsg)
	}
	if run.ExitCode != nil {
		fmt.Printf("Exit Code:       %d\n", run.GetExitCode())
	}
	if run.ChangedFiles > 0 {
		fmt.Printf("Changed Files:   %d\n", run.ChangedFiles)
	}

	return nil
}

// =============================================================================
// Run Create
// =============================================================================

func (a *App) runCreate(args []string) error {
	fs := flag.NewFlagSet("run create", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	taskID := fs.String("task-id", "", "Task ID (required)")
	profileID := fs.String("profile-id", "", "Agent profile ID (required)")
	prompt := fs.String("prompt", "", "Optional override prompt")
	runMode := fs.String("run-mode", "", "Run mode (sandboxed or in_place)")
	executionMode := fs.String("execution-mode", "", "Execution mode (codec_pipe or interactive); interactive launches the real CLI in a live web-console session and is rejected for protected/sandboxed runs")
	forceInPlace := fs.Bool("force-in-place", false, "Force in-place execution")
	idempotencyKey := fs.String("idempotency-key", "", "Idempotency key for safe retries")
	existingSandboxID := fs.String("existing-sandbox-id", "", "Reuse an existing sandbox ID (sandboxed runs only)")
	sandboxConfig := fs.String("sandbox-config", "", "Sandbox config JSON (proto JSON)")
	sandboxConfigFile := fs.String("sandbox-config-file", "", "Path to sandbox config JSON")
	sandboxRetentionMode := fs.String("sandbox-retention-mode", "", "Sandbox retention mode (keep_active, stop_on_terminal, delete_on_terminal)")
	sandboxRetentionTTL := fs.String("sandbox-retention-ttl", "", "Sandbox retention TTL (e.g., 2h, 30m)")
	resultSchema := fs.String("result-schema", "", "JSON Schema for the canonical structured result")
	resultSchemaFile := fs.String("result-schema-file", "", "Path to a JSON Schema for the canonical structured result")
	classify := fs.String("classify", "", "Comma-separated classification values (convenience ResultSpec)")
	structuredExtraction := fs.Bool("structured-extraction", false, "Allow portable extract.structured fallback after deterministic parsing")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *taskID == "" {
		return fmt.Errorf("--task-id is required")
	}
	if *profileID == "" {
		return fmt.Errorf("--profile-id is required")
	}

	req := &apipb.CreateRunRequest{
		TaskId: *taskID,
	}
	if *profileID != "" {
		req.AgentProfileId = protoString(*profileID)
	}
	if *prompt != "" {
		req.Prompt = protoString(*prompt)
	}
	if *idempotencyKey != "" {
		req.IdempotencyKey = protoString(*idempotencyKey)
	}
	if *existingSandboxID != "" {
		req.ExistingSandboxId = protoString(*existingSandboxID)
	}
	if *runMode != "" {
		mode := parseRunMode(*runMode)
		if mode == domainpb.RunMode_RUN_MODE_UNSPECIFIED {
			return fmt.Errorf("invalid run mode: %s", *runMode)
		}
		req.RunMode = &mode
	} else if *forceInPlace {
		mode := domainpb.RunMode_RUN_MODE_IN_PLACE
		req.RunMode = &mode
	}
	if *executionMode != "" {
		mode := parseExecutionMode(*executionMode)
		if mode == domainpb.ExecutionMode_EXECUTION_MODE_UNSPECIFIED {
			return fmt.Errorf("invalid execution mode: %s (want codec_pipe or interactive)", *executionMode)
		}
		req.ExecutionMode = &mode
	}
	spec, err := parseResultSpec(*resultSchema, *resultSchemaFile, *classify, *structuredExtraction)
	if err != nil {
		return err
	}
	if spec != nil {
		req.InlineConfig = &domainpb.RunConfigOverrides{ResultSpec: spec}
	}
	if cfg, err := parseSandboxConfig(*sandboxConfig, *sandboxConfigFile); err != nil {
		return err
	} else {
		cfg, err = applySandboxRetention(cfg, *sandboxRetentionMode, *sandboxRetentionTTL)
		if err != nil {
			return err
		}
		if cfg != nil {
			if req.InlineConfig == nil {
				req.InlineConfig = &domainpb.RunConfigOverrides{}
			}
			req.InlineConfig.SandboxConfig = cfg
		}
	}

	body, run, err := a.services.Runs.Create(req)
	if err != nil {
		return apiError(body, err)
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Created run: %s\n", run.Id)
	fmt.Printf("Status: %s\n", formatEnumValue(run.Status, "RUN_STATUS_", "_"))
	fmt.Printf("Phase: %s\n", formatEnumValue(run.Phase, "RUN_PHASE_", "_"))
	fmt.Printf("Execution Mode: %s\n", formatEnumValue(run.ExecutionMode, "EXECUTION_MODE_", "_"))
	if run.WebConsoleSessionId != "" {
		fmt.Printf("Live Session: %s\n", run.WebConsoleSessionId)
		if run.WebConsoleSessionUrl != "" {
			fmt.Printf("Live Session URL: %s\n", run.WebConsoleSessionUrl)
		}
	}
	return nil
}

func parseResultSpec(schemaText, schemaFile, classification string, extraction bool) (*domainpb.ResultSpec, error) {
	configured := 0
	if strings.TrimSpace(schemaText) != "" {
		configured++
	}
	if strings.TrimSpace(schemaFile) != "" {
		configured++
	}
	if strings.TrimSpace(classification) != "" {
		configured++
	}
	if configured == 0 {
		return nil, nil
	}
	if configured > 1 {
		return nil, fmt.Errorf("use exactly one of --result-schema, --result-schema-file, or --classify")
	}
	mode := domainpb.StructuredExtractionMode_STRUCTURED_EXTRACTION_MODE_DETERMINISTIC_ONLY
	role := ""
	if extraction {
		mode = domainpb.StructuredExtractionMode_STRUCTURED_EXTRACTION_MODE_CONSTRAINED_FALLBACK
		role = "extract.structured"
	}
	spec := &domainpb.ResultSpec{Version: "result-spec/v1", ExtractionMode: mode, ExtractionRole: role}
	if strings.TrimSpace(classification) != "" {
		spec.Kind = domainpb.ResultSpecKind_RESULT_SPEC_KIND_CLASSIFICATION
		for _, value := range strings.Split(classification, ",") {
			if value = strings.TrimSpace(value); value != "" {
				spec.ClassificationValues = append(spec.ClassificationValues, value)
			}
		}
		if len(spec.ClassificationValues) == 0 {
			return nil, fmt.Errorf("--classify must contain at least one non-empty value")
		}
		return spec, nil
	}
	raw := []byte(schemaText)
	if strings.TrimSpace(schemaFile) != "" {
		var err error
		raw, err = os.ReadFile(schemaFile)
		if err != nil {
			return nil, fmt.Errorf("read result schema: %w", err)
		}
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("result schema is not valid JSON")
	}
	spec.Kind = domainpb.ResultSpecKind_RESULT_SPEC_KIND_JSON_SCHEMA
	spec.Schema = raw
	return spec, nil
}

// =============================================================================
// Run Stop
// =============================================================================

func (a *App) runStop(args []string) error {
	fs := flag.NewFlagSet("run stop", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run stop <id>")
	}

	body, resp, err := a.services.Runs.Stop(id)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	changes := []string{fmt.Sprintf("run_id=%s", id)}
	nextCommandID := id
	if resp != nil && resp.Run != nil {
		nextCommandID = resp.Run.Id
		changes = append(changes,
			fmt.Sprintf("status=%s", formatEnumValue(resp.Run.Status, "RUN_STATUS_", "_")),
			fmt.Sprintf("phase=%s", formatEnumValue(resp.Run.Phase, "RUN_PHASE_", "_")),
		)
	} else if resp != nil && resp.Status != "" {
		changes = append(changes, fmt.Sprintf("status=%s", resp.Status))
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{"Run stop requested"},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("agent-manager run get %s", nextCommandID)},
	})
}

// =============================================================================
// Run Get By Tag
// =============================================================================

func (a *App) runGetByTag(args []string) error {
	fs := flag.NewFlagSet("run get-by-tag", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	// Parse with positional tag first
	var tag string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		tag = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if tag == "" {
		return fmt.Errorf("usage: agent-manager run get-by-tag <tag>")
	}

	body, run, err := a.services.Runs.GetByTag(tag)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("ID:              %s\n", run.Id)
	fmt.Printf("Tag:             %s\n", run.Tag)
	fmt.Printf("Task ID:         %s\n", run.TaskId)
	fmt.Printf("Status:          %s\n", formatEnumValue(run.Status, "RUN_STATUS_", "_"))
	fmt.Printf("Phase:           %s\n", formatEnumValue(run.Phase, "RUN_PHASE_", "_"))
	fmt.Printf("Progress:        %d%%\n", run.ProgressPercent)
	if started := formatTimestamp(run.StartedAt); started != "" {
		fmt.Printf("Started:         %s\n", started)
	}
	if run.ErrorMsg != "" {
		fmt.Printf("Error:           %s\n", run.ErrorMsg)
	}

	return nil
}

// =============================================================================
// Run Stop By Tag
// =============================================================================

func (a *App) runStopByTag(args []string) error {
	fs := flag.NewFlagSet("run stop-by-tag", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	// Parse with positional tag first
	var tag string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		tag = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if tag == "" {
		return fmt.Errorf("usage: agent-manager run stop-by-tag <tag>")
	}

	body, resp, err := a.services.Runs.StopByTag(tag)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	changes := []string{fmt.Sprintf("tag=%s", tag)}
	nextCommand := fmt.Sprintf("agent-manager run get-by-tag %s", tag)
	if resp != nil && resp.Run != nil {
		changes = append(changes,
			fmt.Sprintf("run_id=%s", resp.Run.Id),
			fmt.Sprintf("status=%s", formatEnumValue(resp.Run.Status, "RUN_STATUS_", "_")),
			fmt.Sprintf("phase=%s", formatEnumValue(resp.Run.Phase, "RUN_PHASE_", "_")),
		)
		nextCommand = fmt.Sprintf("agent-manager run get %s", resp.Run.Id)
	} else if resp != nil && resp.Status != "" {
		changes = append(changes, fmt.Sprintf("status=%s", resp.Status))
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{"Run stop requested"},
		Changes:     changes,
		NextCommand: []string{nextCommand},
	})
}

// =============================================================================
// Run Stop All
// =============================================================================

func (a *App) runStopAll(args []string) error {
	fs := flag.NewFlagSet("run stop-all", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	tagPrefix := fs.String("tag-prefix", "", "Only stop runs with this tag prefix")
	force := fs.Bool("force", false, "Force termination even if graceful stop fails")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	req := &apipb.StopAllRunsRequest{Force: *force}
	if *tagPrefix != "" {
		req.TagPrefix = protoString(*tagPrefix)
	}
	body, result, err := a.services.Runs.StopAll(req)
	if err != nil {
		return err
	}

	if *jsonOutput || result == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if result != nil {
		fmt.Printf("Stopped:  %d\n", result.StoppedCount)
		if len(result.Failures) > 0 {
			fmt.Printf("Failed:   %d\n", len(result.Failures))
			failedIDs := make([]string, 0, len(result.Failures))
			for _, failure := range result.Failures {
				if failure == nil {
					continue
				}
				failedIDs = append(failedIDs, failure.RunId)
			}
			if len(failedIDs) > 0 {
				fmt.Printf("Failed IDs: %v\n", failedIDs)
			}
		}
	}
	return nil
}

// =============================================================================
// Run Quiesce (Baseline Modes promote drain)
// =============================================================================

func (a *App) runQuiesce(args []string) error {
	fs := flag.NewFlagSet("run quiesce", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	scenario := fs.String("scenario", "", "Target scenario slug to quiesce (required)")
	scopePrefix := fs.String("scope-prefix", "", "Override the working-tree scope (default scenarios/<scenario>)")
	tagPrefix := fs.String("tag-prefix", "", "Also enumerate in-flight runs by this tag prefix (whole-repo runs)")
	excludeRun := fs.String("exclude-run", "", "The promoting run's own ID, excluded from the drain set")
	timeout := fs.String("timeout", "", "Max wait for in-flight runs to terminate (e.g. 5m)")
	force := fs.Bool("force", false, "On timeout, cancel survivors instead of aborting")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *scenario == "" {
		return fmt.Errorf("--scenario is required")
	}

	req := &apipb.QuiesceScenarioRequest{Scenario: *scenario, Force: *force}
	if *scopePrefix != "" {
		req.ScopePrefix = protoString(*scopePrefix)
	}
	if *tagPrefix != "" {
		req.TagPrefix = protoString(*tagPrefix)
	}
	if *excludeRun != "" {
		req.ExcludeRunId = protoString(*excludeRun)
	}
	if *timeout != "" {
		req.Timeout = protoString(*timeout)
	}

	body, result, err := a.services.Runs.Quiesce(req)
	if err != nil {
		return err
	}

	if *jsonOutput || result == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if result.Drained {
		fmt.Printf("✓ %s drained — safe to promote\n", result.Scenario)
	} else if result.Aborted {
		fmt.Printf("✗ %s NOT drained — %d run(s) still in-flight\n", result.Scenario, len(result.InFlight))
	}
	if len(result.Cancelled) > 0 {
		fmt.Printf("Cancelled: %d run(s)\n", len(result.Cancelled))
	}
	for _, ref := range result.InFlight {
		fmt.Printf("  in-flight: %s [%s] %s\n", ref.Id, ref.Status, ref.Tag)
	}
	if result.Reason != "" {
		fmt.Printf("→ %s\n", result.Reason)
	}
	return nil
}

// =============================================================================
// Run Approve
// =============================================================================

func (a *App) runApprove(args []string) error {
	fs := flag.NewFlagSet("run approve", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	actor := fs.String("actor", "", "Who is approving")
	commitMsg := fs.String("commit-msg", "", "Commit message for changes")
	force := fs.Bool("force", false, "Force approval despite conflicts")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run approve <id> [options]")
	}

	req := &apipb.ApproveRunRequest{
		RunId: id,
		Force: *force,
	}
	if trimmed := strings.TrimSpace(*actor); trimmed != "" {
		req.Actor = protoString(trimmed)
	}
	if *commitMsg != "" {
		req.CommitMsg = protoString(*commitMsg)
	}

	body, result, err := a.services.Runs.Approve(id, req)
	if err != nil {
		return err
	}

	if *jsonOutput || result == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if result != nil && result.Success {
		fmt.Printf("Approved run: %s\n", id)
		fmt.Printf("Applied: %d files\n", result.FilesApplied)
		if result.CommitHash != "" {
			fmt.Printf("Commit: %s\n", result.CommitHash)
		}
	} else {
		message := ""
		if result != nil {
			message = result.Message
		}
		fmt.Printf("Approval failed: %s\n", message)
	}
	return nil
}

// =============================================================================
// Run Reject
// =============================================================================

func (a *App) runReject(args []string) error {
	fs := flag.NewFlagSet("run reject", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	actor := fs.String("actor", "", "Who is rejecting")
	reason := fs.String("reason", "", "Reason for rejection")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run reject <id> [options]")
	}

	req := &apipb.RejectRunRequest{
		RunId:  id,
		Reason: *reason,
	}
	if trimmed := strings.TrimSpace(*actor); trimmed != "" {
		req.Actor = protoString(trimmed)
	}

	body, err := a.services.Runs.Reject(id, req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Rejected run: %s\n", id)
	return nil
}

// =============================================================================
// Run Diff
// =============================================================================

func (a *App) runDiff(args []string) error {
	fs := flag.NewFlagSet("run diff", flag.ContinueOnError)

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run diff <id>")
	}

	body, diff, err := a.services.Runs.GetDiff(id)
	if err != nil {
		return err
	}

	// Just print the diff output directly
	if diff != nil && diff.Content != "" {
		fmt.Println(diff.Content)
	} else if diff != nil && len(diff.Files) > 0 {
		fmt.Println("No unified diff content available. Changed files:")
		for _, file := range diff.Files {
			fmt.Printf("- %s (%s, +%d -%d)\n", file.Path, file.ChangeType, file.Additions, file.Deletions)
		}
	} else {
		fmt.Println(string(body))
	}
	return nil
}

// =============================================================================
// Run Events
// =============================================================================

func (a *App) runEvents(args []string) error {
	fs := flag.NewFlagSet("run events", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	follow := fs.Bool("follow", false, "Stream events in real-time (WebSocket)")
	limit := fs.Int("limit", 0, "Maximum number of events to return")
	afterSequence := fs.Int64("after-sequence", -1, "Only return events with sequence greater than this value")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run events <id> [--follow] [--after-sequence N] [--limit N]")
	}

	if *follow {
		return a.streamEvents(id)
	}

	var after *int64
	if *afterSequence >= 0 {
		after = afterSequence
	}
	body, events, err := a.services.Runs.GetEvents(id, *limit, after)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if events == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(events) == 0 {
		fmt.Println("No events found")
		return nil
	}

	fmt.Printf("%-6s  %-12s  %-24s  %s\n", "SEQ", "TYPE", "TIMESTAMP", "DATA")
	fmt.Printf("%-6s  %-12s  %-24s  %s\n", strings.Repeat("-", 6), strings.Repeat("-", 12), strings.Repeat("-", 24), strings.Repeat("-", 40))
	for _, e := range events {
		dataStr := runEventDataString(e)
		if len(dataStr) > 60 {
			dataStr = dataStr[:57] + "..."
		}
		timestamp := formatTimestamp(e.Timestamp)
		if timestamp != "" {
			timestamp = trimTimestamp(timestamp)
		}
		eventType := formatEnumValue(e.EventType, "RUN_EVENT_TYPE_", "_")
		fmt.Printf("%-6d  %-12s  %-24s  %s\n", e.Sequence, eventType, timestamp, dataStr)
	}

	return nil
}

// streamEvents is implemented in cmd_events.go

func runEventDataString(event *domainpb.RunEvent) string {
	if event == nil {
		return ""
	}

	var payload proto.Message
	switch data := event.Data.(type) {
	case *domainpb.RunEvent_Log:
		payload = data.Log
	case *domainpb.RunEvent_Message:
		payload = data.Message
	case *domainpb.RunEvent_MessageDeleted:
		payload = data.MessageDeleted
	case *domainpb.RunEvent_ToolCall:
		payload = data.ToolCall
	case *domainpb.RunEvent_ToolResult:
		payload = data.ToolResult
	case *domainpb.RunEvent_Status:
		payload = data.Status
	case *domainpb.RunEvent_Metric:
		payload = data.Metric
	case *domainpb.RunEvent_Artifact:
		payload = data.Artifact
	case *domainpb.RunEvent_Error:
		payload = data.Error
	case *domainpb.RunEvent_Progress:
		payload = data.Progress
	case *domainpb.RunEvent_Cost:
		payload = data.Cost
	case *domainpb.RunEvent_RateLimit:
		payload = data.RateLimit
	case *domainpb.RunEvent_Compaction:
		payload = data.Compaction
	default:
		return ""
	}

	return marshalProtoJSON(payload)
}

// =============================================================================
// Run Delete
// =============================================================================

func (a *App) runDelete(args []string) error {
	fs := flag.NewFlagSet("run delete", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run delete <id>")
	}

	if !*force {
		fmt.Printf("Delete run %s? [y/N]: ", id)
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := a.services.Runs.Delete(id); err != nil {
		return err
	}

	fmt.Printf("Deleted run: %s\n", id)
	return nil
}

// =============================================================================
// Run Continue
// =============================================================================

func (a *App) runContinue(args []string) error {
	fs := flag.NewFlagSet("run continue", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	message := fs.String("message", "", "Follow-up message (required)")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run continue <id> --message <message>")
	}

	if *message == "" {
		return fmt.Errorf("--message is required")
	}

	req := &domainpb.ContinueRunRequest{
		RunId:   id,
		Message: *message,
	}

	body, run, err := a.services.Runs.Continue(id, req)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Continued run: %s (status: %s)\n", run.Id, formatEnumValue(run.Status, "RUN_STATUS_", "_"))
	return nil
}

// runPark parks a run on externally-owned async work. Invoked from inside an
// agent-manager-controlled run; it authenticates with the run's identity token
// (VROOLI_AGENT_IDENTITY_TOKEN by default) so the server can confirm the caller
// owns the run it is parking.
func (a *App) runPark(args []string) error {
	fs := flag.NewFlagSet("run park", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	producer := fs.String("producer", "", "Producer that resolves the await (e.g. test-genie, git-control-tower) (required)")
	key := fs.String("key", "", "Producer-scoped identifier of the awaited work (required)")
	deadlineUnix := fs.Int64("deadline-unix", 0, "Optional wait deadline as a Unix timestamp (seconds); 0 = default TTL")
	identityToken := fs.String("identity-token", "", "Owning run's identity token (defaults to $"+cliutil.EnvIdentityToken+")")

	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: agent-manager run park <id> --producer <p> --key <k>")
	}
	if *producer == "" || *key == "" {
		return fmt.Errorf("--producer and --key are required")
	}

	token := *identityToken
	if token == "" {
		token = os.Getenv(cliutil.EnvIdentityToken)
	}
	if token == "" {
		return fmt.Errorf("no identity token: set --identity-token or run inside an agent-manager run ($%s)", cliutil.EnvIdentityToken)
	}

	req := &domainpb.ParkRunRequest{
		RunId:         id,
		Producer:      *producer,
		Key:           *key,
		DeadlineUnix:  *deadlineUnix,
		IdentityToken: token,
	}

	body, resp, err := a.services.Runs.Park(id, req)
	if err != nil {
		return err
	}
	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	// The message is the clean turn-ending tool-result the agent should see.
	fmt.Println(resp.Message)
	return nil
}

// runWake wakes a parked run with a result injected as its next turn. This is an
// ops/manual-recovery verb; normal wake is driven by agent-manager's waiter.
func (a *App) runWake(args []string) error {
	fs := flag.NewFlagSet("run wake", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	result := fs.String("result", "", "Awaited result injected as the next turn")
	timedOut := fs.Bool("timed-out", false, "Frame the result as a park-deadline timeout")

	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: agent-manager run wake <id> [--result <r>] [--timed-out]")
	}

	req := &domainpb.WakeRunRequest{
		RunId:    id,
		Result:   *result,
		TimedOut: *timedOut,
	}

	body, resp, err := a.services.Runs.Wake(id, req)
	if err != nil {
		return err
	}
	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if !resp.Success {
		fmt.Println("Run was not parked (no-op); wake is idempotent")
		return nil
	}
	if resp.Run != nil {
		fmt.Printf("Woke run: %s (status: %s)\n", resp.Run.Id, formatEnumValue(resp.Run.Status, "RUN_STATUS_", "_"))
	} else {
		fmt.Println("Wake requested")
	}
	return nil
}

// runAwaitResult re-fetches a run's most recently resolved await result without
// re-running the blocking producer. This is the deterministic retrieval path a
// woken agent uses if it did not receive — or wants to re-read — the result.
func (a *App) runAwaitResult(args []string) error {
	fs := flag.NewFlagSet("run await-result", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: agent-manager run await-result <id>")
	}

	body, resp, err := a.services.Runs.AwaitResult(id)
	if err != nil {
		return err
	}
	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if !resp.Found {
		fmt.Println("No awaited result recorded for this run.")
		return nil
	}
	if resp.Key != "" {
		fmt.Printf("Awaited: %s\n", resp.Key)
	}
	if resp.ResolvedAt != "" {
		fmt.Printf("Resolved at: %s\n", resp.ResolvedAt)
	}
	fmt.Printf("\nResult:\n%s\n", resp.Result)
	return nil
}

func (a *App) runRecover(args []string) error {
	fs := flag.NewFlagSet("run recover", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: agent-manager run recover <id>")
	}

	body, resp, err := a.services.Runs.Recover(id)
	if err != nil {
		return err
	}
	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Recovered:       %t\n", resp.Recovered)
	fmt.Printf("Idempotent:      %t\n", resp.Idempotent)
	if resp.Message != "" {
		fmt.Printf("Message:         %s\n", resp.Message)
	}
	if resp.Run != nil {
		fmt.Printf("Run ID:          %s\n", resp.Run.Id)
		fmt.Printf("Status:          %s\n", formatEnumValue(resp.Run.Status, "RUN_STATUS_", "_"))
	}
	return nil
}

// =============================================================================
// Run Investigate
// =============================================================================

func (a *App) runInvestigate(args []string) error {
	fs := flag.NewFlagSet("run investigate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	runIDs := fs.String("run-ids", "", "Comma-separated run IDs to investigate (required)")
	customContext := fs.String("context", "", "Custom context for investigation")
	depth := fs.String("depth", "standard", "Investigation depth: quick, standard, deep")
	projectRoot := fs.String("project-root", "", "Project root directory")
	scopePaths := fs.String("scope-paths", "", "Comma-separated scope paths")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *runIDs == "" {
		return fmt.Errorf("--run-ids is required")
	}

	ids := strings.Split(*runIDs, ",")
	for i, id := range ids {
		ids[i] = strings.TrimSpace(id)
	}

	req := map[string]interface{}{
		"runIds": ids,
	}
	if *customContext != "" {
		req["customContext"] = *customContext
	}
	if *depth != "" {
		req["depth"] = *depth
	}
	if *projectRoot != "" {
		req["projectRoot"] = *projectRoot
	}
	if *scopePaths != "" {
		paths := strings.Split(*scopePaths, ",")
		for i, p := range paths {
			paths[i] = strings.TrimSpace(p)
		}
		req["scopePaths"] = paths
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}

	body, run, err := a.services.Runs.Investigate(payload)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Created investigation run: %s\n", run.Id)
	return nil
}

// =============================================================================
// Run Apply Investigation
// =============================================================================

func (a *App) runApplyInvestigation(args []string) error {
	fs := flag.NewFlagSet("run apply-investigation", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	customContext := fs.String("context", "", "Custom context for apply run")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run apply-investigation <investigation-run-id>")
	}

	req := map[string]interface{}{
		"investigationRunId": id,
	}
	if *customContext != "" {
		req["customContext"] = *customContext
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}

	body, run, err := a.services.Runs.InvestigationApply(payload)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Created apply run: %s\n", run.Id)
	return nil
}

// =============================================================================
// Run Sandbox Sync
// =============================================================================

func (a *App) runSandboxSync(args []string) error {
	fs := flag.NewFlagSet("run sandbox-sync", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	status := fs.String("status", "", "Status to sync (required)")
	sandboxID := fs.String("sandbox-id", "", "Sandbox ID")
	actor := fs.String("actor", "", "Actor identifier")
	reason := fs.String("reason", "", "Reason for sync")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run sandbox-sync <id> --status <status>")
	}

	if *status == "" {
		return fmt.Errorf("--status is required")
	}

	req := map[string]interface{}{
		"runId":  id,
		"status": *status,
	}
	if *sandboxID != "" {
		req["sandboxId"] = *sandboxID
	}
	if *actor != "" {
		req["actor"] = *actor
	}
	if *reason != "" {
		req["reason"] = *reason
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}

	body, err := a.services.Runs.SandboxSync(id, payload)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Synced run: %s\n", id)
	return nil
}
