// Package variantspace loads and validates the landing-page A/B "variant
// space": the declarative catalog of selection axes (persona, jtbd,
// conversionStyle, …) and the allowed values per axis. Both the variant domain
// (which validates axis selections when creating/updating variants) and the
// variant_space domain (which serves the raw catalog to the admin UI) depend on
// this package, so it lives in internal/ with no database coupling.
package variantspace

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// DefaultJSON is the baked-in fallback catalog used when no variant_space.json
// file is present. It is intentionally minimal — a real scenario ships its own
// .vrooli/variant_space.json.
var DefaultJSON = []byte(`{
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

// Space is the parsed variant space: named axes each holding an ordered set of
// allowed variant values, plus optional combination constraints.
type Space struct {
	Name            string                     `json:"_name"`
	SchemaVersion   int                        `json:"_schemaVersion"`
	Note            string                     `json:"_note,omitempty"`
	AgentGuidelines []string                   `json:"_agentGuidelines,omitempty"`
	Axes            map[string]*AxisDefinition `json:"axes"`
	Constraints     *Constraints               `json:"constraints,omitempty"`
	rawJSON         json.RawMessage            `json:"-"`
}

// AxisDefinition is one selection axis and its allowed variants.
type AxisDefinition struct {
	Note     string        `json:"_note,omitempty"`
	Variants []AxisVariant `json:"variants"`
}

// AxisVariant is a single allowed value on an axis.
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

// Constraints holds cross-axis rules, currently disallowed combinations.
type Constraints struct {
	Note                   string              `json:"_note,omitempty"`
	DisallowedCombinations []map[string]string `json:"disallowedCombinations,omitempty"`
}

// Parse decodes a variant space payload, preserving the raw JSON for verbatim
// serving.
func Parse(data []byte) (*Space, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("variant space payload is empty")
	}
	var space Space
	if err := json.Unmarshal(data, &space); err != nil {
		return nil, fmt.Errorf("parse variant space: %w", err)
	}
	space.rawJSON = cloneBytes(data)
	return &space, nil
}

// Default returns the baked-in fallback space. It panics only if DefaultJSON is
// itself invalid, which is a programming error.
func Default() *Space {
	space, err := Parse(DefaultJSON)
	if err != nil {
		panic(fmt.Sprintf("default variant space invalid: %v", err))
	}
	return space
}

// Load resolves the variant space from VARIANT_SPACE_PATH (or the scenario's
// ../.vrooli/variant_space.json), falling back to the baked-in default on any
// read or parse error.
func Load() *Space {
	data := loadBytes(filePath())
	space, err := Parse(data)
	if err != nil {
		log.Printf("failed to parse variant space: %v; using baked defaults", err)
		return Default()
	}
	return space
}

func filePath() string {
	if override := strings.TrimSpace(os.Getenv("VARIANT_SPACE_PATH")); override != "" {
		return override
	}
	return filepath.Join("..", ".vrooli", "variant_space.json")
}

func loadBytes(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("failed to read variant space at %s: %v", path, err)
		return cloneBytes(DefaultJSON)
	}
	return cloneBytes(data)
}

// JSONBytes returns the original JSON payload for verbatim HTTP serving.
func (s *Space) JSONBytes() []byte {
	if s == nil || len(s.rawJSON) == 0 {
		return DefaultJSON
	}
	return s.rawJSON
}

// ValidateSelection ensures every axis has a valid value and no disallowed
// combination is chosen.
func (s *Space) ValidateSelection(selection map[string]string) error {
	if s == nil {
		return fmt.Errorf("variant space not initialized")
	}
	if len(s.Axes) == 0 {
		return fmt.Errorf("variant space has no axes defined")
	}

	for axisID := range selection {
		if _, ok := s.Axes[axisID]; !ok {
			return fmt.Errorf("unknown axis %s", axisID)
		}
	}

	for axisID, axisDef := range s.Axes {
		value, ok := selection[axisID]
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("axis %s is required", axisID)
		}
		if !axisDef.hasVariant(value) {
			return fmt.Errorf("invalid value '%s' for axis %s", value, axisID)
		}
	}

	if s.Constraints != nil {
		for _, combo := range s.Constraints.DisallowedCombinations {
			if len(combo) == 0 {
				continue
			}
			match := true
			for axisID, axisValue := range combo {
				value, ok := selection[axisID]
				if !ok || value != axisValue {
					match = false
					break
				}
			}
			if match {
				return fmt.Errorf("axis combination %v is disallowed", combo)
			}
		}
	}

	return nil
}

func (a *AxisDefinition) hasVariant(id string) bool {
	for _, v := range a.Variants {
		if v.ID == id {
			return true
		}
	}
	return false
}

func cloneBytes(data []byte) []byte {
	if data == nil {
		return nil
	}
	cloned := make([]byte, len(data))
	copy(cloned, data)
	return cloned
}
