// DOC: docs/concepts/GRAPH.md#health-scoring
package graph

import (
	"context"
	"fmt"
	"path/filepath"
	"prompt-manager/store"
	"strings"
	"sync"
)

// HealthWeights defines tunable metric weights for a node type.
type HealthWeights struct {
	OutgoingEdges          float64 `json:"outgoingEdges"`
	IncomingEdges          float64 `json:"incomingEdges"`
	CodeUsage              float64 `json:"codeUsage"`
	RecentActivity         float64 `json:"recentActivity"`
	SkillContentLength     float64 `json:"skillContentLength"`
	AgentContextLoad       float64 `json:"agentContextLoad"`
	TeamMemberCountBalance float64 `json:"teamMemberCountBalance"`
	TeamRoleCoverage       float64 `json:"teamRoleCoverage"`
	ActionContract         float64 `json:"actionContract"`
	ActionCommand          float64 `json:"actionCommand"`
	ActionExamples         float64 `json:"actionExamples"`
	ActionOwner            float64 `json:"actionOwner"`
}

// CLIHealthConfig defines CLI-specific health policy levers.
type CLIHealthConfig struct {
	NeutralCommands       []string `json:"neutralCommands"`
	ExternalToolScore     float64  `json:"externalToolScore"`
	ScenarioFallbackScore float64  `json:"scenarioFallbackScore"`
}

// HealthConfig defines scoring controls per entity type.
type HealthConfig struct {
	Team   HealthWeights   `json:"team"`
	Agent  HealthWeights   `json:"agent"`
	Skill  HealthWeights   `json:"skill"`
	Action HealthWeights   `json:"action"`
	CLI    CLIHealthConfig `json:"cli"`
}

// HealthConfigProvider reads scoring configuration.
type HealthConfigProvider interface {
	Get(ctx context.Context) (HealthConfig, error)
}

// HealthConfigStore persists graph health config under the scenario store.
type HealthConfigStore struct {
	configDir string
	mu       sync.RWMutex
}

const healthConfigRelativePath = "config/graph-health.json"

// NewHealthConfigStore creates a file-backed health config store.
func NewHealthConfigStore(configDir string) *HealthConfigStore {
	return &HealthConfigStore{configDir: configDir}
}

func (s *HealthConfigStore) path() string {
	return filepath.Join(s.configDir, healthConfigRelativePath)
}

// Get loads config from disk or returns defaults if missing.
func (s *HealthConfigStore) Get(_ context.Context) (HealthConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.path()
	if !store.FileExists(path) {
		return DefaultHealthConfig(), nil
	}

	loaded, err := store.LoadJSON[HealthConfig](path)
	if err != nil {
		return HealthConfig{}, err
	}
	cfg := *loaded
	cfg = withHealthConfigDefaults(cfg)
	if err := ValidateHealthConfig(cfg); err != nil {
		return HealthConfig{}, err
	}
	return cfg, nil
}

// Put validates and saves config to disk.
func (s *HealthConfigStore) Put(_ context.Context, cfg HealthConfig) error {
	cfg = withHealthConfigDefaults(cfg)
	if err := ValidateHealthConfig(cfg); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return store.SaveJSON(s.path(), &cfg)
}

// DefaultHealthConfig returns the canonical health scoring defaults.
func DefaultHealthConfig() HealthConfig {
	defaultWeights := HealthWeights{
		OutgoingEdges:  1.0,
		IncomingEdges:  1.0,
		CodeUsage:      0.5,
		RecentActivity: 0.5,
	}
	return HealthConfig{
		Team: HealthWeights{
			OutgoingEdges:          defaultWeights.OutgoingEdges,
			IncomingEdges:          defaultWeights.IncomingEdges,
			CodeUsage:              defaultWeights.CodeUsage,
			RecentActivity:         defaultWeights.RecentActivity,
			TeamMemberCountBalance: 0.75,
			TeamRoleCoverage:       0.75,
		},
		Agent: HealthWeights{
			OutgoingEdges:    defaultWeights.OutgoingEdges,
			IncomingEdges:    defaultWeights.IncomingEdges,
			CodeUsage:        defaultWeights.CodeUsage,
			RecentActivity:   defaultWeights.RecentActivity,
			AgentContextLoad: 0.75,
		},
		Skill: HealthWeights{
			OutgoingEdges:      defaultWeights.OutgoingEdges,
			IncomingEdges:      defaultWeights.IncomingEdges,
			CodeUsage:          defaultWeights.CodeUsage,
			RecentActivity:     defaultWeights.RecentActivity,
			SkillContentLength: 0.75,
		},
		Action: HealthWeights{
			OutgoingEdges:  defaultWeights.OutgoingEdges,
			IncomingEdges:  defaultWeights.IncomingEdges,
			RecentActivity: defaultWeights.RecentActivity,
			ActionContract: 1.0,
			ActionCommand:  1.0,
			ActionExamples: 0.75,
			ActionOwner:    0.75,
		},
		CLI: CLIHealthConfig{
			NeutralCommands:       []string{"vrooli"},
			ExternalToolScore:     0.0,
			ScenarioFallbackScore: 0.0,
		},
	}
}

func withHealthConfigDefaults(cfg HealthConfig) HealthConfig {
	defaults := DefaultHealthConfig()
	if healthWeightSum(cfg.Action) <= 0 {
		cfg.Action = defaults.Action
	}
	if len(cfg.CLI.NeutralCommands) == 0 {
		cfg.CLI.NeutralCommands = defaults.CLI.NeutralCommands
	}
	return cfg
}

// ValidateHealthConfig checks control-surface safety and completeness.
func ValidateHealthConfig(cfg HealthConfig) error {
	for _, check := range []struct {
		name string
		w    HealthWeights
	}{
		{name: "team", w: cfg.Team},
		{name: "agent", w: cfg.Agent},
		{name: "skill", w: cfg.Skill},
		{name: "action", w: cfg.Action},
	} {
		if err := validateWeights(check.name, check.w); err != nil {
			return err
		}
	}

	if cfg.CLI.ExternalToolScore < 0 || cfg.CLI.ExternalToolScore > 1 {
		return fmt.Errorf("cli.externalToolScore must be between 0 and 1")
	}
	if cfg.CLI.ScenarioFallbackScore < 0 || cfg.CLI.ScenarioFallbackScore > 1 {
		return fmt.Errorf("cli.scenarioFallbackScore must be between 0 and 1")
	}
	if len(cfg.CLI.NeutralCommands) == 0 {
		return fmt.Errorf("cli.neutralCommands must include at least one command")
	}
	for _, cmd := range cfg.CLI.NeutralCommands {
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("cli.neutralCommands must not contain empty values")
		}
	}
	return nil
}

func validateWeights(entity string, w HealthWeights) error {
	values := []struct {
		name string
		val  float64
	}{
		{name: "outgoingEdges", val: w.OutgoingEdges},
		{name: "incomingEdges", val: w.IncomingEdges},
		{name: "codeUsage", val: w.CodeUsage},
		{name: "recentActivity", val: w.RecentActivity},
		{name: "skillContentLength", val: w.SkillContentLength},
		{name: "agentContextLoad", val: w.AgentContextLoad},
		{name: "teamMemberCountBalance", val: w.TeamMemberCountBalance},
		{name: "teamRoleCoverage", val: w.TeamRoleCoverage},
		{name: "actionContract", val: w.ActionContract},
		{name: "actionCommand", val: w.ActionCommand},
		{name: "actionExamples", val: w.ActionExamples},
		{name: "actionOwner", val: w.ActionOwner},
	}

	weightSum := healthWeightSum(w)
	for _, v := range values {
		if v.val < 0 {
			return fmt.Errorf("%s.%s must be >= 0", entity, v.name)
		}
		if v.val > 1 {
			return fmt.Errorf("%s.%s must be <= 1", entity, v.name)
		}
	}
	if weightSum <= 0 {
		return fmt.Errorf("%s must have at least one positive weight", entity)
	}
	return nil
}

func healthWeightSum(w HealthWeights) float64 {
	values := []float64{
		w.OutgoingEdges,
		w.IncomingEdges,
		w.CodeUsage,
		w.RecentActivity,
		w.SkillContentLength,
		w.AgentContextLoad,
		w.TeamMemberCountBalance,
		w.TeamRoleCoverage,
		w.ActionContract,
		w.ActionCommand,
		w.ActionExamples,
		w.ActionOwner,
	}
	sum := 0.0
	for _, val := range values {
		sum += val
	}
	return sum
}
