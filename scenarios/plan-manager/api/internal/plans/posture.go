package plans

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Work-posture derivation. The Greenfield/Brownfield stance of a plan is
// AUTOFILLED — never typed by an authoring agent. It is derived CONSERVATIVELY
// and in AGGREGATE across every scenario the plan's change boundary and
// references touch: if any affected scenario is pilot/production/sunset the plan
// is Brownfield; otherwise Greenfield. The default is Greenfield unless a
// scenario's .vrooli/service.json maturity proves otherwise. This is the one
// place posture is decided; render.go projects it and the wizard only reviews it.
// See docs/concepts/PLAN-MODEL.md (Work Posture).

// Recognized scenario maturity values (mirrors api-core/projectmeta and
// .vrooli/schemas/service.schema.json). Absent/unrecognized => greenfield.
const (
	maturityGreenfield = "greenfield"
	maturityPilot      = "pilot"
	maturityProduction = "production"
	maturitySunset     = "sunset"
)

// MaturityReader reads the declared maturity of a named scenario's
// .vrooli/service.json. It is a seam: the production impl reads the filesystem; a
// nil reader (or a read that fails) degrades posture to Greenfield honestly —
// never a false Brownfield. `ok` is true when the scenario's service.json was
// located and parsed (regardless of whether a maturity key was present).
type MaturityReader interface {
	Maturity(ctx context.Context, scenario string) (maturity string, ok bool, err error)
}

// ResolvePosture derives the work posture for a plan. It honors an explicit
// override or an imported posture and never clobbers it; otherwise it collects
// EVERY scenario the change boundary and references touch and derives the posture
// CONSERVATIVELY across all of them: if any affected scenario is pilot/production/
// sunset the plan is Brownfield (and the detail names the causes); otherwise it is
// Greenfield. No code path uses "first scenario wins." The returned triple is
// (posture, source, detail).
func ResolvePosture(ctx context.Context, p Plan, reader MaturityReader) (WorkPosture, WorkPostureSource, string) {
	// An explicit override or imported posture is authoritative.
	if p.WorkPostureSource == WorkPostureSourceExplicitOverride ||
		p.WorkPostureSource == WorkPostureSourceImportLegacy {
		return p.WorkPosture, p.WorkPostureSource, p.WorkPostureDetail
	}

	scenarios := affectedScenariosForPlan(p)
	if len(scenarios) == 0 {
		return WorkPostureGreenfield, WorkPostureSourceDefault,
			"No affected scenario resolved; defaulting to greenfield."
	}
	if reader == nil {
		return WorkPostureGreenfield, WorkPostureSourceDefault,
			"Maturity reader unavailable; defaulting to greenfield."
	}

	var brownfieldCauses []string
	var unreadable []string
	readAny := false
	for _, scenario := range scenarios {
		maturity, ok, err := reader.Maturity(ctx, scenario)
		if err != nil || !ok {
			unreadable = append(unreadable, scenario)
			continue
		}
		readAny = true
		switch strings.TrimSpace(maturity) {
		case maturityPilot:
			brownfieldCauses = append(brownfieldCauses, fmt.Sprintf("%s (pilot)", scenario))
		case maturityProduction:
			brownfieldCauses = append(brownfieldCauses, fmt.Sprintf("%s (production)", scenario))
		case maturitySunset:
			brownfieldCauses = append(brownfieldCauses, fmt.Sprintf("%s (sunset)", scenario))
		}
	}

	if len(brownfieldCauses) > 0 {
		return WorkPostureBrownfield, WorkPostureSourceServiceMaturity,
			fmt.Sprintf("Brownfield because affected scenario(s) are deployed or limited-live: %s. Preserve external contracts and data unless the plan explicitly authorizes a breaking change.",
				strings.Join(brownfieldCauses, ", "))
	}

	// No scenario maturity could actually be read — honest default, not a derived
	// greenfield claim.
	if !readAny {
		return WorkPostureGreenfield, WorkPostureSourceDefault,
			fmt.Sprintf("Maturity for affected scenario(s) %s could not be read; defaulting to greenfield.",
				strings.Join(unreadable, ", "))
	}

	detail := fmt.Sprintf("All affected scenarios are greenfield: %s.", strings.Join(scenarios, ", "))
	if len(unreadable) > 0 {
		detail += fmt.Sprintf(" Maturity for %s could not be read; treated as greenfield.", strings.Join(unreadable, ", "))
	}
	return WorkPostureGreenfield, WorkPostureSourceServiceMaturity, detail
}

var scenarioRefPattern = regexp.MustCompile(`scenarios/([A-Za-z0-9._-]+)`)

// affectedScenariosForPlan collects every scenario the plan touches, from (in
// no priority order — all contribute): the change boundary's allow globs, each
// phase boundary's allow globs, the plan/phase regression-anchor scenario (legacy
// imported anchors), and `scenarios/<name>/...` code references at plan and phase
// scope. The result is deduplicated and sorted so posture derivation is
// deterministic.
func affectedScenariosForPlan(p Plan) []string {
	seen := map[string]struct{}{}
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name != "" {
			seen[name] = struct{}{}
		}
	}
	for _, s := range p.ChangeBoundary.AffectedScenarios() {
		add(s)
	}
	add(p.RegressionAnchor.Scenario)
	for _, ref := range p.References {
		add(scenarioFromReference(ref))
	}
	for _, ph := range p.Phases {
		for _, s := range ph.ChangeBoundary.AffectedScenarios() {
			add(s)
		}
		for _, ref := range ph.References {
			add(scenarioFromReference(ref))
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func scenarioFromReference(ref Reference) string {
	if ref.Kind != ReferenceCode {
		return ""
	}
	if m := scenarioRefPattern.FindStringSubmatch(ref.Target); m != nil {
		return m[1]
	}
	return ""
}

// PostureBlock returns the rendered guidance block for a posture. Greenfield
// renders the EXACT canonical sentence (with backticks preserved on the
// code-like tokens); Brownfield renders the conservative compatibility note.
func PostureBlock(posture WorkPosture) string {
	switch posture {
	case WorkPostureBrownfield:
		return "This plan targets a deployed or limited-live scenario. Preserve external contracts and data unless the plan explicitly authorizes a breaking change."
	default:
		return "**This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables."
	}
}

// --- production filesystem reader ---

type fsMaturityReader struct{}

// NewFilesystemMaturityReader returns the production MaturityReader. It locates
// the monorepo root by walking up from the working directory until it finds a
// `scenarios/` directory, then reads `scenarios/<name>/.vrooli/service.json`.
func NewFilesystemMaturityReader() MaturityReader { return fsMaturityReader{} }

func (fsMaturityReader) Maturity(_ context.Context, scenario string) (string, bool, error) {
	scenario = strings.TrimSpace(scenario)
	// Only a single, safe path element is a valid scenario name — never a path
	// with separators or traversal. This plus os.Root scoping below makes the
	// service.json read traversal-proof.
	if scenario == "" || scenario == "." || scenario == ".." ||
		strings.ContainsAny(scenario, `/\`) || strings.Contains(scenario, "..") {
		return "", false, nil
	}
	root, ok := findScenariosRoot()
	if !ok {
		return "", false, nil
	}
	// Scope all reads under <root>/scenarios so a crafted name cannot escape it.
	scenariosRoot, err := os.OpenRoot(filepath.Join(root, "scenarios"))
	if err != nil {
		return "", false, nil
	}
	defer func() { _ = scenariosRoot.Close() }()
	f, err := scenariosRoot.Open(filepath.Join(scenario, ".vrooli", "service.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	defer func() { _ = f.Close() }()
	raw, err := io.ReadAll(f)
	if err != nil {
		return "", false, err
	}
	return parseMaturity(raw)
}

// parseMaturity extracts the recognized maturity from a service.json payload.
// ok=true once the document parses; an unrecognized/absent value yields "" so
// ResolvePosture treats it as greenfield (matching the schema default).
func parseMaturity(raw []byte) (string, bool, error) {
	var doc struct {
		Maturity string `json:"maturity"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", false, err
	}
	switch doc.Maturity {
	case maturityGreenfield, maturityPilot, maturityProduction, maturitySunset:
		return doc.Maturity, true, nil
	default:
		return "", true, nil
	}
}

// findScenariosRoot walks up from cwd to the nearest ancestor containing a
// `scenarios/` directory (the monorepo root).
func findScenariosRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "scenarios")); err == nil && info.IsDir() {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
