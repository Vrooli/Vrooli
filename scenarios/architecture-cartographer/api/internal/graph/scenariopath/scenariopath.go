// Package scenariopath resolves a target scenario's language-specific
// project directory — the directory a code-graph producer (Go or
// TypeScript) should be pointed at.
//
// Cartographer is invoked with a scenario *name* (for example
// "web-console"), but the code-graph producers require an absolute
// filesystem path to the project they parse, and that project usually
// lives in a subdirectory (ui/ for TypeScript, api/ or cli/ for Go),
// not at the scenario root. This package bridges that gap with a
// path-first, marker-verified probe: it walks an ordered list of
// candidate subdirectories under <repoRoot>/scenarios/<name> and returns
// the first one that actually contains the language's marker file
// (tsconfig.json / go.mod). "Detect what is really there" rather than
// assume — a scenario with a non-standard stack simply yields no match.
package scenariopath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Candidate is one probe: look for Marker inside Subdir (relative to the
// scenario root). Subdir "." means the scenario root itself.
type Candidate struct {
	Subdir string
	Marker string
}

// Resolver probes a fixed, ordered candidate list under a repo root.
// It performs a fresh filesystem check on every call (no caching) so it
// stays correct as projects are added or removed between extractions.
type Resolver struct {
	repoRoot   string
	candidates []Candidate
}

// NewResolver builds a Resolver. repoRoot is the absolute repository
// root; candidates are probed in order and the first match wins.
func NewResolver(repoRoot string, candidates []Candidate) *Resolver {
	return &Resolver{repoRoot: repoRoot, candidates: candidates}
}

// Resolve returns the absolute path of the first candidate subdirectory
// under <repoRoot>/scenarios/<scenarioName> whose marker file exists.
//
// found == false with a nil error means no candidate matched — the
// scenario has no project of this language at any known location, which
// callers should treat as "nothing to extract" rather than an error.
func (r *Resolver) Resolve(scenarioName string) (path string, found bool, err error) {
	name := strings.TrimSpace(scenarioName)
	if name == "" {
		return "", false, fmt.Errorf("scenariopath: scenario name is required")
	}
	if strings.TrimSpace(r.repoRoot) == "" {
		return "", false, fmt.Errorf("scenariopath: repo root is not configured")
	}
	scenarioRoot := filepath.Join(r.repoRoot, "scenarios", name)
	if name == "control-plane" {
		scenarioRoot = r.repoRoot
	}
	for _, c := range r.candidates {
		dir := filepath.Join(scenarioRoot, c.Subdir)
		if info, statErr := os.Stat(filepath.Join(dir, c.Marker)); statErr == nil && !info.IsDir() {
			return filepath.Clean(dir), true, nil
		}
	}
	return "", false, nil
}
