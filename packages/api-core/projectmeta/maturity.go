package projectmeta

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
)

// Maturity values recognized by service.json. They describe the scenario's
// lifecycle/deploy stage, independent of dev/prod Mode():
//   - greenfield: not yet deployed; no schema migrations are expected, so
//     migration debt is informational rather than a failure.
//   - pilot:      limited live use.
//   - production: serving real data; migration hygiene is enforced.
//   - sunset:     being retired.
const (
	MaturityGreenfield = "greenfield"
	MaturityPilot      = "pilot"
	MaturityProduction = "production"
	MaturitySunset     = "sunset"
)

var (
	maturityOnce   sync.Once
	maturityCached string
)

// Maturity returns the project's declared maturity stage. If service.json is
// missing, unreadable, carries an unrecognized value, or omits the field,
// Maturity returns MaturityGreenfield — the safe default ("not yet deployed,
// so migration debt is informational"). storage-manager consumes this to derive
// storage_stage and to decide whether migration findings are advisory or
// enforced.
func Maturity() string {
	maturityOnce.Do(loadMaturity)
	return maturityCached
}

// IsGreenfield reports whether Maturity() == MaturityGreenfield.
func IsGreenfield() bool { return Maturity() == MaturityGreenfield }

// loadMaturity resolves the maturity by walking up from startDir (or cwd)
// looking for the nearest .vrooli/service.json and reading its top-level
// "maturity" field. It shares the discovery helpers with mode.go.
func loadMaturity() {
	maturityCached = MaturityGreenfield // safe default

	dir := currentStartDir()
	path, ok := findServiceJSON(dir)
	if !ok {
		return
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("projectmeta: read service.json failed; defaulting to greenfield",
			"path", path, "err", err)
		return
	}

	var doc struct {
		Maturity string `json:"maturity"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		slog.Warn("projectmeta: parse service.json failed; defaulting to greenfield",
			"path", path, "err", err)
		return
	}

	switch doc.Maturity {
	case "":
		// Absent field — defaults to greenfield; not a warning.
	case MaturityGreenfield, MaturityPilot, MaturityProduction, MaturitySunset:
		maturityCached = doc.Maturity
	default:
		slog.Warn("projectmeta: unrecognized maturity value; defaulting to greenfield",
			"path", path, "value", doc.Maturity)
	}
}

// resetMaturityForTesting clears the sync.Once-protected maturity cache. Tests
// use this between cases that exercise different service.json contents.
func resetMaturityForTesting() {
	maturityOnce = sync.Once{}
	maturityCached = ""
}
