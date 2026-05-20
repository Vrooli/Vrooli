// Package projectmeta exposes durable metadata about the running scenario's
// project as declared in .vrooli/service.json.
//
// The accessors in this package are the single source of truth for "is this
// scenario running in development or production mode" — the answer that gates
// the test-mode HTTP middleware and the dev-only RoutingService.
package projectmeta

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Mode values recognized by service.json.
const (
	ModeDevelopment = "development"
	ModeProduction  = "production"
)

var (
	once     sync.Once
	cached   string
	cachedOK bool

	// startDir is the directory the mode lookup ascends from. It defaults to
	// os.Getwd() at first use; tests override it via SetStartDirForTesting.
	startDirMu sync.Mutex
	startDir   string
)

// Mode returns the project's declared mode. If service.json is missing,
// unreadable, or carries an unrecognized value, Mode returns
// ModeDevelopment — the safer default for the *gating* of dev-only surfaces
// (forgetting to set the flag manifests as a 404 or test-mode no-op in a
// production deploy that has been validated; see docs for the deploy-time
// validation follow-up).
func Mode() string {
	once.Do(loadMode)
	return cached
}

// IsDevelopment reports whether Mode() == ModeDevelopment.
func IsDevelopment() bool { return Mode() == ModeDevelopment }

// IsProduction reports whether Mode() == ModeProduction.
func IsProduction() bool { return Mode() == ModeProduction }

// loadMode resolves the mode by walking up from startDir (or cwd) looking for
// the nearest .vrooli/service.json and reading its top-level "mode" field.
func loadMode() {
	cached = ModeDevelopment // safe default

	dir := currentStartDir()
	path, ok := findServiceJSON(dir)
	if !ok {
		return
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("projectmeta: read service.json failed; defaulting to development",
			"path", path, "err", err)
		return
	}

	var doc struct {
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		slog.Warn("projectmeta: parse service.json failed; defaulting to development",
			"path", path, "err", err)
		return
	}

	switch doc.Mode {
	case "":
		// Absent field — defaults to development; not a warning.
	case ModeDevelopment, ModeProduction:
		cached = doc.Mode
		cachedOK = true
	default:
		slog.Warn("projectmeta: unrecognized mode value; defaulting to development",
			"path", path, "value", doc.Mode)
	}
}

// findServiceJSON ascends from dir looking for a .vrooli/service.json file.
// Returns the absolute path and true on success.
func findServiceJSON(dir string) (string, bool) {
	for {
		candidate := filepath.Join(dir, ".vrooli", "service.json")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func currentStartDir() string {
	startDirMu.Lock()
	defer startDirMu.Unlock()
	if startDir != "" {
		return startDir
	}
	wd, err := os.Getwd()
	if err != nil {
		return string(filepath.Separator)
	}
	return wd
}

// SetStartDirForTesting overrides the start directory used by the next call to
// Mode/IsDevelopment/IsProduction and resets the cache. Must not be used
// outside tests.
func SetStartDirForTesting(dir string) {
	startDirMu.Lock()
	startDir = dir
	startDirMu.Unlock()
	resetForTesting()
}

// resetForTesting clears the sync.Once-protected cache. Tests use this between
// cases that exercise different service.json contents.
func resetForTesting() {
	once = sync.Once{}
	cached = ""
	cachedOK = false
}

// MustString returns Mode() and is provided for symmetry with other api-core
// must-style helpers. It is identical to Mode() — kept here so a future change
// that adds error-returning loading has a clean upgrade path.
func MustString() string {
	if m := Mode(); m != "" {
		return m
	}
	panic(fmt.Errorf("projectmeta: empty mode after load"))
}
