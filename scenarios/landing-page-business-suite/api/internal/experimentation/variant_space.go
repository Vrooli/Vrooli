package experimentation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"landing-page-business-suite-api/internal/scenarioroot"

	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/logx"
)

var defaultVariantSpaceJSON = []byte(`{
	"_name": "Landing Variant Space (Fallback)",
	"_schemaVersion": 1,
	"axes": {
		"persona": {
			"variants": [
				{ "id": "ops_leader", "label": "Ops Leader" }
			]
		},
		"jtbd": {
			"variants": [
				{ "id": "launch_bundle", "label": "Launch bundle" }
			]
		},
		"conversionStyle": {
			"variants": [
				{ "id": "demo_led", "label": "Demo-led" }
			]
		}
	}
}`)

var (
	variantSpaceBytes   = readVariantSpaceFile()
	defaultVariantSpace = mustLoadVariantSpace()
)

type VariantSpace struct {
	Name            string                     `json:"_name"`
	SchemaVersion   int                        `json:"_schemaVersion"`
	Note            string                     `json:"_note,omitempty"`
	AgentGuidelines []string                   `json:"_agentGuidelines,omitempty"`
	Axes            map[string]*AxisDefinition `json:"axes"`
	Constraints     *VariantSpaceConstraints   `json:"constraints,omitempty"`
	rawJSON         json.RawMessage            `json:"-"`
}

type AxisDefinition struct {
	Note     string        `json:"_note,omitempty"`
	Variants []AxisVariant `json:"variants"`
}

type AxisVariant struct {
	ID            string            `json:"id"`
	Label         string            `json:"label"`
	Description   string            `json:"description,omitempty"`
	Examples      map[string]string `json:"examples,omitempty"`
	DefaultWeight float64           `json:"defaultWeight,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
	Status        string            `json:"status,omitempty"`
	AgentHints    []string          `json:"agentHints,omitempty"`
}

type VariantSpaceConstraints struct {
	Note                   string              `json:"_note,omitempty"`
	DisallowedCombinations []map[string]string `json:"disallowedCombinations,omitempty"`
}

func mustLoadVariantSpace() *VariantSpace {
	space, err := parseVariantSpace(variantSpaceBytes)
	if err != nil {
		logx.Printf("failed to parse variant space: %v; using baked defaults", err)
		space, err = parseVariantSpace(defaultVariantSpaceJSON)
		if err != nil {
			panic(fmt.Sprintf("default variant space invalid: %v", err))
		}
		variantSpaceBytes = append([]byte(nil), space.rawJSON...)
	}
	return space
}

func readVariantSpaceFile() []byte {
	return loadVariantSpaceBytes(variantSpaceFilePath())
}

func variantSpaceFilePath() string {
	if override := strings.TrimSpace(envx.Get("VARIANT_SPACE_PATH")); override != "" {
		return override
	}
	candidates := []string{}
	if root := scenarioroot.Resolve(); root != "" {
		candidates = append(candidates, filepath.Join(root, "config", "variant_space.json"))
	}
	candidates = append(candidates,
		filepath.Join("..", "config", "variant_space.json"),
		filepath.Join(".", "config", "variant_space.json"),
		filepath.Join("..", "..", "config", "variant_space.json"),
	)
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("..", "config", "variant_space.json")
}

func loadVariantSpaceBytes(path string) []byte {
	// #nosec G304 -- path is a scenario config candidate or deployment-controlled override.
	data, err := os.ReadFile(path)
	if err != nil {
		logx.Printf("failed to read variant space at %s: %v", path, err)
		return cloneBytes(defaultVariantSpaceJSON)
	}
	return cloneBytes(data)
}

func parseVariantSpace(data []byte) (*VariantSpace, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("variant space payload is empty")
	}

	var space VariantSpace
	if err := json.Unmarshal(data, &space); err != nil {
		return nil, fmt.Errorf("parse variant space: %w", err)
	}
	space.rawJSON = cloneBytes(data)
	return &space, nil
}

// JSONBytes returns the original JSON payload for serving via HTTP.
func (vs *VariantSpace) JSONBytes() []byte {
	if vs == nil || len(vs.rawJSON) == 0 {
		return variantSpaceBytes
	}
	return vs.rawJSON
}

// ValidateSelection ensures every axis has a valid variant and combination rules are satisfied.
func (vs *VariantSpace) ValidateSelection(selection map[string]string) error {
	if vs == nil {
		return fmt.Errorf("variant space not initialized")
	}
	space := Space{Axes: make(map[string]Axis, len(vs.Axes))}
	for axisID, axisDefinition := range vs.Axes {
		variants := make([]string, 0, len(axisDefinition.Variants))
		for _, variant := range axisDefinition.Variants {
			variants = append(variants, variant.ID)
		}
		space.Axes[axisID] = Axis{Variants: variants}
	}
	if vs.Constraints != nil {
		space.DisallowedCombinations = vs.Constraints.DisallowedCombinations
	}
	return ValidateSelection(space, selection)
}

// DefaultVariantSpace returns the immutable process-default experiment space.
// Callers should treat the returned value as read-only.
func DefaultVariantSpace() *VariantSpace {
	return defaultVariantSpace
}

func cloneBytes(data []byte) []byte {
	if data == nil {
		return nil
	}
	cloned := make([]byte, len(data))
	copy(cloned, data)
	return cloned
}
