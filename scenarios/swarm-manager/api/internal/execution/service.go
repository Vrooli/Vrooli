package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/storage"
)

var errNotFound = errors.New("execution not found")

// DOC: docs/concepts/ARCHITECTURE.md#key-flows
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/TEMPORAL-FLOWS.md

type agentSpawner interface {
	IsEnabled() bool
	SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error)
}

type runInspector interface {
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
}

type runStopper interface {
	StopRun(ctx context.Context, runID string) error
}

// ServiceConfig configures execution service dependencies.
type ServiceConfig struct {
	RootDir      string
	StorePath    string
	PolicyPath   string
	AgentService agentSpawner
	PromptClient promptmanager.Client
}

// Service owns execution lifecycle logic.
type Service struct {
	rootDir      string
	store        Store
	policyStore  *PolicyStore
	agentService agentSpawner
	promptClient promptmanager.Client
	inspector    runInspector
	stopper      runStopper
	mu           sync.Mutex
}

// NewService creates a new execution service.
func NewService(cfg ServiceConfig) *Service {
	rootDir := strings.TrimSpace(cfg.RootDir)
	if rootDir == "" {
		rootDir = pathutil.ResolveScenarioRoot("swarm-manager")
	}

	pc := cfg.PromptClient
	if pc == nil {
		pc = promptmanager.NewHTTPClient()
	}
	service := &Service{
		rootDir:      rootDir,
		store:        NewStore(cfg.StorePath),
		policyStore:  newPolicyStore(cfg.PolicyPath),
		agentService: cfg.AgentService,
		promptClient: pc,
	}
	if inspector, ok := cfg.AgentService.(runInspector); ok {
		service.inspector = inspector
	}
	if stopper, ok := cfg.AgentService.(runStopper); ok {
		service.stopper = stopper
	}
	return service
}

// QueueBacklog creates an execution record and optionally starts it.
func (s *Service) QueueBacklog(ctx context.Context, req CreateRequest) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy, err := s.policyStore.Load()
	if err != nil {
		return Record{}, err
	}

	if strings.TrimSpace(req.BacklogKind) == "" {
		return Record{}, fmt.Errorf("backlog_kind is required")
	}
	if strings.TrimSpace(req.BacklogName) == "" {
		return Record{}, fmt.Errorf("backlog_name is required")
	}
	mode := normalizeMode(req.Mode)
	if mode == "" {
		mode = policy.DefaultMode
	}
	if mode == "" {
		return Record{}, fmt.Errorf("mode must be manual, scheduled, or yolo")
	}
	if err := validateModeDelayInputs(mode, req.DelaySeconds); err != nil {
		return Record{}, err
	}

	item, err := s.loadBacklogItem(req.BacklogKind, req.BacklogName)
	if err != nil {
		return Record{}, err
	}
	if !isQueueableStatus(item.Status) {
		return Record{}, fmt.Errorf("backlog item cannot be queued from current status: %s", item.Status)
	}

	records, err := s.store.Load()
	if err != nil {
		return Record{}, err
	}

	now := nowRFC3339()
	record := Record{
		ExecutionID: idgen.Generate(),
		BacklogKind: strings.ToLower(strings.TrimSpace(req.BacklogKind)),
		BacklogName: strings.TrimSpace(req.BacklogName),
		Mode:        mode,
		Status:      StatusPending,
		StartedBy:   strings.TrimSpace(req.StartedBy),
		Operation:   normalizeOperation(req.Operation),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if record.StartedBy == "" {
		record.StartedBy = "swarm-manager"
	}
	if record.Operation == "" {
		record.Operation = "generator"
	}

	delaySeconds := req.DelaySeconds
	if mode == ModeScheduled && delaySeconds == 0 {
		delaySeconds = policy.DefaultDelaySeconds
	}
	if mode == ModeScheduled && delaySeconds <= 0 {
		return Record{}, fmt.Errorf("scheduled mode requires delay_seconds > 0 (or policy default > 0)")
	}
	scheduledAt, status := plannedSchedule(mode, delaySeconds)
	record.Status = status
	record.ScheduledAt = scheduledAt

	if err := s.updateBacklogStatus(item, "queued"); err != nil {
		return Record{}, err
	}

	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}

	if mode == ModeYOLO {
		return s.startLocked(ctx, record.ExecutionID)
	}
	return record, nil
}

// Policy returns current execution policy.
func (s *Service) Policy(_ context.Context) (Policy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.policyStore.Load()
}

// UpdatePolicy persists execution policy.
func (s *Service) UpdatePolicy(_ context.Context, policy Policy) (Policy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validatePolicyInputs(policy); err != nil {
		return Policy{}, err
	}
	normalized := normalizePolicy(policy)
	if err := s.policyStore.Save(normalized); err != nil {
		return Policy{}, err
	}
	return normalized, nil
}

// Start starts a pending/scheduled/failed execution now.
func (s *Service) Start(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startLocked(ctx, executionID)
}

func (s *Service) startLocked(ctx context.Context, executionID string) (Record, error) {
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := records[idx]
	if record.Status == StatusRunning || record.Status == StatusCompleted {
		return record, nil
	}
	if record.Status == StatusCanceled {
		return Record{}, fmt.Errorf("cannot start canceled execution")
	}

	if s.agentService == nil || !s.agentService.IsEnabled() {
		return Record{}, agentmanager.ErrNotAvailable
	}

	item, err := s.loadBacklogItem(record.BacklogKind, record.BacklogName)
	if err != nil {
		return Record{}, err
	}

	prompt, promptErr := s.fetchProcessingPrompt(ctx, item, record.Operation)
	if promptErr != nil {
		log.Printf("[execution] prompt fetch failed: %v", promptErr)
		prompt = "Use the backlog item folder as context and complete the requested work."
	}

	runResult, err := s.agentService.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind:        item.Kind,
		Name:        item.Name,
		Title:       buildProcessingTitle(item),
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   s.itemDir(item.Kind, item.Name),
		ProjectRoot: ".",
		CreatedBy:   record.StartedBy,
		Purpose:     "process",
	})
	if err != nil {
		return Record{}, err
	}

	record.TaskID = runResult.TaskID
	record.RunID = runResult.RunID
	record.StartedAt = nowRFC3339()
	record.FinishedAt = ""
	record.FailureReason = ""
	record.Status = StatusRunning
	record.UpdatedAt = nowRFC3339()
	record.ScheduledAt = ""
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}
	return record, nil
}

// Cancel cancels a scheduled record before it starts.
func (s *Service) Cancel(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := records[idx]

	switch record.Status {
	case StatusPending, StatusScheduled:
		record.Status = StatusCanceled
		record.UpdatedAt = nowRFC3339()
		record.FinishedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
		if item, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr == nil {
			_ = s.updateBacklogStatus(item, "ready")
		}
		return record, nil
	case StatusRunning:
		if s.stopper == nil {
			return Record{}, fmt.Errorf("cancel is not supported by current agent service")
		}
		if strings.TrimSpace(record.RunID) == "" {
			return Record{}, fmt.Errorf("execution has no run id")
		}
		if err := s.stopper.StopRun(ctx, record.RunID); err != nil {
			return Record{}, err
		}
		record.Status = StatusCanceled
		record.UpdatedAt = nowRFC3339()
		record.FinishedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
		if item, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr == nil {
			_ = s.updateBacklogStatus(item, "ready")
		}
		return record, nil
	default:
		return Record{}, fmt.Errorf("only pending/scheduled/running executions can be canceled")
	}
}

// Retry retries a failed run immediately.
func (s *Service) Retry(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	if records[idx].Status != StatusFailed {
		return Record{}, fmt.Errorf("only failed executions can be retried")
	}
	return s.startLocked(ctx, executionID)
}

// Get returns a single execution by ID after status refresh.
func (s *Service) Get(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.refreshRunningLocked(ctx)
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	return records[idx], nil
}

// List returns executions ordered by created_at descending.
func (s *Service) List(ctx context.Context, filters ListFilters) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_ = s.refreshRunningLocked(ctx)
	records, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	filtered := make([]Record, 0, len(records))
	for _, record := range records {
		if !matchesFilters(record, filters) {
			continue
		}
		filtered = append(filtered, record)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt == filtered[j].CreatedAt {
			return filtered[i].ExecutionID > filtered[j].ExecutionID
		}
		return filtered[i].CreatedAt > filtered[j].CreatedAt
	})

	return filtered, nil
}

// ProcessScheduledStarts starts due scheduled executions.
func (s *Service) ProcessScheduledStarts(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.store.Load()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	started := false
	for i := range records {
		record := records[i]
		if record.Status != StatusScheduled {
			continue
		}
		dueAt, err := time.Parse(time.RFC3339, strings.TrimSpace(record.ScheduledAt))
		if err != nil || dueAt.After(now) {
			continue
		}
		if _, err := s.startLocked(ctx, record.ExecutionID); err == nil {
			started = true
		}
	}
	if started {
		return nil
	}
	return nil
}

func (s *Service) refreshRunningLocked(ctx context.Context) error {
	if s.inspector == nil {
		return nil
	}
	records, err := s.store.Load()
	if err != nil {
		return err
	}

	changed := false
	for i := range records {
		record := &records[i]
		if record.Status != StatusRunning || strings.TrimSpace(record.RunID) == "" {
			continue
		}
		runState, err := s.inspector.GetRunState(ctx, record.RunID)
		if err != nil {
			continue
		}
		nextStatus, reason := mapRunStatus(runState.Status, runState.ErrorMsg)
		if nextStatus == StatusRunning {
			continue
		}
		record.Status = nextStatus
		record.FailureReason = reason
		record.UpdatedAt = nowRFC3339()
		if strings.TrimSpace(runState.FinishedAt) != "" {
			record.FinishedAt = runState.FinishedAt
		} else {
			record.FinishedAt = nowRFC3339()
		}
		if item, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr == nil {
			if nextStatus == StatusCompleted {
				_ = s.updateBacklogStatus(item, "completed")
			} else if nextStatus == StatusFailed || nextStatus == StatusCanceled {
				_ = s.updateBacklogStatus(item, "ready")
			}
		}
		changed = true
	}

	if changed {
		return s.store.Save(records)
	}
	return nil
}

func (s *Service) loadRecordLocked(executionID string) ([]Record, int, error) {
	records, err := s.store.Load()
	if err != nil {
		return nil, -1, err
	}
	trimmed := strings.TrimSpace(executionID)
	for i := range records {
		if records[i].ExecutionID == trimmed {
			return records, i, nil
		}
	}
	return nil, -1, errNotFound
}

func normalizeMode(mode Mode) Mode {
	switch Mode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case ModeManual:
		return ModeManual
	case ModeScheduled:
		return ModeScheduled
	case ModeYOLO:
		return ModeYOLO
	default:
		return ""
	}
}

func normalizeOperation(operation string) string {
	switch strings.ToLower(strings.TrimSpace(operation)) {
	case "", "generator":
		return "generator"
	case "improver":
		return "improver"
	default:
		return "generator"
	}
}

func validateModeDelayInputs(mode Mode, delaySeconds int64) error {
	if delaySeconds < 0 {
		return fmt.Errorf("delay_seconds must be >= 0")
	}
	if mode != ModeScheduled && delaySeconds > 0 {
		return fmt.Errorf("delay_seconds is only supported for scheduled mode")
	}
	return nil
}

func validatePolicyInputs(policy Policy) error {
	mode := normalizeMode(policy.DefaultMode)
	if mode == "" {
		return fmt.Errorf("default_mode must be manual, scheduled, or yolo")
	}
	if policy.DefaultDelaySeconds < 0 {
		return fmt.Errorf("default_delay_seconds must be >= 0")
	}
	if mode == ModeScheduled && policy.DefaultDelaySeconds <= 0 {
		return fmt.Errorf("scheduled default_mode requires default_delay_seconds > 0")
	}
	return nil
}

func plannedSchedule(mode Mode, delaySeconds int64) (string, Status) {
	switch mode {
	case ModeScheduled:
		delay := time.Duration(delaySeconds) * time.Second
		if delay < 0 {
			delay = 0
		}
		return time.Now().UTC().Add(delay).Format(time.RFC3339), StatusScheduled
	case ModeManual:
		return "", StatusPending
	default:
		return "", StatusPending
	}
}

func mapRunStatus(status, errorMsg string) (Status, string) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "starting", "running", "needs_review", "unspecified":
		return StatusRunning, ""
	case "complete":
		return StatusCompleted, ""
	case "failed":
		reason := strings.TrimSpace(errorMsg)
		if reason == "" {
			reason = "agent-manager run failed"
		}
		return StatusFailed, reason
	case "cancelled":
		return StatusCanceled, ""
	default:
		return StatusRunning, ""
	}
}

func matchesFilters(record Record, filters ListFilters) bool {
	if strings.TrimSpace(filters.Status) != "" && string(record.Status) != strings.TrimSpace(filters.Status) {
		return false
	}
	if strings.TrimSpace(filters.Mode) != "" && string(record.Mode) != strings.TrimSpace(filters.Mode) {
		return false
	}
	if strings.TrimSpace(filters.BacklogKind) != "" && record.BacklogKind != strings.TrimSpace(filters.BacklogKind) {
		return false
	}
	if strings.TrimSpace(filters.BacklogName) != "" && record.BacklogName != strings.TrimSpace(filters.BacklogName) {
		return false
	}
	if strings.TrimSpace(filters.StartedBy) != "" && record.StartedBy != strings.TrimSpace(filters.StartedBy) {
		return false
	}
	if strings.TrimSpace(filters.CreatedFrom) != "" {
		from, err := time.Parse(time.RFC3339, strings.TrimSpace(filters.CreatedFrom))
		if err == nil {
			createdAt, createdErr := time.Parse(time.RFC3339, strings.TrimSpace(record.CreatedAt))
			if createdErr == nil && createdAt.Before(from) {
				return false
			}
		}
	}
	if strings.TrimSpace(filters.CreatedTo) != "" {
		to, err := time.Parse(time.RFC3339, strings.TrimSpace(filters.CreatedTo))
		if err == nil {
			createdAt, createdErr := time.Parse(time.RFC3339, strings.TrimSpace(record.CreatedAt))
			if createdErr == nil && createdAt.After(to) {
				return false
			}
		}
	}
	return true
}

type backlogItem struct {
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Status         string   `json:"status"`
	Priority       int      `json:"priority"`
	Tags           []string `json:"tags"`
	Created        string   `json:"created"`
	Updated        string   `json:"updated"`
	Kind           string   `json:"kind"`
	ResearchTarget string   `json:"research_target,omitempty"`
}

func (s *Service) loadBacklogItem(kind, name string) (backlogItem, error) {
	specPath := filepath.Join(s.itemDir(kind, name), "spec.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		return backlogItem{}, err
	}
	var item backlogItem
	if err := json.Unmarshal(data, &item); err != nil {
		return backlogItem{}, err
	}
	item.Name = strings.TrimSpace(name)
	item.Kind = strings.ToLower(strings.TrimSpace(kind))
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return item, nil
}

func (s *Service) updateBacklogStatus(item backlogItem, status string) error {
	item.Status = status
	item.Updated = nowRFC3339()
	return storage.WriteJSONAtomic(filepath.Join(s.itemDir(item.Kind, item.Name), "spec.json"), item)
}

func (s *Service) kindDir(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "idea":
		return filepath.Join(s.rootDir, "ideas")
	case "research":
		return filepath.Join(s.rootDir, "research")
	case "fix":
		return filepath.Join(s.rootDir, "fix")
	case "execute":
		return filepath.Join(s.rootDir, "execute")
	default:
		return filepath.Join(s.rootDir, "ideas")
	}
}

func (s *Service) itemDir(kind, name string) string {
	return filepath.Join(s.kindDir(kind), strings.TrimSpace(name))
}

func isQueueableStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "backlog", "researching", "ready":
		return true
	default:
		return false
	}
}

func buildProcessingTitle(item backlogItem) string {
	label := strings.TrimSpace(item.Title)
	if label == "" {
		label = strings.TrimSpace(item.Name)
	}
	if label == "" {
		label = "backlog item"
	}
	switch item.Kind {
	case "fix":
		return "Apply fix: " + label
	case "execute":
		return "Execute task: " + label
	default:
		return "Generate scenario: " + label
	}
}

// processingSkillIDs maps backlog kind to prompt-manager skill IDs.
var processingSkillIDs = map[string]string{
	"idea":     "swarm-manager-process-idea",
	"fix":      "swarm-manager-process-fix",
	"execute":  "swarm-manager-process-execute",
	"research": "swarm-manager-process-execute",
}

// fetchProcessingPrompt loads a processing prompt from prompt-manager.
func (s *Service) fetchProcessingPrompt(ctx context.Context, item backlogItem, operation string) (string, error) {
	skillID := processingSkillIDs[item.Kind]
	if skillID == "" {
		skillID = "swarm-manager-process-execute"
	}

	vars := map[string]string{
		"ITEM_NAME":        item.Name,
		"ITEM_TITLE":       item.Title,
		"ITEM_DESCRIPTION": item.Description,
		"ITEM_KIND":        item.Kind,
		"ITEM_STATUS":      item.Status,
		"ITEM_PRIORITY":    fmt.Sprintf("%d", item.Priority),
		"ITEM_TAGS":        strings.Join(item.Tags, ", "),
		"ITEM_FOLDER":      s.itemDir(item.Kind, item.Name),
	}

	prompt, err := s.promptClient.ReadSkill(ctx, skillID, vars, true)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(operation) == "improver" {
		prompt = prompt + "\n\nOperation hint: improver (focus on improving an existing scenario).\n"
	}
	return prompt, nil
}
