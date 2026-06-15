// Package pagediscovery is test-genie's single source of truth for enumerating
// the UI pages of a scenario. It reads the scenario's
// .vrooli/lighthouse.json page set and falls back to a single home page when no
// configuration exists, so the smoke phase's all-pages visual capture and any
// future per-page validation share one discovery contract.
//
// This logic was previously duplicated in git-control-tower
// (discoverPagesWithMethod). test-genie owns it now; GCT consumes test-genie's
// run artifacts instead of re-discovering pages.
package pagediscovery

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Method names how a page set was resolved, mirroring the historical GCT
// contract so existing consumers and docs stay aligned.
const (
	// MethodLighthouse means the pages came from an enabled .vrooli/lighthouse.json.
	MethodLighthouse = "lighthouse"
	// MethodFallback means no usable config existed; a single home page is used.
	MethodFallback = "fallback"
	// MethodExplicit means the caller supplied the page paths directly.
	MethodExplicit = "explicit"
)

// Page is one discoverable UI surface of a scenario.
type Page struct {
	// ID is the optional stable identifier from lighthouse.json.
	ID string `json:"id,omitempty"`
	// Path is the route relative to the scenario UI root (e.g. "/", "/backlog").
	Path string `json:"path"`
	// Label is a human-readable name for the page.
	Label string `json:"label,omitempty"`
	// WaitForSelector, when set, is a CSS selector a capturer may wait on before
	// recording the page (carried through for capture producers).
	WaitForSelector string `json:"waitForSelector,omitempty"`
}

// lighthouseConfig is the on-disk shape of .vrooli/lighthouse.json.
type lighthouseConfig struct {
	Enabled bool   `json:"enabled"`
	Pages   []Page `json:"pages"`
}

// FileReader reads a file's bytes. It is the filesystem seam for page discovery
// so tests can supply page configs without touching disk.
//
// seam: FileReader is the page-discovery filesystem seam. Production wires
// OSFileReader (os.ReadFile); tests wire pagediscovery.FakeFileReader (fake.go).
type FileReader interface {
	ReadFile(path string) ([]byte, error)
}

// OSFileReader reads files from the real filesystem.
type OSFileReader struct{}

// ReadFile delegates to os.ReadFile.
func (OSFileReader) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

// Discoverer resolves a scenario's page set from its directory.
type Discoverer struct {
	fs FileReader
}

// New returns a Discoverer reading from the given FileReader; nil uses the real
// filesystem.
func New(fs FileReader) *Discoverer {
	if fs == nil {
		fs = OSFileReader{}
	}
	return &Discoverer{fs: fs}
}

// homePage is the single-page fallback used when no lighthouse config applies.
func homePage() Page { return Page{Path: "/", Label: "Home"} }

// Discover resolves the page set for the scenario rooted at scenarioDir and
// reports how it was resolved. When explicitPaths is non-empty those pages are
// used verbatim (MethodExplicit). Otherwise an enabled .vrooli/lighthouse.json
// with at least one page wins (MethodLighthouse); failing that a single home
// page is returned (MethodFallback). Discover never returns an empty slice.
func (d *Discoverer) Discover(scenarioDir string, explicitPaths []string) ([]Page, string) {
	if len(explicitPaths) > 0 {
		pages := make([]Page, 0, len(explicitPaths))
		for _, p := range explicitPaths {
			pages = append(pages, Page{Path: p, Label: p})
		}
		return pages, MethodExplicit
	}

	path := filepath.Join(scenarioDir, ".vrooli", "lighthouse.json")
	data, err := d.fs.ReadFile(path)
	if err == nil {
		var cfg lighthouseConfig
		if jsonErr := json.Unmarshal(data, &cfg); jsonErr == nil && cfg.Enabled && len(cfg.Pages) > 0 {
			return cfg.Pages, MethodLighthouse
		}
	}

	return []Page{homePage()}, MethodFallback
}
