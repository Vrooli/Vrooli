// DOC: docs/concepts/GRAPH.md#health-scoring
package graph

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"prompt-manager/store"
)

// HealthWeights defines tunable metric weights for a node type.
type HealthWeights struct {
	OutgoingEdges  float64 `json:"outgoingEdges"`
	IncomingEdges  float64 `json:"incomingEdges"`
	CodeUsage      float64 `json:"codeUsage"`
	RecentActivity float64 `json:"recentActivity"`
}

// CLIHealthConfig defines CLI-specific health policy levers.
type CLIHealthConfig struct {
	NeutralCommands       []string `json:"neutralCommands"`
	ExternalToolScore     float64  `json:"externalToolScore"`
	ScenarioFallbackScore float64  `json:"scenarioFallbackScore"`
}

// HealthConfig defines scoring controls per entity type.
type HealthConfig struct {
	Team  HealthWeights   `json:"team"`
	Agent HealthWeights   `json:"agent"`
	Skill HealthWeights   `json:"skill"`
	CLI   CLIHealthConfig `json:"cli"`
}

// HealthConfigProvider reads scoring configuration.
type HealthConfigProvider interface {
	Get(ctx context.Context) (HealthConfig, error)
}

// HealthConfigStore persists graph health config under the scenario store.
type HealthConfigStore struct {
	storeDir string
	mu       sync.RWMutex
}

const healthConfigRelativePath = "config/graph-health.json"

// NewHealthConfigStore creates a file-backed health config store.
func NewHealthConfigStore(storeDir string) *HealthConfigStore {
	return &HealthConfigStore{storeDir: storeDir}
}

func (s *HealthConfigStore) path() string {
	return filepath.Join(s.storeDir, healthConfigRelativePath)
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
	if err := ValidateHealthConfig(cfg); err != nil {
		return HealthConfig{}, err
	}
	return cfg, nil
}

// Put validates and saves config to disk.
func (s *HealthConfigStore) Put(_ context.Context, cfg HealthConfig) error {
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
		Team:  defaultWeights,
		Agent: defaultWeights,
		Skill: defaultWeights,
		CLI: CLIHealthConfig{
			NeutralCommands:       []string{"vrooli"},
			ExternalToolScore:     0.0,
			ScenarioFallbackScore: 0.0,
		},
	}
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
	}

	weightSum := 0.0
	for _, v := range values {
		if v.val < 0 {
			return fmt.Errorf("%s.%s must be >= 0", entity, v.name)
		}
		if v.val > 1 {
			return fmt.Errorf("%s.%s must be <= 1", entity, v.name)
		}
		weightSum += v.val
	}
	if weightSum <= 0 {
		return fmt.Errorf("%s must have at least one positive weight", entity)
	}
	return nil
}
