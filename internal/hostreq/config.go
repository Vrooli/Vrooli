package hostreq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func resolveSafeguardConfig(name string, manifest hostreqkit.SafeguardManifest, recorded map[string]any, required bool) (map[string]any, string, string) {
	if len(manifest.Config) == 0 {
		if len(recorded) == 0 {
			return nil, "", ""
		}
		return nil, fmt.Sprintf("invalid safeguard parameter(s) for %s: manifest declares no config schema", name), ""
	}

	config := make(map[string]any)
	if properties, ok := manifest.Config["properties"].(map[string]any); ok {
		for key, raw := range properties {
			property, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if value, exists := property["default"]; exists {
				config[key] = value
			}
		}
	}
	for key, value := range recorded {
		config[key] = value
	}
	if len(recorded) == 0 && !required {
		if raw, ok := manifest.Config["required"].([]any); ok && len(raw) > 0 {
			missing := make([]string, 0, len(raw))
			for _, value := range raw {
				if key, ok := value.(string); ok && key != "" {
					missing = append(missing, key)
				}
			}
			slices.Sort(missing)
			return config, "", fmt.Sprintf("optional safeguard is unconfigured; set %s with `vrooli-onboarding operator set-safeguard-config --name %s --key <name> --value-json <json>`", strings.Join(missing, ", "), name)
		}
		return config, "", ""
	}

	schemaData, err := json.Marshal(manifest.Config)
	if err != nil {
		return nil, fmt.Sprintf("invalid safeguard parameter(s) for %s: cannot encode config schema: %v", name, err), ""
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("safeguard-config.json", bytes.NewReader(schemaData)); err != nil {
		return nil, fmt.Sprintf("invalid safeguard parameter(s) for %s: cannot compile config schema: %v", name, err), ""
	}
	schema, err := compiler.Compile("safeguard-config.json")
	if err != nil {
		return nil, fmt.Sprintf("invalid safeguard parameter(s) for %s: cannot compile config schema: %v", name, err), ""
	}
	if err := schema.Validate(config); err != nil {
		return nil, fmt.Sprintf("invalid safeguard parameter(s) for %s: %v", name, err), ""
	}
	return config, "", ""
}

func configDiffersFromDefaults(manifest hostreqkit.SafeguardManifest, recorded map[string]any) bool {
	if len(recorded) == 0 {
		return false
	}
	properties, _ := manifest.Config["properties"].(map[string]any)
	for key, value := range recorded {
		property, _ := properties[key].(map[string]any)
		defaultValue, hasDefault := property["default"]
		if !hasDefault || !reflect.DeepEqual(defaultValue, value) {
			return true
		}
	}
	return false
}

// ValidateSafeguardConfig validates an operator-provided config through the
// same manifest-backed resolver used by host requirement resolution. It is a
// write-boundary helper for scenario APIs that persist operator state without
// duplicating JSON-Schema handling.
func ValidateSafeguardConfig(name string, recorded map[string]any) error {
	catalog, err := loadRequirementCatalog()
	if err != nil {
		return err
	}
	manifest, ok := catalog.safeguards[name]
	if !ok {
		return fmt.Errorf("unknown safeguard %q", name)
	}
	// Preserve keys introduced by a newer manifest. They are opaque to this
	// binary and therefore cannot be validated here, but known keys still go
	// through the canonical schema resolver below.
	known := make(map[string]any, len(recorded))
	if properties, ok := manifest.Config["properties"].(map[string]any); ok {
		for key, value := range recorded {
			if _, declared := properties[key]; declared {
				known[key] = value
			}
		}
	}
	_, validationError, _ := resolveSafeguardConfig(name, manifest, known, true)
	if validationError != "" {
		return fmt.Errorf("%s", validationError)
	}
	return nil
}
