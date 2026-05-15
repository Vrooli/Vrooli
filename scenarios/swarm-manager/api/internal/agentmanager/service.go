// Package agentmanager provides a higher-level integration seam for agent-manager.
//
// This service hides HTTP/proto details from handlers and owns profile setup,
// tagging, and spawn orchestration for Swarm Manager.
//
// DOC: docs/concepts/ARCHITECTURE.md#design-principles
// DOC: docs/internal/SEAMS.md
// DOC: docs/internal/INTENT.md#what-not-to-modify-here
package agentmanager

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/scenario"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// freshConversationID mints a new ConversationID for a swarm-manager-spawned
// run. Per Decision D7 of the auditability contract, spawn surfaces SHOULD
// populate ConversationID explicitly rather than rely on agent-manager's
// fallback. Each top-level spawn from swarm-manager (research, backlog,
// initiative) is conceptually a fresh conversation.
func freshConversationID() *string {
	id := uuid.NewString()
	return &id
}

// Service defines the seam handlers depend on.
type Service interface {
	IsEnabled() bool
	IsAvailable(ctx context.Context) bool
	ResolveURL(ctx context.Context) (string, error)
	GetProfileID() string

	SpawnBacklog(ctx context.Context, req BacklogSpawnRequest) (RunResult, error)
	SpawnResearch(ctx context.Context, req ResearchSpawnRequest) (RunResult, error)
	GetRunState(ctx context.Context, runID string) (RunState, error)
	GetRunDiff(ctx context.Context, runID string) (RunDiff, error)
	StopRun(ctx context.Context, runID string) error
	ContinueRun(ctx context.Context, runID string, message string) error
}

// AgentService implements the Service interface.
type AgentService struct {
	client         *HTTPClient
	profileName    string
	profileKey     string
	requiredKeys   []string
	profileID      string
	profileIDs     map[string]string
	mu             sync.RWMutex
	enabled        bool
	settingsReader SettingsReader
}

// AgentServiceConfig contains configuration for the agent service.
type AgentServiceConfig struct {
	ProfileName    string
	ProfileKey     string
	RequiredKeys   []string
	Timeout        time.Duration
	Enabled        bool
	SettingsReader SettingsReader
}

// DefaultServiceConfig returns a baseline configuration for Swarm Manager.
func DefaultServiceConfig() AgentServiceConfig {
	return AgentServiceConfig{
		ProfileName: "swarm-manager",
		ProfileKey:  "swarm-manager/default",
		Timeout:     30 * time.Second,
		Enabled:     true,
	}
}

// NewAgentService creates a new agent service.
func NewAgentService(cfg AgentServiceConfig) *AgentService {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	client := NewHTTPClientWithTimeout(cfg.Timeout)
	return &AgentService{
		client:         client,
		profileName:    strings.TrimSpace(cfg.ProfileName),
		profileKey:     strings.TrimSpace(cfg.ProfileKey),
		requiredKeys:   normalizeProfileKeys(cfg.RequiredKeys),
		profileIDs:     make(map[string]string),
		enabled:        cfg.Enabled,
		settingsReader: cfg.SettingsReader,
	}
}

func normalizeProfileKeys(keys []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func (s *AgentService) requiredProfileKeys() []string {
	return normalizeProfileKeys(append([]string{s.profileKey}, s.requiredKeys...))
}

func (s *AgentService) validateRequiredProfiles(profileIDs map[string]string) error {
	const prefix = "swarm-manager/"
	for _, key := range s.requiredProfileKeys() {
		if !strings.HasPrefix(key, prefix) {
			return fmt.Errorf("required profile %q is not owned by scenario %q", key, "swarm-manager")
		}
		if strings.TrimSpace(profileIDs[key]) == "" {
			return fmt.Errorf("required profile %q was not returned", key)
		}
	}
	return nil
}

// IsEnabled returns whether agent-manager integration is enabled.
func (s *AgentService) IsEnabled() bool {
	return s.enabled
}

// IsAvailable checks if agent-manager is reachable.
func (s *AgentService) IsAvailable(ctx context.Context) bool {
	if !s.enabled {
		return false
	}
	ok, err := s.client.Health(ctx)
	return err == nil && ok
}

// ResolveURL returns the current agent-manager base URL.
func (s *AgentService) ResolveURL(ctx context.Context) (string, error) {
	if !s.enabled {
		return "", fmt.Errorf("agent-manager not enabled")
	}
	return s.client.ResolveURL(ctx)
}

// Initialize ensures the agent profile exists.
// Call this at startup to create/update the swarm-manager profile.
func (s *AgentService) Initialize(ctx context.Context, cfg *ProfileConfig) error {
	if !s.enabled {
		return nil
	}

	resp, err := s.client.ReconcileScenarioProfiles(ctx, scenario.Name())
	if err != nil {
		return fmt.Errorf("reconcile scenario profiles: %w", err)
	}

	s.mu.Lock()
	found := false
	profileIDs := make(map[string]string, len(resp.Results))
	for _, item := range resp.Results {
		profileIDs[item.ProfileKey] = item.ProfileId
		if item.ProfileKey == s.profileKey {
			s.profileID = item.ProfileId
			found = true
		}
	}
	s.profileIDs = profileIDs
	s.mu.Unlock()
	if resp.Failed > 0 {
		return fmt.Errorf("reconcile scenario profiles: %d profile source(s) failed validation", resp.Failed)
	}
	if err := s.validateRequiredProfiles(profileIDs); err != nil {
		return fmt.Errorf("reconcile scenario profiles: %w", err)
	}
	if !found {
		return fmt.Errorf("profile %q was not returned", s.profileKey)
	}

	slog.Info("reconciled agent profiles", "scenario", resp.Scenario, "created", resp.Created, "updated", resp.Updated, "unchanged", resp.Unchanged, "failed", resp.Failed)

	return nil
}

func (s *AgentService) ensureProfilesReconciled(ctx context.Context) error {
	if s.GetProfileID() != "" {
		return nil
	}
	return s.Initialize(ctx, nil)
}

// ResearchSpawnRequest describes a request to spawn an idea research agent.
type ResearchSpawnRequest struct {
	IdeaName    string
	Title       string
	Description string
	Prompt      string
	ScopePath   string
	ProjectRoot string
	CreatedBy   string
	Mode        string
	ProfileKey  string
}

// InitiativeSpawnRequest describes a request to spawn an initiative-scoped
// agent (feedback round, initiative review). Structurally identical to
// BacklogSpawnRequest except that Name identifies an initiative rather
// than a backlog item and Purpose distinguishes the round type
// ("feedback" | "feedback_continue" | "review").
type InitiativeSpawnRequest struct {
	Name               string
	Title              string
	Description        string
	Prompt             string
	ScopePath          string
	ProjectRoot        string
	CreatedBy          string
	Purpose            string
	RoundNumber        int
	RoundSlug          string
	AcceptanceAllow    []string
	AcceptanceDeny     []string
	Creates            []string
	Environment        map[string]string
	ContextAttachments []*domainpb.ContextAttachment
	ProfileKey         string
}

// SessionSpawnRequest describes a durable Swarm Manager-owned conversational
// workflow run. These runs are not scoped to a backlog item or initiative.
type SessionSpawnRequest struct {
	SessionID          string
	Kind               string
	Title              string
	Description        string
	Prompt             string
	ScopePath          string
	ProjectRoot        string
	CreatedBy          string
	Environment        map[string]string
	ContextAttachments []*domainpb.ContextAttachment
	ProfileKey         string
}

// BacklogSpawnRequest describes a request to spawn a backlog agent.
type BacklogSpawnRequest struct {
	Kind               string
	Name               string
	Title              string
	Description        string
	Prompt             string
	ScopePath          string
	ProjectRoot        string
	CreatedBy          string
	Purpose            string
	AcceptanceAllow    []string
	AcceptanceDeny     []string
	Creates            []string
	Environment        map[string]string
	ContextAttachments []*domainpb.ContextAttachment
	ProfileKey         string
}

// RunResult returns agent-manager identifiers.
type RunResult struct {
	TaskID    string
	RunID     string
	BaseURL   string
	CreatedAt string
}

// RunState captures externally visible lifecycle state for a run.
type RunState struct {
	RunID         string
	TaskID        string
	Status        string
	StartedAt     string
	FinishedAt    string
	ErrorMsg      string
	SandboxID     string
	Summary       string
	TokensUsed    int32
	TurnsUsed     int32
	CostEstimate  float64
	ChangedFiles  int32
	ContextTokens int32
}

type RunEventsOptions struct {
	AfterSequence int64
	Limit         int32
}

// RunDiff captures the changed files for a sandboxed run.
type RunDiff struct {
	RunID       string
	SandboxID   string
	GeneratedAt string
	Files       []RunDiffFile
}

// RunDiffFile captures one changed file from a run diff.
type RunDiffFile struct {
	Path       string
	ChangeType string
}

// SpawnResearch creates a research task/run in agent-manager.
func (s *AgentService) SpawnResearch(ctx context.Context, req ResearchSpawnRequest) (RunResult, error) {
	if !s.enabled {
		return RunResult{}, ErrNotAvailable
	}
	if err := s.ensureProfilesReconciled(ctx); err != nil {
		return RunResult{}, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = buildResearchTitle(req.Mode, req.IdeaName)
	}

	scopePath := strings.TrimSpace(req.ScopePath)
	if scopePath == "" {
		scopePath = "."
	}

	projectRoot := strings.TrimSpace(req.ProjectRoot)
	if projectRoot == "" {
		projectRoot = "."
	}

	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		createdBy = "swarm-manager"
	}

	task := &domainpb.Task{
		Title:       title,
		Description: truncateDescription(strings.TrimSpace(req.Description)),
		ScopePath:   scopePath,
		ProjectRoot: projectRoot,
		CreatedBy:   createdBy,
	}

	createdTask, err := s.client.CreateTask(ctx, task)
	if err != nil {
		return RunResult{}, err
	}

	tag := buildResearchTag(req.IdeaName)
	profileRef, err := s.profileRefFor(req.ProfileKey)
	if err != nil {
		return RunResult{}, err
	}
	runReq := &apipb.CreateRunRequest{
		ConversationId: freshConversationID(),
		TaskId:         createdTask.Id,
		ProfileRef:     profileRef,
		Tag:            &tag,
		Force:          true,
	}
	if prompt := strings.TrimSpace(req.Prompt); prompt != "" {
		runReq.Prompt = &prompt
	}

	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		return RunResult{}, err
	}

	baseURL, _ := s.ResolveURL(ctx)

	return RunResult{
		TaskID:    createdTask.Id,
		RunID:     run.Id,
		BaseURL:   baseURL,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// SpawnSession creates a general Swarm Manager session task/run in
// agent-manager.
func (s *AgentService) SpawnSession(ctx context.Context, req SessionSpawnRequest) (RunResult, error) {
	if !s.enabled {
		return RunResult{}, ErrNotAvailable
	}
	if err := s.ensureProfilesReconciled(ctx); err != nil {
		return RunResult{}, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = buildSessionTitle(req.Kind, req.SessionID)
	}

	scopePath, projectRoot, err := resolveScopeAndRoot(req.ScopePath, req.ProjectRoot, nil, nil)
	if err != nil {
		return RunResult{}, err
	}

	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		createdBy = "swarm-manager"
	}

	task := &domainpb.Task{
		Title:              title,
		Description:        truncateDescription(strings.TrimSpace(req.Description)),
		ScopePath:          scopePath,
		ProjectRoot:        projectRoot,
		CreatedBy:          createdBy,
		ContextAttachments: req.ContextAttachments,
	}

	createdTask, err := s.client.CreateTask(ctx, task)
	if err != nil {
		return RunResult{}, err
	}

	tag := buildSessionTag(req.Kind, req.SessionID)
	profileRef, err := s.profileRefFor(req.ProfileKey)
	if err != nil {
		return RunResult{}, err
	}
	runReq := &apipb.CreateRunRequest{
		ConversationId: freshConversationID(),
		TaskId:         createdTask.Id,
		ProfileRef:     profileRef,
		Tag:            &tag,
		Force:          true,
	}
	if prompt := strings.TrimSpace(req.Prompt); prompt != "" {
		runReq.Prompt = &prompt
	}
	if len(req.Environment) > 0 {
		runReq.Environment = req.Environment
	}

	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		return RunResult{}, err
	}

	baseURL, _ := s.ResolveURL(ctx)

	return RunResult{
		TaskID:    createdTask.Id,
		RunID:     run.Id,
		BaseURL:   baseURL,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// SpawnBacklog creates a backlog task/run in agent-manager.
func (s *AgentService) SpawnBacklog(ctx context.Context, req BacklogSpawnRequest) (RunResult, error) {
	if !s.enabled {
		return RunResult{}, ErrNotAvailable
	}
	if err := s.ensureProfilesReconciled(ctx); err != nil {
		return RunResult{}, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = buildBacklogTitle(req.Kind, req.Name, req.Purpose)
	}

	scopePath, projectRoot, err := resolveScopeAndRoot(req.ScopePath, req.ProjectRoot, req.AcceptanceAllow, req.Creates)
	if err != nil {
		return RunResult{}, err
	}

	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		createdBy = "swarm-manager"
	}

	task := &domainpb.Task{
		Title:              title,
		Description:        truncateDescription(strings.TrimSpace(req.Description)),
		ScopePath:          scopePath,
		ProjectRoot:        projectRoot,
		CreatedBy:          createdBy,
		ContextAttachments: req.ContextAttachments,
	}

	createdTask, err := s.client.CreateTask(ctx, task)
	if err != nil {
		return RunResult{}, err
	}

	tag := buildBacklogTag(req.Kind, req.Name, req.Purpose)
	profileRef, err := s.profileRefFor(req.ProfileKey)
	if err != nil {
		return RunResult{}, err
	}
	runReq := &apipb.CreateRunRequest{
		ConversationId: freshConversationID(),
		TaskId:         createdTask.Id,
		ProfileRef:     profileRef,
		Tag:            &tag,
		Force:          true,
	}
	if prompt := strings.TrimSpace(req.Prompt); prompt != "" {
		runReq.Prompt = &prompt
	}
	if len(req.Environment) > 0 {
		runReq.Environment = req.Environment
	}
	applyAcceptanceOverride(runReq, req.AcceptanceAllow, req.AcceptanceDeny)

	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		return RunResult{}, err
	}

	baseURL, _ := s.ResolveURL(ctx)

	return RunResult{
		TaskID:    createdTask.Id,
		RunID:     run.Id,
		BaseURL:   baseURL,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// SpawnInitiative creates an initiative-scoped task/run in agent-manager.
// Mirrors SpawnBacklog; the distinction is captured in the Tag and Title
// so downstream tooling (agent-activity list filters, logs) can
// distinguish feedback rounds from backlog executions.
func (s *AgentService) SpawnInitiative(ctx context.Context, req InitiativeSpawnRequest) (RunResult, error) {
	if !s.enabled {
		return RunResult{}, ErrNotAvailable
	}
	if err := s.ensureProfilesReconciled(ctx); err != nil {
		return RunResult{}, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = buildInitiativeTitle(req.Name, req.Purpose, req.RoundNumber)
	}

	scopePath, projectRoot, err := resolveScopeAndRoot(req.ScopePath, req.ProjectRoot, req.AcceptanceAllow, req.Creates)
	if err != nil {
		return RunResult{}, err
	}

	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		createdBy = "swarm-manager"
	}

	task := &domainpb.Task{
		Title:              title,
		Description:        truncateDescription(strings.TrimSpace(req.Description)),
		ScopePath:          scopePath,
		ProjectRoot:        projectRoot,
		CreatedBy:          createdBy,
		ContextAttachments: req.ContextAttachments,
	}

	createdTask, err := s.client.CreateTask(ctx, task)
	if err != nil {
		return RunResult{}, err
	}

	tag := buildInitiativeTag(req.Name, req.Purpose, req.RoundNumber)
	profileRef, err := s.profileRefFor(req.ProfileKey)
	if err != nil {
		return RunResult{}, err
	}
	runReq := &apipb.CreateRunRequest{
		ConversationId: freshConversationID(),
		TaskId:         createdTask.Id,
		ProfileRef:     profileRef,
		Tag:            &tag,
		Force:          true,
	}
	if prompt := strings.TrimSpace(req.Prompt); prompt != "" {
		runReq.Prompt = &prompt
	}
	if len(req.Environment) > 0 {
		runReq.Environment = req.Environment
	}
	applyAcceptanceOverride(runReq, req.AcceptanceAllow, req.AcceptanceDeny)

	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		return RunResult{}, err
	}

	baseURL, _ := s.ResolveURL(ctx)

	return RunResult{
		TaskID:    createdTask.Id,
		RunID:     run.Id,
		BaseURL:   baseURL,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// applyAcceptanceOverride builds a SandboxAcceptanceConfig allowlist override
// from the caller's allow/deny path globs and attaches it to runReq.InlineConfig.
//
// We deliberately set ONLY the Acceptance field on SandboxConfig here: the
// orchestrator's resolveSandboxConfig field-wise backfills Mode and
// NetworkMode from DefaultSandboxConfig() (Protected, localhost) so this
// helper never accidentally strips the protected-mode default by emitting
// a partially-zeroed config. See PROTECTED_MODE_RUNNERS.md "Default-mode
// policy".
func applyAcceptanceOverride(runReq *apipb.CreateRunRequest, allow, deny []string) {
	if len(allow) == 0 && len(deny) == 0 {
		return
	}
	acceptance := &domainpb.SandboxAcceptanceConfig{
		Mode: domainpb.SandboxAcceptanceMode_SANDBOX_ACCEPTANCE_MODE_ALLOWLIST,
	}
	if len(allow) > 0 {
		acceptance.Allow = &domainpb.SandboxFileCriteria{PathGlobs: allow}
	}
	if len(deny) > 0 {
		acceptance.Deny = &domainpb.SandboxFileCriteria{PathGlobs: deny}
	}
	runReq.InlineConfig = &domainpb.RunConfigOverrides{
		SandboxConfig: &domainpb.SandboxConfig{
			Acceptance: acceptance,
		},
	}
}

// GetRunState resolves run state from agent-manager.
func (s *AgentService) GetRunState(ctx context.Context, runID string) (RunState, error) {
	if !s.enabled {
		return RunState{}, ErrNotAvailable
	}

	run, err := s.client.GetRun(ctx, runID)
	if err != nil {
		return RunState{}, err
	}

	state := RunState{
		RunID:     strings.TrimSpace(run.Id),
		TaskID:    strings.TrimSpace(run.TaskId),
		Status:    normalizeRunStatus(run.Status),
		ErrorMsg:  strings.TrimSpace(run.ErrorMsg),
		SandboxID: strings.TrimSpace(run.GetSandboxId()),
	}
	if run.StartedAt != nil {
		state.StartedAt = run.StartedAt.AsTime().UTC().Format(time.RFC3339)
	}
	if run.EndedAt != nil {
		state.FinishedAt = run.EndedAt.AsTime().UTC().Format(time.RFC3339)
	}
	if run.Summary != nil {
		if run.Summary.Description != "" {
			state.Summary = strings.TrimSpace(run.Summary.Description)
		}
		state.TokensUsed = run.Summary.TokensUsed
		state.TurnsUsed = run.Summary.TurnsUsed
		state.CostEstimate = run.Summary.CostEstimate
		state.ContextTokens = run.Summary.ContextTokens
	}
	state.ChangedFiles = run.ChangedFiles
	return state, nil
}

func (s *AgentService) GetRunEvents(ctx context.Context, runID string, opts RunEventsOptions) ([]*domainpb.RunEvent, bool, error) {
	if !s.enabled {
		return nil, false, ErrNotAvailable
	}
	resp, err := s.client.GetRunEvents(ctx, runID, opts)
	if err != nil {
		return nil, false, err
	}
	return resp.Events, resp.HasMore, nil
}

// GetRunDiff resolves changed files for a sandboxed run.
func (s *AgentService) GetRunDiff(ctx context.Context, runID string) (RunDiff, error) {
	if !s.enabled {
		return RunDiff{}, ErrNotAvailable
	}

	run, err := s.client.GetRun(ctx, runID)
	if err != nil {
		return RunDiff{}, err
	}
	diff, err := s.client.GetRunDiff(ctx, runID)
	if err != nil {
		return RunDiff{}, err
	}

	result := RunDiff{
		RunID:     strings.TrimSpace(diff.RunId),
		SandboxID: strings.TrimSpace(run.GetSandboxId()),
	}
	if diff.GeneratedAt != nil {
		result.GeneratedAt = diff.GeneratedAt.AsTime().UTC().Format(time.RFC3339)
	}
	for _, file := range diff.Files {
		result.Files = append(result.Files, RunDiffFile{
			Path:       strings.TrimSpace(file.Path),
			ChangeType: strings.TrimSpace(file.ChangeType),
		})
	}
	return result, nil
}

// ContinueRun sends a follow-up message to an existing run.
func (s *AgentService) ContinueRun(ctx context.Context, runID string, message string) error {
	if !s.enabled {
		return ErrNotAvailable
	}
	return s.client.ContinueRun(ctx, runID, message)
}

// StopRun requests cancellation of an in-flight run.
func (s *AgentService) StopRun(ctx context.Context, runID string) error {
	if !s.enabled {
		return ErrNotAvailable
	}
	return s.client.StopRun(ctx, runID)
}

func buildResearchTag(ideaName string) string {
	ideaName = strings.TrimSpace(ideaName)
	if ideaName == "" {
		return "swarm-manager:idea:research"
	}
	return fmt.Sprintf("swarm-manager:idea:%s:research", ideaName)
}

func buildResearchTitle(mode, ideaName string) string {
	label := strings.TrimSpace(ideaName)
	if label == "" {
		label = "idea"
	}
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "clarify":
		return "Clarify idea: " + label
	case "suggest":
		return "Suggest improvements: " + label
	case "enhance":
		return "Enhance idea: " + label
	default:
		return "Research idea: " + label
	}
}

func buildInitiativeTag(name, purpose string, round int) string {
	name = strings.TrimSpace(name)
	purpose = strings.TrimSpace(purpose)
	if name == "" {
		name = "initiative"
	}
	tag := fmt.Sprintf("swarm-manager:initiative:%s", name)
	if purpose != "" {
		tag = fmt.Sprintf("%s:%s", tag, purpose)
	}
	if round > 0 {
		tag = fmt.Sprintf("%s:round-%03d", tag, round)
	}
	return tag
}

func buildInitiativeTitle(name, purpose string, round int) string {
	label := strings.TrimSpace(name)
	if label == "" {
		label = "initiative"
	}
	switch strings.ToLower(strings.TrimSpace(purpose)) {
	case "feedback":
		if round > 0 {
			return fmt.Sprintf("Feedback round %d: %s", round, label)
		}
		return "Feedback: " + label
	case "feedback_continue":
		if round > 0 {
			return fmt.Sprintf("Feedback round %d (continue): %s", round, label)
		}
		return "Feedback continue: " + label
	case "review":
		return "Review: " + label
	}
	return "Initiative: " + label
}

func buildSessionTag(kind, sessionID string) string {
	kind = strings.TrimSpace(kind)
	sessionID = strings.TrimSpace(sessionID)
	if kind == "" {
		kind = "session"
	}
	tag := fmt.Sprintf("swarm-manager:session:%s", kind)
	if sessionID != "" {
		tag = fmt.Sprintf("%s:%s", tag, sessionID)
	}
	return tag
}

func buildSessionTitle(kind, sessionID string) string {
	label := strings.TrimSpace(sessionID)
	if label == "" {
		label = "session"
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "meta_orchestration":
		return "Meta-orchestration: " + label
	case "operating_mode_authoring":
		return "Operating mode authoring: " + label
	case "swarm_operations":
		return "Swarm operations: " + label
	default:
		return "Agent session: " + label
	}
}

func buildBacklogTag(kind, name, purpose string) string {
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	purpose = strings.TrimSpace(purpose)
	if kind == "" {
		kind = "backlog"
	}
	tag := fmt.Sprintf("swarm-manager:backlog:%s", kind)
	if name != "" {
		tag = fmt.Sprintf("%s:%s", tag, name)
	}
	if purpose != "" {
		tag = fmt.Sprintf("%s:%s", tag, purpose)
	}
	return tag
}

func buildBacklogTitle(kind, name, purpose string) string {
	label := strings.TrimSpace(name)
	if label == "" {
		label = "backlog item"
	}
	if strings.TrimSpace(purpose) != "" {
		return fmt.Sprintf("%s: %s", capitalizeLabel(strings.TrimSpace(purpose)), label)
	}
	if strings.TrimSpace(kind) != "" {
		return fmt.Sprintf("Backlog %s: %s", strings.TrimSpace(kind), label)
	}
	return "Backlog item: " + label
}

func capitalizeLabel(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

// maxTaskDescriptionLen is the agent-manager limit for task descriptions (64KB).
const maxTaskDescriptionLen = 65536

// truncateDescription ensures the description fits within agent-manager's
// limit. The full prompt is still sent via the CreateRunRequest.Prompt field,
// so the agent receives the complete text regardless of truncation here.
func truncateDescription(desc string) string {
	if len(desc) <= maxTaskDescriptionLen {
		return desc
	}
	const suffix = "\n\n[truncated — full prompt provided via run request]"
	return desc[:maxTaskDescriptionLen-len(suffix)] + suffix
}

func normalizeRunStatus(status domainpb.RunStatus) string {
	switch status {
	case domainpb.RunStatus_RUN_STATUS_PENDING:
		return "pending"
	case domainpb.RunStatus_RUN_STATUS_STARTING:
		return "starting"
	case domainpb.RunStatus_RUN_STATUS_RUNNING:
		return "running"
	case domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW:
		return "needs_review"
	case domainpb.RunStatus_RUN_STATUS_COMPLETE:
		return "complete"
	case domainpb.RunStatus_RUN_STATUS_FAILED:
		return "failed"
	case domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "unspecified"
	}
}
