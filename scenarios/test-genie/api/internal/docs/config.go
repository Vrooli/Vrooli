package docs

import (
	"time"

	repocontract "github.com/vrooli/repo-contract-go"

	"test-genie/internal/shared"
)

// defaultManifestRel returns the canonical scenario-relative path for the
// docs manifest, resolved through the repo contract. An empty repoRoot
// argument causes the helper to fall back to the contract's canonical
// default, which is the desired behavior for callers that need a static
// default before any repo root is known.
func defaultManifestRel() string {
	rel, _ := repocontract.ScenarioDocsManifestRel("")
	return rel
}

// DOC: docs/phases/docs/README.md#configuration
// Settings holds configuration for docs validation loaded from .vrooli/testing.json (docs section).
type Settings struct {
	Markdown   MarkdownSettings  `json:"markdown"`
	Mermaid    MermaidSettings   `json:"mermaid"`
	Links      LinkSettings      `json:"links"`
	Paths      PathSettings      `json:"absolute_paths"`
	ScanPaths  ScanPathSettings  `json:"paths"`
	References *ReferencesConfig `json:"references"`
	Manifest   *ManifestConfig   `json:"manifest"`
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

// ScanPathSettings controls filesystem traversal filters for docs validation.
type ScanPathSettings struct {
	// ExcludeDirs skips directories by name or by relative path prefix.
	ExcludeDirs []string `json:"exclude_dirs"`
	// ExcludeGlobs skips files/dirs by scenario-relative glob. Supports **.
	ExcludeGlobs []string `json:"exclude_globs"`
}

// ReferencesConfig controls bidirectional code↔documentation reference validation.
type ReferencesConfig struct {
	// Enabled toggles reference validation. Default: true.
	Enabled *bool `json:"enabled"`
	// ValidateCodeRefs checks [CODE: ...] references in docs point to valid files. Default: true.
	ValidateCodeRefs *bool `json:"validate_code_refs"`
	// ValidateDocRefs checks // DOC: comments in code point to valid docs. Default: true.
	ValidateDocRefs *bool `json:"validate_doc_refs"`
	// ValidateMarkedRefs checks marked path/doc inline refs in docs. Default: true.
	ValidateMarkedRefs *bool `json:"validate_marked_refs"`
	// CodeExtensions lists file extensions to scan for DOC: comments.
	CodeExtensions []string `json:"code_extensions"`
	// Strict fails on broken references (default: false = warnings only).
	Strict *bool `json:"strict"`
	// SkipDirs lists additional directories to skip when scanning code files.
	SkipDirs []string `json:"skip_dirs"`
}

// ManifestConfig controls docs manifest coverage tracking.
type ManifestConfig struct {
	// Enabled toggles manifest coverage checking. Default: false.
	Enabled *bool `json:"enabled"`
	// RequireAllDocsRegistered warns when docs exist but aren't in manifest. Default: false.
	RequireAllDocsRegistered *bool `json:"require_all_docs_registered"`
	// ManifestPath is the path to the manifest file relative to scenario dir.
	ManifestPath string `json:"manifest_path"`
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
		References: &ReferencesConfig{
			Enabled:            boolPtr(true),
			ValidateCodeRefs:   boolPtr(true),
			ValidateDocRefs:    boolPtr(true),
			ValidateMarkedRefs: boolPtr(true),
			CodeExtensions:     []string{".ts", ".tsx", ".js", ".jsx", ".go", ".py", ".rs", ".java", ".kt"},
			Strict:             boolPtr(false),
			SkipDirs:           nil,
		},
		Manifest: &ManifestConfig{
			Enabled:                  boolPtr(false),
			RequireAllDocsRegistered: boolPtr(false),
			ManifestPath:             defaultManifestRel(),
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

func (s *Settings) referencesEnabled() bool {
	if s == nil || s.References == nil || s.References.Enabled == nil {
		return true
	}
	return *s.References.Enabled
}

func (s *Settings) codeRefsEnabled() bool {
	if s == nil || s.References == nil || s.References.ValidateCodeRefs == nil {
		return true
	}
	return *s.References.ValidateCodeRefs
}

func (s *Settings) docRefsEnabled() bool {
	if s == nil || s.References == nil || s.References.ValidateDocRefs == nil {
		return true
	}
	return *s.References.ValidateDocRefs
}

func (s *Settings) markedRefsEnabled() bool {
	if s == nil || s.References == nil || s.References.ValidateMarkedRefs == nil {
		return true
	}
	return *s.References.ValidateMarkedRefs
}

func (s *Settings) referencesStrict() bool {
	if s == nil || s.References == nil || s.References.Strict == nil {
		return false
	}
	return *s.References.Strict
}

func (s *Settings) codeExtensions() []string {
	if s == nil || s.References == nil || len(s.References.CodeExtensions) == 0 {
		return []string{".ts", ".tsx", ".js", ".jsx", ".go", ".py", ".rs", ".java", ".kt"}
	}
	return s.References.CodeExtensions
}

func (s *Settings) referencesSkipDirs() []string {
	if s == nil || s.References == nil {
		return nil
	}
	return s.References.SkipDirs
}

func (s *Settings) scanExcludeDirs() []string {
	if s == nil {
		return nil
	}
	return s.ScanPaths.ExcludeDirs
}

func (s *Settings) scanExcludeGlobs() []string {
	if s == nil {
		return nil
	}
	return s.ScanPaths.ExcludeGlobs
}

func (s *Settings) manifestEnabled() bool {
	if s == nil || s.Manifest == nil || s.Manifest.Enabled == nil {
		return false
	}
	return *s.Manifest.Enabled
}

func (s *Settings) manifestRequireAll() bool {
	if s == nil || s.Manifest == nil || s.Manifest.RequireAllDocsRegistered == nil {
		return false
	}
	return *s.Manifest.RequireAllDocsRegistered
}

func (s *Settings) manifestPath() string {
	if s == nil || s.Manifest == nil || s.Manifest.ManifestPath == "" {
		return defaultManifestRel()
	}
	return s.Manifest.ManifestPath
}

func boolPtr(v bool) *bool {
	return &v
}
