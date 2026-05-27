package domains

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// APIInternalDir is the scenario-relative root the FolderExtractor scans
// for de-facto domain folders.
const APIInternalDir = "api/internal"

// defaultNonDomainFolders are api/internal/ subdirectories that are
// cross-cutting infrastructure, never product domains. The control surface
// (Phase 6) lets a scenario extend this set; the default mirrors the
// Non-Domains list every cartographer-shaped scenario shares.
var defaultNonDomainFolders = map[string]struct{}{
	"server":     {},
	"module":     {},
	"modules":    {},
	"database":   {},
	"testutil":   {},
	"middleware": {},
	"clock":      {},
	"git":        {},
	"httpc":      {},
	"httpx":      {},
}

// FolderExtractor derives de-facto domains from api/internal/<domain>/
// package folders. It is a lower ladder rung used for convergence against
// the authoritative source, and the zero-config authority for scenarios
// that ship no DOMAINS.md.
type FolderExtractor struct {
	// nonDomains is the set of folder names treated as infrastructure. When
	// nil the package default is used.
	nonDomains map[string]struct{}
}

// NewFolderExtractor returns the production folder extractor using the
// default non-domain set.
func NewFolderExtractor() *FolderExtractor { return &FolderExtractor{} }

// NewFolderExtractorWithExemptions returns a folder extractor that treats
// the given folder names as non-domains in addition to the defaults.
func NewFolderExtractorWithExemptions(extra []string) *FolderExtractor {
	set := make(map[string]struct{}, len(defaultNonDomainFolders)+len(extra))
	for k := range defaultNonDomainFolders {
		set[k] = struct{}{}
	}
	for _, name := range extra {
		set[name] = struct{}{}
	}
	return &FolderExtractor{nonDomains: set}
}

// Source identifies this rung.
func (*FolderExtractor) Source() Source { return SourceAPIFolders }

// APIHandlersDir is the second scenario-relative root the FolderExtractor
// scans. A handler-only domain (e.g., health) lives here without an
// api/internal package, and must still count as an implemented domain.
const APIHandlersDir = "api/handlers"

// Extract scans api/internal/ and api/handlers/ for domain folders and
// unions their names. A scenario with neither directory returns an empty
// extraction. Folders in the non-domain set (infrastructure) are skipped.
func (e *FolderExtractor) Extract(_ context.Context, scenarioDir string) (Extraction, error) {
	exempt := e.nonDomains
	if exempt == nil {
		exempt = defaultNonDomainFolders
	}

	names := map[string]struct{}{}
	for _, root := range []string{APIInternalDir, APIHandlersDir} {
		entries, err := os.ReadDir(filepath.Join(scenarioDir, root))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return Extraction{}, fmt.Errorf("read %s: %w", root, err)
		}
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			name := ent.Name()
			if _, skip := exempt[name]; skip {
				continue
			}
			names[name] = struct{}{}
		}
	}

	out := Extraction{Source: SourceAPIFolders}
	for name := range names {
		out.Domains = append(out.Domains, ExtractedDomain{
			Name:  name,
			Paths: folderPaths(scenarioDir, name),
		})
	}
	sort.Slice(out.Domains, func(i, j int) bool { return out.Domains[i].Name < out.Domains[j].Name })
	return out, nil
}

// folderPaths returns the repo-relative path prefixes a folder-derived
// domain owns: whichever of its api/internal/ and api/handlers/ packages
// exist on disk.
func folderPaths(scenarioDir, name string) []string {
	var paths []string
	for _, root := range []string{APIInternalDir, APIHandlersDir} {
		full := filepath.Join(scenarioDir, root, name)
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			paths = append(paths, root+"/"+name+"/")
		}
	}
	return paths
}
