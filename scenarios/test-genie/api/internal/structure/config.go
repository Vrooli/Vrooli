package structure

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"

	"test-genie/internal/shared"
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

	// ValidatePlaybooks controls whether playbooks structure is validated.
	// When true (default), bas/ is checked for proper structure.
	ValidatePlaybooks bool

	// PlaybooksStrict controls whether playbooks issues block the structure phase.
	// When false (default), issues are reported as informational warnings.
	// When true, issues cause the structure phase to fail.
	PlaybooksStrict bool
}

// configSection represents the structure section of .vrooli/testing.json.
type configSection struct {
	AdditionalDirs  []pathEntry     `json:"additional_dirs"`
	AdditionalFiles []pathEntry     `json:"additional_files"`
	ExcludeDirs     []pathEntry     `json:"exclude_dirs"`
	ExcludeFiles    []pathEntry     `json:"exclude_files"`
	Validations     validationFlags `json:"validations"`
	Playbooks       playbooksConfig `json:"playbooks"`
}

type validationFlags struct {
	ServiceNameMatchesDirectory *bool `json:"service_json_name_matches_directory"`
}

type playbooksConfig struct {
	Enabled *bool `json:"enabled"`
	Strict  *bool `json:"strict"`
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

	section, err := shared.LoadPhaseConfig(scenarioDir, "structure", configSection{})
	if err != nil {
		return nil, err
	}

	exp.AdditionalDirs = entriesToPaths(section.AdditionalDirs)
	exp.AdditionalFiles = entriesToPaths(section.AdditionalFiles)
	exp.ExcludedDirs = entriesToPaths(section.ExcludeDirs)
	exp.ExcludedFiles = entriesToPaths(section.ExcludeFiles)

	if section.Validations.ServiceNameMatchesDirectory != nil {
		exp.ValidateServiceName = *section.Validations.ServiceNameMatchesDirectory
	}

	if section.Playbooks.Enabled != nil {
		exp.ValidatePlaybooks = *section.Playbooks.Enabled
	}
	if section.Playbooks.Strict != nil {
		exp.PlaybooksStrict = *section.Playbooks.Strict
	}

	return exp, nil
}

// DefaultExpectations returns the default structure validation expectations.
func DefaultExpectations() *Expectations {
	return &Expectations{
		ValidateServiceName: true,
		ValidatePlaybooks:   true,
		PlaybooksStrict:     false,
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
