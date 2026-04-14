package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"
)

var (
	allowedViolationTypes = map[string]struct{}{
		"security":  {},
		"standards": {},
	}
	allowedSeverities = []string{"critical", "high", "medium", "low"}
)

type AutomatedFixConfig struct {
	Enabled        bool      `json:"enabled"`
	ViolationTypes []string  `json:"violation_types"`
	Severities     []string  `json:"severities"`
	Strategy       string    `json:"strategy"`
	LoopDelay      int       `json:"loop_delay_seconds"`
	TimeoutSeconds int       `json:"timeout_seconds"`
	MaxFixes       int       `json:"max_fixes"`
	Model          string    `json:"model"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type AutomatedFixConfigInput struct {
	ViolationTypes []string `json:"violation_types"`
	Severities     []string `json:"severities"`
	Strategy       string   `json:"strategy"`
	LoopDelay      *int     `json:"loop_delay_seconds"`
	TimeoutSeconds *int     `json:"timeout_seconds"`
	MaxFixes       *int     `json:"max_fixes"`
	Model          string   `json:"model"`
}

type AutomatedFixRecord struct {
	ID            string     `json:"id"`
	ScenarioName  string     `json:"scenario_name"`
	ViolationType string     `json:"violation_type"`
	Severity      string     `json:"severity"`
	IssueCount    int        `json:"issue_count"`
	Status        string     `json:"status"`
	AppliedAt     time.Time  `json:"applied_at"`
	RolledBackAt  *time.Time `json:"rolled_back_at,omitempty"`
	AgentID       string     `json:"agent_id"`
	AutomationRun string     `json:"automation_run_id,omitempty"`
}

type automatedFixStoreData struct {
	Config AutomatedFixConfig `json:"config"`
}

type automatedFixHistoryData struct {
	History    []AutomatedFixRecord `json:"history"`
	LastUpdate time.Time            `json:"last_update"`
}

type AutomatedFixStore struct {
	mu          sync.RWMutex
	config      AutomatedFixConfig
	history     []AutomatedFixRecord
	configPath  string
	historyPath string
	maxHistory  int
}

var automatedFixStore = initAutomatedFixStore()

const (
	defaultAutomatedFixStrategy = "critical_first"
	defaultLoopDelaySeconds     = 300
	defaultTimeoutSeconds       = 3600
	defaultMaxFixes             = 0
)

func initAutomatedFixStore() *AutomatedFixStore {
	store := &AutomatedFixStore{
		config: AutomatedFixConfig{
			Enabled:        false,
			ViolationTypes: []string{"security"},
			Severities:     []string{"critical", "high"},
			Strategy:       defaultAutomatedFixStrategy,
			LoopDelay:      defaultLoopDelaySeconds,
			TimeoutSeconds: defaultTimeoutSeconds,
			MaxFixes:       defaultMaxFixes,
			Model:          openRouterModel,
			UpdatedAt:      time.Now().UTC(),
		},
		maxHistory: 100,
	}
	store.enablePersistence()
	return store
}

func (s *AutomatedFixStore) enablePersistence() {
	configPath, err := resolveScenarioAuditorStoragePath(storage.ClassConfig, "automated-fix-config.json")
	if err != nil {
		logger.Error("Failed to resolve scenario-auditor automated fix config path", err)
		return
	}
	historyPath, err := resolveScenarioAuditorStoragePath(storage.ClassData, "automated-fix-history.json")
	if err != nil {
		logger.Error("Failed to resolve scenario-auditor automated fix history path", err)
		return
	}

	s.configPath = configPath
	s.historyPath = historyPath
	if err := s.loadFromDisk(); err != nil {
		logger.Error("Failed to load automated fix store", err)
		// Continue with defaults
	}
}

func (s *AutomatedFixStore) loadFromDisk() error {
	if err := s.loadConfigFromDisk(); err != nil {
		return err
	}
	if err := s.loadHistoryFromDisk(); err != nil {
		return err
	}
	return nil
}

func (s *AutomatedFixStore) persistLocked() {
	s.persistConfigLocked()
	s.persistHistoryLocked()
}

func (s *AutomatedFixStore) loadConfigFromDisk() error {
	if strings.TrimSpace(s.configPath) == "" {
		return nil
	}
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var payload automatedFixStoreData
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	s.config = sanitizeConfig(payload.Config)
	return nil
}

func (s *AutomatedFixStore) loadHistoryFromDisk() error {
	if strings.TrimSpace(s.historyPath) == "" {
		return nil
	}
	data, err := os.ReadFile(s.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var payload automatedFixHistoryData
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	s.history = payload.History
	if len(s.history) > s.maxHistory {
		s.history = s.history[len(s.history)-s.maxHistory:]
	}
	return nil
}

func (s *AutomatedFixStore) persistConfigLocked() {
	if strings.TrimSpace(s.configPath) == "" {
		return
	}
	payload := automatedFixStoreData{Config: s.config}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		logger.Error("Failed to serialise automated fix config", err)
		return
	}
	if err := storage.WriteFileAtomic(s.configPath, data, storage.DefaultFilePerm); err != nil {
		logger.Error("Failed to persist automated fix config", err)
	}
}

func (s *AutomatedFixStore) persistHistoryLocked() {
	if strings.TrimSpace(s.historyPath) == "" {
		return
	}
	payload := automatedFixHistoryData{
		History:    append([]AutomatedFixRecord(nil), s.history...),
		LastUpdate: time.Now().UTC(),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		logger.Error("Failed to serialise automated fix history", err)
		return
	}
	if err := storage.WriteFileAtomic(s.historyPath, data, storage.DefaultFilePerm); err != nil {
		logger.Error("Failed to persist automated fix history", err)
	}
}

func (s *AutomatedFixStore) GetConfig() AutomatedFixConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg := s.config
	cfg.ViolationTypes = append([]string(nil), cfg.ViolationTypes...)
	cfg.Severities = append([]string(nil), cfg.Severities...)
	return cfg
}

func (s *AutomatedFixStore) Enable(input AutomatedFixConfigInput) AutomatedFixConfig {
	s.mu.Lock()
	s.applyInputLocked(true, input)
	cfg := s.config
	s.persistLocked()
	s.mu.Unlock()
	cfg.ViolationTypes = append([]string(nil), cfg.ViolationTypes...)
	cfg.Severities = append([]string(nil), cfg.Severities...)
	return cfg
}

func (s *AutomatedFixStore) Disable() AutomatedFixConfig {
	s.mu.Lock()
	s.config.Enabled = false
	s.config.UpdatedAt = time.Now().UTC()
	s.persistLocked()
	cfg := s.config
	s.mu.Unlock()
	cfg.ViolationTypes = append([]string(nil), cfg.ViolationTypes...)
	cfg.Severities = append([]string(nil), cfg.Severities...)
	return cfg
}

func (s *AutomatedFixStore) UpdateConfig(input AutomatedFixConfigInput) AutomatedFixConfig {
	s.mu.Lock()
	s.applyInputLocked(s.config.Enabled, input)
	cfg := s.config
	s.persistLocked()
	s.mu.Unlock()
	cfg.ViolationTypes = append([]string(nil), cfg.ViolationTypes...)
	cfg.Severities = append([]string(nil), cfg.Severities...)
	return cfg
}

func (s *AutomatedFixStore) Append(record AutomatedFixRecord) AutomatedFixRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = append(s.history, record)
	if len(s.history) > s.maxHistory {
		s.history = s.history[len(s.history)-s.maxHistory:]
	}
	s.persistLocked()
	return record
}

func (s *AutomatedFixStore) RecordStart(scenarioName, violationType, severity, agentID string, issueCount int, automationRunID string) AutomatedFixRecord {
	record := AutomatedFixRecord{
		ID:            uuid.NewString(),
		ScenarioName:  scenarioName,
		ViolationType: violationType,
		Severity:      severity,
		IssueCount:    issueCount,
		Status:        "in_progress",
		AppliedAt:     time.Now().UTC(),
		AgentID:       agentID,
		AutomationRun: strings.TrimSpace(automationRunID),
	}
	return s.Append(record)
}

func (s *AutomatedFixStore) RecordCompletion(agentID string, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for idx := range s.history {
		if s.history[idx].AgentID == agentID {
			if success {
				s.history[idx].Status = "applied"
			} else {
				s.history[idx].Status = "failed"
			}
			s.persistLocked()
			return
		}
	}
}

func (s *AutomatedFixStore) RecordRollback(id string) (AutomatedFixRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for idx := range s.history {
		if s.history[idx].ID == id {
			now := time.Now().UTC()
			s.history[idx].Status = "rolled_back"
			s.history[idx].RolledBackAt = &now
			s.persistLocked()
			return s.history[idx], nil
		}
	}
	return AutomatedFixRecord{}, fmt.Errorf("fix %s not found", id)
}

func (s *AutomatedFixStore) History() []AutomatedFixRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	history := make([]AutomatedFixRecord, len(s.history))
	copy(history, s.history)
	return history
}

func (s *AutomatedFixStore) applyInputLocked(enabled bool, input AutomatedFixConfigInput) {
	cleanedTypes := normaliseViolationTypes(input.ViolationTypes)
	if len(cleanedTypes) == 0 {
		cleanedTypes = append([]string(nil), s.config.ViolationTypes...)
	}
	cleanedSeverities := normaliseSeverities(input.Severities)
	if len(cleanedSeverities) == 0 {
		cleanedSeverities = append([]string(nil), s.config.Severities...)
	}

	strategy := sanitizeStrategy(input.Strategy, s.config.Strategy)
	loopDelay := s.config.LoopDelay
	if input.LoopDelay != nil {
		loopDelay = sanitizeLoopDelay(*input.LoopDelay, s.config.LoopDelay)
	}
	timeout := s.config.TimeoutSeconds
	if input.TimeoutSeconds != nil {
		timeout = sanitizeTimeout(*input.TimeoutSeconds, s.config.TimeoutSeconds)
	}
	maxFixes := s.config.MaxFixes
	if input.MaxFixes != nil {
		maxFixes = sanitizeMaxFixes(*input.MaxFixes, s.config.MaxFixes)
	}
	model := sanitizeAutomationModel(input.Model, s.config.Model)

	s.config.Enabled = enabled
	s.config.ViolationTypes = cleanedTypes
	s.config.Severities = cleanedSeverities
	s.config.Strategy = strategy
	s.config.LoopDelay = loopDelay
	s.config.TimeoutSeconds = timeout
	s.config.MaxFixes = maxFixes
	s.config.Model = model
	s.config.UpdatedAt = time.Now().UTC()
}

func sanitizeConfig(cfg AutomatedFixConfig) AutomatedFixConfig {
	cfg.ViolationTypes = normaliseViolationTypes(cfg.ViolationTypes)
	cfg.Severities = normaliseSeverities(cfg.Severities)
	if len(cfg.ViolationTypes) == 0 {
		cfg.ViolationTypes = []string{"security"}
	}
	if len(cfg.Severities) == 0 {
		cfg.Severities = []string{"critical", "high"}
	}
	cfg.Strategy = sanitizeStrategy(cfg.Strategy, defaultAutomatedFixStrategy)
	if strings.TrimSpace(cfg.Strategy) == "" {
		cfg.Strategy = defaultAutomatedFixStrategy
		if cfg.LoopDelay == 0 {
			cfg.LoopDelay = defaultLoopDelaySeconds
		}
		if cfg.TimeoutSeconds == 0 {
			cfg.TimeoutSeconds = defaultTimeoutSeconds
		}
	}
	cfg.LoopDelay = sanitizeLoopDelay(cfg.LoopDelay, defaultLoopDelaySeconds)
	cfg.TimeoutSeconds = sanitizeTimeout(cfg.TimeoutSeconds, defaultTimeoutSeconds)
	cfg.MaxFixes = sanitizeMaxFixes(cfg.MaxFixes, defaultMaxFixes)
	cfg.Model = sanitizeAutomationModel(cfg.Model, openRouterModel)
	return cfg
}

func normaliseViolationTypes(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	unique := make(map[string]struct{})
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if _, ok := allowedViolationTypes[trimmed]; ok {
			unique[trimmed] = struct{}{}
		}
	}
	if len(unique) == 0 {
		return nil
	}
	result := make([]string, 0, len(unique))
	for value := range unique {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func normaliseSeverities(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	allowed := make(map[string]int)
	for idx, value := range allowedSeverities {
		allowed[value] = idx
	}
	unique := make(map[string]struct{})
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if _, ok := allowed[trimmed]; ok {
			unique[trimmed] = struct{}{}
		}
	}
	if len(unique) == 0 {
		return nil
	}
	result := make([]string, 0, len(unique))
	for value := range unique {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		return allowed[result[i]] < allowed[result[j]]
	})
	return result
}

func sanitizeStrategy(value, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "critical_first", "security_first", "low_first", "standards_first":
		return value
	case "":
		return fallback
	default:
		return fallback
	}
}

func sanitizeLoopDelay(value, fallback int) int {
	if value < 0 {
		return fallback
	}
	if value == 0 {
		return 0
	}
	return value
}

func sanitizeTimeout(value, fallback int) int {
	if value < 0 {
		return fallback
	}
	return value
}

func sanitizeMaxFixes(value, fallback int) int {
	if value < 0 {
		return fallback
	}
	return value
}

func sanitizeAutomationModel(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		trimmed = strings.TrimSpace(fallback)
	}
	if trimmed == "" {
		trimmed = openRouterModel
	}
	return normalizeAgentModel(trimmed)
}

func computeSafetyStatus(cfg AutomatedFixConfig) string {
	if !cfg.Enabled {
		return "disabled"
	}

	lowIncluded := false
	mediumIncluded := false
	for _, severity := range cfg.Severities {
		switch severity {
		case "low":
			lowIncluded = true
		case "medium":
			mediumIncluded = true
		}
	}

	switch {
	case lowIncluded:
		return "high-risk"
	case mediumIncluded:
		return "moderate"
	default:
		return "guarded"
	}
}

func highestSeverityLabel(severities []string) string {
	priority := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}
	best := "unknown"
	bestScore := len(priority) + 1
	for _, severity := range severities {
		lower := strings.ToLower(strings.TrimSpace(severity))
		if score, ok := priority[lower]; ok {
			if score < bestScore {
				bestScore = score
				best = lower
			}
		}
	}
	return best
}
