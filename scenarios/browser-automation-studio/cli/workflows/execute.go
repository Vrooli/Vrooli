package workflows

import (
	"browser-automation-studio/cli/internal/appctx"
	"browser-automation-studio/cli/internal/export"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

func printExecuteHelp(cliName string) {
	fmt.Printf("Usage: %s workflow execute [workflow-id|name] [options]\n\n", cliName)
	fmt.Println("Execute a browser automation workflow.")
	fmt.Println()
	fmt.Println("Workflow Sources (use one):")
	fmt.Println("  <workflow-id|name>       Execute a stored workflow by ID or name")
	fmt.Println("  --from-file <path>       Execute workflow from a JSON file")
	fmt.Println("  --step <type> [args...]  Build and execute inline workflow (repeatable)")
	fmt.Println()
	fmt.Println("Execution Options:")
	fmt.Println("  --wait                   Wait for execution to complete")
	fmt.Println("  --output <dir>           Export results to directory (implies --wait)")
	fmt.Println("  --project-root <path>    Base path for resolving subflow workflow_path")
	fmt.Println("  --start-url <url>        Starting URL for workflows without navigate step")
	fmt.Println()
	fmt.Println("Parameters:")
	fmt.Println("  --params <json>          Execution parameters (top-level fields)")
	fmt.Println("  --initial-params <json>  Workflow input parameters (@params/ namespace)")
	fmt.Println("  --initial-store <json>   Pre-seeded runtime store (@store/ namespace)")
	fmt.Println("  --env <json>             Environment variables (@env/ namespace)")
	fmt.Println()
	fmt.Println("Session Management:")
	fmt.Println("  --session-profile <name> Load session profile's cookies/localStorage")
	fmt.Println("  --save-session <name>    Save browser state to profile after execution")
	fmt.Println("  --fresh-session          Ignore any saved session state")
	fmt.Println("  --restore-tabs           Restore tabs from session profile (default: false)")
	fmt.Println()
	fmt.Println("Artifact Collection:")
	fmt.Println("  --record-video           Capture video during execution")
	fmt.Println("  --record-trace           Capture Playwright trace")
	fmt.Println("  --record-har             Capture HAR network archive")
	fmt.Println()
	fmt.Println("Test Data:")
	fmt.Println("  --seed <mode>            Seed data mode: 'applied' or 'needs-applying'")
	fmt.Println("  --seed-scenario <name>   Scenario to apply seed from")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s workflow execute my-workflow --wait\n", cliName)
	fmt.Printf("  %s workflow execute --from-file bas/cases/login.json --output /tmp/results\n", cliName)
	fmt.Printf("  %s workflow execute --step navigate \"http://example.com\" --step screenshot\n", cliName)
	fmt.Printf("  %s workflow execute --from-file bas/flows/checkout.json --session-profile \"Dev Account\"\n", cliName)
	fmt.Printf("  %s workflow execute --from-file bas/actions/login.json --save-session \"Dev Account\"\n", cliName)
	fmt.Println()
}

func runExecute(ctx *appctx.Context, args []string) error {
	// Check for help flag first (before step parsing)
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printExecuteHelp(ctx.Name)
			return nil
		}
	}

	// Parse --step flags first (before other flag parsing)
	steps, remainingArgs, err := ParseSteps(args)
	if err != nil {
		return fmt.Errorf("invalid step: %w", err)
	}
	args = remainingArgs

	workflow := ""

	paramsRaw := "{}"
	initialParamsRaw := ""
	initialStoreRaw := ""
	envRaw := ""
	wait := false
	outputDir := ""  // Legacy: for --output-screenshots (deprecated)
	outputPath := "" // New: for --output (export results to folder)
	projectRoot := ""
	adhoc := false
	requiresVideo := false
	requiresTrace := false
	requiresHAR := false
	fromFile := ""
	startURL := ""
	seedMode := ""
	seedScenario := ""
	sessionProfile := ""
	saveSession := ""
	freshSession := false
	restoreTabs := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--params":
			if i+1 >= len(args) {
				return fmt.Errorf("--params requires a value")
			}
			paramsRaw = args[i+1]
			i++
		case "--initial-params":
			if i+1 >= len(args) {
				return fmt.Errorf("--initial-params requires a value")
			}
			initialParamsRaw = args[i+1]
			i++
		case "--initial-store":
			if i+1 >= len(args) {
				return fmt.Errorf("--initial-store requires a value")
			}
			initialStoreRaw = args[i+1]
			i++
		case "--env":
			if i+1 >= len(args) {
				return fmt.Errorf("--env requires a value")
			}
			envRaw = args[i+1]
			i++
		case "--from-file":
			if i+1 >= len(args) {
				return fmt.Errorf("--from-file requires a value")
			}
			fromFile = args[i+1]
			i++
		case "--wait":
			wait = true
		case "--output-screenshots":
			if i+1 >= len(args) {
				return fmt.Errorf("--output-screenshots requires a value")
			}
			outputDir = args[i+1]
			i++
		case "--output":
			if i+1 >= len(args) {
				return fmt.Errorf("--output requires a value")
			}
			outputPath = args[i+1]
			i++
		case "--project-root":
			if i+1 >= len(args) {
				return fmt.Errorf("--project-root requires a value")
			}
			projectRoot = args[i+1]
			i++
		case "--start-url":
			if i+1 >= len(args) {
				return fmt.Errorf("--start-url requires a value")
			}
			startURL = args[i+1]
			i++
		case "--adhoc":
			adhoc = true
		case "--record-video", "--requires-video":
			requiresVideo = true
		case "--record-trace", "--requires-trace":
			requiresTrace = true
		case "--record-har", "--requires-har":
			requiresHAR = true
		case "--seed":
			if i+1 >= len(args) {
				return fmt.Errorf("--seed requires a value (use 'applied' or 'needs-applying')")
			}
			raw := strings.TrimSpace(strings.ToLower(args[i+1]))
			switch raw {
			case "applied":
				seedMode = raw
			case "needs-applying":
				seedMode = raw
			default:
				return fmt.Errorf("unsupported --seed value %q (use 'applied' or 'needs-applying')", args[i+1])
			}
			i++
		case "--seed-scenario":
			if i+1 >= len(args) {
				return fmt.Errorf("--seed-scenario requires a value")
			}
			seedScenario = strings.TrimSpace(args[i+1])
			i++
		case "--session-profile":
			if i+1 >= len(args) {
				return fmt.Errorf("--session-profile requires a value (profile ID or name)")
			}
			sessionProfile = strings.TrimSpace(args[i+1])
			i++
		case "--save-session":
			if i+1 >= len(args) {
				return fmt.Errorf("--save-session requires a value (profile ID or name)")
			}
			saveSession = strings.TrimSpace(args[i+1])
			i++
		case "--fresh-session":
			freshSession = true
		case "--restore-tabs":
			restoreTabs = true
		default:
			if strings.HasPrefix(args[i], "--") {
				return fmt.Errorf("unknown option: %s", args[i])
			}
			if workflow == "" {
				workflow = args[i]
			} else {
				return fmt.Errorf("unexpected argument: %s", args[i])
			}
		}
	}

	// Validate that we have a workflow source (steps, file, or ID)
	hasSteps := len(steps) > 0
	if !hasSteps && strings.TrimSpace(fromFile) == "" && strings.TrimSpace(workflow) == "" {
		return fmt.Errorf("workflow ID/name, --from-file, or --step flags are required")
	}

	// Validate that we don't mix --step with other sources
	if hasSteps {
		if strings.TrimSpace(workflow) != "" {
			return fmt.Errorf("cannot use --step with workflow ID")
		}
		if strings.TrimSpace(fromFile) != "" {
			return fmt.Errorf("cannot use --step with --from-file")
		}
	}

	// Force wait mode if --output is specified (export requires completion)
	if outputPath != "" {
		wait = true
	}

	// Force wait mode if --save-session is specified (need execution to complete to save state)
	if saveSession != "" {
		wait = true
	}

	// Validate session profile flags
	if sessionProfile != "" && freshSession {
		return fmt.Errorf("cannot use --session-profile with --fresh-session")
	}
	if restoreTabs && sessionProfile == "" {
		return fmt.Errorf("--restore-tabs requires --session-profile")
	}

	// Resolve session profile IDs if names were provided
	var sessionProfileID string
	var saveSessionProfileID string
	if sessionProfile != "" {
		resolved, err := resolveSessionProfile(ctx, sessionProfile)
		if err != nil {
			return fmt.Errorf("--session-profile: %w", err)
		}
		sessionProfileID = resolved.ID
		printSessionProfileLoadInfo(ctx, resolved)
	}
	if saveSession != "" {
		resolved, err := resolveSessionProfile(ctx, saveSession)
		if err != nil {
			// For --save-session, create profile if it doesn't exist
			if strings.Contains(err.Error(), "not found") {
				created, err := createSessionProfile(ctx, saveSession)
				if err != nil {
					return fmt.Errorf("--save-session: failed to create profile: %w", err)
				}
				saveSessionProfileID = created.ID
				fmt.Printf("Created new session profile: %s (%s)\n", created.Name, created.ID[:8])
			} else {
				return fmt.Errorf("--save-session: %w", err)
			}
		} else {
			saveSessionProfileID = resolved.ID
			printSaveSessionProfileInfo(ctx, resolved)
		}
	}

	if hasSteps {
		fmt.Printf("Executing inline workflow with %d steps\n", len(steps))
	} else if fromFile != "" {
		fmt.Printf("Executing workflow file: %s\n", fromFile)
	} else {
		fmt.Printf("Executing workflow: %s\n", workflow)
	}

	if projectRoot == "" && fromFile != "" {
		absFile, err := filepath.Abs(fromFile)
		if err != nil {
			return fmt.Errorf("resolve --from-file path: %w", err)
		}
		projectRoot = findBasRoot(filepath.Dir(absFile))
		if projectRoot == "" {
			return fmt.Errorf("unable to infer --project-root from file path; set --project-root /abs/path/to/bas")
		}
	}

	if projectRoot == "" {
		if envRoot := strings.TrimSpace(os.Getenv("BAS_PROJECT_ROOT")); envRoot != "" {
			projectRoot = envRoot
		} else if ctx.ScenarioRoot != "" {
			candidate := filepath.Join(ctx.ScenarioRoot, "bas")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				projectRoot = candidate
			}
		}
	}

	if projectRoot != "" {
		adhoc = true
	}

	fmt.Printf("API URL: %s\n", ctx.Core.APIBase())
	if projectRoot != "" {
		fmt.Printf("Project root: %s\n", projectRoot)
	} else {
		fmt.Println("Project root: (not set)")
		fmt.Println("WARN: Subflows using workflow_path may fail. Provide --project-root /abs/path/to/bas")
	}

	if adhoc {
		fmt.Println("Execution mode: adhoc")
		if wait {
			fmt.Println("Wait mode: client-side polling (adhoc returns immediately)")
		}
	} else {
		fmt.Println("Execution mode: direct")
	}

	var params map[string]any
	if err := json.Unmarshal([]byte(paramsRaw), &params); err != nil {
		return fmt.Errorf("invalid JSON for --params")
	}

	// Normalize params: move unknown fields to initial_params
	params = normalizeExecutionParams(params)

	// Merge --initial-params if provided
	if initialParamsRaw != "" {
		var initialParams map[string]any
		if err := json.Unmarshal([]byte(initialParamsRaw), &initialParams); err != nil {
			return fmt.Errorf("invalid JSON for --initial-params")
		}
		params = mergeIntoInitialParams(params, initialParams)
	}

	// Merge --initial-store if provided
	if initialStoreRaw != "" {
		var initialStore map[string]any
		if err := json.Unmarshal([]byte(initialStoreRaw), &initialStore); err != nil {
			return fmt.Errorf("invalid JSON for --initial-store")
		}
		params = mergeIntoInitialStore(params, initialStore)
	}

	// Merge --env if provided
	if envRaw != "" {
		var envVars map[string]any
		if err := json.Unmarshal([]byte(envRaw), &envVars); err != nil {
			return fmt.Errorf("invalid JSON for --env")
		}
		params = mergeIntoEnv(params, envVars)
	}

	if projectRoot != "" {
		params["projectRoot"] = projectRoot
		fmt.Println("Project root injected into parameters as projectRoot.")
	}

	// Inject session profile ID if specified (and not --fresh-session)
	if sessionProfileID != "" && !freshSession {
		params["session_profile_id"] = sessionProfileID
	}

	// Inject restore_tabs flag (only meaningful with session profile)
	if restoreTabs {
		params["restore_tabs"] = true
	}

	// Inject save session profile ID if specified
	if saveSessionProfileID != "" {
		params["save_session_profile_id"] = saveSessionProfileID
	}
	startURLFromParams := false
	if startURL == "" {
		if v, ok := params["start_url"].(string); ok {
			startURL = strings.TrimSpace(v)
			startURLFromParams = startURL != ""
		}
		if startURL == "" {
			if v, ok := params["startUrl"].(string); ok {
				startURL = strings.TrimSpace(v)
				startURLFromParams = startURL != ""
			}
		}
	}
	delete(params, "start_url")
	delete(params, "startUrl")
	if startURL != "" {
		params["startUrl"] = startURL
	}
	if startURL != "" {
		if startURLFromParams {
			fmt.Printf("Start URL (from params): %s\n", startURL)
		} else {
			fmt.Printf("Start URL: %s\n", startURL)
		}
	}

	if seedMode == "applied" {
		env, ok := params["env"].(map[string]any)
		if !ok || env == nil {
			env = map[string]any{}
		}
		env["seed_applied"] = true
		params["env"] = env
		fmt.Println("Seed flag: applied")
	}

	if seedMode == "needs-applying" && seedScenario == "" {
		seedScenario = ctx.Name
	}

	var seedCleanupToken string
	if seedMode == "needs-applying" && strings.EqualFold(seedScenario, ctx.Name) {
		fmt.Println("Seed mode: needs-applying (self-seed handshake via test-genie)")
		seedResp, err := applySeedViaTestGenie(seedScenario)
		if err != nil {
			return err
		}
		seedCleanupToken = seedResp.CleanupToken
		fmt.Printf("Seed cleanup token: %s\n", seedCleanupToken)
		mergeSeedStateIntoParams(params, seedResp.SeedState)

		env, ok := params["env"].(map[string]any)
		if !ok || env == nil {
			env = map[string]any{}
		}
		env["seed_applied"] = true
		params["env"] = env

		seedMode = "applied"
		if err := refreshScenarioAPI(ctx, ctx.Name); err != nil {
			return err
		}
		fmt.Println("Seed applied via test-genie; proceeding with execution.")
	}

	var response []byte

	// Force adhoc mode for inline steps
	if hasSteps {
		adhoc = true
	}

	if adhoc {
		workflowID, err := resolveWorkflowID(ctx, workflow)
		if fromFile == "" && !hasSteps {
			if err != nil {
				return err
			}
		}

		payload := map[string]any{
			"wait_for_completion": wait,
			"parameters":          params,
		}

		if hasSteps {
			// Build workflow from inline steps
			flowDef, err := BuildWorkflowFromSteps(steps)
			if err != nil {
				return fmt.Errorf("build workflow from steps: %w", err)
			}
			payload["flow_definition"] = flowDef
			payload["metadata"] = flowDef["metadata"]
		} else if fromFile != "" {
			// If --project-root was explicitly provided and fromFile is relative,
			// try resolving against project-root
			resolvedFile := fromFile
			projectRootCandidate := ""
			if projectRoot != "" && !filepath.IsAbs(fromFile) {
				projectRootCandidate = filepath.Join(projectRoot, fromFile)
				if _, err := os.Stat(projectRootCandidate); err == nil {
					resolvedFile = projectRootCandidate
				}
			}

			data, err := os.ReadFile(resolvedFile)
			if err != nil {
				if projectRootCandidate != "" && projectRootCandidate != resolvedFile {
					return fmt.Errorf("file not found: %s (also tried: %s)", fromFile, projectRootCandidate)
				}
				return fmt.Errorf("file not found: %s", fromFile)
			}
			var rawContent map[string]any
			if err := json.Unmarshal(data, &rawContent); err != nil {
				return fmt.Errorf("invalid JSON in %s", fromFile)
			}

			// Extract flow_definition if present (wrapper format from flows/ directory)
			// Otherwise use the entire file content (direct format from actions/, cases/)
			var flowDef any
			if nestedDef, ok := rawContent["flow_definition"]; ok {
				flowDef = nestedDef
			} else {
				flowDef = rawContent
			}

			payload["flow_definition"] = flowDef
			payload["metadata"] = buildAdhocMetadata(rawContent, fromFile)
		} else {
			workflowDetail, err := getWorkflow(ctx, workflowID)
			if err != nil {
				return err
			}
			if len(workflowDetail.Workflow.FlowDefinition) == 0 || string(workflowDetail.Workflow.FlowDefinition) == "null" {
				return fmt.Errorf("missing flow_definition for workflow %s", workflowID)
			}
			payload["flow_definition"] = json.RawMessage(workflowDetail.Workflow.FlowDefinition)
			if name := strings.TrimSpace(workflowDetail.Workflow.Name); name != "" {
				payload["metadata"] = map[string]any{"name": name}
			}
		}

		executePath := ctx.Core.APIPath("/workflows/execute-adhoc")
		executePath = appendExecuteQuery(executePath, requiresVideo, requiresTrace, requiresHAR, seedMode, seedScenario)
		response, err = ctx.Core.APIClient.Request("POST", executePath, nil, payload)
		if err != nil {
			return err
		}
	} else {
		executePath := ctx.Core.APIPath("/workflows/" + workflow + "/execute")
		executePath = appendExecuteQuery(executePath, requiresVideo, requiresTrace, requiresHAR, seedMode, seedScenario)
		payload := map[string]any{
			"parameters":          params,
			"wait_for_completion": wait,
		}
		var err error
		response, err = ctx.Core.APIClient.Request("POST", executePath, nil, payload)
		if err != nil {
			return err
		}
	}

	var execResp executeResponse
	if err := json.Unmarshal(response, &execResp); err != nil || execResp.ExecutionID == "" {
		fmt.Println("error: failed to start execution")
		fmt.Println(string(response))
		if adhoc {
			fmt.Println("Note: adhoc executions can still start even if the API times out.")
			fmt.Println("Check running executions with: browser-automation-studio execution list")
		} else if projectRoot == "" {
			fmt.Println("Hint: set --project-root /abs/path/to/bas so workflow_path subflows can resolve.")
		}
		return fmt.Errorf("execution start failed")
	}

	executionID := execResp.ExecutionID
	fmt.Println("OK: Execution started!")
	fmt.Printf("Execution ID: %s\n", executionID)

	seedCleanupScheduled := false
	if seedCleanupToken != "" {
		if err := scheduleSeedCleanup(ctx, executionID, seedScenario, seedCleanupToken); err != nil {
			fmt.Printf("WARN: seed cleanup scheduling failed: %v\n", err)
			fmt.Printf("Manual cleanup: test-genie playbooks-seed cleanup --scenario %s --token %s\n", seedScenario, seedCleanupToken)
		} else {
			fmt.Println("Seed cleanup scheduled after execution completes.")
			seedCleanupScheduled = true
		}
	}

	recordingsRoot := ""
	if ctx.ScenarioRoot != "" {
		recordingsRoot = filepath.Join(ctx.ScenarioRoot, "data", "recordings", executionID)
	}

	if !wait {
		fmt.Println("")
		fmt.Printf("Artifacts will be available after completion. Watch with: browser-automation-studio execution watch %s\n", executionID)
		if recordingsRoot != "" {
			fmt.Printf("Find more info at: %s\n", filepath.Join(recordingsRoot, "README.md"))
		}
	}

	var waitFailed bool
	var waitTimedOut bool
	var waitMissingExecution bool
	var waitLastStatus string
	var waitFailureMessage string

	if wait {
		fmt.Println("Waiting for completion...")
		maxAttempts := 60
		lastStatus := ""
		completed := false
		failed := false
		missingExecution := false
		consecutiveErrors := 0
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			if err := refreshScenarioAPIBase(ctx, ctx.Name); err != nil {
				fmt.Print(".")
				time.Sleep(5 * time.Second)
				continue
			}
			statusResp, err := ctx.Core.Get("/executions/"+executionID, nil)
			if err != nil {
				if isExecutionNotFoundErr(err) {
					fmt.Println("")
					fmt.Printf("ERROR: Execution record not found (id: %s).\n", executionID)
					fmt.Println("This can happen when the test subject restarts into a new database after seed cleanup.")
					fmt.Println("Hint: re-run without --wait or use test-genie playbooks to orchestrate seeds.")
					missingExecution = true
					failed = true
					completed = true
					break
				}
				consecutiveErrors++
				if consecutiveErrors >= 3 && seedCleanupToken != "" {
					fmt.Println("")
					fmt.Println("WARN: API unreachable after seed apply; the test subject may have restarted and the execution record may no longer be available.")
					missingExecution = true
					failed = true
					completed = true
					break
				}
				if consecutiveErrors >= 2 {
					if err := waitForScenarioRecovery(ctx, ctx.Name, 15*time.Second); err != nil {
						fmt.Print(".")
						time.Sleep(5 * time.Second)
						continue
					}
					consecutiveErrors = 0
				}
				fmt.Println(".")
				time.Sleep(5 * time.Second)
				continue
			}
			consecutiveErrors = 0
			status := normalizeExecutionStatus(extractString(statusResp, "status"))
			lastStatus = status
			if status == "completed" {
				fmt.Println("OK: Execution completed successfully")
				completed = true
				break
			}
			if status == "failed" || status == "cancelled" {
				fmt.Println("ERROR: Execution failed")
				errorMessage := extractString(statusResp, "error")
				if errorMessage == "" {
					errorMessage = extractString(statusResp, "error_message")
				}
				if errorMessage != "" {
					fmt.Printf("Error: %s\n", errorMessage)
				}
				waitFailureMessage = errorMessage
				failed = true
				completed = true
				break
			}
			fmt.Print(".")
			time.Sleep(5 * time.Second)
		}
		if completed {
			// Export results if --output was specified (works for both success and failure)
			if outputPath != "" && !missingExecution {
				if err := export.ExportExecution(ctx, executionID, outputPath); err != nil {
					fmt.Printf("WARN: Export failed: %v\n", err)
					// Fall back to printing artifact URLs if export fails
					if !missingExecution {
						printCollectedArtifacts(ctx, executionID, recordingsRoot, failed, requiresVideo, requiresTrace, requiresHAR)
					}
				} else {
					fmt.Printf("\nResults exported to: %s\n", outputPath)
					fmt.Printf("Read %s/README.md for details\n", outputPath)
				}
			} else if !missingExecution {
				// No --output specified, print artifact URLs as before
				printCollectedArtifacts(ctx, executionID, recordingsRoot, failed, requiresVideo, requiresTrace, requiresHAR)
			}
			if seedCleanupToken != "" && !seedCleanupScheduled {
				if err := cleanupSeedViaTestGenie(seedScenario, seedCleanupToken); err != nil {
					fmt.Printf("WARN: seed cleanup failed: %v\n", err)
					fmt.Printf("Manual cleanup: test-genie playbooks-seed cleanup --scenario %s --token %s\n", seedScenario, seedCleanupToken)
				} else {
					fmt.Println("Seed cleanup completed.")
				}
			} else if seedCleanupToken != "" {
				fmt.Println("Seed cleanup already scheduled; no manual cleanup needed.")
			}
		}
		if !completed {
			if lastStatus == "" {
				lastStatus = "unknown"
			}
			fmt.Println("")
			fmt.Printf("TIMEOUT: Execution did not finish after %d seconds (last status: %s). Use: browser-automation-studio execution watch %s\n", maxAttempts*5, lastStatus, executionID)
			if seedCleanupToken != "" {
				fmt.Println("Hint: if the test subject restarted during seed cleanup, the execution record may no longer be visible.")
			}
			if seedCleanupToken != "" && !seedCleanupScheduled {
				if err := scheduleSeedCleanup(ctx, executionID, seedScenario, seedCleanupToken); err != nil {
					fmt.Printf("WARN: seed cleanup scheduling failed: %v\n", err)
					fmt.Printf("Manual cleanup: test-genie playbooks-seed cleanup --scenario %s --token %s\n", seedScenario, seedCleanupToken)
				} else {
					fmt.Println("Seed cleanup scheduled after execution completes.")
				}
			} else if seedCleanupToken != "" {
				fmt.Println("Seed cleanup already scheduled; no manual cleanup needed.")
			}
		}

		waitFailed = failed
		waitTimedOut = !completed
		waitMissingExecution = missingExecution
		waitLastStatus = lastStatus
	}

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err == nil {
			fmt.Printf("Screenshots saved to: %s\n", outputDir)
		}
	}

	if waitFailed {
		if waitMissingExecution {
			return fmt.Errorf("execution %s failed: execution record unavailable", executionID)
		}
		if waitFailureMessage != "" {
			return fmt.Errorf("execution %s failed: %s", executionID, waitFailureMessage)
		}
		return fmt.Errorf("execution %s failed", executionID)
	}
	if waitTimedOut {
		if waitLastStatus == "" {
			waitLastStatus = "unknown"
		}
		return fmt.Errorf("execution %s timed out (last status: %s)", executionID, waitLastStatus)
	}

	return nil
}

func isExecutionNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "execution_not_found") || strings.Contains(message, "execution not found")
}

func refreshScenarioAPIBase(ctx *appctx.Context, scenarioName string) error {
	if ctx == nil || ctx.Core == nil {
		return fmt.Errorf("CLI context not configured")
	}
	if strings.TrimSpace(scenarioName) == "" {
		return fmt.Errorf("scenario name is required to resolve API base")
	}
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	base, err := resolveScenarioBase(ctxWithTimeout, scenarioName)
	if err != nil {
		return err
	}
	base = strings.TrimRight(base, "/")
	current := strings.TrimRight(ctx.Core.APIRootBase(), "/")
	if base != "" && base != current {
		ctx.Core.APIOverride = base
		fmt.Printf("Re-resolved API base: %s\n", strings.TrimRight(base, "/")+"/api/v1")
	}
	return nil
}

func waitForScenarioRecovery(ctx *appctx.Context, scenarioName string, timeout time.Duration) error {
	if ctx == nil || ctx.Core == nil {
		return fmt.Errorf("CLI context not configured")
	}
	if strings.TrimSpace(scenarioName) == "" {
		return fmt.Errorf("scenario name is required to resolve API base")
	}
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	base, err := resolveScenarioBase(ctxWithTimeout, scenarioName)
	if err != nil {
		return err
	}
	if err := waitForScenarioHealth(ctxWithTimeout, base); err != nil {
		return err
	}
	ctx.Core.APIOverride = strings.TrimRight(base, "/")
	return nil
}

func buildAdhocMetadata(flowDef any, filePath string) map[string]any {
	name := extractWorkflowName(flowDef)
	if name == "" {
		base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		name = strings.TrimSpace(base)
	}
	if name == "" {
		return nil
	}
	return map[string]any{"name": name}
}

func extractWorkflowName(flowDef any) string {
	flowMap, ok := flowDef.(map[string]any)
	if !ok {
		return ""
	}
	if name, ok := flowMap["name"].(string); ok {
		return strings.TrimSpace(name)
	}
	meta, ok := flowMap["metadata"].(map[string]any)
	if !ok {
		return ""
	}
	if name, ok := meta["name"].(string); ok {
		return strings.TrimSpace(name)
	}
	return ""
}

func appendExecuteQuery(base string, requiresVideo bool, requiresTrace bool, requiresHAR bool, seedMode string, seedScenario string) string {
	pairs := make([]string, 0, 5)
	if requiresVideo {
		pairs = append(pairs, "requires_video=true")
	}
	if requiresTrace {
		pairs = append(pairs, "requires_trace=true")
	}
	if requiresHAR {
		pairs = append(pairs, "requires_har=true")
	}
	if seedMode == "needs-applying" {
		pairs = append(pairs, "seed=needs-applying")
		if seedScenario != "" {
			pairs = append(pairs, fmt.Sprintf("seed_scenario=%s", url.QueryEscape(seedScenario)))
		}
	}
	if len(pairs) == 0 {
		return base
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + strings.Join(pairs, "&")
}

func printCollectedArtifacts(ctx *appctx.Context, executionID, recordingsRoot string, failed bool, requiresVideo bool, requiresTrace bool, requiresHAR bool) {
	fmt.Println("")
	fmt.Println("Execution artifacts")

	hasTimeline := false
	hasScreenshots := false
	hasVideos := false
	hasTraces := false
	hasHAR := false

	if recordingsRoot != "" {
		if fileExists(filepath.Join(recordingsRoot, "timeline.proto.json")) || fileExists(filepath.Join(recordingsRoot, "timeline.json")) {
			hasTimeline = true
		}

		artifactsRoot := filepath.Join(recordingsRoot, "artifacts")
		hasScreenshots = dirHasFiles(filepath.Join(artifactsRoot, "screenshots"))
		hasVideos = dirHasFiles(filepath.Join(artifactsRoot, "videos"))
		hasTraces = dirHasFiles(filepath.Join(artifactsRoot, "traces"))
		hasHAR = dirHasFiles(filepath.Join(artifactsRoot, "har"))
	}

	if hasTimeline {
		fmt.Printf("Timeline: %s/executions/%s/timeline\n", ctx.Core.APIBase(), executionID)
	}
	if hasScreenshots {
		fmt.Printf("Screenshots: %s/executions/%s/screenshots\n", ctx.Core.APIBase(), executionID)
	}
	if hasVideos {
		fmt.Printf("Recorded videos: %s/executions/%s/recorded-videos\n", ctx.Core.APIBase(), executionID)
	}
	if hasTraces {
		fmt.Printf("Traces: %s/executions/%s/recorded-traces\n", ctx.Core.APIBase(), executionID)
	}
	if hasHAR {
		fmt.Printf("HAR files: %s/executions/%s/recorded-har\n", ctx.Core.APIBase(), executionID)
	}
	if !hasTimeline && !hasScreenshots && !hasVideos && !hasTraces && !hasHAR {
		fmt.Println("No artifacts detected yet.")
	}

	if recordingsRoot != "" {
		fmt.Printf("Find more info at: %s\n", filepath.Join(recordingsRoot, "README.md"))
	}

	if !hasVideos && requiresVideo {
		fmt.Println("Video capture was requested but no recordings were produced.")
		fmt.Println("Check playwright-driver logs for video capture errors.")
	}
	if !hasTraces && requiresTrace {
		fmt.Println("Trace capture was requested but no traces were produced.")
		fmt.Println("Check playwright-driver logs for trace capture errors.")
	}
	if !hasHAR && requiresHAR {
		fmt.Println("HAR capture was requested but no HAR files were produced.")
		fmt.Println("Check playwright-driver logs for HAR capture errors.")
	}

	if !hasVideos && !requiresVideo {
		optional := "To collect video recordings, rerun with: --requires-video"
		if failed {
			optional = "To collect video recordings on a retry, rerun with: --requires-video"
		}
		fmt.Println(optional)
	}
	if !hasTraces && !requiresTrace {
		optional := "To collect traces, rerun with: --requires-trace"
		if failed {
			optional = "To collect traces on a retry, rerun with: --requires-trace"
		}
		fmt.Println(optional)
	}
	if !hasHAR && !requiresHAR {
		optional := "To collect HAR files, rerun with: --requires-har"
		if failed {
			optional = "To collect HAR files on a retry, rerun with: --requires-har"
		}
		fmt.Println(optional)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func dirHasFiles(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return true
		}
	}
	return false
}

func resolveWorkflowID(ctx *appctx.Context, workflow string) (string, error) {
	if isUUID(workflow) {
		return workflow, nil
	}
	workflows, _, err := listWorkflows(ctx)
	if err != nil {
		return "", err
	}
	matches := []string{}
	for _, item := range workflows {
		entry := item
		if entry.ID == "" && entry.Workflow != nil {
			entry = *entry.Workflow
		}
		if entry.Name == workflow {
			matches = append(matches, entry.ID)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("workflow not found by name '%s'", workflow)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple workflows match name '%s'", workflow)
	}
	return matches[0], nil
}

func findBasRoot(startDir string) string {
	dir := strings.TrimSpace(startDir)
	for dir != "" && dir != "." && dir != string(filepath.Separator) {
		if filepath.Base(dir) == "bas" {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func isUUID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 36 {
		return false
	}
	return strings.Count(value, "-") == 4
}

func extractString(data []byte, key string) string {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}
	if value, ok := payload[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func normalizeExecutionStatus(raw string) string {
	switch strings.TrimSpace(strings.ToUpper(raw)) {
	case "EXECUTION_STATUS_COMPLETED", "COMPLETED":
		return "completed"
	case "EXECUTION_STATUS_FAILED", "FAILED":
		return "failed"
	case "EXECUTION_STATUS_CANCELLED", "CANCELLED", "CANCELED":
		return "cancelled"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

type seedApplyResponse struct {
	SeedState    map[string]any `json:"seed_state"`
	CleanupToken string         `json:"cleanup_token"`
}

type seedCleanupResponse struct {
	Status string `json:"status"`
}

func applySeedViaTestGenie(seedScenario string) (*seedApplyResponse, error) {
	baseURL, err := resolveTestGenieAPIV1()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/scenarios/%s/playbooks/seed/apply", baseURL, url.PathEscape(seedScenario))
	payload := map[string]any{"retain": false}

	var resp seedApplyResponse
	if err := postJSON(endpoint, payload, &resp); err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.CleanupToken) == "" {
		return nil, fmt.Errorf("test-genie seed apply returned empty cleanup_token")
	}
	return &resp, nil
}

func cleanupSeedViaTestGenie(seedScenario, cleanupToken string) error {
	baseURL, err := resolveTestGenieAPIV1()
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/scenarios/%s/playbooks/seed/cleanup", baseURL, url.PathEscape(seedScenario))
	payload := map[string]any{"cleanup_token": cleanupToken}

	var resp seedCleanupResponse
	if err := postJSON(endpoint, payload, &resp); err != nil {
		return err
	}
	if strings.TrimSpace(resp.Status) == "" {
		return fmt.Errorf("test-genie seed cleanup returned empty status")
	}
	return nil
}

func scheduleSeedCleanup(ctx *appctx.Context, executionID, seedScenario, cleanupToken string) error {
	if ctx == nil || ctx.Core == nil {
		return fmt.Errorf("CLI context is not configured")
	}
	payload := map[string]any{
		"cleanup_token": cleanupToken,
	}
	if strings.TrimSpace(seedScenario) != "" {
		payload["seed_scenario"] = seedScenario
	}
	_, err := ctx.Core.Request("POST", "/executions/"+executionID+"/seed-cleanup", nil, payload)
	return err
}

func mergeSeedStateIntoParams(params map[string]any, seedState map[string]any) {
	if params == nil || len(seedState) == 0 {
		return
	}
	initialParams, ok := params["initial_params"].(map[string]any)
	if !ok || initialParams == nil {
		initialParams = map[string]any{}
	}
	for key, value := range seedState {
		initialParams[key] = value
	}
	params["initial_params"] = initialParams
}

func resolveTestGenieAPIV1() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	base, err := discovery.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		return "", fmt.Errorf("resolve test-genie API: %w", err)
	}
	return strings.TrimRight(base, "/") + "/api/v1", nil
}

func postJSON(endpoint string, payload any, dest any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("request failed (%s): %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, dest); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func refreshScenarioAPI(ctx *appctx.Context, scenarioName string) error {
	if ctx == nil || ctx.Core == nil {
		return fmt.Errorf("CLI context not configured")
	}
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	base, err := resolveScenarioBase(ctxWithTimeout, scenarioName)
	if err != nil {
		return err
	}
	ctx.Core.APIOverride = base
	fmt.Printf("Re-resolved API base: %s\n", strings.TrimRight(base, "/")+"/api/v1")

	if err := waitForScenarioHealth(ctxWithTimeout, base); err != nil {
		return err
	}
	return nil
}

func resolveScenarioBase(ctx context.Context, scenarioName string) (string, error) {
	if strings.TrimSpace(scenarioName) == "" {
		return "", fmt.Errorf("scenario name is required to resolve API base")
	}
	base, err := discovery.ResolveScenarioURLDefault(ctx, scenarioName)
	if err != nil {
		return "", fmt.Errorf("resolve %s API: %w", scenarioName, err)
	}
	return strings.TrimRight(base, "/"), nil
}

func waitForScenarioHealth(ctx context.Context, base string) error {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return fmt.Errorf("API base is empty while waiting for health")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	paths := []string{"/health", "/api/v1/health"}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		for _, path := range paths {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
			if err != nil {
				continue
			}
			resp, err := client.Do(req)
			if err == nil && resp != nil {
				resp.Body.Close()
				if resp.StatusCode < 400 {
					return nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for API health at %s", base)
		case <-ticker.C:
		}
	}
}

// knownExecutionParamFields is the set of fields that are part of the ExecutionParameters proto.
var knownExecutionParamFields = map[string]bool{
	"initial_params":          true,
	"initial_store":           true,
	"env":                     true,
	"projectRoot":             true,
	"startUrl":                true,
	"start_url":               true,
	"session_profile_id":      true,
	"save_session_profile_id": true,
	"restore_tabs":            true, // Tab restoration from session profile
}

// normalizeExecutionParams moves unknown fields to initial_params.
// This allows users to pass custom workflow parameters directly in --params
// without needing to nest them in initial_params.
func normalizeExecutionParams(params map[string]any) map[string]any {
	if params == nil {
		return params
	}

	// Find unknown fields
	unknownFields := make(map[string]any)
	for key, value := range params {
		if !knownExecutionParamFields[key] {
			unknownFields[key] = value
		}
	}

	// If no unknown fields, return as-is
	if len(unknownFields) == 0 {
		return params
	}

	// Move unknown fields to initial_params
	initialParams, _ := params["initial_params"].(map[string]any)
	if initialParams == nil {
		initialParams = make(map[string]any)
	}

	for key, value := range unknownFields {
		// Don't overwrite existing values in initial_params
		if _, exists := initialParams[key]; !exists {
			initialParams[key] = value
		}
		delete(params, key)
	}

	params["initial_params"] = initialParams
	return params
}

// mergeIntoInitialParams merges additional values into params["initial_params"].
func mergeIntoInitialParams(params map[string]any, values map[string]any) map[string]any {
	if params == nil {
		params = make(map[string]any)
	}
	initialParams, _ := params["initial_params"].(map[string]any)
	if initialParams == nil {
		initialParams = make(map[string]any)
	}
	for k, v := range values {
		initialParams[k] = v
	}
	params["initial_params"] = initialParams
	return params
}

// mergeIntoInitialStore merges additional values into params["initial_store"].
func mergeIntoInitialStore(params map[string]any, values map[string]any) map[string]any {
	if params == nil {
		params = make(map[string]any)
	}
	initialStore, _ := params["initial_store"].(map[string]any)
	if initialStore == nil {
		initialStore = make(map[string]any)
	}
	for k, v := range values {
		initialStore[k] = v
	}
	params["initial_store"] = initialStore
	return params
}

// mergeIntoEnv merges additional values into params["env"].
func mergeIntoEnv(params map[string]any, values map[string]any) map[string]any {
	if params == nil {
		params = make(map[string]any)
	}
	env, _ := params["env"].(map[string]any)
	if env == nil {
		env = make(map[string]any)
	}
	for k, v := range values {
		env[k] = v
	}
	params["env"] = env
	return params
}

// sessionProfileInfo holds basic session profile information for resolution.
type sessionProfileInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type sessionProfileListResponse struct {
	Profiles []sessionProfileInfo `json:"profiles"`
}

// resolveSessionProfile resolves a profile identifier (ID, short ID prefix, or name) to a profile.
// Resolution order: exact ID match → short ID prefix match → exact name match.
func resolveSessionProfile(ctx *appctx.Context, identifier string) (*sessionProfileInfo, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, fmt.Errorf("profile identifier is required")
	}

	body, err := ctx.Core.Get("/recordings/sessions", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	var resp sessionProfileListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse profiles: %w", err)
	}

	// First, try exact ID match
	for _, p := range resp.Profiles {
		if p.ID == identifier {
			return &p, nil
		}
	}

	// Then try short ID prefix match (minimum 4 characters to avoid accidental matches)
	if len(identifier) >= 4 && len(identifier) < 36 && !strings.Contains(identifier, " ") {
		var prefixMatches []sessionProfileInfo
		for _, p := range resp.Profiles {
			if strings.HasPrefix(p.ID, identifier) {
				prefixMatches = append(prefixMatches, p)
			}
		}
		if len(prefixMatches) == 1 {
			return &prefixMatches[0], nil
		}
		if len(prefixMatches) > 1 {
			return nil, fmt.Errorf("ambiguous short ID '%s': %d profiles match. Use full ID or more characters", identifier, len(prefixMatches))
		}
	}

	// Finally, try exact name match
	var nameMatches []sessionProfileInfo
	for _, p := range resp.Profiles {
		if p.Name == identifier {
			nameMatches = append(nameMatches, p)
		}
	}

	if len(nameMatches) == 0 {
		return nil, fmt.Errorf("session profile not found: %s", identifier)
	}
	if len(nameMatches) > 1 {
		return nil, fmt.Errorf("ambiguous profile name '%s': %d profiles match. Use profile ID instead", identifier, len(nameMatches))
	}

	return &nameMatches[0], nil
}

// createSessionProfile creates a new session profile.
func createSessionProfile(ctx *appctx.Context, name string) (*sessionProfileInfo, error) {
	payload := map[string]string{"name": name}
	body, err := ctx.Core.Request("POST", "/recordings/sessions", nil, payload)
	if err != nil {
		return nil, err
	}
	var profile sessionProfileInfo
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// sessionStorageState holds storage state information for a session profile.
type sessionStorageState struct {
	Cookies []sessionCookie     `json:"cookies"`
	Origins []sessionOrigin     `json:"origins"`
	Stats   sessionStorageStats `json:"stats"`
}

type sessionCookie struct {
	Name    string  `json:"name"`
	Domain  string  `json:"domain"`
	Expires float64 `json:"expires"`
}

type sessionOrigin struct {
	Origin       string                    `json:"origin"`
	LocalStorage []sessionLocalStorageItem `json:"localStorage"`
}

type sessionLocalStorageItem struct {
	Name string `json:"name"`
}

type sessionStorageStats struct {
	CookieCount       int `json:"cookieCount"`
	LocalStorageCount int `json:"localStorageCount"`
	OriginCount       int `json:"originCount"`
}

// getSessionStorageState retrieves the storage state for a session profile.
func getSessionStorageState(ctx *appctx.Context, profileID string) (*sessionStorageState, error) {
	body, err := ctx.Core.Get("/recordings/sessions/"+profileID+"/storage", nil)
	if err != nil {
		return nil, err
	}
	var state sessionStorageState
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// countExpiredSessionCookies returns the number of expired cookies in the storage state.
func countExpiredSessionCookies(cookies []sessionCookie) int {
	now := float64(time.Now().Unix())
	count := 0
	for _, c := range cookies {
		// Session cookies (expires <= 0) don't expire
		if c.Expires > 0 && c.Expires < now {
			count++
		}
	}
	return count
}

// printSessionProfileLoadInfo prints information about the loaded session profile.
func printSessionProfileLoadInfo(ctx *appctx.Context, profile *sessionProfileInfo) {
	state, err := getSessionStorageState(ctx, profile.ID)
	if err != nil {
		// Silently continue if we can't get storage state - profile will still be used
		fmt.Printf("Using session profile: %s (%s)\n", profile.Name, profile.ID[:8])
		return
	}

	// Print basic load confirmation
	fmt.Printf("Using session profile: %s (%s) - %d cookies, %d localStorage items\n",
		profile.Name, profile.ID[:8], state.Stats.CookieCount, state.Stats.LocalStorageCount)

	// Warn about expired cookies
	expiredCount := countExpiredSessionCookies(state.Cookies)
	if expiredCount > 0 {
		fmt.Printf("WARN: Session profile has %d expired cookie(s). Authentication may fail.\n", expiredCount)
		fmt.Printf("      Consider re-authenticating with --save-session %q\n", profile.Name)
	}
}

// printSaveSessionProfileInfo prints information about the existing profile that will be updated.
func printSaveSessionProfileInfo(ctx *appctx.Context, profile *sessionProfileInfo) {
	state, err := getSessionStorageState(ctx, profile.ID)
	if err != nil {
		// Silently continue if we can't get storage state
		fmt.Printf("Will update existing profile: %s (%s)\n", profile.Name, profile.ID[:8])
		return
	}

	// Show what will be overwritten
	if state.Stats.CookieCount > 0 || state.Stats.LocalStorageCount > 0 {
		fmt.Printf("Will update existing profile: %s (%s) - replacing %d cookies, %d localStorage items\n",
			profile.Name, profile.ID[:8], state.Stats.CookieCount, state.Stats.LocalStorageCount)
	} else {
		fmt.Printf("Will update existing profile: %s (%s) - currently empty\n",
			profile.Name, profile.ID[:8])
	}
}
