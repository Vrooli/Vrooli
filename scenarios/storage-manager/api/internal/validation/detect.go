package validation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Detection is the resolved storage surface for a scenario: its API-surface
// language and the domains its code is partitioned into. Produced by a
// Detector (code-facts-backed in production, filesystem in tests/fallback).
type Detection struct {
	// Language is the API-surface language ("go", "typescript", "python", "").
	Language string
	// Domains are the code-facts file-domains the scenario's code lives in.
	Domains []string
}

// Detector resolves a scenario's API language + domains. The seam lets the
// service run with code-facts in production and a deterministic filesystem
// detector in tests, without a running code-facts service.
type Detector interface {
	Detect(ctx context.Context, scenario, scenarioDir string) Detection
}

// FilesystemDetector resolves language + domains purely from on-disk layout.
// It is both the test detector and the production fallback when code-facts is
// unavailable — so detection is never silently empty.
type FilesystemDetector struct{}

var _ Detector = FilesystemDetector{}

// Detect classifies the API surface language from its build manifest and the
// domains from the api/internal and api/handlers directory names.
func (FilesystemDetector) Detect(_ context.Context, _ string, scenarioDir string) Detection {
	return Detection{
		Language: filesystemLanguage(scenarioDir),
		Domains:  filesystemDomains(scenarioDir),
	}
}

// filesystemLanguage returns the API-surface language inferred from the
// manifest present under api/. Go is authoritative via go.mod; TS via
// package.json; Python via pyproject.toml / requirements.txt.
func filesystemLanguage(scenarioDir string) string {
	apiDir := filepath.Join(scenarioDir, "api")
	switch {
	case fileExists(filepath.Join(apiDir, "go.mod")):
		return "go"
	case fileExists(filepath.Join(apiDir, "package.json")):
		return "typescript"
	case fileExists(filepath.Join(apiDir, "pyproject.toml")), fileExists(filepath.Join(apiDir, "requirements.txt")):
		return "python"
	default:
		return ""
	}
}

// infraDirs are directory names under api/internal and api/handlers that are
// cross-cutting infrastructure, not storage domains.
var infraDirs = map[string]struct{}{
	"clock": {}, "database": {}, "httpc": {}, "httpx": {}, "middleware": {},
	"module": {}, "modules": {}, "server": {}, "testutil": {}, "health": {},
	"config": {}, "metrics": {}, "preflight": {}, "version": {},
}

// filesystemDomains lists the storage domains a scenario partitions its code
// into, inferred from api/internal/<domain> and api/handlers/<domain>
// directory names (excluding cross-cutting infrastructure). Deterministic.
func filesystemDomains(scenarioDir string) []string {
	seen := map[string]struct{}{}
	for _, parent := range []string{
		filepath.Join(scenarioDir, "api", "internal"),
		filepath.Join(scenarioDir, "api", "handlers"),
	} {
		entries, err := os.ReadDir(parent)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if _, infra := infraDirs[name]; infra || strings.HasPrefix(name, ".") {
				continue
			}
			seen[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for d := range seen {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// serviceJSON is the minimal projection of .vrooli/service.json the detector
// reads: declared resources (engines), scenario dependencies (backup-target
// detection), the maturity stage, and an optional backup block.
//
// resources and scenarios are captured as RawMessage because the fleet carries
// TWO shapes: newer scenarios declare them as JSON arrays of ids
// (["postgres","redis"]) while older scenarios use keyed object maps
// ({"postgres":{...}}). dependencyKeys normalizes both so the whole fleet
// classifies correctly — parsing only one shape silently misclassifies the
// majority of deployed scenarios.
type serviceJSON struct {
	Maturity     string `json:"maturity"`
	Dependencies struct {
		Resources json.RawMessage `json:"resources"`
		Scenarios json.RawMessage `json:"scenarios"`
	} `json:"dependencies"`
	Backup json.RawMessage `json:"backup"`
}

// dependencyKeys normalizes a service.json dependency block — a JSON array of
// string ids, an array of {type|id} objects, or an object map keyed by id —
// into a sorted, de-duplicated, lower-cased id slice. Returns nil for an
// absent or unrecognized block.
func dependencyKeys(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var arrStr []string
	if json.Unmarshal(raw, &arrStr) == nil {
		return normalizeKeys(arrStr)
	}
	var arrObj []map[string]any
	if json.Unmarshal(raw, &arrObj) == nil {
		var ids []string
		for _, m := range arrObj {
			if v, ok := m["type"].(string); ok && v != "" {
				ids = append(ids, v)
			} else if v, ok := m["id"].(string); ok && v != "" {
				ids = append(ids, v)
			}
		}
		return normalizeKeys(ids)
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) == nil {
		ids := make([]string, 0, len(obj))
		for k := range obj {
			ids = append(ids, k)
		}
		return normalizeKeys(ids)
	}
	return nil
}

// normalizeKeys lower-cases, trims, drops empties, de-duplicates, and sorts.
func normalizeKeys(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// detectEngines classifies the storage engines a scenario uses from its
// declared service.json resources plus filesystem evidence (a SQLite driver
// import / system.sql implies SQLite even when not listed as a resource).
func detectEngines(scenarioDir string) []Engine {
	set := map[Engine]struct{}{}
	if sj, ok := readServiceJSON(scenarioDir); ok {
		for _, r := range dependencyKeys(sj.Dependencies.Resources) {
			switch r {
			case "postgres", "postgresql":
				set[EnginePostgres] = struct{}{}
			case "qdrant":
				set[EngineQdrant] = struct{}{}
			case "redis":
				set[EngineRedis] = struct{}{}
			}
		}
	}
	if usesSQLite(scenarioDir) {
		set[EngineSQLite] = struct{}{}
	}
	out := make([]Engine, 0, len(set))
	for e := range set {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// dataPersistingEngines is the set of engines that hold durable state worth a
// backup target. Redis is treated as an ephemeral cache (not durable here), so
// it is excluded; SQLite/Postgres/Qdrant/file all persist.
func isDataPersisting(engines []Engine) bool {
	for _, e := range engines {
		switch e {
		case EngineSQLite, EnginePostgres, EngineQdrant, EngineFile:
			return true
		}
	}
	return false
}

// HasBackupTarget reports whether the scenario at scenarioDir statically
// declares a backup target. Exported so the fleet inventory can classify
// backup readiness from the same signal the BACKUP_TARGET_MISSING analyzer uses.
func HasBackupTarget(scenarioDir string) bool { return hasBackupTarget(scenarioDir) }

// IsDataPersisting reports whether the engine set includes a durable store
// (SQLite/Postgres/Qdrant/file; Redis is treated as ephemeral cache). Exported
// for the fleet inventory's backup-readiness rollup.
func IsDataPersisting(engines []Engine) bool { return isDataPersisting(engines) }

// hasBackupTarget reports whether a scenario statically declares a backup
// target: either a `backup` block in service.json or a dependency on
// data-backup-manager. Runtime registrations are invisible to a static
// analyzer, so this is the honest, declaration-based signal — the backup
// finding it drives is advisory (L4), never a hard gate.
func hasBackupTarget(scenarioDir string) bool {
	sj, ok := readServiceJSON(scenarioDir)
	if !ok {
		return false
	}
	if len(sj.Backup) > 0 && string(sj.Backup) != "null" {
		return true
	}
	for _, s := range dependencyKeys(sj.Dependencies.Scenarios) {
		if s == "data-backup-manager" {
			return true
		}
	}
	return false
}

// usesSQLite reports whether the API surface imports a SQLite driver or embeds
// a SQL schema — the cheap, language-agnostic SQLite signal.
func usesSQLite(scenarioDir string) bool {
	gomod := filepath.Join(scenarioDir, "api", "go.mod")
	if data, err := os.ReadFile(gomod); err == nil {
		if strings.Contains(string(data), "modernc.org/sqlite") || strings.Contains(string(data), "mattn/go-sqlite3") {
			return true
		}
	}
	return fileExists(filepath.Join(scenarioDir, "api", "internal", "database", "system.sql"))
}

// deriveStorageStage resolves the scenario's deploy/greenfield stage from the
// service.json maturity field (defaulting to greenfield) and reports whether a
// committed migrations/ directory exists. The stage is informational — it
// gates whether migration findings are advisory, never a hard failure.
func deriveStorageStage(scenarioDir string) (stage string, hasMigrations bool) {
	stage = "greenfield"
	if sj, ok := readServiceJSON(scenarioDir); ok {
		switch strings.ToLower(strings.TrimSpace(sj.Maturity)) {
		case "pilot", "production", "sunset", "greenfield":
			stage = strings.ToLower(strings.TrimSpace(sj.Maturity))
		}
	}
	if info, err := os.Stat(filepath.Join(scenarioDir, "migrations")); err == nil && info.IsDir() {
		hasMigrations = true
	} else if info, err := os.Stat(filepath.Join(scenarioDir, "api", "migrations")); err == nil && info.IsDir() {
		hasMigrations = true
	}
	return stage, hasMigrations
}

func readServiceJSON(scenarioDir string) (serviceJSON, bool) {
	data, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return serviceJSON{}, false
	}
	var sj serviceJSON
	if err := json.Unmarshal(data, &sj); err != nil {
		return serviceJSON{}, false
	}
	return sj, true
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
