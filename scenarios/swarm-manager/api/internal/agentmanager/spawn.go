package agentmanager

import (
	"context"
	"fmt"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

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

// RunResult returns agent-manager identifiers.
type RunResult struct {
	TaskID    string
	RunID     string
	BaseURL   string
	CreatedAt string
}

// SpawnSession creates a general Swarm Manager session task/run in
// agent-manager.
func (s *AgentService) SpawnSession(ctx context.Context, req SessionSpawnRequest) (RunResult, error) {
	if !s.enabled {
		return RunResult{}, ErrNotAvailable
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
