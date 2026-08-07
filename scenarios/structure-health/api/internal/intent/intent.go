// Package intent is structure-health's typed reader of a target scenario's
// declared service.json — the "intent" half of the reconcile. It is deliberately
// permissive (unknown fields are ignored, free-form steps are kept as-is) so it
// reads any scenario's manifest, not just the react-vite/Go shape.
package intent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Intent is the subset of service.json structure-health reconciles against code
// facts. Raw is the original decoded document for rules that need fields this
// typed view does not surface.
type Intent struct {
	Name        string
	DisplayName string
	CLIEnabled  bool
	CLICommand  string
	Ports       map[string]Port
	Lifecycle   Lifecycle
	Deps        Dependencies
	Raw         map[string]any
}

// Resolution records whether a target kind has a declared intent document.
// Presence is kept separate from the typed value so source-only targets do
// not look like they declared an empty scenario manifest.
type Resolution struct {
	Value    Intent
	Declared bool
	Source   string
}

// Port is a declared port binding.
type Port struct {
	EnvVar      string `json:"env_var"`
	Port        int    `json:"port"`
	Range       string `json:"range"`
	Description string `json:"description"`
}

// Lifecycle holds the lifecycle phases structure-health inspects.
type Lifecycle struct {
	Health  Health
	Setup   Phase
	Develop Phase
	Test    Phase
	Stop    Phase
}

// Health is the lifecycle health contract.
type Health struct {
	Endpoints          map[string]string `json:"endpoints"`
	Checks             []HealthCheck     `json:"checks"`
	StartupGracePeriod int               `json:"startup_grace_period"`
}

// HealthCheck is one declared health probe.
type HealthCheck struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Target   string `json:"target"`
	Critical bool   `json:"critical"`
}

// Phase is a lifecycle phase with its steps and an optional freshness condition.
type Phase struct {
	Description string     `json:"description"`
	Steps       []Step     `json:"steps"`
	Condition   *Condition `json:"condition"`
}

// Step is a single lifecycle step.
type Step struct {
	Name        string     `json:"name"`
	Run         string     `json:"run"`
	Description string     `json:"description"`
	Background  bool       `json:"background"`
	Condition   *Condition `json:"condition"`
}

// Condition gates a phase or step; its Checks are the freshness/provisioning
// checks structure-health validates for presence and well-formedness.
type Condition struct {
	FileExists string       `json:"file_exists"`
	Checks     []FreshCheck `json:"checks"`
}

// FreshCheck is one freshness/provisioning check (binaries, cli, ui-bundle, …).
type FreshCheck struct {
	Type       string   `json:"type"`
	Targets    []string `json:"targets"`
	Command    string   `json:"command"`
	BundlePath string   `json:"bundle_path"`
	SourceDir  string   `json:"source_dir"`
}

// Dependencies are the declared scenario/resource dependencies, each a flat
// object map keyed by canonical identifier.
type Dependencies struct {
	Scenarios map[string]Dependency `json:"scenarios"`
	Resources map[string]Dependency `json:"resources"`
}

// Dependency is one declared dependency edge. Name mirrors the map key for
// rules that carry the value around without its key.
type Dependency struct {
	Name            string `json:"name"`
	Enabled         *bool  `json:"enabled"`
	Required        *bool  `json:"required"`
	StartupPolicy   string `json:"startup_policy"`
	FreshnessPolicy string `json:"freshness_policy"`
}

// ServiceJSONRelPath is the canonical service.json location within a scenario.
const ServiceJSONRelPath = ".vrooli/service.json"

// Load reads and parses a scenario's service.json from its root directory.
func Load(scenarioRoot string) (Intent, error) {
	path := filepath.Join(scenarioRoot, filepath.FromSlash(ServiceJSONRelPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		return Intent{}, fmt.Errorf("read %s: %w", path, err)
	}
	return Parse(raw)
}

// Resolve selects the declaration source for a target kind. Scenarios use the
// historical service.json contract. Other kinds deliberately return an absent
// declaration until their kind-specific packs define an authoritative source.
func Resolve(targetKind, targetRoot string) (Resolution, error) {
	kind := strings.ToLower(strings.TrimSpace(targetKind))
	kind = strings.TrimPrefix(kind, "validation_target_kind_")
	if kind == "" {
		kind = "scenario"
	}
	if kind != "scenario" {
		return Resolution{Source: "none"}, nil
	}
	in, err := Load(targetRoot)
	if err != nil {
		return Resolution{Source: ServiceJSONRelPath}, err
	}
	return Resolution{Value: in, Declared: true, Source: ServiceJSONRelPath}, nil
}

// Parse decodes service.json bytes into the typed Intent view.
func Parse(raw []byte) (Intent, error) {
	var doc struct {
		Service struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"service"`
		CLI struct {
			Enabled bool   `json:"enabled"`
			Command string `json:"command"`
		} `json:"cli"`
		Ports     map[string]Port `json:"ports"`
		Lifecycle struct {
			Health  Health `json:"health"`
			Setup   Phase  `json:"setup"`
			Develop Phase  `json:"develop"`
			Test    Phase  `json:"test"`
			Stop    Phase  `json:"stop"`
		} `json:"lifecycle"`
		Dependencies Dependencies `json:"dependencies"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Intent{}, fmt.Errorf("parse service.json: %w", err)
	}
	var rawMap map[string]any
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return Intent{}, fmt.Errorf("parse service.json document: %w", err)
	}
	return Intent{
		Name:        doc.Service.Name,
		DisplayName: doc.Service.DisplayName,
		CLIEnabled:  doc.CLI.Enabled,
		CLICommand:  doc.CLI.Command,
		Ports:       doc.Ports,
		Lifecycle: Lifecycle{
			Health:  doc.Lifecycle.Health,
			Setup:   doc.Lifecycle.Setup,
			Develop: doc.Lifecycle.Develop,
			Test:    doc.Lifecycle.Test,
			Stop:    doc.Lifecycle.Stop,
		},
		Deps: doc.Dependencies,
		Raw:  rawMap,
	}, nil
}

// FreshCheckByType returns the setup-condition freshness checks of a given type.
func (i Intent) FreshCheckByType(checkType string) []FreshCheck {
	var out []FreshCheck
	if i.Lifecycle.Setup.Condition == nil {
		return out
	}
	for _, c := range i.Lifecycle.Setup.Condition.Checks {
		if c.Type == checkType {
			out = append(out, c)
		}
	}
	return out
}
