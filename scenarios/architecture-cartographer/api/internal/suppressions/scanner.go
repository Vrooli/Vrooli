package suppressions

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// scannableExtensions are the source file types the FileScanner reads for
// markers. Markers live in real source, not generated or vendored trees.
var scannableExtensions = map[string]struct{}{
	".go": {}, ".ts": {}, ".tsx": {}, ".js": {}, ".jsx": {}, ".py": {}, ".sh": {}, ".sql": {},
}

// skipDirs are directories the scanner never descends into.
var skipDirs = map[string]struct{}{
	"node_modules": {}, ".git": {}, "gen": {}, "dist": {}, "build": {}, "vendor": {}, ".vrooli": {},
}

// Scanner reads a scenario's source tree and returns every suppression
// marker it finds.
//
// seam: Scanner is the substitution boundary for marker discovery.
// Production wires FileScanner (walks the filesystem); tests pass a fake.
// Registered in docs/internal/SEAMS.md.
type Scanner interface {
	// Scan walks scenarioDir and returns all markers (valid and malformed;
	// callers filter with Validate/IsActive).
	Scan(ctx context.Context, scenarioDir string) ([]Marker, error)
}

// FileScanner is the production filesystem scanner.
type FileScanner struct{}

// NewFileScanner returns the production scanner.
func NewFileScanner() *FileScanner { return &FileScanner{} }

var _ Scanner = (*FileScanner)(nil)

// Scan walks scenarioDir for source files and parses every marker. Marker
// File paths are returned scenario-relative with forward slashes.
func (s *FileScanner) Scan(_ context.Context, scenarioDir string) ([]Marker, error) {
	var markers []Marker
	err := filepath.WalkDir(scenarioDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := scannableExtensions[strings.ToLower(filepath.Ext(path))]; !ok {
			return nil
		}
		rel, relErr := filepath.Rel(scenarioDir, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		fileMarkers, scanErr := scanFile(path, rel)
		if scanErr != nil {
			return scanErr
		}
		markers = append(markers, fileMarkers...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan suppressions: %w", err)
	}
	sort.Slice(markers, func(i, j int) bool {
		if markers[i].File != markers[j].File {
			return markers[i].File < markers[j].File
		}
		return markers[i].Line < markers[j].Line
	})
	return markers, nil
}

func scanFile(absPath, relPath string) ([]Marker, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []Marker
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		m, ok := ParseMarker(sc.Text())
		if !ok {
			continue
		}
		m.File = relPath
		m.Line = line
		out = append(out, m)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", relPath, err)
	}
	return out, nil
}

// Locator resolves a scenario name to its on-disk root. domains'
// RepoScenarioLocator satisfies this structurally (no import needed).
type Locator interface {
	Locate(scenario string) (string, error)
}

// Clock is the slim time seam for expiry evaluation.
type Clock = TimeSource

type TimeSource interface {
	Now() time.Time
}

// Provider resolves + scans + filters a scenario's active, valid markers.
// It is the seam the conflicts handler depends on.
type Provider interface {
	Active(ctx context.Context, scenario string) ([]Marker, error)
}

type provider struct {
	locator Locator
	scanner Scanner
	clock   Clock
}

// NewProvider wires a production marker provider.
func NewProvider(locator Locator, scanner Scanner, clock Clock) Provider {
	return &provider{locator: locator, scanner: scanner, clock: clock}
}

var _ Provider = (*provider)(nil)

// Active returns only well-formed, non-expired markers for the scenario.
func (p *provider) Active(ctx context.Context, scenario string) ([]Marker, error) {
	dir, err := p.locator.Locate(scenario)
	if err != nil {
		return nil, err
	}
	all, err := p.scanner.Scan(ctx, dir)
	if err != nil {
		return nil, err
	}
	now := p.clock.Now()
	out := make([]Marker, 0, len(all))
	for _, m := range all {
		if m.Validate() && m.IsActive(now) {
			out = append(out, m)
		}
	}
	return out, nil
}
