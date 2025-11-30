package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var variantSpaceBytes = readVariantSpaceFile()
var defaultVariantSpace = mustLoadVariantSpace()

type VariantSpace struct {
	Name            string                            `json:"_name"`
	SchemaVersion   int                               `json:"_schemaVersion"`
	Note            string                            `json:"_note,omitempty"`
	AgentGuidelines []string                          `json:"_agentGuidelines,omitempty"`
	Axes            map[string]*AxisDefinition        `json:"axes"`
	Constraints     *VariantSpaceConstraints          `json:"constraints,omitempty"`
	rawJSON         json.RawMessage                   `json:"-"`
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
	data := readVariantSpaceFile()

	var space VariantSpace
	if err := json.Unmarshal(data, &space); err != nil {
		panic(fmt.Sprintf("failed to parse variant space: %v", err))
	}
	space.rawJSON = make([]byte, len(data))
	copy(space.rawJSON, data)
	return &space
}

func readVariantSpaceFile() []byte {
	path := filepath.Join("..", ".vrooli", "variant_space.json")
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("failed to read variant space file %s: %v", path, err))
	}
	return data
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
	if len(vs.Axes) == 0 {
		return fmt.Errorf("variant space has no axes defined")
	}

	// Check unknown axes
	for axisID := range selection {
		if _, ok := vs.Axes[axisID]; !ok {
			return fmt.Errorf("unknown axis %s", axisID)
		}
	}

	// Ensure each axis has a value and that value exists.
	for axisID, axisDef := range vs.Axes {
		value, ok := selection[axisID]
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("axis %s is required", axisID)
		}
		if !axisDef.hasVariant(value) {
			return fmt.Errorf("invalid value '%s' for axis %s", value, axisID)
		}
	}

	// Verify disallowed combos.
	if vs.Constraints != nil {
		for _, combo := range vs.Constraints.DisallowedCombinations {
			if combo == nil || len(combo) == 0 {
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
