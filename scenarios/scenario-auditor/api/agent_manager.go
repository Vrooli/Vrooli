package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	agentStatusRunning   = "running"
	agentStatusStopping  = "stopping"
	agentStatusStopped   = "stopped"
	agentStatusCompleted = "completed"
	agentStatusFailed    = "failed"
)

const (
	agentActionStandardsFix     = "standards_fix"
	agentActionVulnerabilityFix = "vulnerability_fix"
	agentActionCreateRule       = "create_rule"
	agentActionEditRule         = "edit_rule"

	// Rule test action labels, kept for backward compatibility with logging/history
	agentActionAddRuleTests = "add_rule_tests"
	agentActionFixRuleTests = "fix_rule_tests"
)

const (
	defaultOpenRouterModel = "openrouter/x-ai/grok-code-fast-1"
	defaultAllowedTools    = "read,write,edit,bash"

	agentManagerProfileKey   = "scenario-auditor-opencode"
	agentManagerProfileName  = "Scenario Auditor OpenCode"
	agentManagerPollInterval = 2 * time.Second
	agentManagerRequestTTL   = 30 * time.Second
	agentManagerWatchTTL     = 2 * time.Hour
	agentHistoryLimit        = 50
	metadataAttachmentKey    = "scenario-auditor-metadata"
)

var openRouterModel = resolveOpenRouterModel()

// AgentInfo represents an active or completed agent run tracked by scenario-auditor.
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

type agentExecution struct {
	RunID         string
	TaskID        string
	StartedAt     time.Time
	StopRequested bool
	Metadata      auditorAgentMetadata
}

type auditorAgentMetadata struct {
	Name               string            `json:"name,omitempty"`
	Label              string            `json:"label,omitempty"`
	Action             string            `json:"action,omitempty"`
	RuleID             string            `json:"rule_id,omitempty"`
	Scenario           string            `json:"scenario,omitempty"`
	Model              string            `json:"model,omitempty"`
	PromptLength       int               `json:"prompt_length,omitempty"`
	AllowedTools       []string          `json:"allowed_tools,omitempty"`
	MaxTurns           int32             `json:"max_turns,omitempty"`
	TaskTimeoutSeconds int32             `json:"task_timeout_seconds,omitempty"`
	IssueIDs           []string          `json:"issue_ids,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

type AgentManager struct {
	mu                 sync.RWMutex
	active             map[string]*agentExecution
	history            map[string]AgentInfo
	completionRecorded map[string]bool
	client             *agentManagerClient
	logger             *Logger
}

func NewAgentManager() *AgentManager {
	return &AgentManager{
		active:             make(map[string]*agentExecution),
		history:            make(map[string]AgentInfo),
		completionRecorded: make(map[string]bool),
		client:             newAgentManagerClient(agentManagerRequestTTL),
		logger:             NewLogger(),
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
	base := 300
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

	repoCtx, err := repoContext()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repo context: %w", err)
	}
	scenarioRoot := strings.TrimSpace(repoCtx.ScenarioAuditorRoot())

	// Build a repo-relative scope path for the sandbox. agent-manager forwards
	// scope_path as VROOLI_SANDBOX_SCOPE, and the CLI sandbox resolver
	// (repocontract.ScenarioScopeMatch) requires a relative form like
	// "scenarios/<name>"; absolute paths fail the prefix match silently.
	// When cfg.Scenario is empty, fall back to the auditor's own scope so the
	// agent's edits land somewhere coherent rather than escaping the overlay.
	auditTarget := strings.TrimSpace(cfg.Scenario)
	if auditTarget == "" {
		auditTarget = "scenario-auditor"
	}
	sandboxScope := path.Join("scenarios", auditTarget)

	cfg.Model = normalizeAgentModel(cfg.Model)
	allowedTools := configuredAllowedTools()
	maxTurns := configuredMaxTurns(len(cfg.IssueIDs))
	taskTimeoutSeconds := configuredTaskTimeout(len(cfg.IssueIDs))

	metadata := cloneMetadata(cfg.Metadata)
	if cfg.Scenario != "" {
		metadata["scenario"] = cfg.Scenario
	}
	if len(cfg.IssueIDs) > 0 {
		metadata["issue_count"] = strconv.Itoa(len(cfg.IssueIDs))
	}

	agentID := uuid.NewString()
	startedAt := time.Now().UTC()
	taskTitle := fallbackAgentName(cfg.Name, cfg.Label, cfg.Action, cfg.RuleID)
	agentMeta := auditorAgentMetadata{
		Name:               cfg.Name,
		Label:              cfg.Label,
		Action:             cfg.Action,
		RuleID:             cfg.RuleID,
		Scenario:           cfg.Scenario,
		Model:              cfg.Model,
		PromptLength:       len([]rune(cfg.Prompt)),
		AllowedTools:       append([]string(nil), allowedTools...),
		MaxTurns:           int32(maxTurns),
		TaskTimeoutSeconds: int32(taskTimeoutSeconds),
		IssueIDs:           append([]string(nil), cfg.IssueIDs...),
		Metadata:           metadata,
	}

	ctx, cancel := context.WithTimeout(context.Background(), agentManagerRequestTTL)
	defer cancel()

	if err := am.ensureProfile(ctx); err != nil {
		return nil, err
	}

	task, err := am.client.CreateTask(ctx, &domainpb.Task{
		Title:       taskTitle,
		Description: cfg.Prompt,
		ScopePath:   sandboxScope,
		ProjectRoot: scenarioRoot,
		CreatedBy:   serviceName,
		CreatedAt:   timestamppb.New(startedAt),
		UpdatedAt:   timestamppb.New(startedAt),
		ContextAttachments: []*domainpb.ContextAttachment{
			buildMetadataAttachment(agentMeta),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create agent-manager task: %w", err)
	}

	tag := agentID
	// RunMode is left unset so the orchestrator derives sandboxed via
	// the profile's SandboxConfig.Mode (Protected). Sandboxed mode is
	// required to get VROOLI_SANDBOX_* env vars injected, which keeps
	// the auditor's CLI helpers (cliutil.ResolveScenarioPath in
	// cli/internal/support) operating on the agent's overlay rather
	// than the real repo.
	run, err := am.client.CreateRun(ctx, &apipb.CreateRunRequest{
		TaskId:         task.Id,
		Tag:            &tag,
		Force:          true,
		IdempotencyKey: stringPtr("scenario-auditor:" + agentID),
		ProfileRef: &apipb.ProfileRef{
			ProfileKey: agentManagerProfileKey,
			Defaults:   am.defaultProfile(),
		},
		InlineConfig: &domainpb.RunConfigOverrides{
			Model:                stringPtr(cfg.Model),
			MaxTurns:             int32Ptr(int32(maxTurns)),
			Timeout:              durationpb.New(time.Duration(taskTimeoutSeconds) * time.Second),
			AllowedTools:         append([]string(nil), allowedTools...),
			SkipPermissionPrompt: boolPtr(true),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create agent-manager run: %w", err)
	}

	am.mu.Lock()
	am.active[agentID] = &agentExecution{
		RunID:     run.Id,
		TaskID:    task.Id,
		StartedAt: startedAt,
		Metadata:  agentMeta,
	}
	delete(am.history, agentID)
	am.mu.Unlock()

	info := am.composeAgentInfo(agentID, run, task, agentMeta, startedAt, false)
	am.logger.Info(fmt.Sprintf("Started agent %s via agent-manager run %s", agentID, run.Id))

	go am.watchRun(agentID, run.Id, task.Id, startedAt, agentMeta)

	return &info, nil
}

func (am *AgentManager) StopAgent(agentID string) error {
	run, task, meta, startedAt, _, err := am.loadRunDetails(agentID)
	if err != nil {
		return err
	}
	if run == nil {
		return fmt.Errorf("agent not found")
	}
	if isTerminalRunStatus(run.Status) {
		info := am.composeAgentInfo(agentID, run, task, meta, startedAt, false)
		am.cacheTerminalInfo(agentID, info)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), agentManagerRequestTTL)
	defer cancel()
	if err := am.client.StopRunByTag(ctx, agentID); err != nil {
		return fmt.Errorf("stop agent-manager run %s: %w", agentID, err)
	}

	am.mu.Lock()
	if active := am.active[agentID]; active != nil {
		active.StopRequested = true
	}
	am.mu.Unlock()

	am.logger.Info(fmt.Sprintf("Stopping agent %s via agent-manager", agentID))
	return nil
}

func (am *AgentManager) GetAgent(agentID string) (*AgentInfo, bool) {
	run, task, meta, startedAt, stopRequested, err := am.loadRunDetails(agentID)
	if err != nil || run == nil {
		return nil, false
	}

	info := am.composeAgentInfo(agentID, run, task, meta, startedAt, stopRequested)
	if isTerminalRunStatus(run.Status) {
		am.cacheTerminalInfo(agentID, info)
		am.recordCompletion(agentID, run.Status == domainpb.RunStatus_RUN_STATUS_COMPLETE)
		return nil, false
	}
	return &info, true
}

func (am *AgentManager) GetAgentHistory(agentID string) (*AgentInfo, bool) {
	am.mu.RLock()
	if info, ok := am.history[agentID]; ok {
		copyInfo := info
		am.mu.RUnlock()
		return &copyInfo, true
	}
	am.mu.RUnlock()

	run, task, meta, startedAt, stopRequested, err := am.loadRunDetails(agentID)
	if err != nil || run == nil || !isTerminalRunStatus(run.Status) {
		return nil, false
	}

	info := am.composeAgentInfo(agentID, run, task, meta, startedAt, stopRequested)
	am.cacheTerminalInfo(agentID, info)
	am.recordCompletion(agentID, run.Status == domainpb.RunStatus_RUN_STATUS_COMPLETE)
	return &info, true
}

func (am *AgentManager) ListAgents() []AgentInfo {
	am.mu.RLock()
	agentIDs := make([]string, 0, len(am.active))
	for agentID := range am.active {
		agentIDs = append(agentIDs, agentID)
	}
	am.mu.RUnlock()

	result := make([]AgentInfo, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		if info, ok := am.GetAgent(agentID); ok {
			result = append(result, *info)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.Before(result[j].StartedAt)
	})
	return result
}

func (am *AgentManager) AgentLogPath(agentID string) string {
	run, _, _, _, _, err := am.loadRunDetails(agentID)
	if err == nil && run != nil && strings.TrimSpace(run.LogPath) != "" {
		return strings.TrimSpace(run.LogPath)
	}
	ctx, err := repoContext()
	if err != nil {
		return filepath.Join("logs", "agents", fmt.Sprintf("%s.log", agentID))
	}
	return filepath.Join(ctx.ScenarioAuditorRoot(), "logs", "agents", fmt.Sprintf("%s.log", agentID))
}

func (am *AgentManager) watchRun(agentID, runID, taskID string, startedAt time.Time, meta auditorAgentMetadata) {
	ctx, cancel := context.WithTimeout(context.Background(), agentManagerWatchTTL)
	defer cancel()

	run, err := am.client.WaitForRun(ctx, runID, agentManagerPollInterval)
	if err != nil {
		am.logger.Error(fmt.Sprintf("wait for agent-manager run %s failed", runID), err)
		return
	}

	task, taskErr := am.fetchTask(taskID)
	if taskErr != nil {
		am.logger.Error(fmt.Sprintf("fetch agent-manager task %s failed", taskID), taskErr)
	}

	stopRequested := false
	am.mu.RLock()
	if active := am.active[agentID]; active != nil {
		stopRequested = active.StopRequested
	}
	am.mu.RUnlock()

	info := am.composeAgentInfo(agentID, run, task, meta, startedAt, stopRequested)
	am.cacheTerminalInfo(agentID, info)
	am.recordCompletion(agentID, run.Status == domainpb.RunStatus_RUN_STATUS_COMPLETE)

	switch run.Status {
	case domainpb.RunStatus_RUN_STATUS_COMPLETE:
		am.logger.Info(fmt.Sprintf("Agent %s completed via agent-manager", agentID))
	case domainpb.RunStatus_RUN_STATUS_CANCELLED:
		am.logger.Info(fmt.Sprintf("Agent %s cancelled via agent-manager", agentID))
	default:
		am.logger.Error(fmt.Sprintf("agent-manager run %s failed", runID), errors.New(strings.TrimSpace(run.ErrorMsg)))
	}
}

func (am *AgentManager) ensureProfile(ctx context.Context) error {
	_, err := am.client.EnsureProfile(ctx, &apipb.EnsureProfileRequest{
		ProfileKey:     agentManagerProfileKey,
		Defaults:       am.defaultProfile(),
		UpdateExisting: false,
	})
	if err != nil {
		return fmt.Errorf("ensure scenario-auditor profile: %w", err)
	}
	return nil
}

func (am *AgentManager) defaultProfile() *domainpb.AgentProfile {
	timeout := int32(configuredTaskTimeout(1))
	return &domainpb.AgentProfile{
		Name:                 agentManagerProfileName,
		ProfileKey:           agentManagerProfileKey,
		Description:          "OpenCode profile for scenario-auditor automated fixes",
		RunnerType:           domainpb.RunnerType_RUNNER_TYPE_OPENCODE,
		Model:                openRouterModel,
		MaxTurns:             int32(configuredMaxTurns(1)),
		Timeout:              durationpb.New(time.Duration(timeout) * time.Second),
		AllowedTools:         configuredAllowedTools(),
		SkipPermissionPrompt: true,
		// Run sandboxed so the auditor CLI's sandbox-aware path resolution
		// (cliutil.ResolveScenarioPath) gets activated by VROOLI_SANDBOX_*.
		// ManualReview defaults to false so audit fixes flow into the canonical
		// repo with provenance recorded for traceability.
		SandboxConfig: &domainpb.SandboxConfig{Mode: domainpb.SandboxMode_SANDBOX_MODE_PROTECTED},
		CreatedBy:     serviceName,
	}
}

func buildMetadataAttachment(meta auditorAgentMetadata) *domainpb.ContextAttachment {
	body, _ := json.Marshal(meta)
	return &domainpb.ContextAttachment{
		Type:     "note",
		Label:    "Scenario Auditor Metadata",
		Key:      metadataAttachmentKey,
		Content:  string(body),
		Summary:  "Scenario-auditor metadata used to reconstruct fix status and history.",
		Format:   "json",
		Priority: "high",
	}
}

func (am *AgentManager) loadRunDetails(agentID string) (*domainpb.Run, *domainpb.Task, auditorAgentMetadata, time.Time, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), agentManagerRequestTTL)
	defer cancel()

	run, err := am.client.GetRunByTag(ctx, agentID)
	if err != nil {
		return nil, nil, auditorAgentMetadata{}, time.Time{}, false, err
	}
	if run == nil {
		return nil, nil, auditorAgentMetadata{}, time.Time{}, false, nil
	}

	am.mu.RLock()
	active := am.active[agentID]
	am.mu.RUnlock()

	task, err := am.fetchTask(run.TaskId)
	if err != nil {
		return nil, nil, auditorAgentMetadata{}, time.Time{}, false, err
	}

	meta := auditorAgentMetadata{}
	startedAt := runCreatedAt(run)
	stopRequested := false
	if active != nil {
		meta = active.Metadata
		startedAt = active.StartedAt
		stopRequested = active.StopRequested
	}

	if meta.isZero() {
		meta = metadataFromTask(task)
	}
	if startedAt.IsZero() {
		startedAt = runCreatedAt(run)
	}

	return run, task, meta, startedAt, stopRequested, nil
}

func (am *AgentManager) fetchTask(taskID string) (*domainpb.Task, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), agentManagerRequestTTL)
	defer cancel()
	return am.client.GetTask(ctx, taskID)
}

func (am *AgentManager) composeAgentInfo(agentID string, run *domainpb.Run, task *domainpb.Task, meta auditorAgentMetadata, startedAt time.Time, stopRequested bool) AgentInfo {
	if run == nil {
		return AgentInfo{}
	}

	name := meta.Name
	if strings.TrimSpace(name) == "" && task != nil {
		name = strings.TrimSpace(task.Title)
	}
	name = fallbackAgentName(name, meta.Label, meta.Action, meta.RuleID)

	if startedAt.IsZero() {
		startedAt = runCreatedAt(run)
	}

	var endedAt *time.Time
	if ts := timestampToTime(run.EndedAt); !ts.IsZero() {
		ended := ts
		endedAt = &ended
	}

	durationEnd := time.Now().UTC()
	if endedAt != nil {
		durationEnd = *endedAt
	}

	model := strings.TrimSpace(meta.Model)
	if model == "" && run.GetResolvedConfig() != nil {
		model = strings.TrimSpace(run.GetResolvedConfig().GetModel())
	}
	if model == "" {
		model = openRouterModel
	}

	metadata := cloneMetadata(meta.Metadata)
	metadata["agent_manager_run_id"] = run.Id
	metadata["agent_manager_task_id"] = run.TaskId
	if strings.TrimSpace(run.LogPath) != "" {
		metadata["log_path"] = strings.TrimSpace(run.LogPath)
	}
	if strings.TrimSpace(run.SessionId) != "" {
		metadata["session_id"] = strings.TrimSpace(run.SessionId)
	}
	if task != nil && strings.TrimSpace(task.ProjectRoot) != "" {
		metadata["project_root"] = strings.TrimSpace(task.ProjectRoot)
	}
	if len(meta.AllowedTools) > 0 {
		metadata["allowed_tools"] = strings.Join(meta.AllowedTools, ",")
	}
	if meta.MaxTurns > 0 {
		metadata["max_turns"] = strconv.Itoa(int(meta.MaxTurns))
	}
	if meta.TaskTimeoutSeconds > 0 {
		metadata["task_timeout"] = strconv.Itoa(int(meta.TaskTimeoutSeconds))
	}

	info := AgentInfo{
		ID:              agentID,
		Name:            name,
		Label:           meta.Label,
		Action:          meta.Action,
		RuleID:          meta.RuleID,
		Scenario:        meta.Scenario,
		Model:           model,
		Status:          mapRunStatus(run.Status, stopRequested),
		StartedAt:       startedAt,
		EndedAt:         endedAt,
		DurationSeconds: int(durationEnd.Sub(startedAt).Seconds()),
		Command:         buildAgentCommand(model, meta.AllowedTools, meta.MaxTurns, meta.TaskTimeoutSeconds),
		PromptLength:    meta.PromptLength,
		Metadata:        metadata,
		IssueIDs:        append([]string(nil), meta.IssueIDs...),
		Error:           strings.TrimSpace(run.ErrorMsg),
	}
	if info.DurationSeconds < 0 {
		info.DurationSeconds = 0
	}
	return info
}

func (am *AgentManager) cacheTerminalInfo(agentID string, info AgentInfo) {
	am.mu.Lock()
	defer am.mu.Unlock()

	delete(am.active, agentID)
	am.history[agentID] = info
	if len(am.history) <= agentHistoryLimit {
		return
	}

	type entry struct {
		id      string
		started time.Time
	}
	entries := make([]entry, 0, len(am.history))
	for id, item := range am.history {
		entries = append(entries, entry{id: id, started: item.StartedAt})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].started.Before(entries[j].started)
	})
	for len(entries) > agentHistoryLimit {
		delete(am.history, entries[0].id)
		entries = entries[1:]
	}
}

func (am *AgentManager) recordCompletion(agentID string, success bool) {
	am.mu.Lock()
	if am.completionRecorded[agentID] {
		am.mu.Unlock()
		return
	}
	am.completionRecorded[agentID] = true
	am.mu.Unlock()
	automatedFixStore.RecordCompletion(agentID, success)
}

func (m auditorAgentMetadata) isZero() bool {
	return m.Name == "" &&
		m.Label == "" &&
		m.Action == "" &&
		m.RuleID == "" &&
		m.Scenario == "" &&
		m.Model == "" &&
		m.PromptLength == 0 &&
		len(m.AllowedTools) == 0 &&
		m.MaxTurns == 0 &&
		m.TaskTimeoutSeconds == 0 &&
		len(m.IssueIDs) == 0 &&
		len(m.Metadata) == 0
}

func metadataFromTask(task *domainpb.Task) auditorAgentMetadata {
	if task == nil {
		return auditorAgentMetadata{}
	}
	for _, attachment := range task.ContextAttachments {
		if attachment == nil || attachment.Key != metadataAttachmentKey {
			continue
		}
		var meta auditorAgentMetadata
		if err := json.Unmarshal([]byte(attachment.Content), &meta); err == nil {
			if meta.Metadata == nil {
				meta.Metadata = map[string]string{}
			}
			return meta
		}
	}
	return auditorAgentMetadata{
		Name:         strings.TrimSpace(task.Title),
		PromptLength: len([]rune(task.Description)),
		Metadata:     map[string]string{},
	}
}

func configuredAllowedTools() []string {
	raw := strings.TrimSpace(os.Getenv("SCENARIO_AUDITOR_ALLOWED_TOOLS"))
	if raw == "" {
		raw = defaultAllowedTools
	}
	parts := strings.Split(raw, ",")
	tools := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		tool := strings.TrimSpace(part)
		if tool == "" {
			continue
		}
		key := strings.ToLower(tool)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tools = append(tools, tool)
	}
	return tools
}

func configuredMaxTurns(issueCount int) int {
	value := strings.TrimSpace(os.Getenv("SCENARIO_AUDITOR_AGENT_MAX_TURNS"))
	if value == "" {
		return estimateMaxTurns(issueCount)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return estimateMaxTurns(issueCount)
	}
	return parsed
}

func configuredTaskTimeout(issueCount int) int {
	value := strings.TrimSpace(os.Getenv("SCENARIO_AUDITOR_AGENT_TIMEOUT"))
	if value == "" {
		return estimateTaskTimeout(issueCount)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return estimateTaskTimeout(issueCount)
	}
	return parsed
}

func buildAgentCommand(model string, allowedTools []string, maxTurns, taskTimeoutSeconds int32) []string {
	command := []string{"agent-manager", "create-run", "--runner", "opencode", "--model", model}
	if len(allowedTools) > 0 {
		command = append(command, "--allowed-tools", strings.Join(allowedTools, ","))
	}
	if maxTurns > 0 {
		command = append(command, "--max-turns", strconv.Itoa(int(maxTurns)))
	}
	if taskTimeoutSeconds > 0 {
		command = append(command, "--task-timeout", strconv.Itoa(int(taskTimeoutSeconds)))
	}
	return command
}

func mapRunStatus(status domainpb.RunStatus, stopRequested bool) string {
	if stopRequested && !isTerminalRunStatus(status) {
		return agentStatusStopping
	}
	switch status {
	case domainpb.RunStatus_RUN_STATUS_COMPLETE:
		return agentStatusCompleted
	case domainpb.RunStatus_RUN_STATUS_FAILED:
		return agentStatusFailed
	case domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return agentStatusStopped
	default:
		return agentStatusRunning
	}
}

func isTerminalRunStatus(status domainpb.RunStatus) bool {
	switch status {
	case domainpb.RunStatus_RUN_STATUS_COMPLETE,
		domainpb.RunStatus_RUN_STATUS_FAILED,
		domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return true
	default:
		return false
	}
}

func runCreatedAt(run *domainpb.Run) time.Time {
	if ts := timestampToTime(run.GetStartedAt()); !ts.IsZero() {
		return ts
	}
	if ts := timestampToTime(run.GetCreatedAt()); !ts.IsZero() {
		return ts
	}
	return time.Now().UTC()
}

func timestampToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	value := ts.AsTime().UTC()
	if value.IsZero() {
		return time.Time{}
	}
	return value
}

func stringPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
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
	case agentActionEditRule:
		if label != "" {
			return label
		}
		if ruleID != "" {
			return fmt.Sprintf("Edit %s", ruleID)
		}
		return "Edit rule"
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
