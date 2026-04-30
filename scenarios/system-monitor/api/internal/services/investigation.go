package services

// DOC: docs/concepts/ARCHITECTURE.md#investigation-flow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"system-monitor-api/internal/agentmanager"
	"system-monitor-api/internal/apierrors"
	"system-monitor-api/internal/config"
	"system-monitor-api/internal/models"
	"system-monitor-api/internal/repository"
)

// InvestigationService handles anomaly investigations
type InvestigationService struct {
	log            *slog.Logger
	config         *config.Config
	repo           repository.InvestigationRepository
	metricsSrc     MetricsSource
	agentSvc       AgentExecutor
	clock          Clock
	configStore    ConfigStore
	promptStore    ConfigStore
	mu             sync.RWMutex
	cooldownPeriod time.Duration
	lastTrigger    time.Time
	triggers       map[string]*models.TriggerConfig
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

// InvestigationOption configures an InvestigationService.
type InvestigationOption func(*InvestigationService)

// WithInvestigationClock sets the clock used by the investigation service.
func WithInvestigationClock(c Clock) InvestigationOption {
	return func(s *InvestigationService) { s.clock = c }
}

// WithConfigStore sets the config store used by the investigation service.
func WithConfigStore(cs ConfigStore) InvestigationOption {
	return func(s *InvestigationService) { s.configStore = cs }
}

// WithPromptStore sets the prompt template store used by the investigation service.
func WithPromptStore(cs ConfigStore) InvestigationOption {
	return func(s *InvestigationService) { s.promptStore = cs }
}

// WithLogger sets the structured logger used by the investigation service.
func WithLogger(l *slog.Logger) InvestigationOption {
	return func(s *InvestigationService) { s.log = l }
}

// NewInvestigationService creates a new investigation service
func NewInvestigationService(cfg *config.Config, repo repository.InvestigationRepository, metricsSrc MetricsSource, agentSvc AgentExecutor, opts ...InvestigationOption) *InvestigationService {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	s := &InvestigationService{
		config:         cfg,
		repo:           repo,
		metricsSrc:     metricsSrc,
		agentSvc:       agentSvc,
		clock:          RealClock{},
		cooldownPeriod: 5 * time.Minute, // Default 5 minutes
		lastTrigger:    time.Time{},     // Start with zero time - no cooldown initially
		triggers:       make(map[string]*models.TriggerConfig),
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.log == nil {
		s.log = slog.Default()
	}
	if s.configStore == nil {
		s.configStore = &FileConfigStore{basePath: ResolveConfigBasePath()}
	}
	if s.promptStore == nil {
		s.promptStore = &FileConfigStore{basePath: ResolvePromptBasePath()}
	}

	// Load triggers from configuration file, fallback to defaults if not found
	if err := s.loadTriggersFromConfig(); err != nil {
		// Initialize default triggers if config not found
		s.initializeDefaultTriggers()
	}

	// Initialize agent profile if agent-manager is enabled
	if agentSvc != nil && agentSvc.IsEnabled() {
		go s.initializeAgentProfile()
	}

	return s
}

// Shutdown cancels the shutdown context, causing any in-flight investigations
// to be cancelled.
func (s *InvestigationService) Shutdown() {
	s.shutdownCancel()
}

// initializeAgentProfile ensures the agent profile exists in agent-manager.
func (s *InvestigationService) initializeAgentProfile() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.agentSvc.Initialize(ctx, agentmanager.DefaultProfileConfig()); err != nil {
		s.log.Warn("failed to initialize agent profile", "error", err)
	}
}

// TriggerInvestigation starts a new investigation
func (s *InvestigationService) TriggerInvestigation(ctx context.Context, autoFix bool, note string) (*models.Investigation, error) {
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
	if !s.lastTrigger.IsZero() {
		elapsed := s.clock.Since(s.lastTrigger)
		if elapsed < s.cooldownPeriod {
			return nil, apierrors.Cooldown(int((s.cooldownPeriod - elapsed).Seconds()))
		}
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
		return nil, apierrors.Internal("failed to create investigation", err)
	}

	// Update last trigger time only after successful creation
	s.lastTrigger = now

	// Start investigation in background with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error("investigation panicked", "investigation_id", investigationID, "panic", r)
				bgCtx := context.Background()
				_ = s.UpdateInvestigationStatus(bgCtx, investigationID, models.StatusFailed)
				_ = s.UpdateInvestigationFindings(bgCtx, investigationID,
					fmt.Sprintf("Investigation failed due to internal error: %v", r), nil)
			}
		}()
		s.runInvestigation(investigationID, autoFix, note)
	}()

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

// runInvestigation performs the actual investigation
func (s *InvestigationService) runInvestigation(investigationID string, autoFix bool, note string) {
	ctx, cancel := context.WithTimeout(s.shutdownCtx, 10*time.Minute)

	// Update status to in_progress
	if err := s.UpdateInvestigationStatus(ctx, investigationID, models.StatusInProgress); err != nil {
		s.log.Error("failed to update investigation status", "investigation_id", investigationID, "error", err)
	}
	if err := s.UpdateInvestigationProgress(ctx, investigationID, 10); err != nil {
		s.log.Error("failed to update investigation progress", "investigation_id", investigationID, "error", err)
	}

	// Collect current metrics for context
	cpuUsage := 15.0
	memoryUsage := 45.0
	tcpConnections := 50
	if s.metricsSrc != nil {
		if fresh, err := s.metricsSrc.GetCurrentMetricsFresh(ctx); err == nil && fresh != nil {
			cpuUsage = fresh.CPUUsage
			memoryUsage = fresh.MemoryUsage
			tcpConnections = fresh.TCPConnections
		}
	}
	timestamp := s.clock.Now().Format(time.RFC3339)

	// Execute investigation via agent-manager
	findings, details, ok := s.performInvestigation(investigationID, cpuUsage, memoryUsage, tcpConnections, timestamp, autoFix, note)

	// Cancel the long-running context and create a fresh one for final updates
	cancel()
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Update investigation with findings
	if err := s.UpdateInvestigationFindings(ctx, investigationID, findings, details); err != nil {
		s.log.Error("failed to update investigation findings", "investigation_id", investigationID, "error", err)
	}
	if err := s.UpdateInvestigationProgress(ctx, investigationID, 100); err != nil {
		s.log.Error("failed to update investigation progress", "investigation_id", investigationID, "error", err)
	}
	current, err := s.repo.GetInvestigation(ctx, investigationID)
	if err == nil && current != nil {
		status := strings.ToLower(strings.TrimSpace(current.Status))
		if models.IsTerminalStatus(status) {
			return
		}
	}
	if ok {
		if err := s.UpdateInvestigationStatus(ctx, investigationID, models.StatusCompleted); err != nil {
			s.log.Error("failed to update investigation status", "investigation_id", investigationID, "status", models.StatusCompleted, "error", err)
		}
	} else {
		if err := s.UpdateInvestigationStatus(ctx, investigationID, models.StatusFailed); err != nil {
			s.log.Error("failed to update investigation status", "investigation_id", investigationID, "status", models.StatusFailed, "error", err)
		}
	}
}

// performInvestigation executes the investigation logic via agent-manager.
func (s *InvestigationService) performInvestigation(investigationID string, cpuUsage, memoryUsage float64, tcpConnections int, timestamp string, autoFix bool, note string) (string, map[string]interface{}, bool) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if !s.agentSvc.IsAvailable(ctx) {
		return "Investigation failed: agent-manager is not available", map[string]interface{}{
			"error":          "agent-manager not available",
			"auto_fix":       autoFix,
			"operation_mode": operationMode,
		}, false
	}

	prompt := s.buildInvestigationPrompt(investigationID, cpuUsage, memoryUsage, tcpConnections, timestamp, autoFix, note)
	workingDir := resolveInvestigationWorkingDir()

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

func resolveInvestigationWorkingDir() string {
	if custom := strings.TrimSpace(os.Getenv("SYSTEM_MONITOR_INVESTIGATION_ROOT")); custom != "" {
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

// initializeDefaultTriggers sets up default trigger configurations
func (s *InvestigationService) initializeDefaultTriggers() {
	s.triggers["high_cpu"] = &models.TriggerConfig{
		ID:          "high_cpu",
		Name:        "High CPU Usage",
		Description: "Triggers when CPU usage exceeds threshold",
		Icon:        "cpu",
		Enabled:     true,
		AutoFix:     false,
		Threshold:   85.0,
		Unit:        "%",
		Condition:   "above",
	}

	s.triggers["memory_pressure"] = &models.TriggerConfig{
		ID:          "memory_pressure",
		Name:        "Memory Pressure",
		Description: "Triggers when available memory falls below threshold",
		Icon:        "database",
		Enabled:     true,
		AutoFix:     true,
		Threshold:   10.0,
		Unit:        "%",
		Condition:   "below",
	}

	s.triggers["disk_space"] = &models.TriggerConfig{
		ID:          "disk_space",
		Name:        "Low Disk Space",
		Description: "Triggers when disk usage exceeds threshold",
		Icon:        "hard-drive",
		Enabled:     true,
		AutoFix:     true,
		Threshold:   90.0,
		Unit:        "%",
		Condition:   "above",
	}

	s.triggers["network_connections"] = &models.TriggerConfig{
		ID:          "network_connections",
		Name:        "Excessive Network Connections",
		Description: "Triggers when active connections exceed normal levels",
		Icon:        "network",
		Enabled:     false,
		AutoFix:     false,
		Threshold:   1000.0,
		Unit:        " connections",
		Condition:   "above",
	}

	s.triggers["process_anomaly"] = &models.TriggerConfig{
		ID:          "process_anomaly",
		Name:        "Process Anomaly",
		Description: "Triggers when zombie or high-resource processes detected",
		Icon:        "zap",
		Enabled:     true,
		AutoFix:     false,
		Threshold:   5.0,
		Unit:        " processes",
		Condition:   "above",
	}
}

// GetCooldownStatus returns the current cooldown status
func (s *InvestigationService) GetCooldownStatus(ctx context.Context) (*models.CooldownStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	remainingSeconds := 0
	isReady := true

	if !s.lastTrigger.IsZero() {
		elapsed := s.clock.Since(s.lastTrigger)
		if elapsed < s.cooldownPeriod {
			remainingSeconds = int((s.cooldownPeriod - elapsed).Seconds())
			isReady = false
		}
	}

	return &models.CooldownStatus{
		CooldownPeriodSeconds: int(s.cooldownPeriod.Seconds()),
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

// GetTriggers returns all trigger configurations
func (s *InvestigationService) GetTriggers(ctx context.Context) (map[string]*models.TriggerConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	triggers := make(map[string]*models.TriggerConfig)
	for k, v := range s.triggers {
		triggers[k] = &models.TriggerConfig{
			ID:          v.ID,
			Name:        v.Name,
			Description: v.Description,
			Icon:        v.Icon,
			Enabled:     v.Enabled,
			AutoFix:     v.AutoFix,
			Threshold:   v.Threshold,
			Unit:        v.Unit,
			Condition:   v.Condition,
		}
	}

	return triggers, nil
}

// UpdateTrigger updates a trigger configuration
func (s *InvestigationService) UpdateTrigger(ctx context.Context, id string, enabled *bool, autoFix *bool, threshold *float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	trigger, exists := s.triggers[id]
	if !exists {
		return apierrors.NotFound("trigger", id)
	}

	if enabled != nil {
		trigger.Enabled = *enabled
	}
	if autoFix != nil {
		trigger.AutoFix = *autoFix
	}
	if threshold != nil {
		trigger.Threshold = *threshold
	}

	// Save to configuration file
	if err := s.saveTriggersToConfig(); err != nil {
		return apierrors.Internal("Unable to save trigger configuration", err)
	}
	return nil
}

// UpdateCooldownPeriod updates the cooldown period for investigations
func (s *InvestigationService) UpdateCooldownPeriod(ctx context.Context, periodSeconds int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cooldownPeriod = time.Duration(periodSeconds) * time.Second

	// Save to configuration file
	if err := s.saveTriggersToConfig(); err != nil {
		return apierrors.Internal("Unable to save cooldown configuration", err)
	}
	return nil
}

// loadTriggersFromConfig loads trigger configuration from JSON file
func (s *InvestigationService) loadTriggersFromConfig() error {
	data, err := s.configStore.ReadConfig("investigation-triggers.json")
	if err != nil {
		return err
	}

	var config struct {
		Cooldown struct {
			PeriodSeconds int `json:"period_seconds"`
		} `json:"cooldown"`
		Triggers []struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Icon        string  `json:"icon"`
			Enabled     bool    `json:"enabled"`
			AutoFix     bool    `json:"auto_fix"`
			Threshold   float64 `json:"threshold"`
			Unit        string  `json:"unit"`
			Condition   string  `json:"condition"`
		} `json:"triggers"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Update cooldown period
	s.cooldownPeriod = time.Duration(config.Cooldown.PeriodSeconds) * time.Second

	// Load triggers
	for _, t := range config.Triggers {
		s.triggers[t.ID] = &models.TriggerConfig{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Icon:        t.Icon,
			Enabled:     t.Enabled,
			AutoFix:     t.AutoFix,
			Threshold:   t.Threshold,
			Unit:        t.Unit,
			Condition:   t.Condition,
		}
	}

	return nil
}

// =============================================================================
// Agent Configuration Methods
// =============================================================================

// AgentConfigResponse represents the current agent configuration.
type AgentConfigResponse struct {
	Enabled         bool     `json:"enabled"`
	ProfileID       string   `json:"profile_id,omitempty"`
	ProfileName     string   `json:"profile_name"`
	RunnerType      string   `json:"runner_type"`
	Model           string   `json:"model"`
	MaxTurns        int32    `json:"max_turns"`
	TimeoutSeconds  int32    `json:"timeout_seconds"`
	AllowedTools    []string `json:"allowed_tools"`
	SkipPermissions bool     `json:"skip_permissions"`
	// SandboxMode is the proto enum string ("off"/"tracking"/"protected"/"unspecified").
	// system-monitor agents run with Mode=Off because they manipulate
	// host system state. Replaces the older RequiresSandbox/RequiresApproval
	// JSON pair removed in the agent-manager Phase 1 reliability pass.
	SandboxMode string `json:"sandbox_mode"`
}

// RunnerResponse represents an available runner.
type RunnerResponse struct {
	Type            string   `json:"type"`
	Name            string   `json:"name"`
	Available       bool     `json:"available"`
	Message         string   `json:"message,omitempty"`
	InstallHint     string   `json:"install_hint,omitempty"`
	SupportedModels []string `json:"supported_models,omitempty"`
}

// AgentStatusResponse represents the agent-manager status.
type AgentStatusResponse struct {
	Enabled      bool             `json:"enabled"`
	Available    bool             `json:"available"`
	ProfileID    string           `json:"profile_id,omitempty"`
	ActiveRuns   int              `json:"active_runs"`
	RunnerStatus []RunnerResponse `json:"runners,omitempty"`
	AgentManager string           `json:"agent_manager_url,omitempty"`
	LastError    string           `json:"last_error,omitempty"`
}

// GetAgentConfig returns the current agent configuration.
func (s *InvestigationService) GetAgentConfig(ctx context.Context) (*AgentConfigResponse, error) {
	if s.agentSvc == nil || !s.agentSvc.IsEnabled() {
		// Return defaults when agent-manager is not enabled
		defaultCfg := agentmanager.DefaultProfileConfig()
		return &AgentConfigResponse{
			Enabled:         false,
			ProfileName:     s.config.AgentManager.ProfileName,
			RunnerType:      runnerTypeToString(defaultCfg.RunnerType),
			Model:           defaultCfg.Model,
			MaxTurns:        defaultCfg.MaxTurns,
			TimeoutSeconds:  defaultCfg.TimeoutSeconds,
			AllowedTools:    defaultCfg.AllowedTools,
			SkipPermissions: defaultCfg.SkipPermissions,
			SandboxMode:     sandboxModeToString(defaultCfg.SandboxMode),
		}, nil
	}

	profile, err := s.agentSvc.GetProfile(ctx)
	if err != nil {
		// Return defaults with error context
		defaultCfg := agentmanager.DefaultProfileConfig()
		return &AgentConfigResponse{
			Enabled:         true,
			ProfileName:     s.config.AgentManager.ProfileName,
			RunnerType:      runnerTypeToString(defaultCfg.RunnerType),
			Model:           defaultCfg.Model,
			MaxTurns:        defaultCfg.MaxTurns,
			TimeoutSeconds:  defaultCfg.TimeoutSeconds,
			AllowedTools:    defaultCfg.AllowedTools,
			SkipPermissions: defaultCfg.SkipPermissions,
			SandboxMode:     sandboxModeToString(defaultCfg.SandboxMode),
		}, nil
	}

	timeoutSecs := int32(600)
	if profile.Timeout != nil {
		timeoutSecs = int32(profile.Timeout.AsDuration().Seconds())
	}

	return &AgentConfigResponse{
		Enabled:         true,
		ProfileID:       profile.Id,
		ProfileName:     profile.Name,
		RunnerType:      runnerTypeToString(profile.RunnerType),
		Model:           profile.Model,
		MaxTurns:        profile.MaxTurns,
		TimeoutSeconds:  timeoutSecs,
		AllowedTools:    profile.AllowedTools,
		SkipPermissions: profile.SkipPermissionPrompt,
		SandboxMode:     profileSandboxModeString(profile),
	}, nil
}

// sandboxModeToString renders an agent-manager SandboxMode enum as the
// stable JSON string used in the system-monitor REST API. Empty input
// (Unspecified) is rendered as "" so callers can distinguish "the
// profile didn't set a mode" from "the profile set Off explicitly".
func sandboxModeToString(m domainpb.SandboxMode) string {
	switch m {
	case domainpb.SandboxMode_SANDBOX_MODE_OFF:
		return "off"
	case domainpb.SandboxMode_SANDBOX_MODE_TRACKING:
		return "tracking"
	case domainpb.SandboxMode_SANDBOX_MODE_PROTECTED:
		return "protected"
	default:
		return ""
	}
}

// profileSandboxModeString returns the SandboxMode of an agent-manager
// profile as a string. Empty when SandboxConfig is nil.
func profileSandboxModeString(profile *domainpb.AgentProfile) string {
	if profile == nil || profile.SandboxConfig == nil {
		return ""
	}
	return sandboxModeToString(profile.SandboxConfig.Mode)
}

// stringToSandboxMode parses the string accepted by the system-monitor
// REST API back to the agent-manager SandboxMode enum. Empty / unknown
// values map to Unspecified so the orchestrator's DefaultSandboxConfig
// kicks in.
func stringToSandboxMode(s string) domainpb.SandboxMode {
	switch s {
	case "off":
		return domainpb.SandboxMode_SANDBOX_MODE_OFF
	case "tracking":
		return domainpb.SandboxMode_SANDBOX_MODE_TRACKING
	case "protected":
		return domainpb.SandboxMode_SANDBOX_MODE_PROTECTED
	default:
		return domainpb.SandboxMode_SANDBOX_MODE_UNSPECIFIED
	}
}

// GetAvailableRunners returns available runners from agent-manager.
func (s *InvestigationService) GetAvailableRunners(ctx context.Context) ([]RunnerResponse, error) {
	if s.agentSvc == nil || !s.agentSvc.IsEnabled() {
		// Agent-manager is required; return a disabled placeholder
		return []RunnerResponse{
			{
				Type:      "agent-manager",
				Name:      "agent-manager",
				Available: false,
				Message:   "agent-manager is required for investigations",
			},
		}, nil
	}

	runners, err := s.agentSvc.GetAvailableRunners(ctx)
	if err != nil {
		return nil, apierrors.Internal("Unable to retrieve available runners", err)
	}

	result := make([]RunnerResponse, 0, len(runners))
	for _, r := range runners {
		result = append(result, RunnerResponse{
			Type:            r.Name,
			Name:            r.Name,
			Available:       r.Available,
			Message:         r.Message,
			InstallHint:     r.InstallHint,
			SupportedModels: r.SupportedModels,
		})
	}

	return result, nil
}

// UpdateAgentConfig updates the agent profile configuration.
//
// sandboxMode replaces the previous (requiresSandbox bool, requiresApproval bool)
// pair. Accepted strings: "off", "tracking", "protected", "" (unspecified —
// agent-manager applies DefaultSandboxConfig).
func (s *InvestigationService) UpdateAgentConfig(ctx context.Context, runnerType, model string, maxTurns, timeoutSeconds int32, allowedTools []string, skipPermissions bool, sandboxMode string) (*AgentConfigResponse, error) {
	if s.agentSvc == nil || !s.agentSvc.IsEnabled() {
		return nil, apierrors.Unavailable("agent-manager")
	}

	cfg := &agentmanager.ProfileConfig{
		RunnerType:      stringToRunnerType(runnerType),
		Model:           model,
		MaxTurns:        maxTurns,
		TimeoutSeconds:  timeoutSeconds,
		AllowedTools:    allowedTools,
		SkipPermissions: skipPermissions,
		SandboxMode:     stringToSandboxMode(sandboxMode),
	}

	// Apply defaults if not provided
	defaultCfg := agentmanager.DefaultProfileConfig()
	if cfg.Model == "" {
		cfg.Model = defaultCfg.Model
	}
	if cfg.MaxTurns == 0 {
		cfg.MaxTurns = defaultCfg.MaxTurns
	}
	if cfg.TimeoutSeconds == 0 {
		cfg.TimeoutSeconds = defaultCfg.TimeoutSeconds
	}
	if len(cfg.AllowedTools) == 0 {
		cfg.AllowedTools = defaultCfg.AllowedTools
	}

	profile, err := s.agentSvc.UpdateProfile(ctx, cfg)
	if err != nil {
		return nil, apierrors.Internal("Failed to update agent configuration", err)
	}

	timeoutSecs := int32(600)
	if profile.Timeout != nil {
		timeoutSecs = int32(profile.Timeout.AsDuration().Seconds())
	}

	return &AgentConfigResponse{
		Enabled:         true,
		ProfileID:       profile.Id,
		ProfileName:     profile.Name,
		RunnerType:      runnerTypeToString(profile.RunnerType),
		Model:           profile.Model,
		MaxTurns:        profile.MaxTurns,
		TimeoutSeconds:  timeoutSecs,
		AllowedTools:    profile.AllowedTools,
		SkipPermissions: profile.SkipPermissionPrompt,
		SandboxMode:     profileSandboxModeString(profile),
	}, nil
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

	// Get runner status
	runners, err := s.agentSvc.GetAvailableRunners(ctx)
	if err == nil {
		for _, r := range runners {
			status.RunnerStatus = append(status.RunnerStatus, RunnerResponse{
				Type:            r.Name,
				Name:            r.Name,
				Available:       r.Available,
				Message:         r.Message,
				SupportedModels: r.SupportedModels,
			})
		}
	} else {
		status.LastError = err.Error()
	}

	return status, nil
}

// runnerTypeToString converts RunnerType enum to string.
func runnerTypeToString(rt domainpb.RunnerType) string {
	switch rt {
	case domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE:
		return "claude-code"
	case domainpb.RunnerType_RUNNER_TYPE_CODEX:
		return "codex"
	case domainpb.RunnerType_RUNNER_TYPE_OPENCODE:
		return "opencode"
	default:
		return "unknown"
	}
}

// stringToRunnerType converts string to RunnerType enum.
func stringToRunnerType(s string) domainpb.RunnerType {
	switch strings.ToLower(s) {
	case "claude-code", "claude_code":
		return domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE
	case "codex":
		return domainpb.RunnerType_RUNNER_TYPE_CODEX
	case "opencode":
		return domainpb.RunnerType_RUNNER_TYPE_OPENCODE
	default:
		return domainpb.RunnerType_RUNNER_TYPE_CODEX
	}
}

// saveTriggersToConfig saves trigger configuration to JSON file
func (s *InvestigationService) saveTriggersToConfig() error {
	// Read existing configuration to preserve extra fields
	existingData, err := s.configStore.ReadConfig("investigation-triggers.json")
	var existingConfig map[string]interface{}
	if err == nil {
		if err := json.Unmarshal(existingData, &existingConfig); err != nil {
			s.log.Warn("failed to unmarshal existing config", "error", err)
			existingConfig = make(map[string]interface{})
		}
	} else {
		existingConfig = make(map[string]interface{})
	}

	// Get existing triggers as a map for easier lookup
	existingTriggers := make(map[string]map[string]interface{})
	if triggers, ok := existingConfig["triggers"].([]interface{}); ok {
		for _, t := range triggers {
			if trigger, ok := t.(map[string]interface{}); ok {
				if id, ok := trigger["id"].(string); ok {
					existingTriggers[id] = trigger
				}
			}
		}
	}

	// Update triggers with current values while preserving extra fields
	var triggers []map[string]interface{}
	for _, t := range s.triggers {
		trigger := make(map[string]interface{})

		// Start with existing trigger data if it exists
		if existing, ok := existingTriggers[t.ID]; ok {
			for k, v := range existing {
				trigger[k] = v
			}
		}

		// Update with current values
		trigger["id"] = t.ID
		trigger["name"] = t.Name
		trigger["description"] = t.Description
		trigger["icon"] = t.Icon
		trigger["enabled"] = t.Enabled
		trigger["auto_fix"] = t.AutoFix
		trigger["threshold"] = t.Threshold
		trigger["unit"] = t.Unit
		trigger["condition"] = t.Condition

		triggers = append(triggers, trigger)
	}

	// Build final config preserving other top-level fields
	config := existingConfig
	config["version"] = "1.0.0"
	config["cooldown"] = map[string]interface{}{
		"period_seconds": int(s.cooldownPeriod.Seconds()),
		"description":    "Minimum time between automatic investigations to prevent spam",
	}
	config["triggers"] = triggers

	// Update metadata
	if metadata, ok := config["metadata"].(map[string]interface{}); ok {
		metadata["last_modified"] = s.clock.Now().Format(time.RFC3339)
	} else {
		config["metadata"] = map[string]interface{}{
			"last_modified":  s.clock.Now().Format(time.RFC3339),
			"config_version": "1.0.0",
		}
	}

	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	return s.configStore.WriteConfig("investigation-triggers.json", data)
}
