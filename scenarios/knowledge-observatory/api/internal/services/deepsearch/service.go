package deepsearch

// DOC: docs/concepts/ARCHITECTURE.md#documentation-deep-search-flow
// DOC: docs/reference/api-endpoints.md#documentation-deep-search
import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const deepSearchSkillID = "documentation-search"

// JobStore persists deep search jobs.
type JobStore interface {
	CreateJob(ctx context.Context, req DeepSearchRequest) (string, error)
	GetJob(ctx context.Context, jobID string) (DeepSearchJob, bool, error)
	MarkRunning(ctx context.Context, jobID, runID string, startedAt time.Time) error
	UpdateProgress(ctx context.Context, jobID, progress string) error
	CompleteJob(ctx context.Context, jobID string, results []DeepSearchResult, completedAt time.Time) error
	FailJob(ctx context.Context, jobID, message string, completedAt time.Time) error
}

// Service coordinates agent-powered deep search.
type Service struct {
	scenariosRoot string
	repoRoot      string
	agent         AgentClient
	store         JobStore
	skills        SkillProvider
	parser        ResultParser
	now           func() time.Time
	sleep         func(time.Duration)
	pollInterval  time.Duration
}

// NewService constructs a deep search service.
func NewService(scenariosRoot string, agent AgentClient, store JobStore, skills SkillProvider, parser ResultParser) (*Service, error) {
	scenariosRoot = strings.TrimSpace(scenariosRoot)
	if scenariosRoot == "" {
		return nil, ErrScenarioRootEmpty
	}
	info, err := os.Stat(scenariosRoot)
	if err != nil || !info.IsDir() {
		return nil, ErrScenarioRootEmpty
	}
	if agent == nil {
		return nil, ErrAgentUnavailable
	}
	if store == nil {
		return nil, ErrJobStoreUnavailable
	}
	if parser == nil {
		parser = &JSONParser{}
	}
	repoRoot := filepath.Dir(scenariosRoot)
	if repoRoot == scenariosRoot {
		repoRoot = ""
	}
	return &Service{
		scenariosRoot: scenariosRoot,
		repoRoot:      repoRoot,
		agent:         agent,
		store:         store,
		skills:        skills,
		parser:        parser,
		now:           time.Now,
		sleep:         time.Sleep,
		pollInterval:  2 * time.Second,
	}, nil
}

// StartSearch launches a deep search job and returns the initial job status.
func (s *Service) StartSearch(ctx context.Context, req DeepSearchRequest) (*DeepSearchJob, error) {
	if s == nil || s.store == nil {
		return nil, ErrJobStoreUnavailable
	}
	if s.agent == nil {
		return nil, ErrAgentUnavailable
	}
	if err := req.normalize(); err != nil {
		return nil, err
	}
	if err := s.validateScope(req); err != nil {
		return nil, err
	}

	jobID, err := s.store.CreateJob(ctx, req)
	if err != nil {
		return nil, err
	}

	scopePath, projectRoot, err := s.resolveScope(req)
	if err != nil {
		_ = s.store.FailJob(ctx, jobID, err.Error(), s.now())
		return nil, err
	}

	prompt, err := s.buildPrompt(ctx, req, scopePath)
	if err != nil {
		_ = s.store.FailJob(ctx, jobID, err.Error(), s.now())
		return nil, err
	}

	runID, err := s.spawnAgent(ctx, jobID, req, prompt, scopePath, projectRoot)
	if err != nil {
		_ = s.store.FailJob(ctx, jobID, err.Error(), s.now())
		return nil, err
	}

	startedAt := s.now()
	if err := s.store.MarkRunning(ctx, jobID, runID, startedAt); err != nil {
		return nil, err
	}

	job := &DeepSearchJob{
		JobID:      jobID,
		Status:     StatusRunning,
		StartedAt:  &startedAt,
		AgentRunID: runID,
	}

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	go s.trackRun(jobID, runID, timeout, req.MaxResults)

	return job, nil
}

// GetJob returns the job status, refreshing from agent-manager if needed.
func (s *Service) GetJob(ctx context.Context, jobID string) (*DeepSearchJob, error) {
	if s == nil || s.store == nil {
		return nil, ErrJobStoreUnavailable
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, ErrJobIDRequired
	}
	job, found, err := s.store.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrJobNotFound
	}
	if job.Status == StatusRunning && job.AgentRunID != "" && s.agent != nil {
		s.refreshJob(ctx, jobID, job.AgentRunID)
		updated, _, err := s.store.GetJob(ctx, jobID)
		if err == nil {
			job = updated
		}
	}
	return &job, nil
}

func (s *Service) spawnAgent(ctx context.Context, jobID string, req DeepSearchRequest, prompt, scopePath, projectRoot string) (string, error) {
	if err := s.agent.EnsureProfile(ctx); err != nil {
		return "", err
	}
	title := fmt.Sprintf("Deep documentation search: %s", truncate(req.Query, 64))
	runReq := AgentRunRequest{
		Title:       title,
		Description: "Agent-powered deep documentation search",
		Prompt:      prompt,
		ScopePath:   scopePath,
		ProjectRoot: projectRoot,
		Tag:         fmt.Sprintf("doc-deep-search-%s", jobID),
		Timeout:     time.Duration(req.TimeoutSeconds) * time.Second,
	}
	return s.agent.CreateRun(ctx, runReq)
}

func (s *Service) trackRun(jobID, runID string, timeout time.Duration, maxResults int) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	var lastSeq int64
	var lastProgress string

	for {
		select {
		case <-ctx.Done():
			_ = s.store.FailJob(context.Background(), jobID, "deep search timed out", s.now())
			return
		case <-ticker.C:
			run, err := s.agent.GetRun(ctx, runID)
			if err != nil {
				lastProgress = s.updateProgress(jobID, lastProgress, "Awaiting agent status...")
				continue
			}
			if run == nil {
				_ = s.store.FailJob(context.Background(), jobID, "agent run not found", s.now())
				return
			}

			progress, seq := s.readProgress(ctx, runID, lastSeq)
			if seq > lastSeq {
				lastSeq = seq
			}
			if progress != "" && progress != lastProgress {
				lastProgress = s.updateProgress(jobID, lastProgress, progress)
			}

			switch run.Status {
			case RunStatusComplete:
				if err := s.finalizeRun(ctx, jobID, runID, maxResults); err != nil {
					_ = s.store.FailJob(context.Background(), jobID, err.Error(), s.now())
				}
				return
			case RunStatusFailed, RunStatusCancelled:
				message := strings.TrimSpace(run.Error)
				if message == "" {
					message = fmt.Sprintf("agent run %s", run.Status)
				}
				_ = s.store.FailJob(context.Background(), jobID, message, s.now())
				return
			}
		}
	}
}

func (s *Service) refreshJob(ctx context.Context, jobID, runID string) {
	if s.agent == nil {
		return
	}
	run, err := s.agent.GetRun(ctx, runID)
	if err != nil || run == nil {
		return
	}
	switch run.Status {
	case RunStatusComplete:
		_ = s.finalizeRun(ctx, jobID, runID, maxMaxResults)
	case RunStatusFailed, RunStatusCancelled:
		message := strings.TrimSpace(run.Error)
		if message == "" {
			message = fmt.Sprintf("agent run %s", run.Status)
		}
		_ = s.store.FailJob(ctx, jobID, message, s.now())
	}
}

func (s *Service) finalizeRun(ctx context.Context, jobID, runID string, maxResults int) error {
	events, err := s.agent.GetRunEvents(ctx, runID, 0)
	if err != nil {
		return err
	}
	output, progress := extractLatestOutput(events)
	if progress != "" {
		_ = s.store.UpdateProgress(ctx, jobID, progress)
	}
	if strings.TrimSpace(output) == "" {
		return fmt.Errorf("agent returned no output")
	}
	results, err := s.parser.Parse(ctx, output)
	if err != nil {
		return err
	}
	results = s.normalizeResults(results, maxResults)
	return s.store.CompleteJob(ctx, jobID, results, s.now())
}

func (s *Service) readProgress(ctx context.Context, runID string, afterSequence int64) (string, int64) {
	events, err := s.agent.GetRunEvents(ctx, runID, afterSequence)
	if err != nil || len(events) == 0 {
		return "", afterSequence
	}
	progress := ""
	lastSeq := afterSequence
	for _, event := range events {
		if event.Sequence > lastSeq {
			lastSeq = event.Sequence
		}
		if event.Type == EventProgress {
			progress = formatProgress(event)
		}
	}
	return progress, lastSeq
}

func (s *Service) updateProgress(jobID, prev, progress string) string {
	if strings.TrimSpace(progress) == "" || progress == prev {
		return prev
	}
	_ = s.store.UpdateProgress(context.Background(), jobID, progress)
	return progress
}

func (s *Service) normalizeResults(results []DeepSearchResult, maxResults int) []DeepSearchResult {
	trimmed := make([]DeepSearchResult, 0, len(results))
	for _, res := range results {
		path := strings.TrimSpace(res.Path)
		if path == "" {
			continue
		}
		path = s.repoRelative(path)
		relevance := res.Relevance
		if relevance < 0 {
			relevance = 0
		}
		if relevance > 1 {
			relevance = 1
		}
		references := make([]string, 0, len(res.References))
		for _, ref := range res.References {
			ref = strings.TrimSpace(ref)
			if ref != "" {
				references = append(references, s.repoRelative(ref))
			}
		}
		trimmed = append(trimmed, DeepSearchResult{
			Path:        path,
			Relevance:   relevance,
			Summary:     strings.TrimSpace(res.Summary),
			MatchReason: strings.TrimSpace(res.MatchReason),
			References:  references,
			Snippet:     strings.TrimSpace(res.Snippet),
		})
	}
	if maxResults <= 0 {
		maxResults = maxMaxResults
	}
	if len(trimmed) > maxResults {
		return trimmed[:maxResults]
	}
	return trimmed
}

func extractLatestOutput(events []AgentRunEvent) (string, string) {
	var output string
	var progress string
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		switch event.Type {
		case EventMessage:
			if output == "" && strings.EqualFold(event.Role, "assistant") {
				output = event.Content
			}
		case EventProgress:
			if progress == "" {
				progress = formatProgress(event)
			}
		}
		if output != "" && progress != "" {
			break
		}
	}
	return output, progress
}

func formatProgress(event AgentRunEvent) string {
	label := strings.TrimSpace(event.ProgressAction)
	if label == "" {
		label = strings.TrimSpace(event.ProgressPhase)
	}
	if label == "" {
		return fmt.Sprintf("%d%%", event.ProgressPercent)
	}
	return fmt.Sprintf("%d%% - %s", event.ProgressPercent, label)
}

func (s *Service) resolveScope(req DeepSearchRequest) (string, string, error) {
	switch req.Scope {
	case ScopeGlobal:
		root := s.repoRoot
		if root == "" {
			root = s.scenariosRoot
		}
		return root, root, nil
	case ScopeScenario:
		path, err := s.scenarioPath(req.Scenario)
		if err != nil {
			return "", "", err
		}
		return path, path, nil
	case ScopePath:
		path, err := s.resolveBasePath(req.BasePath)
		if err != nil {
			return "", "", err
		}
		return path, path, nil
	default:
		return "", "", ErrScopeInvalid
	}
}

func (s *Service) validateScope(req DeepSearchRequest) error {
	if req.Scope == ScopeScenario {
		_, err := s.scenarioPath(req.Scenario)
		return err
	}
	if req.Scope == ScopePath {
		_, err := s.resolveBasePath(req.BasePath)
		return err
	}
	return nil
}

func (s *Service) scenarioPath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", ErrScenarioRequired
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", ErrScenarioRequired
	}
	path := filepath.Join(s.scenariosRoot, name)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", ErrScenarioRequired
	}
	return path, nil
}

func (s *Service) resolveBasePath(base string) (string, error) {
	if strings.TrimSpace(base) == "" {
		return "", ErrBasePathRequired
	}
	base = filepath.Clean(base)
	var abs string
	if filepath.IsAbs(base) {
		abs = base
	} else if s.repoRoot != "" {
		abs = filepath.Join(s.repoRoot, base)
	} else {
		abs = base
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", ErrBasePathRequired
	}
	if s.repoRoot != "" {
		rel, err := filepath.Rel(s.repoRoot, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", ErrBasePathInvalid
		}
	}
	return abs, nil
}

func (s *Service) repoRelative(path string) string {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return filepath.ToSlash(path)
	}
	if s.repoRoot == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(s.repoRoot, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func (s *Service) buildPrompt(ctx context.Context, req DeepSearchRequest, scopePath string) (string, error) {
	var skill string
	if s.skills != nil {
		content, err := s.skills.GetSkill(ctx, deepSearchSkillID)
		if err == nil {
			skill = strings.TrimSpace(content)
		}
	}
	scopeLine := req.Scope
	if req.Scope == ScopeScenario {
		scopeLine = fmt.Sprintf("%s (scenario=%s)", req.Scope, req.Scenario)
	} else if req.Scope == ScopePath {
		scopeLine = fmt.Sprintf("%s (path=%s)", req.Scope, req.BasePath)
	}
	builder := &strings.Builder{}
	builder.WriteString("You are the Documentation Search agent for the Vrooli codebase.\n")
	if skill != "" {
		builder.WriteString("\n=== Documentation Search Skill ===\n")
		builder.WriteString(skill)
		builder.WriteString("\n=== End Skill ===\n")
	}
	builder.WriteString("\nSearch Request:\n")
	builder.WriteString(fmt.Sprintf("- Query: %s\n", req.Query))
	builder.WriteString(fmt.Sprintf("- Scope: %s\n", scopeLine))
	builder.WriteString(fmt.Sprintf("- Scope root: %s\n", scopePath))
	builder.WriteString(fmt.Sprintf("- Follow references: %t\n", req.FollowRefs))
	builder.WriteString(fmt.Sprintf("- Max results: %d\n", req.MaxResults))
	builder.WriteString("\nRules:\n")
	builder.WriteString("- Use read-only tools only (Read, Glob, Grep).\n")
	builder.WriteString("- Focus on documentation files and references.\n")
	if !req.FollowRefs {
		builder.WriteString("- Do not follow references beyond the first matching file.\n")
	} else {
		builder.WriteString("- Follow referenced docs when they improve relevance.\n")
	}
	builder.WriteString("- Rank results by relevance (0-1). Provide concise summaries.\n")
	builder.WriteString("\nReturn JSON only as an array of objects with fields:\n")
	builder.WriteString("- path (string)\n- relevance (number 0-1)\n- summary (string)\n- match_reason (string)\n- references (array of strings)\n- snippet (string)\n")
	builder.WriteString("\nReturn JSON only, no markdown or commentary.\n")
	return builder.String(), nil
}

func truncate(value string, max int) string {
	if max <= 0 {
		return ""
	}
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return strings.TrimSpace(value[:max-1]) + "..."
}
