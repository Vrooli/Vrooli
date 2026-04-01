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
	"log"
	"strings"
	"sync"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/durationpb"
)

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
}

// AgentService implements the Service interface.
type AgentService struct {
	client         *HTTPClient
	profileName    string
	profileKey     string
	profileID      string
	mu             sync.RWMutex
	enabled        bool
	settingsReader SettingsReader
}

// AgentServiceConfig contains configuration for the agent service.
type AgentServiceConfig struct {
	ProfileName    string
	ProfileKey     string
	Timeout        time.Duration
	Enabled        bool
	SettingsReader SettingsReader
}

// DefaultServiceConfig returns a baseline configuration for Swarm Manager.
func DefaultServiceConfig() AgentServiceConfig {
	return AgentServiceConfig{
		ProfileName: "swarm-manager",
		ProfileKey:  "swarm-manager",
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
		enabled:        cfg.Enabled,
		settingsReader: cfg.SettingsReader,
	}
}

// SetSettingsReader assigns a SettingsReader for runtime profile config resolution.
// This is safe to call after construction, before any spawns occur.
func (s *AgentService) SetSettingsReader(r SettingsReader) {
	s.settingsReader = r
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
	if cfg == nil {
		cfg = s.resolveProfileConfig()
	}

	resp, err := s.client.EnsureProfile(ctx, &apipb.EnsureProfileRequest{
		ProfileKey:     s.profileKey,
		Defaults:       s.buildProfile(cfg),
		UpdateExisting: false,
	})
	if err != nil {
		return fmt.Errorf("ensure profile: %w", err)
	}

	s.mu.Lock()
	if resp.Profile != nil {
		s.profileID = resp.Profile.Id
	}
	s.mu.Unlock()

	if resp.Created {
		log.Printf("[agent-manager] Created profile '%s' (id=%s)", s.profileName, s.profileID)
	} else {
		log.Printf("[agent-manager] Resolved profile '%s' (id=%s)", s.profileName, s.profileID)
	}

	return nil
}

// ProfileConfig contains agent profile configuration.
type ProfileConfig struct {
	RunnerType       domainpb.RunnerType
	Model            string
	ModelPreset      domainpb.ModelPreset
	MaxTurns         int32
	TimeoutSeconds   int32
	AllowedTools     []string
	SkipPermissions  bool
	RequiresSandbox  bool
	RequiresApproval bool
}

// SettingsReader provides agent settings from an external source (e.g. settings store)
// without creating a direct import dependency on the settings package.
type SettingsReader interface {
	LoadAgentSettings() (maxTurns, timeoutSeconds int32, requiresApproval bool, err error)
}

// ProfileConfigFromSettings creates a ProfileConfig by overlaying settings values
// on top of the defaults. Zero/negative values for maxTurns or timeoutSeconds
// are ignored, preserving the default.
func ProfileConfigFromSettings(maxTurns, timeoutSeconds int32, requiresApproval bool) *ProfileConfig {
	cfg := DefaultProfileConfig()
	if maxTurns > 0 {
		cfg.MaxTurns = maxTurns
	}
	if timeoutSeconds > 0 {
		cfg.TimeoutSeconds = timeoutSeconds
	}
	cfg.RequiresApproval = requiresApproval
	return cfg
}

// DefaultProfileConfig returns the default configuration for swarm-manager agents.
func DefaultProfileConfig() *ProfileConfig {
	return &ProfileConfig{
		RunnerType:  domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		ModelPreset: domainpb.ModelPreset_MODEL_PRESET_SMART,
		MaxTurns:    60,
		// 15 minute timeout for research and implementation prep.
		TimeoutSeconds: 900,
		AllowedTools: []string{
			"Read",
			"Write",
			"Edit",
			"Glob",
			"Grep",
			"Bash",
		},
		SkipPermissions:  false,
		RequiresSandbox:  false,
		RequiresApproval: true,
	}
}

func (s *AgentService) buildProfile(cfg *ProfileConfig) *domainpb.AgentProfile {
	return &domainpb.AgentProfile{
		Name:                 s.profileName,
		ProfileKey:           s.profileKey,
		Description:          "Agent profile for swarm-manager research and execution",
		RunnerType:           cfg.RunnerType,
		Model:                cfg.Model,
		ModelPreset:          cfg.ModelPreset,
		MaxTurns:             cfg.MaxTurns,
		Timeout:              durationpb.New(time.Duration(cfg.TimeoutSeconds) * time.Second),
		AllowedTools:         cfg.AllowedTools,
		SkipPermissionPrompt: cfg.SkipPermissions,
		RequiresSandbox:      cfg.RequiresSandbox,
		RequiresApproval:     cfg.RequiresApproval,
		CreatedBy:            "swarm-manager",
	}
}

// resolveProfileConfig returns a ProfileConfig derived from the settings store
// when available, falling back to hardcoded defaults on error or when no
// SettingsReader is configured.
func (s *AgentService) resolveProfileConfig() *ProfileConfig {
	if s.settingsReader != nil {
		maxTurns, timeout, approval, err := s.settingsReader.LoadAgentSettings()
		if err == nil {
			return ProfileConfigFromSettings(maxTurns, timeout, approval)
		}
		log.Printf("[agent-manager] settings read failed, using defaults: %v", err)
	}
	return DefaultProfileConfig()
}

func (s *AgentService) defaultProfileRef() *apipb.ProfileRef {
	if s.profileKey == "" {
		return nil
	}
	return &apipb.ProfileRef{
		ProfileKey: s.profileKey,
		Defaults:   s.buildProfile(s.resolveProfileConfig()),
	}
}

// GetProfileID returns the current profile ID.
func (s *AgentService) GetProfileID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.profileID
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
}

// BacklogSpawnRequest describes a request to spawn a backlog agent.
type BacklogSpawnRequest struct {
	Kind            string
	Name            string
	Title           string
	Description     string
	Prompt          string
	ScopePath       string
	ProjectRoot     string
	CreatedBy       string
	Purpose         string
	AcceptanceAllow []string
	AcceptanceDeny  []string
	Environment     map[string]string
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
	RunID      string
	TaskID     string
	Status     string
	StartedAt  string
	FinishedAt string
	ErrorMsg   string
	SandboxID  string
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
	runReq := &apipb.CreateRunRequest{
		TaskId:     createdTask.Id,
		ProfileRef: s.defaultProfileRef(),
		Tag:        &tag,
		Force:      true,
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

// SpawnBacklog creates a backlog task/run in agent-manager.
func (s *AgentService) SpawnBacklog(ctx context.Context, req BacklogSpawnRequest) (RunResult, error) {
	if !s.enabled {
		return RunResult{}, ErrNotAvailable
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = buildBacklogTitle(req.Kind, req.Name, req.Purpose)
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

	tag := buildBacklogTag(req.Kind, req.Name, req.Purpose)
	runReq := &apipb.CreateRunRequest{
		TaskId:     createdTask.Id,
		ProfileRef: s.defaultProfileRef(),
		Tag:        &tag,
		Force:      true,
	}
	if prompt := strings.TrimSpace(req.Prompt); prompt != "" {
		runReq.Prompt = &prompt
	}
	if len(req.Environment) > 0 {
		runReq.Environment = req.Environment
	}
	if len(req.AcceptanceAllow) > 0 || len(req.AcceptanceDeny) > 0 {
		acceptance := &domainpb.SandboxAcceptanceConfig{
			Mode: domainpb.SandboxAcceptanceMode_SANDBOX_ACCEPTANCE_MODE_ALLOWLIST,
		}
		if len(req.AcceptanceAllow) > 0 {
			acceptance.Allow = &domainpb.SandboxFileCriteria{PathGlobs: req.AcceptanceAllow}
		}
		if len(req.AcceptanceDeny) > 0 {
			acceptance.Deny = &domainpb.SandboxFileCriteria{PathGlobs: req.AcceptanceDeny}
		}
		runReq.InlineConfig = &domainpb.RunConfigOverrides{
			SandboxConfig: &domainpb.SandboxConfig{
				Acceptance: acceptance,
			},
		}
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
	return state, nil
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
