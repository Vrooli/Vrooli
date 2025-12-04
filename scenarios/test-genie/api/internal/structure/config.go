package structure

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Expectations holds the configuration for structure validation, loaded from
// .vrooli/testing.json. It specifies which paths to check and which validations
// to perform.
type Expectations struct {
	// AdditionalDirs lists extra directories that must exist beyond the standard set.
	AdditionalDirs []string

	// AdditionalFiles lists extra files that must exist beyond the standard set.
	AdditionalFiles []string

	// ExcludedDirs lists directories to skip from the standard required set.
	ExcludedDirs []string

	// ExcludedFiles lists files to skip from the standard required set.
	ExcludedFiles []string

	// ValidateServiceName controls whether service.json name must match the scenario directory.
	ValidateServiceName bool
}

// configDocument represents the structure of .vrooli/testing.json.
type configDocument struct {
	Structure configSection `json:"structure"`
}

type configSection struct {
	AdditionalDirs  []pathEntry     `json:"additional_dirs"`
	AdditionalFiles []pathEntry     `json:"additional_files"`
	ExcludeDirs     []pathEntry     `json:"exclude_dirs"`
	ExcludeFiles    []pathEntry     `json:"exclude_files"`
	Validations     validationFlags `json:"validations"`
}

type validationFlags struct {
	ServiceNameMatchesDirectory *bool `json:"service_json_name_matches_directory"`
}

// pathEntry supports both string and object forms in JSON:
//
//	"path/to/dir"
//	{"path": "path/to/dir"}
type pathEntry struct {
	Path string
}

func (e *pathEntry) UnmarshalJSON(data []byte) error {
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

// LoadExpectations reads structure validation expectations from .vrooli/testing.json.
// If the file doesn't exist, default expectations are returned.
func LoadExpectations(scenarioDir string) (*Expectations, error) {
	exp := DefaultExpectations()
	configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return exp, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", configPath, err)
	}

	var doc configDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", configPath, err)
	}

	exp.AdditionalDirs = entriesToPaths(doc.Structure.AdditionalDirs)
	exp.AdditionalFiles = entriesToPaths(doc.Structure.AdditionalFiles)
	exp.ExcludedDirs = entriesToPaths(doc.Structure.ExcludeDirs)
	exp.ExcludedFiles = entriesToPaths(doc.Structure.ExcludeFiles)

	if doc.Structure.Validations.ServiceNameMatchesDirectory != nil {
		exp.ValidateServiceName = *doc.Structure.Validations.ServiceNameMatchesDirectory
	}

	return exp, nil
}

// DefaultExpectations returns the default structure validation expectations.
func DefaultExpectations() *Expectations {
	return &Expectations{
		ValidateServiceName: true,
	}
}

// entriesToPaths extracts and canonicalizes paths from path entries.
func entriesToPaths(entries []pathEntry) []string {
	var paths []string
	for _, entry := range entries {
		if clean := canonicalizePath(entry.Path); clean != "" {
			paths = append(paths, clean)
		}
	}
	return paths
}

// canonicalizePath normalizes a path for consistent comparison.
func canonicalizePath(path string) string {
	clean := filepath.Clean(path)
	clean = strings.TrimPrefix(clean, "./")
	if clean == "." {
		return ""
	}
	return filepath.ToSlash(clean)
}
