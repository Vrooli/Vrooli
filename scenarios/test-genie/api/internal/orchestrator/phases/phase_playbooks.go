package phases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"test-genie/internal/orchestrator/workspace"
)

const (
	playbookRegistryName   = "registry.json"
	playbookFolderName     = "playbooks"
	playbookResolverScript = "scripts/scenarios/testing/playbooks/resolve-workflow.py"
	playbookArtifactRoot   = "coverage/automation"
	playbookScenarioName   = "browser-automation-studio"
)

var (
	errMissingWorkflow = errors.New("workflow definition missing")
	errResolveWorkflow = errors.New("workflow resolver failed")
)

type playbookRegistry struct {
	Scenario   string          `json:"scenario"`
	Generated  string          `json:"generated_at"`
	Playbooks  []playbookEntry `json:"playbooks"`
	Deprecated []playbookEntry `json:"deprecated_playbooks,omitempty"`
}

type basExecutionStatus struct {
	Status        string  `json:"status"`
	Progress      float64 `json:"progress"`
	CurrentStep   string  `json:"current_step"`
	FailureReason string  `json:"failure_reason"`
	Error         string  `json:"error"`
}

type playbookEntry struct {
	File         string   `json:"file"`
	Description  string   `json:"description"`
	Order        string   `json:"order"`
	Requirements []string `json:"requirements"`
	Fixtures     []string `json:"fixtures"`
	Reset        string   `json:"reset"`
}

type playbookOutcome struct {
	ExecutionID string
	Duration    time.Duration
	Stats       string
}

type playbookResult struct {
	Entry        playbookEntry
	Outcome      *playbookOutcome
	Err          error
	ArtifactPath string
}

func runPlaybooksPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if os.Getenv("TEST_GENIE_SKIP_PLAYBOOKS") == "1" {
		message := "playbooks phase disabled via TEST_GENIE_SKIP_PLAYBOOKS"
		logPhaseWarn(logWriter, message)
		return RunReport{Observations: []Observation{NewObservation(message)}}
	}

	if !scenarioHasUI(env.ScenarioDir) {
		message := "ui/ directory missing; skipping UI workflow validation"
		logPhaseWarn(logWriter, message)
		return RunReport{Observations: []Observation{NewObservation(message)}}
	}

	registry, err := loadPlaybookRegistry(env.TestDir)
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Regenerate test/playbooks/registry.json via playbook builder.",
		}
	}
	playbooks := registry.Playbooks
	if len(playbooks) == 0 {
		message := "no workflows registered under test/playbooks/"
		logPhaseWarn(logWriter, message)
		return RunReport{Observations: []Observation{NewObservation(message)}}
	}

	apiBase, err := ensureBASAPI(ctx, logWriter)
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           "Start the browser-automation-studio scenario so workflows can execute.",
		}
	}

	cleanupSeeds, err := applyPlaybookSeeds(ctx, env, logWriter)
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Fix playbook seeds under test/playbooks/__seeds.",
		}
	}
	if cleanupSeeds != nil {
		defer cleanupSeeds()
	}

	uiBaseURL, _ := resolveScenarioBaseURL(ctx, env.ScenarioName)
	httpClient := &http.Client{Timeout: 15 * time.Second}
	var observations []Observation
	results := make([]playbookResult, 0, len(playbooks))

	for _, entry := range playbooks {
		select {
		case <-ctx.Done():
			return RunReport{Err: ctx.Err(), FailureClassification: FailureClassSystem}
		default:
		}

		outcome, execErr := executePlaybook(ctx, httpClient, env, apiBase, uiBaseURL, entry, logWriter)
		if execErr != nil {
			artifact := ""
			if outcome != nil && outcome.ExecutionID != "" {
				if path, dumpErr := dumpTimeline(httpClient, apiBase, outcome.ExecutionID, env, entry.File, nil); dumpErr == nil {
					artifact = path
				}
			}
			results = append(results, playbookResult{Entry: entry, Outcome: outcome, Err: execErr, ArtifactPath: artifact})
			_ = writePlaybookPhaseResults(env, results)

			classification := FailureClassSystem
			if errors.Is(execErr, errMissingWorkflow) || errors.Is(execErr, errResolveWorkflow) {
				classification = FailureClassMisconfiguration
			}
			return RunReport{
				Err:                   execErr,
				FailureClassification: classification,
				Remediation:           "Inspect the workflow definition and Browser Automation Studio logs.",
				Observations:          observations,
			}
		}

		results = append(results, playbookResult{Entry: entry, Outcome: outcome})
		if outcome != nil && outcome.Stats != "" {
			observations = append(observations, NewSuccessObservation(fmt.Sprintf("%s completed %s", entry.File, outcome.Stats)))
		} else {
			observations = append(observations, NewSuccessObservation(fmt.Sprintf("%s completed", entry.File)))
		}
		logPhaseStep(logWriter, "workflow %s completed", entry.File)
	}

	if err := writePlaybookPhaseResults(env, results); err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassSystem,
			Remediation:           "Ensure coverage/phase-results directory is writable.",
			Observations:          observations,
		}
	}

	logPhaseStep(logWriter, "playbook workflows executed: %d", len(playbooks))
	return RunReport{Observations: observations}
}

func loadPlaybookRegistry(testDir string) (playbookRegistry, error) {
	var registry playbookRegistry
	registryPath := filepath.Join(testDir, playbookFolderName, playbookRegistryName)
	data, err := os.ReadFile(registryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return registry, fmt.Errorf("playbook registry not found at %s", registryPath)
		}
		return registry, err
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return registry, fmt.Errorf("failed to parse %s: %w", registryPath, err)
	}
	if len(registry.Playbooks) == 0 && len(registry.Deprecated) > 0 {
		registry.Playbooks = registry.Deprecated
	}
	sort.SliceStable(registry.Playbooks, func(i, j int) bool {
		left := registry.Playbooks[i].Order
		right := registry.Playbooks[j].Order
		if left == right {
			return registry.Playbooks[i].File < registry.Playbooks[j].File
		}
		return left < right
	})
	return registry, nil
}

func ensureBASAPI(ctx context.Context, logWriter io.Writer) (string, error) {
	apiPort, err := resolveScenarioPort(ctx, playbookScenarioName, "API_PORT")
	if err != nil {
		logPhaseWarn(logWriter, "browser-automation-studio port lookup failed: %v", err)
		if startErr := startScenario(ctx, playbookScenarioName, logWriter); startErr != nil {
			return "", fmt.Errorf("failed to start browser-automation-studio: %w", startErr)
		}
		apiPort, err = resolveScenarioPort(ctx, playbookScenarioName, "API_PORT")
		if err != nil {
			return "", fmt.Errorf("browser-automation-studio API port unavailable: %w", err)
		}
	}
	apiBase := fmt.Sprintf("http://127.0.0.1:%s/api/v1", apiPort)
	if err := waitForBASHealth(ctx, apiBase); err != nil {
		return "", err
	}
	return apiBase, nil
}

func executePlaybook(
	ctx context.Context,
	client *http.Client,
	env workspace.Environment,
	apiBase string,
	uiBaseURL string,
	entry playbookEntry,
	logWriter io.Writer,
) (*playbookOutcome, error) {
	workflowPath := entry.File
	if !filepath.IsAbs(workflowPath) {
		workflowPath = filepath.Join(env.ScenarioDir, filepath.FromSlash(workflowPath))
	}
	if _, err := os.Stat(workflowPath); err != nil {
		return nil, fmt.Errorf("%w: %s", errMissingWorkflow, workflowPath)
	}

	rawDefinition, err := resolveWorkflowDefinition(ctx, env, workflowPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errResolveWorkflow, err)
	}
	if uiBaseURL != "" {
		substitutePlaceholders(rawDefinition, uiBaseURL)
	}

	payload := map[string]any{
		"flow_definition":     cleanWorkflowDefinition(rawDefinition),
		"parameters":          map[string]any{},
		"wait_for_completion": false,
		"metadata": map[string]any{
			"name": entry.Description,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/workflows/execute-adhoc", apiBase), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("workflow execution request failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(data)))
	}
	var payloadResp struct {
		ExecutionID string `json:"execution_id"`
	}
	if err := json.Unmarshal(data, &payloadResp); err != nil {
		return nil, fmt.Errorf("execution response decode failed: %w", err)
	}
	if payloadResp.ExecutionID == "" {
		return nil, fmt.Errorf("execution id missing in response: %s", strings.TrimSpace(string(data)))
	}
	logPhaseStep(logWriter, "workflow %s queued with execution id %s", entry.File, payloadResp.ExecutionID)

	outcome := &playbookOutcome{ExecutionID: payloadResp.ExecutionID}
	start := time.Now()
	waitErr := waitForWorkflow(ctx, client, apiBase, payloadResp.ExecutionID)
	outcome.Duration = time.Since(start)
	if waitErr != nil {
		return outcome, waitErr
	}

	timelineData, err := fetchTimelineJSON(ctx, client, apiBase, payloadResp.ExecutionID)
	if err == nil {
		outcome.Stats = summarizeTimeline(timelineData)
	}
	return outcome, nil
}

func resolveWorkflowDefinition(ctx context.Context, env workspace.Environment, workflowPath string) (map[string]any, error) {
	script := filepath.Join(env.AppRoot, playbookResolverScript)
	var output []byte
	if stat, err := os.Stat(script); err == nil && !stat.IsDir() {
		cmdName := "python3"
		if _, err := exec.LookPath(cmdName); err != nil {
			cmdName = "python"
		}
		if _, err := exec.LookPath(cmdName); err == nil {
			cmd := exec.CommandContext(ctx, cmdName, script, "--workflow", workflowPath, "--scenario", env.ScenarioDir)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err == nil {
				output = stdout.Bytes()
			} else {
				return nil, fmt.Errorf("workflow resolver failed: %s", stderr.String())
			}
		}
	}
	if len(output) == 0 {
		var err error
		output, err = os.ReadFile(workflowPath)
		if err != nil {
			return nil, err
		}
	}
	var doc map[string]any
	if err := json.Unmarshal(output, &doc); err != nil {
		return nil, fmt.Errorf("invalid workflow JSON: %w", err)
	}
	return doc, nil
}

func substitutePlaceholders(doc any, baseURL string) {
	switch value := doc.(type) {
	case map[string]any:
		for k, v := range value {
			value[k] = substituteValue(v, baseURL)
		}
	case []any:
		for idx, v := range value {
			value[idx] = substituteValue(v, baseURL)
		}
	}
}

func substituteValue(v any, baseURL string) any {
	switch typed := v.(type) {
	case string:
		processed := strings.ReplaceAll(typed, "${BASE_URL}", baseURL)
		if port := extractPort(baseURL); port != "" {
			processed = strings.ReplaceAll(processed, "{{UI_PORT}}", port)
		}
		return processed
	case map[string]any, []any:
		substitutePlaceholders(typed, baseURL)
		return typed
	default:
		return v
	}
}

func cleanWorkflowDefinition(doc map[string]any) map[string]any {
	root := doc
	if inner, ok := doc["flow_definition"].(map[string]any); ok {
		root = inner
	}
	result := map[string]any{}
	if nodes, ok := root["nodes"]; ok {
		result["nodes"] = cleanNodes(nodes)
	}
	if edges, ok := root["edges"]; ok {
		result["edges"] = edges
	}
	if settings, ok := root["settings"]; ok {
		result["settings"] = settings
	}
	return result
}

func cleanNodes(value any) any {
	nodes, ok := value.([]any)
	if !ok {
		return value
	}
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		if node == nil {
			continue
		}
		data, _ := node["data"].(map[string]any)
		if data == nil {
			continue
		}
		if wf, ok := data["workflowDefinition"].(map[string]any); ok {
			data["workflowDefinition"] = cleanWorkflowDefinition(wf)
		}
	}
	return nodes
}

func waitForWorkflow(ctx context.Context, client *http.Client, apiBase, executionID string) error {
	deadline := time.Now().Add(3 * time.Minute)
	lastStatus := ""
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("workflow execution timed out after %s", 3*time.Minute)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		status, err := fetchExecutionStatus(ctx, client, apiBase, executionID)
		if err != nil {
			return err
		}
		normalized := strings.ToLower(status.Status)
		if normalized == "completed" || normalized == "success" {
			return nil
		}
		if normalized == "failed" || normalized == "error" || normalized == "errored" {
			if status.FailureReason != "" {
				return fmt.Errorf("workflow failed: %s", status.FailureReason)
			}
			if status.Error != "" {
				return fmt.Errorf("workflow failed: %s", status.Error)
			}
			return fmt.Errorf("workflow failed with status %s", status.Status)
		}
		if status.Status != lastStatus {
			lastStatus = status.Status
		}
		time.Sleep(1 * time.Second)
	}
}

func fetchExecutionStatus(ctx context.Context, client *http.Client, apiBase, executionID string) (basExecutionStatus, error) {
	var payload basExecutionStatus
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/executions/%s", apiBase, executionID), nil)
	if err != nil {
		return payload, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return payload, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return payload, fmt.Errorf("execution lookup failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(data)))
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func fetchTimelineJSON(ctx context.Context, client *http.Client, apiBase, executionID string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/executions/%s/timeline", apiBase, executionID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("timeline fetch failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func summarizeTimeline(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var doc struct {
		Frames []struct {
			StepType string `json:"step_type"`
			Status   string `json:"status"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return ""
	}
	totalSteps := len(doc.Frames)
	totalAsserts := 0
	assertPassed := 0
	for _, frame := range doc.Frames {
		if frame.StepType == "assert" {
			totalAsserts++
			if strings.EqualFold(frame.Status, "completed") {
				assertPassed++
			}
		}
	}
	if totalAsserts > 0 {
		return fmt.Sprintf(" (%d steps, %d/%d assertions passed)", totalSteps, assertPassed, totalAsserts)
	}
	if totalSteps > 0 {
		return fmt.Sprintf(" (%d steps)", totalSteps)
	}
	return ""
}

func dumpTimeline(client *http.Client, apiBase, executionID string, env workspace.Environment, workflowFile string, timelineData []byte) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if len(timelineData) == 0 {
		data, err := fetchTimelineJSON(ctx, client, apiBase, executionID)
		if err != nil {
			return "", err
		}
		timelineData = data
	}

	targetDir := filepath.Join(env.ScenarioDir, playbookArtifactRoot)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	filename := sanitizeWorkflowArtifactName(workflowFile)
	path := filepath.Join(targetDir, filename+".timeline.json")
	if err := os.WriteFile(path, timelineData, 0o644); err != nil {
		return "", err
	}
	if rel, err := filepath.Rel(env.AppRoot, path); err == nil {
		return rel, nil
	}
	return path, nil
}

func sanitizeWorkflowArtifactName(name string) string {
	name = strings.ReplaceAll(name, string(filepath.Separator), "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)
	return name
}

func writePlaybookPhaseResults(env workspace.Environment, results []playbookResult) error {
	if len(results) == 0 {
		return nil
	}
	phaseDir := filepath.Join(env.ScenarioDir, "coverage", "phase-results")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		return err
	}
	output := map[string]any{
		"phase":      Playbooks.String(),
		"scenario":   env.ScenarioName,
		"tests":      len(results),
		"errors":     0,
		"status":     "passed",
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	var requirementEntries []map[string]any
	for _, result := range results {
		status := "passed"
		evidence := result.Entry.File
		if result.Outcome != nil {
			if result.Outcome.Stats != "" {
				evidence = fmt.Sprintf("%s%s", result.Entry.File, result.Outcome.Stats)
			}
			if result.Outcome.Duration > 0 {
				evidence = fmt.Sprintf("%s in %s", evidence, result.Outcome.Duration.Truncate(time.Millisecond))
			}
		}
		if result.Err != nil {
			status = "failed"
			evidence = fmt.Sprintf("%s failed: %v", result.Entry.File, result.Err)
		}
		if result.ArtifactPath != "" {
			evidence = fmt.Sprintf("%s (artifact: %s)", evidence, result.ArtifactPath)
		}
		if result.Err != nil {
			output["errors"] = output["errors"].(int) + 1
		}
		for _, reqID := range result.Entry.Requirements {
			requirementEntries = append(requirementEntries, map[string]any{
				"id":         reqID,
				"status":     status,
				"phase":      Playbooks.String(),
				"evidence":   evidence,
				"updated_at": time.Now().UTC().Format(time.RFC3339),
			})
		}
	}
	if output["errors"].(int) > 0 {
		output["status"] = "failed"
	}
	output["requirements"] = requirementEntries

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(phaseDir, "playbooks.json"), data, 0o644)
}

func applyPlaybookSeeds(ctx context.Context, env workspace.Environment, logWriter io.Writer) (func(), error) {
	seedsDir := filepath.Join(env.ScenarioDir, "test", playbookFolderName, "__seeds")
	applyPath := filepath.Join(seedsDir, "apply.sh")
	if _, err := os.Stat(applyPath); err != nil {
		return nil, nil
	}
	if err := runSeedScript(ctx, env, applyPath, logWriter); err != nil {
		return nil, err
	}
	cleanupPath := filepath.Join(seedsDir, "cleanup.sh")
	cleanup := func() {}
	if _, err := os.Stat(cleanupPath); err == nil {
		cleanup = func() {
			_ = runSeedScript(context.Background(), env, cleanupPath, logWriter)
		}
	}
	return cleanup, nil
}

func runSeedScript(ctx context.Context, env workspace.Environment, scriptPath string, logWriter io.Writer) error {
	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	cmd.Dir = env.ScenarioDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("TEST_GENIE_SCENARIO_DIR=%s", env.ScenarioDir),
		fmt.Sprintf("TEST_GENIE_APP_ROOT=%s", env.AppRoot),
	)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("seed script %s failed: %w", scriptPath, err)
	}
	return nil
}

func scenarioHasUI(scenarioDir string) bool {
	info, err := os.Stat(filepath.Join(scenarioDir, "ui"))
	return err == nil && info.IsDir()
}

func resolveScenarioPort(ctx context.Context, scenarioName, portName string) (string, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", scenarioName, portName)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("vrooli port lookup failed: %v: %s", err, stderr.String())
	}
	value := strings.TrimSpace(stdout.String())
	if value == "" {
		return "", fmt.Errorf("port lookup returned empty output")
	}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if strings.TrimSpace(parts[0]) == portName {
				value = strings.TrimSpace(parts[1])
				break
			}
		}
	}
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "\r")
	if _, err := strconv.Atoi(value); err != nil {
		return "", fmt.Errorf("invalid port value %q", value)
	}
	return value, nil
}

func resolveScenarioBaseURL(ctx context.Context, scenarioName string) (string, error) {
	port, err := resolveScenarioPort(ctx, scenarioName, "UI_PORT")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://127.0.0.1:%s", port), nil
}

func startScenario(ctx context.Context, scenarioName string, logWriter io.Writer) error {
	logPhaseStep(logWriter, "starting scenario %s for BAS workflow execution", scenarioName)
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "start", scenarioName, "--clean-stale")
	var stderr bytes.Buffer
	cmd.Stdout = logWriter
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start %s: %v %s", scenarioName, err, stderr.String())
	}
	return nil
}

func waitForBASHealth(ctx context.Context, apiBase string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(45 * time.Second)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("browser-automation-studio health check timeout")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/health", apiBase), nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode < 300 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
}

func extractPort(baseURL string) string {
	if baseURL == "" {
		return ""
	}
	start := strings.LastIndex(baseURL, ":")
	if start == -1 {
		return ""
	}
	return baseURL[start+1:]
}
