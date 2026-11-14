package services

// DOC: docs/concepts/ARCHITECTURE.md#investigation-flow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/owneridentity"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/agentmanager"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/logx"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// InvestigationService handles anomaly investigations
type InvestigationService struct {
	log                logx.Logger
	config             *config.Config
	repo               repository.InvestigationRepository
	metricsSrc         MetricsSource
	agentSvc           AgentExecutor
	settings           *SettingsManager
	clock              Clock
	promptStore        ConfigStore
	mu                 sync.RWMutex
	lastTrigger        time.Time
	shutdownCtx        context.Context
	shutdownCancel     context.CancelFunc
	jobMu              sync.Mutex
	jobCancels         map[string]context.CancelFunc
	jobsClosed         bool
	jobWG              sync.WaitGroup
	profileInitDone    chan struct{}
	profileInitStarted bool
}

// InvestigationOption configures an InvestigationService.
type InvestigationOption func(*InvestigationService)

// WithInvestigationClock sets the clock used by the investigation service.
func WithInvestigationClock(c Clock) InvestigationOption {
	return func(s *InvestigationService) { s.clock = c }
}

// WithPromptStore sets the prompt template store used by the investigation service.
func WithPromptStore(cs ConfigStore) InvestigationOption {
	return func(s *InvestigationService) { s.promptStore = cs }
}

// WithLogger sets the structured logger used by the investigation service.
func WithLogger(l logx.Logger) InvestigationOption {
	return func(s *InvestigationService) { s.log = l }
}

// NewInvestigationService creates a new investigation service
func NewInvestigationService(cfg *config.Config, repo repository.InvestigationRepository, metricsSrc MetricsSource, agentSvc AgentExecutor, settings *SettingsManager, opts ...InvestigationOption) (*InvestigationService, error) {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	if settings == nil {
		shutdownCancel()
		return nil, fmt.Errorf("investigation settings manager is required")
	}
	s := &InvestigationService{
		config:          cfg,
		repo:            repo,
		metricsSrc:      metricsSrc,
		agentSvc:        agentSvc,
		settings:        settings,
		clock:           RealClock{},
		lastTrigger:     time.Time{}, // Start with zero time - no cooldown initially
		shutdownCtx:     shutdownCtx,
		shutdownCancel:  shutdownCancel,
		jobCancels:      make(map[string]context.CancelFunc),
		profileInitDone: completedSignal(),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.log == nil {
		s.log = logx.Default()
	}
	if s.promptStore == nil {
		s.promptStore = &FileConfigStore{basePath: ResolvePromptBasePath()}
	}

	return s, nil
}

func completedSignal() chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

// StartBackground begins the bounded agent-profile initialization. It is
// separate from construction so a persisted lifecycle fence can be checked
// before any background owner starts in a replacement process.
func (s *InvestigationService) StartBackground() {
	s.jobMu.Lock()
	if s.profileInitStarted {
		s.jobMu.Unlock()
		return
	}
	s.profileInitStarted = true
	if s.agentSvc == nil || !s.agentSvc.IsEnabled() {
		s.jobMu.Unlock()
		return
	}
	done := make(chan struct{})
	s.profileInitDone = done
	agentSvc := s.agentSvc
	s.jobMu.Unlock()
	go func() {
		defer close(done)
		s.initializeAgentProfile(agentSvc)
	}()
}

// Quiesce cancels accepted investigations and joins every investigation
// goroutine. A timeout is an incomplete owner receipt; callers must not stop
// the process while it is returned.
func (s *InvestigationService) Quiesce(ctx context.Context) error {
	s.jobMu.Lock()
	s.jobsClosed = true
	for _, cancel := range s.jobCancels {
		cancel()
	}
	profileDone := s.profileInitDone
	s.jobMu.Unlock()
	if profileDone != nil {
		select {
		case <-profileDone:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	done := make(chan struct{})
	go func() {
		s.jobWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Resume reopens investigation admission after a prepared maintenance
// operation is abandoned before process replacement.
func (s *InvestigationService) Resume() {
	s.jobMu.Lock()
	s.jobsClosed = false
	s.jobMu.Unlock()
}

// ActiveJobs reports accepted investigation workers not yet joined.
func (s *InvestigationService) ActiveJobs() int {
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	return len(s.jobCancels)
}

// Shutdown cancels all server-lifetime investigation work and waits for its
// terminal persistence before the repository is closed.
func (s *InvestigationService) Shutdown() {
	s.shutdownCancel()
	_ = s.Quiesce(context.Background())
}

// initializeAgentProfile ensures the agent profile exists in agent-manager.
func (s *InvestigationService) initializeAgentProfile(agentSvc AgentExecutor) {
	ctx, cancel := context.WithTimeout(s.shutdownCtx, 30*time.Second)
	defer cancel()

	if err := agentSvc.Initialize(ctx); err != nil {
		s.log.Warn("failed to initialize agent profile", "error", err)
	}
}

// TriggerInvestigation starts a new investigation
func (s *InvestigationService) TriggerInvestigation(ctx context.Context, autoFix bool, note string) (*models.Investigation, error) {
	if err := owneridentity.RequireCreateRunCaller(ctx, time.Now(), true); err != nil {
		return nil, err
	}
	if s.agentSvc == nil || !s.agentSvc.IsEnabled() {
		return nil, apierrors.Unavailable("agent-manager")
	}
	availabilityCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if !s.agentSvc.IsAvailable(availabilityCtx) {
		return nil, apierrors.Unavailable("agent-manager")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check cooldown
	cooldownPeriod := s.currentCooldownPeriod()
	if !s.lastTrigger.IsZero() {
		elapsed := s.clock.Since(s.lastTrigger)
		if elapsed < cooldownPeriod {
			return nil, apierrors.Cooldown(int((cooldownPeriod - elapsed).Seconds()))
		}
	}

	s.jobMu.Lock()
	if s.jobsClosed {
		s.jobMu.Unlock()
		return nil, apierrors.Unavailable("lifecycle maintenance")
	}
	jobCtx, cancelJob := context.WithCancel(s.shutdownCtx)
	workerCtx, authErr := owneridentity.CarryCreateRunCaller(ctx, jobCtx, time.Now(), true)
	if authErr != nil {
		cancelJob()
		s.jobMu.Unlock()
		return nil, authErr
	}
	now := s.clock.Now()

	// Generate investigation ID
	investigationID := fmt.Sprintf("inv_%d", now.Unix())

	// Create investigation
	investigation := &models.Investigation{
		ID:        investigationID,
		Status:    models.StatusQueued,
		AnomalyID: fmt.Sprintf("ai_investigation_%d", now.Unix()),
		StartTime: now,
		Findings:  "Investigation queued for processing...",
		Progress:  0,
		Details:   make(map[string]interface{}),
		Steps:     []models.InvestigationStep{},
	}

	// Add auto_fix and note to details
	if autoFix {
		investigation.Details["auto_fix"] = true
	}
	if note != "" {
		investigation.Details["note"] = note
	}

	// Save to repository
	if err := s.repo.CreateInvestigation(ctx, investigation); err != nil {
		cancelJob()
		s.jobMu.Unlock()
		return nil, apierrors.Internal("failed to create investigation", err)
	}

	// Update last trigger time only after successful creation
	s.lastTrigger = now

	// Register the accepted async worker before releasing the shared job lock.
	// Quiesce closes this same gate before it snapshots/cancels workers.
	s.jobCancels[investigationID] = cancelJob
	s.jobWG.Add(1)
	go func(callerCtx context.Context) {
		defer func() {
			cancelJob()
			s.jobMu.Lock()
			delete(s.jobCancels, investigationID)
			s.jobMu.Unlock()
			s.jobWG.Done()
		}()
		defer func() {
			if r := recover(); r != nil {
				s.log.Error("investigation panicked", "investigation_id", investigationID, "panic", r)
				// Best-effort persistence of the failure even if the context is
				// already cancelled by shutdown.
				cleanupCtx, cancelCleanup := detachedCleanupContext(callerCtx, 5*time.Second)
				defer cancelCleanup()
				_ = s.UpdateInvestigationStatus(cleanupCtx, investigationID, models.StatusFailed)
				_ = s.UpdateInvestigationFindings(cleanupCtx, investigationID,
					fmt.Sprintf("Investigation failed due to internal error: %v", r), nil)
			}
		}()
		s.runInvestigation(callerCtx, investigationID, autoFix, note)
	}(workerCtx)
	s.jobMu.Unlock()

	return investigation, nil
}

// GetInvestigation retrieves an investigation by ID
func (s *InvestigationService) GetInvestigation(ctx context.Context, id string) (*models.Investigation, error) {
	inv, err := s.repo.GetInvestigation(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apierrors.NotFound("investigation", id)
		}
		return nil, apierrors.Internal("Unable to retrieve investigation", err)
	}
	return inv, nil
}

// GetLatestInvestigation retrieves the most recent investigation
func (s *InvestigationService) GetLatestInvestigation(ctx context.Context) (*models.Investigation, error) {
	investigation, err := s.repo.GetLatestInvestigation(ctx)
	if err != nil {
		// Only return default for "no investigations found"; propagate real errors
		if errors.Is(err, repository.ErrNotFound) {
			return &models.Investigation{
				ID:        "inv_default",
				Status:    models.StatusQueued,
				AnomalyID: "none",
				StartTime: s.clock.Now(),
				Findings:  "No investigations have been triggered yet. Click 'RUN ANOMALY CHECK' to start an investigation.",
			}, nil
		}
		return nil, apierrors.Internal("Unable to retrieve latest investigation", err)
	}
	return investigation, nil
}

// ListInvestigations returns recent investigations sorted by start time
func (s *InvestigationService) ListInvestigations(ctx context.Context, limit int) ([]*models.Investigation, error) {
	if limit <= 0 {
		limit = 20
	}

	investigations, err := s.repo.ListInvestigations(ctx, repository.InvestigationFilter{})
	if err != nil {
		return nil, apierrors.Internal("Unable to list investigations", err)
	}

	sort.Slice(investigations, func(i, j int) bool {
		return investigations[i].StartTime.After(investigations[j].StartTime)
	})

	if len(investigations) > limit {
		investigations = investigations[:limit]
	}

	return investigations, nil
}

// UpdateInvestigationStatus updates the status of an investigation
func (s *InvestigationService) UpdateInvestigationStatus(ctx context.Context, id string, status string) error {
	investigation, err := s.repo.GetInvestigation(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apierrors.NotFound("investigation", id)
		}
		return apierrors.Internal("Unable to update investigation status", err)
	}

	investigation.Status = status
	if status == models.StatusCompleted || status == models.StatusFailed {
		now := s.clock.Now()
		investigation.EndTime = &now
	}

	if err := s.repo.UpdateInvestigation(ctx, investigation); err != nil {
		return apierrors.Internal("Unable to update investigation status", err)
	}
	return nil
}

// UpdateInvestigationFindings updates the findings of an investigation
func (s *InvestigationService) UpdateInvestigationFindings(ctx context.Context, id string, findings string, details map[string]interface{}) error {
	investigation, err := s.repo.GetInvestigation(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apierrors.NotFound("investigation", id)
		}
		return apierrors.Internal("Unable to update investigation findings", err)
	}

	investigation.Findings = findings
	for k, v := range details {
		investigation.Details[k] = v
	}

	if err := s.repo.UpdateInvestigation(ctx, investigation); err != nil {
		return apierrors.Internal("Unable to update investigation findings", err)
	}
	return nil
}

// UpdateInvestigationProgress updates the progress of an investigation
func (s *InvestigationService) UpdateInvestigationProgress(ctx context.Context, id string, progress int) error {
	investigation, err := s.repo.GetInvestigation(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apierrors.NotFound("investigation", id)
		}
		return apierrors.Internal("Unable to update investigation progress", err)
	}

	investigation.Progress = progress
	if err := s.repo.UpdateInvestigation(ctx, investigation); err != nil {
		return apierrors.Internal("Unable to update investigation progress", err)
	}
	return nil
}

// AddInvestigationStep adds a step to an investigation
func (s *InvestigationService) AddInvestigationStep(ctx context.Context, id string, step models.InvestigationStep) error {
	step.StartTime = s.clock.Now()
	if err := s.repo.SaveInvestigationStep(ctx, id, &step); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apierrors.NotFound("investigation", id)
		}
		return apierrors.Internal("Unable to add investigation step", err)
	}
	return nil
}

// runInvestigation performs the actual investigation.
func (s *InvestigationService) runInvestigation(parent context.Context, investigationID string, autoFix bool, note string) {
	workerCtx, cancelWorker := context.WithTimeout(parent, 10*time.Minute)
	defer cancelWorker()

	// Update status to in_progress
	if err := s.UpdateInvestigationStatus(workerCtx, investigationID, models.StatusInProgress); err != nil {
		s.log.Error("failed to update investigation status", "investigation_id", investigationID, "error", err)
	}
	if err := s.UpdateInvestigationProgress(workerCtx, investigationID, 10); err != nil {
		s.log.Error("failed to update investigation progress", "investigation_id", investigationID, "error", err)
	}

	// Collect current metrics for context
	cpuUsage := 15.0
	memoryUsage := 45.0
	tcpConnections := 50
	if s.metricsSrc != nil {
		if fresh, err := s.metricsSrc.GetCurrentMetricsFresh(workerCtx); err == nil && fresh != nil {
			cpuUsage = fresh.CPUUsage
			memoryUsage = fresh.MemoryUsage
			tcpConnections = fresh.TCPConnections
		}
	}
	timestamp := s.clock.Now().Format(time.RFC3339)

	// Execute investigation via agent-manager
	findings, details, ok := s.performInvestigation(workerCtx, investigationID, cpuUsage, memoryUsage, tcpConnections, timestamp, autoFix, note)

	// Final status must survive worker cancellation but remain bounded. Keeping
	// the parent's values preserves trace and caller metadata for repository hooks.
	cancelWorker()
	cleanupCtx, cancelCleanup := detachedCleanupContext(parent, 30*time.Second)
	defer cancelCleanup()

	// Update investigation with findings
	if err := s.UpdateInvestigationFindings(cleanupCtx, investigationID, findings, details); err != nil {
		s.log.Error("failed to update investigation findings", "investigation_id", investigationID, "error", err)
	}
	if err := s.UpdateInvestigationProgress(cleanupCtx, investigationID, 100); err != nil {
		s.log.Error("failed to update investigation progress", "investigation_id", investigationID, "error", err)
	}
	current, err := s.repo.GetInvestigation(cleanupCtx, investigationID)
	if err == nil && current != nil {
		status := strings.ToLower(strings.TrimSpace(current.Status))
		if models.IsTerminalStatus(status) {
			return
		}
	}
	if ok {
		if err := s.UpdateInvestigationStatus(cleanupCtx, investigationID, models.StatusCompleted); err != nil {
			s.log.Error("failed to update investigation status", "investigation_id", investigationID, "status", models.StatusCompleted, "error", err)
		}
	} else {
		if err := s.UpdateInvestigationStatus(cleanupCtx, investigationID, models.StatusFailed); err != nil {
			s.log.Error("failed to update investigation status", "investigation_id", investigationID, "status", models.StatusFailed, "error", err)
		}
	}
}

func detachedCleanupContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(parent), timeout)
}

// performInvestigation executes the investigation logic via agent-manager.
func (s *InvestigationService) performInvestigation(ctx context.Context, investigationID string, cpuUsage, memoryUsage float64, tcpConnections int, timestamp string, autoFix bool, note string) (string, map[string]interface{}, bool) {
	operationMode := "report-only"
	if autoFix {
		operationMode = "auto-fix"
	}

	// Agent-manager is required
	if s.agentSvc == nil || !s.agentSvc.IsEnabled() {
		return "Investigation failed: agent-manager is not enabled", map[string]interface{}{
			"error":          "agent-manager not enabled",
			"auto_fix":       autoFix,
			"operation_mode": operationMode,
		}, false
	}

	if !s.agentSvc.IsAvailable(ctx) {
		return "Investigation failed: agent-manager is not available", map[string]interface{}{
			"error":          "agent-manager not available",
			"auto_fix":       autoFix,
			"operation_mode": operationMode,
		}, false
	}

	prompt := s.buildInvestigationPrompt(investigationID, cpuUsage, memoryUsage, tcpConnections, timestamp, autoFix, note)
	workingDir := resolveInvestigationWorkingDir(s.config.Monitoring.InvestigationRoot)

	return s.performAgentManagerInvestigation(ctx, investigationID, prompt, workingDir, autoFix, operationMode)
}

// performAgentManagerInvestigation uses agent-manager for execution.
func (s *InvestigationService) performAgentManagerInvestigation(ctx context.Context, investigationID, prompt, workingDir string, autoFix bool, operationMode string) (string, map[string]interface{}, bool) {
	result, err := s.agentSvc.Execute(ctx, agentmanager.ExecuteRequest{
		InvestigationID: investigationID,
		Prompt:          prompt,
		WorkingDir:      workingDir,
	})

	baseDetails := map[string]interface{}{
		"agent_source":   "agent-manager",
		"auto_fix":       autoFix,
		"operation_mode": operationMode,
	}

	if err != nil {
		s.log.Error("agent-manager execution failed", "investigation_id", investigationID, "error", err)
		baseDetails["agent_error"] = truncateAgentLog(err.Error(), 400)
		return fmt.Sprintf("Investigation failed: %v", err), baseDetails, false
	}

	if result.Success {
		details := map[string]interface{}{
			"source":           "agent-manager",
			"run_id":           result.RunID,
			"risk_level":       "low",
			"duration_seconds": result.DurationSeconds,
			"tokens_used":      result.TokensUsed,
			"cost_estimate":    result.CostEstimate,
		}
		for k, v := range baseDetails {
			details[k] = v
		}
		return strings.TrimSpace(result.Output), details, true
	}

	// Execution failed
	details := make(map[string]interface{})
	for k, v := range baseDetails {
		details[k] = v
	}
	details["run_id"] = result.RunID
	if result.ErrorMessage != "" {
		details["agent_error"] = truncateAgentLog(result.ErrorMessage, 400)
	}
	if result.RateLimited {
		details["agent_rate_limited"] = true
	}
	if result.Timeout {
		details["agent_timeout"] = true
	}

	return fmt.Sprintf("Investigation completed with issues: %s", result.ErrorMessage), details, false
}

// buildInvestigationPrompt builds the prompt for the investigation agent
func (s *InvestigationService) buildInvestigationPrompt(investigationID string, cpuUsage, memoryUsage float64, tcpConnections int, timestamp string, autoFix bool, note string) string {
	operationMode := "report-only"
	if autoFix {
		operationMode = "auto-fix"
	}

	prompt, err := s.loadPromptTemplate(investigationID, cpuUsage, memoryUsage, tcpConnections, timestamp, operationMode, note, autoFix)
	if err != nil {
		noteLine := ""
		if trimmed := strings.TrimSpace(note); trimmed != "" {
			noteLine = fmt.Sprintf("\nUser Note: %s", trimmed)
		}
		return fmt.Sprintf(`System Anomaly Investigation
Investigation ID: %s
API Base URL: %s
Operation Mode: %s
CPU: %.2f%%, Memory: %.2f%%, TCP Connections: %d%s
Analyze system for anomalies and provide findings.`,
			investigationID, s.resolveAPIBaseURL(), operationMode, cpuUsage, memoryUsage, tcpConnections, noteLine)
	}
	return prompt
}

// loadPromptTemplate loads and processes the prompt template
func (s *InvestigationService) loadPromptTemplate(investigationID string, cpuUsage, memoryUsage float64, tcpConnections int, timestamp, operationMode, note string, autoFix bool) (string, error) {
	promptContent, err := s.promptStore.ReadConfig("anomaly-check.md")
	if err != nil {
		return "", fmt.Errorf("could not read prompt file: %v", err)
	}

	prompt := string(promptContent)
	prompt = strings.ReplaceAll(prompt, "{{CPU_USAGE}}", fmt.Sprintf("%.2f", cpuUsage))
	prompt = strings.ReplaceAll(prompt, "{{MEMORY_USAGE}}", fmt.Sprintf("%.2f", memoryUsage))
	prompt = strings.ReplaceAll(prompt, "{{TCP_CONNECTIONS}}", strconv.Itoa(tcpConnections))
	prompt = strings.ReplaceAll(prompt, "{{TIMESTAMP}}", timestamp)
	prompt = strings.ReplaceAll(prompt, "{{INVESTIGATION_ID}}", investigationID)
	prompt = strings.ReplaceAll(prompt, "{{API_BASE_URL}}", s.resolveAPIBaseURL())
	prompt = strings.ReplaceAll(prompt, "{{OPERATION_MODE}}", operationMode)

	trimmedNote := strings.TrimSpace(note)
	if trimmedNote == "" {
		prompt = strings.ReplaceAll(prompt, "{{USER_NOTE}}", "No specific instructions provided.")
	} else {
		prompt = strings.ReplaceAll(prompt, "{{USER_NOTE}}", trimmedNote)
	}

	prompt = applyConditionalSections(prompt, autoFix)

	return prompt, nil
}

func resolveInvestigationWorkingDir(configuredRoot string) string {
	if custom := strings.TrimSpace(configuredRoot); custom != "" {
		if info, err := os.Stat(custom); err == nil && info.IsDir() {
			return custom
		}
	}

	if scenarioRoot, err := resolveSystemMonitorScenarioRoot(); err == nil {
		if info, statErr := os.Stat(scenarioRoot); statErr == nil && info.IsDir() {
			return scenarioRoot
		}
	}
	if repoRoot, err := repocontract.ResolveRepoRoot(); err == nil {
		return repoRoot
	}
	return ""
}

// resolveSystemMonitorScenarioRoot is retained only for the agent's optional
// working-directory seam. Catalog, config, prompt, and run-history resolution
// never call it; those product surfaces are storage-owned or embedded.
func resolveSystemMonitorScenarioRoot() (string, error) {
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return "", err
	}
	return repocontract.ResolveScenarioPath(repoRoot, "system-monitor")
}

func truncateAgentLog(raw string, limit int) string {
	trimmed := strings.TrimSpace(raw)
	if limit <= 0 || len(trimmed) <= limit {
		return trimmed
	}
	if limit < 4 {
		return trimmed[:limit]
	}
	return trimmed[:limit-3] + "..."
}

func applyConditionalSections(prompt string, autoFix bool) string {
	if autoFix {
		prompt = removeConditionalSection(prompt, "{{#IF_REPORT_ONLY}}", "{{/IF_REPORT_ONLY}}")
		prompt = strings.ReplaceAll(prompt, "{{#IF_AUTO_FIX}}", "")
		prompt = strings.ReplaceAll(prompt, "{{/IF_AUTO_FIX}}", "")
	} else {
		prompt = removeConditionalSection(prompt, "{{#IF_AUTO_FIX}}", "{{/IF_AUTO_FIX}}")
		prompt = strings.ReplaceAll(prompt, "{{#IF_REPORT_ONLY}}", "")
		prompt = strings.ReplaceAll(prompt, "{{/IF_REPORT_ONLY}}", "")
	}
	return prompt
}

func removeConditionalSection(input, startToken, endToken string) string {
	for {
		start := strings.Index(input, startToken)
		if start == -1 {
			break
		}
		end := strings.Index(input[start+len(startToken):], endToken)
		if end == -1 {
			break
		}
		end += start + len(startToken)
		input = input[:start] + input[end+len(endToken):]
	}
	return input
}

func (s *InvestigationService) resolveAPIBaseURL() string {
	return s.config.Server.APIBaseURL
}

// GetCooldownStatus returns the current cooldown status
func (s *InvestigationService) GetCooldownStatus(ctx context.Context) (*models.CooldownStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cooldownPeriod := s.currentCooldownPeriod()

	remainingSeconds := 0
	isReady := true

	if !s.lastTrigger.IsZero() {
		elapsed := s.clock.Since(s.lastTrigger)
		if elapsed < cooldownPeriod {
			remainingSeconds = int((cooldownPeriod - elapsed).Seconds())
			isReady = false
		}
	}

	return &models.CooldownStatus{
		CooldownPeriodSeconds: int(cooldownPeriod.Seconds()),
		RemainingSeconds:      remainingSeconds,
		LastTriggerTime:       s.lastTrigger,
		IsReady:               isReady,
	}, nil
}

// ResetCooldown resets the cooldown timer
func (s *InvestigationService) ResetCooldown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastTrigger = time.Time{} // Reset to zero time
	return nil
}

// GetInvestigationAgentStatus returns the investigation for an agent ID.
func (s *InvestigationService) GetInvestigationAgentStatus(ctx context.Context, id string) (*models.Investigation, error) {
	inv, err := s.repo.GetInvestigation(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apierrors.NotFound("investigation", id)
		}
		return nil, apierrors.Internal("Unable to retrieve investigation status", err)
	}
	return inv, nil
}

// StopInvestigationAgent attempts to stop a running investigation agent.
func (s *InvestigationService) StopInvestigationAgent(ctx context.Context, id string) error {
	investigation, err := s.repo.GetInvestigation(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apierrors.NotFound("investigation", id)
		}
		return apierrors.Internal("Unable to stop investigation", err)
	}

	if models.IsTerminalStatus(investigation.Status) {
		return nil
	}

	if s.agentSvc != nil && s.agentSvc.IsEnabled() {
		tag := fmt.Sprintf("system-monitor-%s", id)
		run, runErr := s.agentSvc.GetRunByTag(ctx, tag)
		if runErr == nil && run != nil {
			if stopErr := s.agentSvc.StopRun(ctx, run.Id); stopErr != nil {
				return stopErr
			}
		}
	}

	return s.UpdateInvestigationStatus(ctx, id, models.StatusStopped)
}

const legacyTriggerSettingsMessage = "automatic trigger settings are retired; use System Settings for thresholds and investigation policy, and Notification Hub for delivery preferences"

// GetTriggers is retained as an explicit migration response for older clients.
// Deprecated: the duplicate trigger configuration store has been removed.
func (s *InvestigationService) GetTriggers(context.Context) (map[string]*models.TriggerConfig, error) {
	return nil, apierrors.Validation("legacy_trigger_settings", legacyTriggerSettingsMessage)
}

// UpdateTrigger rejects writes to the retired trigger configuration surface.
// Deprecated: update canonical System Settings instead.
func (s *InvestigationService) UpdateTrigger(context.Context, string, *bool, *bool, *float64) error {
	return apierrors.Validation("legacy_trigger_settings", legacyTriggerSettingsMessage)
}

// UpdateCooldownPeriod updates the cooldown period for investigations
func (s *InvestigationService) UpdateCooldownPeriod(_ context.Context, periodSeconds int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if periodSeconds < 1 || periodSeconds > 86400 {
		return apierrors.Validation("cooldown_period_seconds", "must be between 1 and 86400")
	}
	settings := s.settings.GetSettings()
	settings.CooldownPeriodSeconds = periodSeconds
	if err := s.settings.UpdateSettings(settings); err != nil {
		return apierrors.Internal("Unable to save cooldown configuration", err)
	}
	return nil
}

func (s *InvestigationService) currentCooldownPeriod() time.Duration {
	return time.Duration(s.settings.GetSettings().CooldownPeriodSeconds) * time.Second
}

// =============================================================================
// Agent Configuration Methods
// =============================================================================

// AgentStatusResponse represents the agent-manager status.
type AgentStatusResponse struct {
	Enabled      bool   `json:"enabled"`
	Available    bool   `json:"available"`
	ProfileID    string `json:"profile_id,omitempty"`
	ActiveRuns   int    `json:"active_runs"`
	AgentManager string `json:"agent_manager_url,omitempty"`
	LastError    string `json:"last_error,omitempty"`
}

// GetAgentStatus returns the current agent-manager status.
func (s *InvestigationService) GetAgentStatus(ctx context.Context) (*AgentStatusResponse, error) {
	status := &AgentStatusResponse{
		Enabled: s.agentSvc != nil && s.agentSvc.IsEnabled(),
	}

	if !status.Enabled {
		return status, nil
	}

	if url, err := s.agentSvc.ResolveURL(ctx); err == nil {
		status.AgentManager = url
	} else {
		status.LastError = err.Error()
	}

	status.Available = s.agentSvc.IsAvailable(ctx)
	status.ProfileID = s.agentSvc.GetProfileID()

	// Get active runs
	runs, err := s.agentSvc.ListActiveRuns(ctx)
	if err == nil {
		status.ActiveRuns = len(runs)
	}

	return status, nil
}
