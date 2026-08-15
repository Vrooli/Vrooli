// Package models owns the rerank model catalog and the durable active-model
// switch used by the managed-service driver.
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

const defaultRole = "rerank.default"

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

type Measurement struct {
	SuiteID      string  `json:"suite_id"`
	MeasuredAt   string  `json:"measured_at"`
	SampleCount  int     `json:"sample_count"`
	Top1         float64 `json:"top1"`
	Top3         float64 `json:"top3"`
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
		path = policyPathFromEnvironment()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read reranker model policy: %w", err)
	}
	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return Policy{}, fmt.Errorf("parse reranker model policy: %w", err)
	}
	if err := policy.Validate(); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func (p Policy) Validate() error {
	if strings.TrimSpace(p.SchemaVersion) == "" || len(p.Roles) == 0 || len(p.Models) == 0 {
		return errors.New("reranker model policy requires schema_version, roles, and models")
	}
	role, ok := p.Roles[defaultRole]
	if !ok || strings.TrimSpace(role.Model) == "" {
		return fmt.Errorf("reranker model policy requires role %q", defaultRole)
	}
	if p.Constraints.FloorBytes <= 0 || p.Constraints.PreferredBytes <= p.Constraints.FloorBytes {
		return errors.New("reranker model policy requires floor_bytes below preferred_bytes")
	}
	for name, model := range p.Models {
		if strings.TrimSpace(name) == "" || len(model.Capabilities) == 0 {
			return fmt.Errorf("reranker model %q is incomplete", name)
		}
		if !contains(model.Capabilities, "rerank") {
			return fmt.Errorf("reranker model %q lacks rerank capability", name)
		}
		if model.Measurement.SuiteID != "router.routing" || model.Measurement.SampleCount <= 0 || strings.TrimSpace(model.Measurement.Evidence) == "" {
			return fmt.Errorf("reranker model %q lacks a completed router.routing measurement", name)
		}
	}
	for _, name := range append([]string{role.Model}, role.Fallbacks...) {
		if _, ok := p.Models[name]; !ok {
			return fmt.Errorf("role %q references missing model %q", defaultRole, name)
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

func policyPathFromEnvironment() string {
	if path := strings.TrimSpace(os.Getenv("RERANKER_MODEL_POLICY_PATH")); path != "" {
		return path
	}
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT", "RESOURCE_ROOT"} {
		if root := strings.TrimSpace(os.Getenv(key)); root != "" {
			return filepath.Join(root, "resources", "reranker", "model-policy.json")
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Application Support", "vrooli", "resources", "reranker", "model-policy.json")
		}
		return filepath.Join(home, ".local", "share", "vrooli", "resources", "reranker", "model-policy.json")
	}
	return filepath.Join("resources", "reranker", "model-policy.json")
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
