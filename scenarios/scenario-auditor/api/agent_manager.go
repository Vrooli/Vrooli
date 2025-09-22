package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	agentStatusRunning   = "running"
	agentStatusStopping  = "stopping"
	agentStatusStopped   = "stopped"
	agentStatusCompleted = "completed"
	agentStatusFailed    = "failed"
)

const (
	agentActionAddRuleTests     = "add_rule_tests"
	agentActionFixRuleTests     = "fix_rule_tests"
	agentActionStandardsFix     = "standards_fix"
	agentActionVulnerabilityFix = "vulnerability_fix"
	agentActionCreateRule       = "create_rule"
)

const (
	// defaultOpenRouterModel must include the provider prefix so resource-opencode
	// resolves it correctly via OpenRouter (provider/model syntax).
	defaultOpenRouterModel = "openrouter/x-ai/grok-code-fast-1"

	// defaultAllowedTools ensures agents can inspect and edit files without
	// prompting for permissions during automated runs.
	defaultAllowedTools = "read,write,edit,bash"
)

var openRouterModel = resolveOpenRouterModel()

// AgentInfo represents an active agent process that the API is tracking.
type AgentInfo struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Label           string            `json:"label"`
	Action          string            `json:"action"`
	RuleID          string            `json:"rule_id,omitempty"`
	Scenario        string            `json:"scenario,omitempty"`
	Model           string            `json:"model"`
	Status          string            `json:"status"`
	StartedAt       time.Time         `json:"started_at"`
	EndedAt         *time.Time        `json:"ended_at,omitempty"`
	DurationSeconds int               `json:"duration_seconds"`
	Command         []string          `json:"command"`
	PromptLength    int               `json:"prompt_length"`
	PID             int               `json:"pid,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	IssueIDs        []string          `json:"issue_ids,omitempty"`
	Error           string            `json:"error,omitempty"`
}

type agentInstance struct {
	info    AgentInfo
	cancel  context.CancelFunc
	execCmd *exec.Cmd
	logPath string
}

type AgentManager struct {
	mu      sync.RWMutex
	agents  map[string]*agentInstance
	logger  *Logger
	history map[string]AgentInfo
}

func NewAgentManager() *AgentManager {
	return &AgentManager{
		agents:  make(map[string]*agentInstance),
		logger:  NewLogger(),
		history: make(map[string]AgentInfo),
	}
}

func resolveOpenRouterModel() string {
	if override := strings.TrimSpace(os.Getenv("SCENARIO_AUDITOR_AGENT_MODEL")); override != "" {
		if strings.EqualFold(override, "default") {
			return defaultOpenRouterModel
		}
		if strings.Contains(override, "/") && !strings.HasPrefix(strings.ToLower(override), "openrouter/") {
			return "openrouter/" + override
		}
		return override
	}
	return defaultOpenRouterModel
}

func normalizeAgentModel(requested string) string {
	trimmed := strings.TrimSpace(requested)
	if trimmed == "" || strings.EqualFold(trimmed, "default") {
		return openRouterModel
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "openrouter/") || strings.HasPrefix(lower, "opencode/") || strings.HasPrefix(lower, "openai/") || strings.HasPrefix(lower, "anthropic/") || strings.HasPrefix(lower, "google/") || strings.HasPrefix(lower, "x-ai/") || strings.HasPrefix(lower, "mistral/") || strings.HasPrefix(lower, "deepseek/") {
		return trimmed
	}
	if strings.Contains(trimmed, "/") {
		return "openrouter/" + trimmed
	}
	return trimmed
}

func estimateMaxTurns(issueCount int) int {
	if issueCount < 1 {
		issueCount = 1
	}
	base := 12
	perIssue := 4
	maxTurns := 100
	estimate := base + perIssue*(issueCount-1)
	if estimate > maxTurns {
		estimate = maxTurns
	}
	return estimate
}

func estimateTaskTimeout(issueCount int) int {
	if issueCount < 1 {
		issueCount = 1
	}
	base := 300 // seconds
	perIssue := 90
	maxTimeout := 1800
	estimate := base + perIssue*(issueCount-1)
	if estimate > maxTimeout {
		estimate = maxTimeout
	}
	return estimate
}

func cloneMetadata(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

type AgentStartConfig struct {
	Label    string
	Name     string
	Action   string
	RuleID   string
	Scenario string
	IssueIDs []string
	Prompt   string
	Model    string
	Metadata map[string]string
}

func (am *AgentManager) StartAgent(cfg AgentStartConfig) (*AgentInfo, error) {
	if strings.TrimSpace(cfg.Prompt) == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	cfg.Model = normalizeAgentModel(cfg.Model)

	if cfg.Metadata == nil {
		cfg.Metadata = make(map[string]string)
	}
	if cfg.Scenario != "" {
		cfg.Metadata["scenario"] = cfg.Scenario
	}
	issueCount := len(cfg.IssueIDs)
	if issueCount > 0 {
		cfg.Metadata["issue_count"] = fmt.Sprintf("%d", issueCount)
	}

	agentID := uuid.New().String()
	allowedTools := strings.TrimSpace(os.Getenv("SCENARIO_AUDITOR_ALLOWED_TOOLS"))
	if allowedTools == "" {
		allowedTools = defaultAllowedTools
	}

	maxTurnsValue := strings.TrimSpace(os.Getenv("SCENARIO_AUDITOR_AGENT_MAX_TURNS"))
	if maxTurnsValue == "" {
		maxTurnsValue = strconv.Itoa(estimateMaxTurns(issueCount))
	}

	taskTimeoutValue := strings.TrimSpace(os.Getenv("SCENARIO_AUDITOR_AGENT_TIMEOUT"))
	if taskTimeoutValue == "" {
		taskTimeoutValue = strconv.Itoa(estimateTaskTimeout(issueCount))
	}

	command := []string{"agents", "run", "--model", cfg.Model, "--prompt", cfg.Prompt}
	if allowedTools != "" {
		command = append(command, "--allowed-tools", allowedTools)
	}

	if maxTurnsValue != "" {
		command = append(command, "--max-turns", maxTurnsValue)
	}

	if taskTimeoutValue != "" {
		command = append(command, "--task-timeout", taskTimeoutValue)
	}

	scenarioRoot := getScenarioRoot()
	logsDir := filepath.Join(scenarioRoot, "logs", "agents")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create agent logs directory: %w", err)
	}

	logPath := filepath.Join(logsDir, fmt.Sprintf("%s.log", agentID))
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent log file: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "resource-opencode", command...)
	cmd.Dir = scenarioRoot
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logFile.Close()
		cancel()
		return nil, fmt.Errorf("failed to connect stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logFile.Close()
		cancel()
		return nil, fmt.Errorf("failed to connect stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		cancel()
		return nil, fmt.Errorf("failed to start agent process: %w", err)
	}

	metadata := cloneMetadata(cfg.Metadata)
	metadata["log_path"] = logPath
	if allowedTools != "" {
		metadata["allowed_tools"] = allowedTools
	}
	if maxTurnsValue != "" {
		metadata["max_turns"] = maxTurnsValue
	}
	if taskTimeoutValue != "" {
		metadata["task_timeout"] = taskTimeoutValue
	}

	agentInfo := AgentInfo{
		ID:        agentID,
		Name:      fallbackAgentName(cfg.Name, cfg.Label, cfg.Action, cfg.RuleID),
		Label:     cfg.Label,
		Action:    cfg.Action,
		RuleID:    cfg.RuleID,
		Scenario:  cfg.Scenario,
		Model:     cfg.Model,
		Status:    agentStatusRunning,
		StartedAt: time.Now().UTC(),
		Command: func() []string {
			base := []string{"resource-opencode", "agents", "run", "--model", cfg.Model}
			if allowedTools != "" {
				base = append(base, "--allowed-tools", allowedTools)
			}
			if maxTurnsValue != "" {
				base = append(base, "--max-turns", maxTurnsValue)
			}
			if taskTimeoutValue != "" {
				base = append(base, "--task-timeout", taskTimeoutValue)
			}
			return base
		}(),
		PromptLength: len([]rune(cfg.Prompt)),
		PID:          cmd.Process.Pid,
		Metadata:     metadata,
		IssueIDs:     append([]string(nil), cfg.IssueIDs...),
	}

	agent := &agentInstance{
		info:    agentInfo,
		cancel:  cancel,
		execCmd: cmd,
		logPath: logPath,
	}

	am.mu.Lock()
	am.agents[agentID] = agent
	am.mu.Unlock()

	am.logger.Info(fmt.Sprintf("Started agent %s (%s)", agentID, agentInfo.Name))

	// Stream stdout/stderr to log file
	go func() {
		defer logFile.Close()
		multi := io.MultiReader(stdout, stderr)
		scanner := bufio.NewScanner(multi)
		scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)
		writer := bufio.NewWriter(logFile)
		defer writer.Flush()
		for scanner.Scan() {
			line := scanner.Text()
			writer.WriteString(line)
			writer.WriteByte('\n')
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			am.logger.Error(fmt.Sprintf("log streaming failed for agent %s", agentID), err)
		}
	}()

	// Wait for completion and record history
	go func() {
		err := cmd.Wait()
		endTime := time.Now().UTC()

		am.mu.Lock()
		if _, exists := am.agents[agentID]; exists {
			delete(am.agents, agentID)
		}

		finalInfo := agent.info
		finalInfo.EndedAt = &endTime
		finalInfo.DurationSeconds = int(endTime.Sub(finalInfo.StartedAt).Seconds())
		if err != nil {
			finalInfo.Status = agentStatusFailed
			failureMessage, summary := summariseAgentFailure(agent.logPath, err)
			if failureMessage != "" {
				finalInfo.Error = failureMessage
			} else {
				finalInfo.Error = err.Error()
			}
			if summary != "" {
				if finalInfo.Metadata == nil {
					finalInfo.Metadata = make(map[string]string)
				}
				finalInfo.Metadata["failure_reason"] = summary
			}
		} else if finalInfo.Status == agentStatusStopping {
			finalInfo.Status = agentStatusStopped
		} else {
			finalInfo.Status = agentStatusCompleted
		}
		if finalInfo.Metadata == nil {
			finalInfo.Metadata = make(map[string]string)
		}
		finalInfo.Metadata["log_path"] = agent.logPath
		am.history[agentID] = finalInfo
		if len(am.history) > 50 {
			for key := range am.history {
				if key == agentID {
					continue
				}
				delete(am.history, key)
				if len(am.history) <= 50 {
					break
				}
			}
		}
		am.mu.Unlock()

		if err != nil {
			automatedFixStore.RecordCompletion(agentID, false)
			am.logger.Error(fmt.Sprintf("agent %s exited with error", agentID), err)
		} else {
			automatedFixStore.RecordCompletion(agentID, true)
			am.logger.Info(fmt.Sprintf("Agent %s completed", agentID))
		}
	}()

	return &agentInfo, nil
}

func (am *AgentManager) StopAgent(agentID string) error {
	am.mu.Lock()
	agent, exists := am.agents[agentID]
	if !exists {
		am.mu.Unlock()
		return fmt.Errorf("agent not found")
	}
	agent.info.Status = agentStatusStopping
	am.agents[agentID] = agent
	am.mu.Unlock()

	am.logger.Info(fmt.Sprintf("Stopping agent %s", agentID))
	agent.cancel()
	if agent.execCmd != nil && agent.execCmd.Process != nil {
		_ = agent.execCmd.Process.Kill()
	}

	return nil
}

func (am *AgentManager) GetAgent(agentID string) (*AgentInfo, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	agent, exists := am.agents[agentID]
	if !exists {
		return nil, false
	}

	info := agent.info
	info.DurationSeconds = int(time.Since(info.StartedAt).Seconds())
	return &info, true
}

func (am *AgentManager) GetAgentHistory(agentID string) (*AgentInfo, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	info, exists := am.history[agentID]
	if !exists {
		return nil, false
	}
	copy := info
	return &copy, true
}

func (am *AgentManager) ListAgents() []AgentInfo {
	am.mu.RLock()
	defer am.mu.RUnlock()
	result := make([]AgentInfo, 0, len(am.agents))
	for _, agent := range am.agents {
		info := agent.info
		info.DurationSeconds = int(time.Since(info.StartedAt).Seconds())
		result = append(result, info)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.Before(result[j].StartedAt)
	})

	return result
}

func (am *AgentManager) AgentLogPath(agentID string) string {
	am.mu.RLock()
	agent, exists := am.agents[agentID]
	am.mu.RUnlock()
	if exists {
		return agent.logPath
	}

	logsDir := filepath.Join(getScenarioRoot(), "logs", "agents")
	return filepath.Join(logsDir, fmt.Sprintf("%s.log", agentID))
}

func fallbackAgentName(name, label, action, ruleID string) string {
	if name != "" {
		return name
	}
	if label != "" {
		return label
	}
	switch action {
	case agentActionAddRuleTests:
		if ruleID != "" {
			return fmt.Sprintf("Add tests for %s", ruleID)
		}
		return "Add rule tests"
	case agentActionFixRuleTests:
		if ruleID != "" {
			return fmt.Sprintf("Fix tests for %s", ruleID)
		}
		return "Fix rule tests"
	case agentActionCreateRule:
		if label != "" {
			return label
		}
		return "Create new rule"
	case agentActionStandardsFix:
		if label != "" {
			return label
		}
		return "Standards fix agent"
	case agentActionVulnerabilityFix:
		if label != "" {
			return label
		}
		return "Vulnerability fix agent"
	default:
		if ruleID != "" {
			return fmt.Sprintf("Agent for %s", ruleID)
		}
		return "Scenario agent"
	}
}

var (
	scenarioRootOnce sync.Once
	scenarioRootPath string
)

func summariseAgentFailure(logPath string, execErr error) (string, string) {
	message := strings.TrimSpace(execErr.Error())
	logSummary, err := extractFailureSummary(logPath)
	if err != nil || logSummary == "" {
		return message, ""
	}
	if message == "" {
		return logSummary, logSummary
	}
	if strings.Contains(logSummary, message) {
		return logSummary, logSummary
	}
	return fmt.Sprintf("%s (process reported: %s)", logSummary, message), logSummary
}

func extractFailureSummary(logPath string) (string, error) {
	data, err := os.ReadFile(logPath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	var fallback string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if fallback == "" {
			fallback = line
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "providermodelnotfounderror") {
			modelID := extractFieldValue(line, "modelID")
			if modelID == "" {
				modelID = extractFieldValue(line, "model")
			}
			return fmt.Sprintf("ProviderModelNotFoundError: model %s is not available for resource-opencode. Configure provider credentials or set SCENARIO_AUDITOR_AGENT_MODEL to a supported model.", safeValue(modelID)), nil
		}
		if strings.Contains(lower, "invalid api key") {
			return "Authentication failed for resource-opencode provider (invalid API key). Update credentials and retry.", nil
		}
		if strings.Contains(lower, "error") {
			return line, nil
		}
	}
	if fallback != "" {
		if len(fallback) > 400 {
			fallback = fallback[:400]
		}
		return fallback, nil
	}
	return "", nil
}

func extractFieldValue(line, key string) string {
	keyEq := key + "="
	for _, part := range strings.Fields(line) {
		if strings.HasPrefix(part, keyEq) {
			return strings.Trim(part[len(keyEq):], "\" ")
		}
	}
	return ""
}

func safeValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(unspecified)"
	}
	return value
}

func getScenarioRoot() string {
	scenarioRootOnce.Do(func() {
		wd, err := os.Getwd()
		if err != nil {
			scenarioRootPath = "."
			return
		}

		// When running from api directory, the scenario root is one level up.
		candidate := filepath.Clean(filepath.Join(wd, ".."))
		if _, err := os.Stat(filepath.Join(candidate, "rules")); err == nil {
			scenarioRootPath = candidate
			return
		}

		// Fallback to working directory
		scenarioRootPath = wd
	})

	return scenarioRootPath
}
