// Package models owns the PII-detection model catalog and the durable
// active-model switch used by the managed-service driver. It mirrors the
// reranker resource's policy contract, adapted for token-classification PII
// models (capability "detect_pii") served over the Go + ONNX-Runtime process.
package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const defaultRole = "pii.small"

// detectCapability is the capability every PII model must declare.
const detectCapability = "detect_pii"

type Policy struct {
	SchemaVersion string           `json:"schema_version"`
	Roles         map[string]Role  `json:"roles"`
	Models        map[string]Model `json:"models"`
	Constraints   Constraints      `json:"constraints"`
}

type Role struct {
	Model                string   `json:"model"`
	Fallbacks            []string `json:"fallbacks"`
	RequiredCapabilities []string `json:"required_capabilities"`
	Description          string   `json:"description"`
	Preference           int      `json:"preference"`
	Provenance           Evidence `json:"provenance"`
}

type Model struct {
	Family          string      `json:"family"`
	Capabilities    []string    `json:"capabilities"`
	ContextWindow   int         `json:"context_window_tokens"`
	DiskBytes       int64       `json:"disk_bytes"`
	ResidentBytes   int64       `json:"resident_bytes"`
	DefaultEligible bool        `json:"default_eligible"`
	UseCaseNotes    string      `json:"use_case_notes"`
	Caveats         []string    `json:"caveats"`
	Measurement     Measurement `json:"measurement"`
}

// Measurement carries the PII eval figures. Unlike the reranker, a completed
// measurement is NOT required to load the policy: the models are acquired with
// pinned checksums but have no neutral benchmark, so our own corpus is the only
// authoritative source and it is populated after acquisition. The zero-value is
// a declared-but-unmeasured model, which is honest, not an error.
type Measurement struct {
	SuiteID      string  `json:"suite_id"`
	MeasuredAt   string  `json:"measured_at"`
	SampleCount  int     `json:"sample_count"`
	Precision    float64 `json:"precision"`
	Recall       float64 `json:"recall"`
	F1           float64 `json:"f1"`
	LatencyP95MS int64   `json:"latency_p95_ms"`
	Evidence     string  `json:"evidence"`
}

type Evidence struct {
	SourceKind  string `json:"source_kind"`
	Confidence  string `json:"confidence"`
	Source      string `json:"source"`
	ObservedAt  string `json:"observed_at"`
	SampleCount int    `json:"sample_count"`
}

type Constraints struct {
	PreferredBytes             int64  `json:"preferred_bytes"`
	FloorBytes                 int64  `json:"floor_bytes"`
	ResidentModelBudgetPercent int    `json:"resident_model_budget_percent"`
	MaxLoadedModelsPolicy      int    `json:"max_loaded_models_policy"`
	Role                       string `json:"role"`
	MeasurementRequired        bool   `json:"measurement_required"`
}

func Load(path string) (Policy, error) {
	if strings.TrimSpace(path) == "" {
		path = policyPath(os.Getenv)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read pii-detector model policy: %w", err)
	}
	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return Policy{}, fmt.Errorf("parse pii-detector model policy: %w", err)
	}
	if err := policy.Validate(); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func (p Policy) Validate() error {
	if strings.TrimSpace(p.SchemaVersion) == "" || len(p.Roles) == 0 || len(p.Models) == 0 {
		return errors.New("pii-detector model policy requires schema_version, roles, and models")
	}
	role, ok := p.Roles[defaultRole]
	if !ok || strings.TrimSpace(role.Model) == "" {
		return fmt.Errorf("pii-detector model policy requires role %q", defaultRole)
	}
	if p.Constraints.FloorBytes <= 0 || p.Constraints.PreferredBytes <= p.Constraints.FloorBytes {
		return errors.New("pii-detector model policy requires floor_bytes below preferred_bytes")
	}
	for name, model := range p.Models {
		if strings.TrimSpace(name) == "" || len(model.Capabilities) == 0 {
			return fmt.Errorf("pii-detector model %q is incomplete", name)
		}
		if !contains(model.Capabilities, detectCapability) {
			return fmt.Errorf("pii-detector model %q lacks %s capability", name, detectCapability)
		}
		// Measurement is advisory by default. When an operator flips
		// measurement_required (after our corpus eval exists) it becomes a
		// hard gate, mirroring the reranker's measured-policy discipline.
		if p.Constraints.MeasurementRequired {
			if model.Measurement.SuiteID != "pii.detection" || model.Measurement.SampleCount <= 0 || strings.TrimSpace(model.Measurement.Evidence) == "" {
				return fmt.Errorf("pii-detector model %q lacks a completed pii.detection measurement", name)
			}
		}
	}
	// Every role's model and fallbacks must resolve to a known model, and the
	// fallback chain must stay within the measured catalog.
	for roleName, r := range p.Roles {
		if strings.TrimSpace(r.Model) == "" {
			return fmt.Errorf("role %q has no model", roleName)
		}
		for _, name := range append([]string{r.Model}, r.Fallbacks...) {
			if _, ok := p.Models[name]; !ok {
				return fmt.Errorf("role %q references missing model %q", roleName, name)
			}
		}
	}
	return nil
}

func (p Policy) ModelNames() []string {
	result := make([]string, 0, len(p.Models))
	for name := range p.Models {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (p Policy) RoleNames() []string {
	result := make([]string, 0, len(p.Roles))
	for name := range p.Roles {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// Allowed reports whether modelName is the role's model or one of its measured
// fallbacks — the only models the activate path will accept for a role.
func (p Policy) Allowed(roleName, modelName string) bool {
	role, ok := p.Roles[roleName]
	if !ok {
		return false
	}
	if role.Model == modelName {
		return true
	}
	return contains(role.Fallbacks, modelName)
}

// DefaultModelForRole returns the role's primary model, falling back to the
// default role when the requested role is unknown.
func (p Policy) DefaultModelForRole(roleName string) (string, bool) {
	if role, ok := p.Roles[roleName]; ok {
		return role.Model, true
	}
	if role, ok := p.Roles[defaultRole]; ok {
		return role.Model, true
	}
	return "", false
}

// policyPath is the single model-policy.json resolver used by both Load and the
// catalog/policy commands. Resolution order: the explicit path override, then a
// known source-root env, then the per-OS user data dir, then a repo-relative
// last resort. getEnv is injected so it is unit-testable.
func policyPath(getEnv func(string) string) string {
	if path := strings.TrimSpace(getEnv("PII_DETECTOR_MODEL_POLICY_PATH")); path != "" {
		return path
	}
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT", "RESOURCE_ROOT"} {
		if root := strings.TrimSpace(getEnv(key)); root != "" {
			return filepath.Join(root, "resources", "pii-detector", "model-policy.json")
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Application Support", "vrooli", "resources", "pii-detector", "model-policy.json")
		}
		return filepath.Join(home, ".local", "share", "vrooli", "resources", "pii-detector", "model-policy.json")
	}
	return filepath.Join("resources", "pii-detector", "model-policy.json")
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
