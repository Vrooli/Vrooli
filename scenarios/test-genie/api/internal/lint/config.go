package lint

import (
	"test-genie/internal/shared"
)

// Settings holds the configuration for lint validation, loaded from
// .vrooli/testing.json. It specifies which languages to lint and how
// strict to be about findings.
type Settings struct {
	// Go configures Go linting.
	Go LanguageSettings

	// Node configures TypeScript/JavaScript linting.
	Node LanguageSettings

	// Python configures Python linting.
	Python LanguageSettings
}

// LanguageSettings configures linting for a specific language.
type LanguageSettings struct {
	// Enabled controls whether this language is linted. Default: true (if detected)
	Enabled *bool

	// Strict treats all lint warnings as errors (fails the phase). Default: false
	Strict bool
}

// IsEnabled returns true if the language is enabled (default true if not specified).
func (s LanguageSettings) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// configSection represents the lint section of .vrooli/testing.json.
type configSection struct {
	Languages languagesConfig `json:"languages"`
}

type languagesConfig struct {
	Go     *languageConfig `json:"go"`
	Node   *languageConfig `json:"node"`
	Python *languageConfig `json:"python"`
}

type languageConfig struct {
	Enabled *bool `json:"enabled"`
	Strict  *bool `json:"strict"`
}

// LoadSettings reads lint validation settings from .vrooli/testing.json.
// If the file doesn't exist or the lint section is missing, default settings are returned.
func LoadSettings(scenarioDir string) (*Settings, error) {
	settings := DefaultSettings()

	section, err := shared.LoadPhaseConfig(scenarioDir, "lint", configSection{})
	if err != nil {
		return nil, err
	}

	// Apply Go settings
	if section.Languages.Go != nil {
		settings.Go.Enabled = section.Languages.Go.Enabled
		if section.Languages.Go.Strict != nil {
			settings.Go.Strict = *section.Languages.Go.Strict
		}
	}

	// Apply Node settings
	if section.Languages.Node != nil {
		settings.Node.Enabled = section.Languages.Node.Enabled
		if section.Languages.Node.Strict != nil {
			settings.Node.Strict = *section.Languages.Node.Strict
		}
	}

	// Apply Python settings
	if section.Languages.Python != nil {
		settings.Python.Enabled = section.Languages.Python.Enabled
		if section.Languages.Python.Strict != nil {
			settings.Python.Strict = *section.Languages.Python.Strict
		}
	}

	return settings, nil
}

// DefaultSettings returns the default lint validation settings.
// By default, all detected languages are linted with non-strict mode
// (lint warnings pass, only type errors fail).
func DefaultSettings() *Settings {
	return &Settings{
		Go:     LanguageSettings{},
		Node:   LanguageSettings{},
		Python: LanguageSettings{},
	}
}
