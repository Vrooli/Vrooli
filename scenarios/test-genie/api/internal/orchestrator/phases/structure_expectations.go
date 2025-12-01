package phases

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type structureExpectations struct {
	AdditionalDirs      []string
	AdditionalFiles     []string
	ExcludedDirs        []string
	ExcludedFiles       []string
	ValidateServiceName bool
	ValidateJSONFiles   bool
}

type structureConfigDocument struct {
	Structure structureConfigSection `json:"structure"`
}

type structureConfigSection struct {
	AdditionalDirs  []structurePathEntry     `json:"additional_dirs"`
	AdditionalFiles []structurePathEntry     `json:"additional_files"`
	ExcludeDirs     []structurePathEntry     `json:"exclude_dirs"`
	ExcludeFiles    []structurePathEntry     `json:"exclude_files"`
	Validations     structureValidationFlags `json:"validations"`
}

type structureValidationFlags struct {
	ServiceNameMatchesDirectory *bool `json:"service_json_name_matches_directory"`
	CheckJSONValidity           *bool `json:"check_json_validity"`
}

type structurePathEntry struct {
	Path string
}

func (e *structurePathEntry) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		return json.Unmarshal(data, &e.Path)
	}
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	e.Path = payload.Path
	return nil
}

func loadStructureExpectations(scenarioDir string) (*structureExpectations, error) {
	cfg := defaultStructureExpectations()
	configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", configPath, err)
	}
	var doc structureConfigDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", configPath, err)
	}

	cfg.AdditionalDirs = entriesToPaths(doc.Structure.AdditionalDirs)
	cfg.AdditionalFiles = entriesToPaths(doc.Structure.AdditionalFiles)
	cfg.ExcludedDirs = entriesToPaths(doc.Structure.ExcludeDirs)
	cfg.ExcludedFiles = entriesToPaths(doc.Structure.ExcludeFiles)

	if doc.Structure.Validations.ServiceNameMatchesDirectory != nil {
		cfg.ValidateServiceName = *doc.Structure.Validations.ServiceNameMatchesDirectory
	}
	if doc.Structure.Validations.CheckJSONValidity != nil {
		cfg.ValidateJSONFiles = *doc.Structure.Validations.CheckJSONValidity
	}
	return cfg, nil
}

func defaultStructureExpectations() *structureExpectations {
	return &structureExpectations{
		ValidateServiceName: true,
		ValidateJSONFiles:   true,
	}
}

func entriesToPaths(entries []structurePathEntry) []string {
	var paths []string
	for _, entry := range entries {
		if clean := canonicalStructurePath(entry.Path); clean != "" {
			paths = append(paths, clean)
		}
	}
	return paths
}
