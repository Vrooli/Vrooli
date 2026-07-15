package agentmanager

import (
	"context"
	"fmt"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

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
	// ManualReview, when true, spawns the run with SandboxConfig.ManualReview so
	// agent-manager holds it at needs_review (overlay NOT merged) until the
	// caller approves. The Baseline Modes pre-merge engagement hold uses this to
	// open a shadow restore point from the actual diff before the merge lands.
	ManualReview bool
}

// RunResult returns agent-manager identifiers.
type RunResult struct {
	TaskID    string
	RunID     string
	BaseURL   string
	CreatedAt string
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

	tag, err := buildResearchTag(req.IdeaName)
	if err != nil {
		return RunResult{}, err
	}
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
	// A created run with no id is a failed spawn: nothing is trackable and the
	// caller would otherwise persist an unpollable "starting" record. Fail closed.
	if strings.TrimSpace(run.GetId()) == "" {
		return RunResult{}, fmt.Errorf("%w: agent-manager created a run with no id", ErrRequestFailed)
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
	// A created run with no id is a failed spawn: nothing is trackable and the
	// caller would otherwise persist an unpollable "starting" record. Fail closed.
	if strings.TrimSpace(run.GetId()) == "" {
		return RunResult{}, fmt.Errorf("%w: agent-manager created a run with no id", ErrRequestFailed)
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
	if req.ManualReview {
		applyManualReview(runReq)
	}

	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		return RunResult{}, err
	}
	// A created run with no id is a failed spawn: nothing is trackable and the
	// caller would otherwise persist an unpollable "starting" record. Fail closed.
	if strings.TrimSpace(run.GetId()) == "" {
		return RunResult{}, fmt.Errorf("%w: agent-manager created a run with no id", ErrRequestFailed)
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
	// A created run with no id is a failed spawn: nothing is trackable and the
	// caller would otherwise persist an unpollable "starting" record. Fail closed.
	if strings.TrimSpace(run.GetId()) == "" {
		return RunResult{}, fmt.Errorf("%w: agent-manager created a run with no id", ErrRequestFailed)
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

// applyManualReview sets SandboxConfig.ManualReview=true on the run request so
// agent-manager defers the overlay merge and holds the run at needs_review.
// Like applyAcceptanceOverride, it sets ONLY the one field it owns; the
// orchestrator backfills Mode/NetworkMode from DefaultSandboxConfig(), so this
// never strips the protected-mode default.
func applyManualReview(runReq *apipb.CreateRunRequest) {
	if runReq.InlineConfig == nil {
		runReq.InlineConfig = &domainpb.RunConfigOverrides{}
	}
	if runReq.InlineConfig.SandboxConfig == nil {
		runReq.InlineConfig.SandboxConfig = &domainpb.SandboxConfig{}
	}
	runReq.InlineConfig.SandboxConfig.ManualReview = true
}
