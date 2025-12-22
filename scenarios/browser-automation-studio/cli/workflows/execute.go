package workflows

import (
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

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/api-core/discovery"
)

func runExecute(ctx *appctx.Context, args []string) error {
	workflow := ""

	paramsRaw := "{}"
	wait := false
	outputDir := ""
	projectRoot := ""
	adhoc := false
	requiresVideo := false
	requiresTrace := false
	requiresHAR := false
	fromFile := ""
	startURL := ""
	seedMode := ""
	seedScenario := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--params":
			if i+1 >= len(args) {
				return fmt.Errorf("--params requires a value")
			}
			paramsRaw = args[i+1]
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

	if strings.TrimSpace(fromFile) == "" && strings.TrimSpace(workflow) == "" {
		return fmt.Errorf("workflow ID/name or --from-file is required")
	}

	if fromFile != "" {
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

	fmt.Printf("API URL: %s\n", ctx.ResolvedAPIV1Base())
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

	if projectRoot != "" {
		params["projectRoot"] = projectRoot
		fmt.Println("Project root injected into parameters as projectRoot.")
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
	if adhoc {
		workflowID, err := resolveWorkflowID(ctx, workflow)
		if fromFile == "" {
			if err != nil {
				return err
			}
		}

		payload := map[string]any{
			"wait_for_completion": wait,
			"parameters":          params,
		}

		if fromFile != "" {
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("file not found: %s", fromFile)
			}
			var flowDef any
			if err := json.Unmarshal(data, &flowDef); err != nil {
				return fmt.Errorf("invalid JSON in %s", fromFile)
			}
			payload["flow_definition"] = flowDef
			payload["metadata"] = buildAdhocMetadata(flowDef, fromFile)
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

		executePath := ctx.APIPath("/workflows/execute-adhoc")
		executePath = appendExecuteQuery(executePath, requiresVideo, requiresTrace, requiresHAR, seedMode, seedScenario)
		response, err = ctx.Core.APIClient.Request("POST", executePath, nil, payload)
		if err != nil {
			return err
		}
	} else {
		executePath := ctx.APIPath("/workflows/" + workflow + "/execute")
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

	if wait {
		fmt.Println("Waiting for completion...")
		maxAttempts := 60
		lastStatus := ""
		completed := false
		failed := false
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			statusResp, err := ctx.Core.APIClient.Get(ctx.APIPath("/executions/"+executionID), nil)
			if err != nil {
				fmt.Println(".")
				time.Sleep(5 * time.Second)
				continue
			}
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
				failed = true
				completed = true
				break
			}
			fmt.Print(".")
			time.Sleep(5 * time.Second)
		}
		if completed {
			printCollectedArtifacts(ctx, executionID, recordingsRoot, failed, requiresVideo, requiresTrace, requiresHAR)
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
	}

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err == nil {
			fmt.Printf("Screenshots saved to: %s\n", outputDir)
		}
	}

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
		fmt.Printf("Timeline: %s/executions/%s/timeline\n", ctx.ResolvedAPIV1Base(), executionID)
	}
	if hasScreenshots {
		fmt.Printf("Screenshots: %s/executions/%s/screenshots\n", ctx.ResolvedAPIV1Base(), executionID)
	}
	if hasVideos {
		fmt.Printf("Recorded videos: %s/executions/%s/recorded-videos\n", ctx.ResolvedAPIV1Base(), executionID)
	}
	if hasTraces {
		fmt.Printf("Traces: %s/executions/%s/recorded-traces\n", ctx.ResolvedAPIV1Base(), executionID)
	}
	if hasHAR {
		fmt.Printf("HAR files: %s/executions/%s/recorded-har\n", ctx.ResolvedAPIV1Base(), executionID)
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
	path := ctx.APIPath("/executions/" + executionID + "/seed-cleanup")
	_, err := ctx.Core.APIClient.Request("POST", path, nil, payload)
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

	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	base, err := resolver.ResolveScenarioURLDefault(ctx, "test-genie")
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
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	base, err := resolver.ResolveScenarioURLDefault(ctx, scenarioName)
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
