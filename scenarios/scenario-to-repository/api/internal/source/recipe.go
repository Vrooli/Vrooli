package source

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

type Snapshot struct {
	Scenario     string    `json:"scenario"`
	SourceDigest string    `json:"sourceDigest"`
	CapturedAt   time.Time `json:"capturedAt"`
	Closure      Closure   `json:"closure"`
}

func ParseRecipe(data []byte) (Recipe, string, error) {
	var recipe Recipe
	if err := yaml.Unmarshal(data, &recipe); err != nil {
		return Recipe{}, "", fmt.Errorf("parse export recipe: %w", err)
	}
	if err := recipe.Validate(); err != nil {
		return Recipe{}, "", err
	}
	digest, err := recipeDigestCanonical(recipe)
	if err != nil {
		return Recipe{}, "", err
	}
	return recipe, digest, nil
}

func (r Recipe) Validate() error {
	if r.SchemaVersion != 1 {
		return fmt.Errorf("unsupported recipe schema version %d", r.SchemaVersion)
	}
	if r.Scenario == "" {
		return fmt.Errorf("recipe scenario is required")
	}
	if r.Mode != "buildable_source" {
		return fmt.Errorf("unsupported recipe mode %q", r.Mode)
	}
	if r.ArchiveFormat != "tar_gzip" || !r.Deterministic {
		return fmt.Errorf("recipe must request deterministic tar_gzip")
	}
	return nil
}

func recipeDigestCanonical(recipe Recipe) (string, error) {
	data, err := json.Marshal(recipe)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

// CaptureSnapshot hashes the admitted closure twice around a stat/read pass.
// A changed source is refused instead of producing a mutable snapshot that
// only looks immutable to downstream approval.
func CaptureSnapshot(root, scenario string) (Snapshot, error) {
	first, err := AnalyzeClosure(root, scenario, "")
	if err != nil {
		return Snapshot{}, err
	}
	first.SourceDigest = digestFiles(first.Files)
	second, err := AnalyzeClosure(root, scenario, "")
	if err != nil {
		return Snapshot{}, err
	}
	second.SourceDigest = digestFiles(second.Files)
	if first.SourceDigest != second.SourceDigest {
		return Snapshot{}, fmt.Errorf("source changed during capture")
	}
	first.SourceDigest = second.SourceDigest
	first.ClosureDigest = second.ClosureDigest
	return Snapshot{Scenario: scenario, SourceDigest: first.SourceDigest, CapturedAt: time.Now().UTC(), Closure: first}, nil
}

func digestFiles(files []File) string {
	ordered := append([]File(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].SourcePath < ordered[j].SourcePath })
	data, _ := json.Marshal(ordered)
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}

func ReadRecipeFile(path string) (Recipe, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Recipe{}, "", err
	}
	return ParseRecipe(data)
}
