package lint

import (
	"fmt"

	"test-genie/internal/lint/execution"
	"test-genie/internal/shared"
)

const (
	HandlerGoModule      = "go_module"
	HandlerNodePackage   = "node_package"
	HandlerPythonProject = "python_project"
)

// Config holds runtime inputs for lint validation.
type Config struct {
	ScenarioDir   string
	ScenarioName  string
	CommandLookup LookupFunc
	CommandRunner execution.Runner
	Settings      *Settings
}

// Settings holds the configuration for lint validation, loaded from testing.json.
type Settings struct {
	Handlers   map[string]HandlerSettings   `json:"handlers"`
	Policy     PolicySettings               `json:"policy"`
	Components map[string]ComponentSettings `json:"components"`
	Ignore     []string                     `json:"ignore"`
}

// HandlerSettings configures one lint handler.
type HandlerSettings struct {
	Enabled *bool `json:"enabled,omitempty"`
	Strict  bool  `json:"strict"`
}

// ComponentSettings overrides lint behavior for one top-level component.
type ComponentSettings struct {
	Enabled *bool  `json:"enabled,omitempty"`
	Handler string `json:"handler,omitempty"`
	Strict  *bool  `json:"strict,omitempty"`
}

// PolicySettings configures unmatched/unconfigured component behavior.
type PolicySettings struct {
	UnconfiguredCommonComponents map[string]PolicySeverity `json:"unconfigured_common_components"`
	UnmatchedCodeComponents      PolicySeverity            `json:"unmatched_code_components"`
}

type configSection struct {
	Handlers   map[string]HandlerSettings   `json:"handlers"`
	Policy     PolicySettings               `json:"policy"`
	Components map[string]ComponentSettings `json:"components"`
	Ignore     []string                     `json:"ignore"`
}

// EnabledOrDefault returns the effective enabled value.
func (s HandlerSettings) EnabledOrDefault() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// StrictForComponent returns component-level strictness if overridden.
func (s HandlerSettings) StrictForComponent(component ComponentSettings) bool {
	if component.Strict != nil {
		return *component.Strict
	}
	return s.Strict
}

// ComponentEnabled returns true unless explicitly disabled.
func (s ComponentSettings) ComponentEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// LoadSettings reads lint settings from .vrooli/testing.json.
func LoadSettings(scenarioDir string) (*Settings, error) {
	settings := DefaultSettings()

	section, err := shared.LoadPhaseConfig(scenarioDir, "lint", configSection{})
	if err != nil {
		return nil, err
	}

	if len(section.Handlers) > 0 {
		for id, handler := range section.Handlers {
			settings.Handlers[id] = handler
		}
	}
	if len(section.Policy.UnconfiguredCommonComponents) > 0 {
		settings.Policy.UnconfiguredCommonComponents = make(map[string]PolicySeverity, len(section.Policy.UnconfiguredCommonComponents))
		for name, severity := range section.Policy.UnconfiguredCommonComponents {
			if err := validatePolicySeverity(severity); err != nil {
				return nil, fmt.Errorf("invalid lint.policy.unconfigured_common_components[%s]: %w", name, err)
			}
			settings.Policy.UnconfiguredCommonComponents[name] = severity
		}
	}
	if section.Policy.UnmatchedCodeComponents != "" {
		if err := validatePolicySeverity(section.Policy.UnmatchedCodeComponents); err != nil {
			return nil, fmt.Errorf("invalid lint.policy.unmatched_code_components: %w", err)
		}
		settings.Policy.UnmatchedCodeComponents = section.Policy.UnmatchedCodeComponents
	}
	if len(section.Components) > 0 {
		for name, component := range section.Components {
			settings.Components[name] = component
		}
	}
	if len(section.Ignore) > 0 {
		settings.Ignore = append([]string(nil), section.Ignore...)
	}

	return settings, nil
}

func validatePolicySeverity(severity PolicySeverity) error {
	switch severity {
	case PolicySeverityIgnore, PolicySeverityInfo, PolicySeverityWarning, PolicySeverityError:
		return nil
	default:
		return fmt.Errorf("unsupported severity %q", severity)
	}
}

// DefaultSettings returns the default lint validation settings.
func DefaultSettings() *Settings {
	return &Settings{
		Handlers: map[string]HandlerSettings{
			HandlerGoModule:      {},
			HandlerNodePackage:   {},
			HandlerPythonProject: {},
		},
		Policy: PolicySettings{
			UnconfiguredCommonComponents: map[string]PolicySeverity{
				"api": PolicySeverityError,
				"ui":  PolicySeverityWarning,
				"cli": PolicySeverityWarning,
			},
			UnmatchedCodeComponents: PolicySeverityWarning,
		},
		Components: map[string]ComponentSettings{},
		Ignore: []string{
			".git",
			".idea",
			".vscode",
			"assets",
			"coverage",
			"dist",
			"build",
			"docs",
			"node_modules",
			"test",
			"tmp",
			"vendor",
		},
	}
}
