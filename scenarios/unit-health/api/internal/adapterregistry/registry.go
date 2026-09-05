package adapterregistry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"unit-health/internal/adapters"
	"unit-health/internal/adapters/reactvitest"
)

// Registry is the single dispatch point for adapter-owned static analyzers.
// Framework-specific registration is isolated here; validation only consumes
// the normalized adapters.Analyzer contract.
type Registry struct {
	analyzers map[string]adapters.Analyzer
}

// NormalizeFramework applies only adapter-registry aliases. The validation
// kernel compares declared and observed values through this function instead
// of carrying ecosystem-specific aliases itself.
func NormalizeFramework(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "react-vite" {
		return "vite"
	}
	return value
}

func DefaultFramework(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "typescript", "javascript", "node":
		return "vitest"
	case "go":
		return "go test"
	default:
		return ""
	}
}

func MinimumCoverageFloor(adapterID, framework, language, className string) float64 {
	if strings.Contains(strings.ToLower(className), "react_vite") || strings.EqualFold(framework, "vitest") || strings.EqualFold(adapterID, "react-vitest") {
		return 85
	}
	if strings.EqualFold(language, "go") {
		return 75
	}
	return 0
}

func IsConfigFile(base string) bool {
	return base == "testing.json" || strings.HasPrefix(base, "vite.config.") || strings.HasPrefix(base, "vitest.config.")
}

func IsLockFile(base string) bool {
	return strings.HasSuffix(base, "-lock.yaml") || strings.HasSuffix(base, "-lock.json") || base == "go.sum" || base == "Cargo.lock" || base == "poetry.lock" || base == "Pipfile.lock"
}

func HasMarkerFile(root, language string) bool {
	markers := map[string][]string{
		"rust": {"Cargo.toml"},
	}
	for _, marker := range markers[strings.ToLower(strings.TrimSpace(language))] {
		if info, err := filepath.Abs(filepath.Join(root, marker)); err == nil {
			if stat, statErr := os.Stat(info); statErr == nil && !stat.IsDir() {
				return true
			}
		}
	}
	return false
}

func New() *Registry {
	return &Registry{analyzers: map[string]adapters.Analyzer{}}
}

func Default() *Registry {
	r := New()
	_ = r.Register(reactvitest.Analyzer{})
	return r
}

func (r *Registry) Register(analyzer adapters.Analyzer) error {
	if r == nil || analyzer == nil {
		return fmt.Errorf("adapter analyzer registry: analyzer is required")
	}
	identity := analyzer.Identity()
	if strings.TrimSpace(identity.ID) == "" {
		return fmt.Errorf("adapter analyzer registry: analyzer identity is required")
	}
	if _, exists := r.analyzers[identity.ID]; exists {
		return fmt.Errorf("adapter analyzer registry: duplicate analyzer %s", identity.ID)
	}
	r.analyzers[identity.ID] = analyzer
	return nil
}

func (r *Registry) Resolve(adapterID, language, framework string) (adapters.Analyzer, bool) {
	if r == nil {
		return nil, false
	}
	if strings.TrimSpace(adapterID) != "" {
		analyzer, ok := r.analyzers[adapterID]
		if ok && !analyzer.Matches(adapters.Match{Language: language, Framework: framework}) {
			return nil, false
		}
		return analyzer, ok
	}
	// Compatibility for pre-v2 in-memory fixtures: resolution is still routed
	// through the registry, and the alias is not applied to arbitrary language
	// surfaces.
	if strings.EqualFold(language, "typescript") && (strings.EqualFold(framework, "vite") || strings.EqualFold(framework, "vitest")) {
		analyzer, ok := r.analyzers["react-vitest"]
		return analyzer, ok
	}
	return nil, false
}
