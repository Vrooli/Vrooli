package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type ServiceManifest struct {
	Service struct {
		Name string `json:"name"`
	} `json:"service"`
	Environment map[string]string `json:"environment"`
	Lifecycle struct {
		Health struct {
			Checks []json.RawMessage `json:"checks"`
		} `json:"health"`
	} `json:"lifecycle"`
	Dependencies struct {
		Resources map[string]struct {
			Required bool   `json:"required"`
			Enabled  bool   `json:"enabled"`
			Type     string `json:"type"`
		} `json:"resources"`
	} `json:"dependencies"`
}

func LoadServiceManifest(path string) (*ServiceManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest %s: %w", path, err)
	}
	var manifest ServiceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest %s: %w", path, err)
	}
	return &manifest, nil
}

func (m *ServiceManifest) RequiredResources() []string {
	if m == nil || m.Dependencies.Resources == nil {
		return nil
	}
	var resources []string
	for key, value := range m.Dependencies.Resources {
		if value.Required {
			resources = append(resources, key)
		}
	}
	return resources
}

// SQLitePathEnvVars returns environment variable names that look like
// scenario-specific SQLite path overrides.
func (m *ServiceManifest) SQLitePathEnvVars() []string {
	if m == nil || len(m.Environment) == 0 {
		return nil
	}
	var keys []string
	for key := range m.Environment {
		upper := strings.ToUpper(key)
		if strings.Contains(upper, "SQLITE") && strings.HasSuffix(upper, "_PATH") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
