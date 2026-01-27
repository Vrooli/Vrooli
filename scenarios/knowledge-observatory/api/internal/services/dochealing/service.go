package dochealing

// DOC: docs/concepts/ARCHITECTURE.md#documentation-healing-flow
// DOC: docs/reference/api-endpoints.md#documentation-healing
import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"knowledge-observatory/internal/docschema"
	"knowledge-observatory/internal/services/dochealth"
)

const docHealingSkillID = "documentation-health"

// JobStore persists healing job state.
type JobStore interface {
	CreateJob(ctx context.Context, req HealRequest, healthBefore *float64) (string, error)
	GetJob(ctx context.Context, jobID string) (HealJob, bool, error)
	MarkRunning(ctx context.Context, jobID, runID string, startedAt time.Time) error
	UpdateProgress(ctx context.Context, jobID, progress string) error
	UpdateReview(ctx context.Context, jobID string, diff *DiffPreview, healthAfter *float64, status string, completedAt time.Time) error
	UpdateError(ctx context.Context, jobID, message string) error
	MarkApproved(ctx context.Context, jobID, actor string, approvedAt time.Time, healthAfter *float64) error
	MarkRejected(ctx context.Context, jobID, actor, reason string, completedAt time.Time) error
	FailJob(ctx context.Context, jobID, message string, completedAt time.Time) error
}

// Service coordinates documentation healing jobs.
type Service struct {
	scenariosRoot string
	repoRoot      string
	agent         AgentClient
	store         JobStore
	health        *dochealth.Service
	skills        SkillProvider
	now           func() time.Time
	pollInterval  time.Duration
}

// NewService constructs a healing service.
func NewService(scenariosRoot string, health *dochealth.Service, agent AgentClient, store JobStore, skills SkillProvider) (*Service, error) {
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
	if health == nil {
		return nil, ErrHealthUnavailable
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
		health:        health,
		skills:        skills,
		now:           time.Now,
		pollInterval:  2 * time.Second,
	}, nil
}

// StartHealing launches a healing job and returns initial status.
func (s *Service) StartHealing(ctx context.Context, req HealRequest) (*HealJob, error) {
	if s == nil || s.store == nil {
		return nil, ErrJobStoreUnavailable
	}
	if s.agent == nil {
		return nil, ErrAgentUnavailable
	}
	if s.health == nil {
		return nil, ErrHealthUnavailable
	}
	if err := req.normalize(); err != nil {
		return nil, err
	}
	scenarioPath, err := s.scenarioPath(req.ScenarioName)
	if err != nil {
		return nil, err
	}

	healthResult, err := s.health.ValidateScenario(ctx, req.ScenarioName)
	if err != nil {
		return nil, err
	}
	healthBefore := healthResult.Validation.HealthScore
	if len(req.Issues) == 0 {
		req.Issues = issuesFromValidation(healthResult.Validation)
	}

	jobID, err := s.store.CreateJob(ctx, req, &healthBefore)
	if err != nil {
		return nil, err
	}

	prompt, err := s.buildPrompt(ctx, req, scenarioPath, healthResult.Validation)
	if err != nil {
		_ = s.store.FailJob(ctx, jobID, err.Error(), s.now())
		return nil, err
	}

	projectRoot := s.repoRoot
	if projectRoot == "" {
		projectRoot = s.scenariosRoot
	}
	runID, err := s.spawnAgent(ctx, jobID, req, prompt, scenarioPath, projectRoot)
	if err != nil {
		_ = s.store.FailJob(ctx, jobID, err.Error(), s.now())
		return nil, err
	}

	startedAt := s.now()
	if err := s.store.MarkRunning(ctx, jobID, runID, startedAt); err != nil {
		return nil, err
	}

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	go s.trackRun(jobID, runID, req, &healthBefore, timeout)

	return &HealJob{
		JobID:        jobID,
		ScenarioName: req.ScenarioName,
		Status:       StatusRunning,
		StartedAt:    &startedAt,
		AgentRunID:   runID,
		HealthBefore: &healthBefore,
		AutoApprove:  req.AutoApprove,
		DryRun:       req.DryRun,
	}, nil
}

// GetJob returns the job status, refreshing from agent-manager if needed.
func (s *Service) GetJob(ctx context.Context, jobID string) (*HealJob, error) {
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
		s.refreshJob(ctx, jobID, job.AgentRunID, job)
		updated, _, err := s.store.GetJob(ctx, jobID)
		if err == nil {
			job = updated
		}
	}
	return &job, nil
}

// ApproveJob approves a healing job and applies changes.
func (s *Service) ApproveJob(ctx context.Context, jobID, actor string) (*HealJob, error) {
	if s == nil || s.store == nil {
		return nil, ErrJobStoreUnavailable
	}
	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != StatusNeedsReview {
		return nil, ErrJobNotApprovable
	}
	if job.AgentRunID == "" {
		return nil, ErrJobNotApprovable
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "knowledge-observatory"
	}
	if _, err := s.agent.ApproveRun(ctx, ApprovalRequest{
		RunID: job.AgentRunID,
		Actor: actor,
	}); err != nil {
		return nil, err
	}
	healthAfter := s.computeHealthScore(ctx, job.ScenarioName)
	if err := s.store.MarkApproved(ctx, jobID, actor, s.now(), healthAfter); err != nil {
		return nil, err
	}
	return s.GetJob(ctx, jobID)
}

// RejectJob rejects a healing job and discards changes.
func (s *Service) RejectJob(ctx context.Context, jobID, actor, reason string) (*HealJob, error) {
	if s == nil || s.store == nil {
		return nil, ErrJobStoreUnavailable
	}
	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != StatusNeedsReview {
		return nil, ErrJobNotRejectable
	}
	if job.AgentRunID == "" {
		return nil, ErrJobNotRejectable
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "knowledge-observatory"
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "Rejected by operator"
	}
	if err := s.agent.RejectRun(ctx, RejectRequest{
		RunID:  job.AgentRunID,
		Actor:  actor,
		Reason: reason,
	}); err != nil {
		return nil, err
	}
	if err := s.store.MarkRejected(ctx, jobID, actor, reason, s.now()); err != nil {
		return nil, err
	}
	return s.GetJob(ctx, jobID)
}

func (s *Service) trackRun(jobID, runID string, req HealRequest, healthBefore *float64, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	var lastSeq int64
	var lastProgress string

	for {
		select {
		case <-ctx.Done():
			_ = s.store.FailJob(context.Background(), jobID, "healing job timed out", s.now())
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
				if err := s.finalizeRun(ctx, jobID, runID, req, healthBefore, run.Summary); err != nil {
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

func (s *Service) refreshJob(ctx context.Context, jobID, runID string, job HealJob) {
	if s.agent == nil {
		return
	}
	run, err := s.agent.GetRun(ctx, runID)
	if err != nil || run == nil {
		return
	}
	switch run.Status {
	case RunStatusComplete:
		_ = s.finalizeRun(ctx, jobID, runID, HealRequest{
			ScenarioName: job.ScenarioName,
			AutoApprove:  job.AutoApprove,
			DryRun:       job.DryRun,
			Issues:       nil,
		}, job.HealthBefore, run.Summary)
	case RunStatusFailed, RunStatusCancelled:
		message := strings.TrimSpace(run.Error)
		if message == "" {
			message = fmt.Sprintf("agent run %s", run.Status)
		}
		_ = s.store.FailJob(ctx, jobID, message, s.now())
	}
}

func (s *Service) finalizeRun(ctx context.Context, jobID, runID string, req HealRequest, healthBefore *float64, summary string) error {
	diff, err := s.agent.GetRunDiff(ctx, runID)
	if err != nil {
		return err
	}
	if err := s.validateDiffPaths(req.ScenarioName, diff); err != nil {
		return err
	}
	preview := buildDiffPreview(diff, summary)
	healthAfter := s.estimateHealthAfter(req.ScenarioName, diff)

	status := StatusNeedsReview
	if preview == nil || len(preview.Files) == 0 {
		status = StatusApproved
		if healthAfter == nil && healthBefore != nil {
			healthAfter = healthBefore
		}
	}
	if err := s.store.UpdateReview(ctx, jobID, preview, healthAfter, status, s.now()); err != nil {
		return err
	}

	if status == StatusNeedsReview && req.AutoApprove && !req.DryRun && healthBefore != nil && healthAfter != nil && *healthAfter > *healthBefore {
		if _, err := s.ApproveJob(ctx, jobID, "auto-approve"); err != nil {
			_ = s.store.UpdateError(ctx, jobID, fmt.Sprintf("auto-approve failed: %s", err.Error()))
		}
	}
	return nil
}

func (s *Service) spawnAgent(ctx context.Context, jobID string, req HealRequest, prompt, scopePath, projectRoot string) (string, error) {
	if err := s.agent.EnsureProfile(ctx); err != nil {
		return "", err
	}
	title := fmt.Sprintf("Documentation healing: %s", req.ScenarioName)
	runReq := AgentRunRequest{
		Title:       title,
		Description: "Agent-powered documentation healing",
		Prompt:      prompt,
		ScopePath:   scopePath,
		ProjectRoot: projectRoot,
		Tag:         fmt.Sprintf("doc-heal-%s", jobID),
		Timeout:     time.Duration(req.TimeoutSeconds) * time.Second,
	}
	return s.agent.CreateRun(ctx, runReq)
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
		return "", ErrScenarioNotFound
	}
	return path, nil
}

func (s *Service) computeHealthScore(ctx context.Context, scenarioName string) *float64 {
	if s.health == nil {
		return nil
	}
	result, err := s.health.ValidateScenario(ctx, scenarioName)
	if err != nil || result == nil || result.Validation == nil {
		return nil
	}
	score := result.Validation.HealthScore
	return &score
}

func (s *Service) estimateHealthAfter(scenarioName string, diff *RunDiff) *float64 {
	if diff == nil {
		return nil
	}
	scenarioPath, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil
	}
	docSet, err := listDocFiles(scenarioPath)
	if err != nil {
		return nil
	}
	for _, file := range diff.Files {
		rel, ok := s.toScenarioRelative(scenarioPath, file.Path)
		if !ok || !isDocPath(rel) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(file.ChangeType)) {
		case "added", "created":
			docSet[rel] = true
		case "deleted", "removed":
			delete(docSet, rel)
		}
	}
	tmpRoot, err := os.MkdirTemp("", "ko-docheal-*")
	if err != nil {
		return nil
	}
	defer os.RemoveAll(tmpRoot)

	tmpScenario := filepath.Join(tmpRoot, scenarioName)
	for rel := range docSet {
		target := filepath.Join(tmpScenario, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil
		}
		if err := os.WriteFile(target, []byte(""), 0o644); err != nil {
			return nil
		}
	}

	result, err := docschema.ValidateScenarioDocumentation(tmpScenario)
	if err != nil {
		return nil
	}
	score := result.HealthScore
	return &score
}

func (s *Service) toScenarioRelative(scenarioPath, value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	clean := filepath.Clean(value)
	var candidates []string
	if filepath.IsAbs(clean) {
		candidates = append(candidates, clean)
	} else {
		if s.repoRoot != "" {
			candidates = append(candidates, filepath.Join(s.repoRoot, clean))
		}
		candidates = append(candidates, filepath.Join(scenarioPath, clean))
	}
	for _, abs := range candidates {
		rel, err := filepath.Rel(scenarioPath, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		return filepath.ToSlash(rel), true
	}
	return "", false
}

func listDocFiles(scenarioPath string) (map[string]bool, error) {
	files := make(map[string]bool)
	for _, name := range []string{"README.md", "PRD.md"} {
		path := filepath.Join(scenarioPath, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			files[name] = true
		}
	}
	docsDir := filepath.Join(scenarioPath, "docs")
	if info, err := os.Stat(docsDir); err == nil && info.IsDir() {
		err = filepath.WalkDir(docsDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(scenarioPath, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if isDocPath(rel) {
				files[rel] = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func isDocPath(rel string) bool {
	ext := strings.ToLower(filepath.Ext(rel))
	return ext == ".md" || ext == ".json"
}

func isAllowedDocChange(rel string) bool {
	ext := strings.ToLower(filepath.Ext(rel))
	if ext != ".md" && ext != ".json" && ext != ".txt" {
		return false
	}
	normalized := filepath.ToSlash(rel)
	if strings.HasPrefix(normalized, "docs/") {
		return true
	}
	return !strings.Contains(normalized, "/")
}

func (s *Service) validateDiffPaths(scenarioName string, diff *RunDiff) error {
	if diff == nil {
		return nil
	}
	scenarioPath, err := s.scenarioPath(scenarioName)
	if err != nil {
		return err
	}
	for _, file := range diff.Files {
		rel, ok := s.toScenarioRelative(scenarioPath, file.Path)
		if !ok || !isAllowedDocChange(rel) {
			return fmt.Errorf("diff path not allowed: %s", file.Path)
		}
	}
	return nil
}

func issuesFromValidation(validation *docschema.ValidationResult) []string {
	if validation == nil {
		return nil
	}
	issues := make([]string, 0, len(validation.MissingDocs)+len(validation.MisplacedDocs)+len(validation.ExtraDocs))
	for _, missing := range validation.MissingDocs {
		expected := missing.ExpectedPath()
		label := fmt.Sprintf("Missing %s", string(missing))
		if expected != "" {
			label = fmt.Sprintf("%s (%s)", label, expected)
		}
		issues = append(issues, label)
	}
	for _, misplaced := range validation.MisplacedDocs {
		issues = append(issues, fmt.Sprintf("Misplaced %s -> %s", misplaced.ActualPath, misplaced.ExpectedPath))
	}
	for _, extra := range validation.ExtraDocs {
		issues = append(issues, fmt.Sprintf("Extra %s", extra))
	}
	sort.Strings(issues)
	return issues
}

func (s *Service) buildPrompt(ctx context.Context, req HealRequest, scenarioPath string, validation *docschema.ValidationResult) (string, error) {
	var skill string
	if s.skills != nil {
		content, err := s.skills.GetSkill(ctx, docHealingSkillID)
		if err == nil {
			skill = strings.TrimSpace(content)
		}
	}

	builder := &strings.Builder{}
	builder.WriteString("You are the Documentation Healing agent for the Vrooli codebase.\n")
	if skill != "" {
		builder.WriteString("\n=== Documentation Health Skill ===\n")
		builder.WriteString(skill)
		builder.WriteString("\n=== End Skill ===\n")
	}
	builder.WriteString("\nHealing Request:\n")
	builder.WriteString(fmt.Sprintf("- Scenario: %s\n", req.ScenarioName))
	builder.WriteString(fmt.Sprintf("- Scenario root: %s\n", scenarioPath))
	if validation != nil {
		builder.WriteString(fmt.Sprintf("- Current health score: %.2f\n", validation.HealthScore))
	}
	if len(req.Issues) > 0 {
		builder.WriteString("- Target issues:\n")
		for _, issue := range req.Issues {
			builder.WriteString(fmt.Sprintf("  - %s\n", issue))
		}
	}
	builder.WriteString("\nRules:\n")
	builder.WriteString("- Only modify documentation files (.md/.json/.txt) within the scenario docs or root.\n")
	builder.WriteString("- Do not change application code or business logic.\n")
	builder.WriteString("- Prefer moving/mending docs to match the standard layout.\n")
	builder.WriteString("- Keep changes minimal and focused on documentation health.\n")
	builder.WriteString("\nDeliverable:\n")
	builder.WriteString("- Apply fixes directly in the workspace.\n")
	builder.WriteString("- Provide a concise summary of what changed.\n")
	return builder.String(), nil
}
