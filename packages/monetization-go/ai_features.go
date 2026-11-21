package monetization

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const aiFeatureManifestPath = ".vrooli/monetization.json"

var aiFeatureIDPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)

// AIFeature is the scenario-owned declaration of one bounded AI action. The
// generated registry contains its limits and schema, never prompt text.
type AIFeature struct {
	ID                      string          `json:"id"`
	Role                    string          `json:"role"`
	Profile                 string          `json:"profile"`
	MaxInputBytes           int             `json:"max_input_bytes"`
	MaxOutputBytes          int             `json:"max_output_bytes"`
	MaxUserTurns            int             `json:"max_user_turns"`
	Schema                  json.RawMessage `json:"schema"`
	PinnedInstructionDigest string          `json:"pinned_instruction_digest,omitempty"`
}

// AIFeatureRegistryEntry binds a scenario-owned contract to its stable feature
// identifier. Gateway code consumes this generated view, not scenario files at
// request time.
type AIFeatureRegistryEntry struct {
	Scenario string    `json:"scenario"`
	Feature  AIFeature `json:"feature"`
}

type AIFeatureRegistry struct {
	Version  int                      `json:"version"`
	Features []AIFeatureRegistryEntry `json:"features"`
}

type aiFeatureDeclaration struct {
	AIFeatures json.RawMessage `json:"ai_features"`
}

// BuildAIFeatureRegistry reads declarations from each direct child of
// <root>/scenarios. It validates and sorts the result so generation is stable.
func BuildAIFeatureRegistry(root string) (AIFeatureRegistry, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" {
		return AIFeatureRegistry{}, errors.New("repository root is required")
	}
	scenariosRoot := filepath.Join(root, "scenarios")
	children, err := os.ReadDir(scenariosRoot)
	if err != nil {
		return AIFeatureRegistry{}, fmt.Errorf("read scenarios directory: %w", err)
	}
	registry := AIFeatureRegistry{Version: 1, Features: []AIFeatureRegistryEntry{}}
	seen := map[string]string{}
	for _, child := range children {
		if !child.IsDir() {
			continue
		}
		scenario := child.Name()
		manifest := filepath.Join(scenariosRoot, scenario, aiFeatureManifestPath)
		features, found, err := readScenarioAIFeatures(manifest)
		if err != nil {
			return AIFeatureRegistry{}, err
		}
		if !found {
			continue
		}
		if err := appendScenarioAIFeatures(&registry, seen, scenario, manifest, features); err != nil {
			return AIFeatureRegistry{}, err
		}
	}
	sort.Slice(registry.Features, func(i, j int) bool {
		left, right := registry.Features[i], registry.Features[j]
		if left.Scenario != right.Scenario {
			return left.Scenario < right.Scenario
		}
		return left.Feature.ID < right.Feature.ID
	})
	return registry, nil
}

func readScenarioAIFeatures(manifest string) ([]json.RawMessage, bool, error) {
	data, err := os.ReadFile(manifest)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", manifest, err)
	}
	var declaration aiFeatureDeclaration
	if err := json.Unmarshal(data, &declaration); err != nil {
		return nil, false, fmt.Errorf("parse %s: %w", manifest, err)
	}
	var features []json.RawMessage
	if len(bytes.TrimSpace(declaration.AIFeatures)) > 0 {
		if err := json.Unmarshal(declaration.AIFeatures, &features); err != nil {
			return nil, false, fmt.Errorf("%s ai_features must be an array: %w", manifest, err)
		}
	}
	return features, true, nil
}

func appendScenarioAIFeatures(registry *AIFeatureRegistry, seen map[string]string, scenario, manifest string, features []json.RawMessage) error {
	for index, rawFeature := range features {
		feature, err := decodeAIFeature(rawFeature)
		if err != nil {
			return fmt.Errorf("%s ai_features[%d]: %w", manifest, index, err)
		}
		if err := validateAIFeature(feature); err != nil {
			return fmt.Errorf("%s ai_features[%d]: %w", manifest, index, err)
		}
		if previous, ok := seen[feature.ID]; ok {
			return fmt.Errorf("duplicate AI feature %q in %s and %s", feature.ID, previous, manifest)
		}
		seen[feature.ID] = manifest
		registry.Features = append(registry.Features, AIFeatureRegistryEntry{Scenario: scenario, Feature: feature})
	}
	return nil
}

func decodeAIFeature(raw []byte) (AIFeature, error) {
	var feature AIFeature
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&feature); err != nil {
		return AIFeature{}, err
	}
	return feature, nil
}

func validateAIFeature(feature AIFeature) error {
	if err := validateAIFeatureIdentity(feature); err != nil {
		return err
	}
	if err := validateAIFeatureLimits(feature); err != nil {
		return err
	}
	if err := validateAIFeatureSchema(feature.Schema); err != nil {
		return err
	}
	return validateAIFeatureDigest(feature.PinnedInstructionDigest)
}

func validateAIFeatureIdentity(feature AIFeature) error {
	if !aiFeatureIDPattern.MatchString(feature.ID) {
		return errors.New("id must match ^[a-z][a-z0-9._-]*$")
	}
	if strings.TrimSpace(feature.Role) == "" || feature.Role != strings.TrimSpace(feature.Role) {
		return errors.New("role must be non-empty and trimmed")
	}
	if strings.TrimSpace(feature.Profile) == "" || feature.Profile != strings.TrimSpace(feature.Profile) {
		return errors.New("profile must be non-empty and trimmed")
	}
	return nil
}

func validateAIFeatureLimits(feature AIFeature) error {
	if feature.MaxInputBytes <= 0 || feature.MaxOutputBytes <= 0 {
		return errors.New("max_input_bytes and max_output_bytes must be positive")
	}
	if feature.MaxUserTurns != 1 {
		return errors.New("max_user_turns must be exactly 1")
	}
	return nil
}

func validateAIFeatureSchema(raw json.RawMessage) error {
	if len(bytes.TrimSpace(raw)) == 0 || !json.Valid(raw) {
		return errors.New("schema must contain valid JSON")
	}
	var schema map[string]json.RawMessage
	if err := json.Unmarshal(raw, &schema); err != nil || schema == nil {
		return errors.New("schema must be a JSON object")
	}
	return nil
}

func validateAIFeatureDigest(value string) error {
	if digest := strings.TrimSpace(value); digest != "" && !isSHA256(digest) {
		return errors.New("pinned_instruction_digest must be a lowercase SHA-256 hex digest")
	}
	return nil
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

// RenderAIFeatureRegistry returns canonical bytes suitable for a committed
// generated registry. Prompt text is not part of the declaration or output.
func RenderAIFeatureRegistry(registry AIFeatureRegistry) ([]byte, error) {
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// CheckAIFeatureRegistry compares generated output with the committed artifact.
func CheckAIFeatureRegistry(root, artifactPath string) error {
	registry, err := BuildAIFeatureRegistry(root)
	if err != nil {
		return err
	}
	want, err := RenderAIFeatureRegistry(registry)
	if err != nil {
		return err
	}
	got, err := os.ReadFile(artifactPath)
	if err != nil {
		return err
	}
	if !bytes.Equal(got, want) {
		return errors.New("AI feature registry is stale; run go run ./packages/monetization-go/cmd/ai-feature-registry")
	}
	return nil
}
