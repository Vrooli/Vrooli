// Responsibility: retain cmd runs declarations within their original package.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"os"
	"strings"
	"time"
)

func (a *App) runAttach(args []string) error {
	fs := flag.NewFlagSet("run attach", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	harnessKind := fs.String("harness-kind", "", "Stable harness kind, such as claude-code or codex")
	harnessSessionID := fs.String("harness-session-id", "", "Harness-owned session identifier")
	taskID := fs.String("task-id", "", "Optional task UUID to associate with the session")
	processID := fs.Int("process-id", 0, "Optional harness process ID")
	harnessTitle := fs.String("harness-title", "", "Optional title from harness transcript metadata")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*harnessKind) == "" || strings.TrimSpace(*harnessSessionID) == "" {
		return fmt.Errorf("usage: agent-manager run attach --harness-kind kind --harness-session-id id [--task-id uuid] [--process-id pid] [--harness-title title]")
	}
	if *processID < 0 {
		return fmt.Errorf("--process-id must be positive when provided")
	}
	req := &apipb.AttachRunRequest{
		HarnessKind:      *harnessKind,
		HarnessSessionId: *harnessSessionID,
	}
	if *taskID != "" {
		req.TaskId = proto.String(*taskID)
	}
	if *processID > 0 {
		value := int32(*processID)
		req.ProcessId = &value
	}
	if *harnessTitle != "" {
		req.HarnessTitle = proto.String(*harnessTitle)
	}
	body, response, err := a.services.Runs.Attach(req)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOutput || response == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	runID := ""
	if response.Run != nil {
		runID = response.Run.Id
	}
	fmt.Printf("Attached run: %s\nIdentity token: %s\n", runID, response.IdentityToken)
	if response.ExpiresAt != nil {
		fmt.Printf("Token expires: %s\n", response.ExpiresAt.AsTime().Format(time.RFC3339))
	}
	return nil
}

func (a *App) runDetach(args []string) error {
	fs := flag.NewFlagSet("run detach", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	reason := fs.String("reason", "", "Optional operator reason for detaching the session")
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: agent-manager run detach <id> [--reason text]")
	}
	body, response, err := a.services.Runs.Detach(id, *reason)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOutput || response == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	status := "terminal"
	if response.Run != nil {
		status = formatEnumValue(response.Run.Status, "RUN_STATUS_", "_")
	}
	fmt.Printf("Detached run: %s\nStatus: %s\n", id, status)
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
	executionMode := fs.String("execution-mode", "", "Execution mode (codec_pipe or interactive); interactive launches the real CLI in a live web-console session, supports sandbox tracking, and rejects protected sandbox mode")
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
	effort := fs.String("effort", "", "Reasoning effort (low, medium, high, xhigh, max)")
	model := fs.String("model", "", "Per-run model override")
	until := fs.String("until", "", "Engine-owned completion test for this run")
	workloadKey := fs.String("workload-key", "", "Stable workload key for grouping repeated work")
	workloadKind := fs.String("workload-kind", "", "Workload kind (workflow_node, scheduled, interactive, adhoc, imported)")
	parentRunID := fs.String("parent-run-id", "", "Parent run ID for durable child lineage")

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
	if *parentRunID != "" {
		req.ParentRunId = protoString(*parentRunID)
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
	if *effort != "" {
		if req.InlineConfig == nil {
			req.InlineConfig = &domainpb.RunConfigOverrides{}
		}
		req.InlineConfig.Effort = protoString(*effort)
	}
	if *model != "" {
		if req.InlineConfig == nil {
			req.InlineConfig = &domainpb.RunConfigOverrides{}
		}
		req.InlineConfig.Model = protoString(*model)
	}
	if *until != "" {
		if req.InlineConfig == nil {
			req.InlineConfig = &domainpb.RunConfigOverrides{}
		}
		req.InlineConfig.Until = protoString(*until)
	}
	if *workloadKey != "" || *workloadKind != "" {
		if req.Environment == nil {
			req.Environment = map[string]string{}
		}
		if *workloadKey != "" {
			req.Environment["VROOLI_WORKLOAD_KEY"] = *workloadKey
		}
		if *workloadKind != "" {
			req.Environment["VROOLI_WORKLOAD_KIND"] = *workloadKind
		}
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
