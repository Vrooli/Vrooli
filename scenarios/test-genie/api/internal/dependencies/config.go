package dependencies

import (
	"fmt"
	"strings"

	"test-genie/internal/shared"
)

// Settings is the dependency-phase policy loaded from .vrooli/testing.json.
type Settings struct {
	Strict          bool                       `json:"strict"`
	RuntimeVersions map[string]string          `json:"runtime_versions"`
	GoModules       GoModuleSettings           `json:"go_modules"`
	NodePackages    NodePackageSettings        `json:"node_packages"`
	Resources       ResourceHealthSettings     `json:"resources"`
	Scenarios       ScenarioDependencySettings `json:"scenarios"`
}

type GoModuleSettings struct {
	Enabled                bool `json:"enabled"`
	TidyDiff               bool `json:"tidy_diff"`
	Build                  bool `json:"build"`
	LocalReplaceResolution bool `json:"local_replace_resolution"`
}

type NodePackageSettings struct {
	Enabled            bool `json:"enabled"`
	RequireNodeModules bool `json:"require_node_modules"`
	LockfileRequired   bool `json:"lockfile_required"`
}

type ResourceHealthSettings struct {
	HealthPolicy                  string   `json:"health_policy"`
	AllowUnknownHealthWhenRunning bool     `json:"allow_unknown_health_when_running"`
	Skip                          []string `json:"skip"`
}

type ScenarioDependencySettings struct {
	Enabled      bool   `json:"enabled"`
	HealthPolicy string `json:"health_policy"`
}

func DefaultSettings() Settings {
	return Settings{
		Strict: true,
		RuntimeVersions: map[string]string{
			"go":      ">=1.21",
			"node":    ">=18.0.0",
			"python3": ">=3.10.0",
		},
		GoModules: GoModuleSettings{
			Enabled:                true,
			TidyDiff:               true,
			LocalReplaceResolution: true,
		},
		NodePackages: NodePackageSettings{
			Enabled:            true,
			RequireNodeModules: true,
			LockfileRequired:   true,
		},
		Resources: ResourceHealthSettings{
			HealthPolicy:                  "fail",
			AllowUnknownHealthWhenRunning: true,
		},
		Scenarios: ScenarioDependencySettings{
			Enabled:      true,
			HealthPolicy: "fail",
		},
	}
}

func LoadSettings(scenarioDir string) (Settings, error) {
	settings := DefaultSettings()
	if err := shared.MergePhaseConfig(scenarioDir, "dependencies", &settings); err != nil {
		return DefaultSettings(), err
	}
	normalizeSettings(&settings)
	if err := validateSettings(settings); err != nil {
		return DefaultSettings(), err
	}
	return settings, nil
}

func normalizeSettings(s *Settings) {
	if s.RuntimeVersions == nil {
		s.RuntimeVersions = DefaultSettings().RuntimeVersions
	}
	s.Resources.HealthPolicy = normalizePolicy(s.Resources.HealthPolicy, "fail")
	s.Scenarios.HealthPolicy = normalizePolicy(s.Scenarios.HealthPolicy, "fail")
}

func normalizePolicy(value, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return fallback
	case "fail", "warn", "skip":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return value
	}
}

func validateSettings(s Settings) error {
	for _, policy := range []struct {
		name  string
		value string
	}{
		{name: "resources.health_policy", value: s.Resources.HealthPolicy},
		{name: "scenarios.health_policy", value: s.Scenarios.HealthPolicy},
	} {
		switch policy.value {
		case "fail", "warn", "skip":
		default:
			return fmt.Errorf("dependencies.%s must be one of fail, warn, skip", policy.name)
		}
	}
	for command, constraint := range s.RuntimeVersions {
		if strings.TrimSpace(constraint) == "" {
			continue
		}
		if _, err := ParseVersionConstraint(constraint); err != nil {
			return fmt.Errorf("dependencies.runtime_versions.%s: %w", command, err)
		}
	}
	return nil
}
