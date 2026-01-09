package docs

import (
	"time"

	"test-genie/internal/shared"
)

// Settings holds configuration for docs validation loaded from .vrooli/testing.json (docs section).
type Settings struct {
	Markdown MarkdownSettings `json:"markdown"`
	Mermaid  MermaidSettings  `json:"mermaid"`
	Links    LinkSettings     `json:"links"`
	Paths    PathSettings     `json:"absolute_paths"`
}

type MarkdownSettings struct {
	// Enabled controls markdown validations (syntax fences, link extraction). Default: true.
	Enabled *bool `json:"enabled"`
}

type MermaidSettings struct {
	// Enabled controls mermaid validation. Default: true when mermaid code fences exist.
	Enabled *bool `json:"enabled"`
	// Strict controls whether mermaid parse errors fail the phase (default true).
	Strict *bool `json:"strict"`
}

type LinkSettings struct {
	// Enabled toggles link validation. Default: true.
	Enabled *bool `json:"enabled"`
	// Ignore lists link prefixes/globs to skip (e.g., "http://localhost:*").
	Ignore []string `json:"ignore"`
	// MaxConcurrency sets concurrent external link checks. Default: 6.
	MaxConcurrency int `json:"max_concurrency"`
	// TimeoutMs is per-request timeout for external link checks. Default: 5000ms.
	TimeoutMs int `json:"timeout_ms"`
	// StrictExternal fails on external timeouts/connection errors when true. Default: false.
	StrictExternal *bool `json:"strict_external"`
}

type PathSettings struct {
	// Enabled toggles absolute path detection. Default: true.
	Enabled *bool `json:"enabled"`
	// Allow lists absolute path prefixes that are permitted (e.g., "/api/").
	Allow []string `json:"allow"`
}

// LoadSettings reads the docs section from testing.json.
func LoadSettings(scenarioDir string) (*Settings, error) {
	settings := DefaultSettings()
	if err := shared.MergePhaseConfig(scenarioDir, "docs", settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// DefaultSettings returns sensible defaults.
func DefaultSettings() *Settings {
	return &Settings{
		Markdown: MarkdownSettings{},
		Mermaid: MermaidSettings{
			Enabled: boolPtr(true),
			Strict:  boolPtr(true),
		},
		Links: LinkSettings{
			Enabled:        boolPtr(true),
			MaxConcurrency: 6,
			TimeoutMs:      5000,
		},
		Paths: PathSettings{
			Enabled: boolPtr(true),
		},
	}
}

func (s *Settings) mermaidEnabled() bool {
	if s == nil || s.Mermaid.Enabled == nil {
		return true
	}
	return *s.Mermaid.Enabled
}

func (s *Settings) mermaidStrict() bool {
	if s == nil || s.Mermaid.Strict == nil {
		return true
	}
	return *s.Mermaid.Strict
}

func (s *Settings) linksEnabled() bool {
	if s == nil || s.Links.Enabled == nil {
		return true
	}
	return *s.Links.Enabled
}

func (s *Settings) linksTimeout() time.Duration {
	timeout := time.Duration(s.Links.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		return 5 * time.Second
	}
	return timeout
}

func (s *Settings) linksConcurrency() int {
	if s.Links.MaxConcurrency <= 0 {
		return 6
	}
	return s.Links.MaxConcurrency
}

func (s *Settings) linksStrictExternal() bool {
	if s == nil || s.Links.StrictExternal == nil {
		return false
	}
	return *s.Links.StrictExternal
}

func (s *Settings) pathsEnabled() bool {
	if s == nil || s.Paths.Enabled == nil {
		return true
	}
	return *s.Paths.Enabled
}

func (s *Settings) markdownEnabled() bool {
	if s == nil || s.Markdown.Enabled == nil {
		return true
	}
	return *s.Markdown.Enabled
}

func boolPtr(v bool) *bool {
	return &v
}
