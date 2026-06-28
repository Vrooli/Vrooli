package plans

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Work-posture derivation. The Greenfield/Brownfield stance of a plan is
// AUTOFILLED from the associated scenario's maturity — never typed by an
// authoring agent. The default is Greenfield unless a scenario's
// .vrooli/service.json maturity proves otherwise. This is the one place posture
// is decided; render.go projects it and the wizard only reviews it. See
// docs/concepts/PLAN-MODEL.md (Work Posture).

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
// override or an imported posture and never clobbers it; otherwise it resolves
// the associated scenario and reads its maturity through the seam, defaulting to
// Greenfield. The returned triple is (posture, source, detail).
func ResolvePosture(ctx context.Context, p Plan, reader MaturityReader) (WorkPosture, WorkPostureSource, string) {
	// An explicit override or imported posture is authoritative.
	if p.WorkPostureSource == WorkPostureSourceExplicitOverride ||
		p.WorkPostureSource == WorkPostureSourceImportLegacy {
		return p.WorkPosture, p.WorkPostureSource, p.WorkPostureDetail
	}

	scenario := scenarioForPlan(p)
	if scenario == "" || reader == nil {
		return WorkPostureGreenfield, WorkPostureSourceDefault,
			"No associated scenario resolved; defaulting to greenfield."
	}

	maturity, ok, err := reader.Maturity(ctx, scenario)
	if err != nil || !ok {
		return WorkPostureGreenfield, WorkPostureSourceDefault,
			fmt.Sprintf("Could not read maturity for scenario %q; defaulting to greenfield.", scenario)
	}

	switch strings.TrimSpace(maturity) {
	case maturityPilot:
		return WorkPostureBrownfield, WorkPostureSourceServiceMaturity,
			fmt.Sprintf("Scenario %q is in pilot (limited live use); preserve external contracts unless a breaking change is explicitly authorized.", scenario)
	case maturityProduction:
		return WorkPostureBrownfield, WorkPostureSourceServiceMaturity,
			fmt.Sprintf("Scenario %q is in production (serving real data); preserve external contracts and data unless a breaking change is explicitly authorized.", scenario)
	case maturitySunset:
		return WorkPostureBrownfield, WorkPostureSourceServiceMaturity,
			fmt.Sprintf("Scenario %q is being retired (sunset); prefer non-invasive changes and avoid new surface area.", scenario)
	case maturityGreenfield, "":
		return WorkPostureGreenfield, WorkPostureSourceServiceMaturity,
			fmt.Sprintf("Scenario %q maturity is greenfield.", scenario)
	default:
		return WorkPostureGreenfield, WorkPostureSourceServiceMaturity,
			fmt.Sprintf("Scenario %q has an unrecognized maturity %q; defaulting to greenfield.", scenario, maturity)
	}
}

var scenarioRefPattern = regexp.MustCompile(`scenarios/([A-Za-z0-9._-]+)`)

// scenarioForPlan derives the scenario a plan is about from its strongest
// signals, in priority order: the regression anchor's scenario baseline, then a
// `scenarios/<name>/...` code reference (plan or phase scope). Returns "" when no
// scenario can be associated.
func scenarioForPlan(p Plan) string {
	if s := strings.TrimSpace(p.RegressionAnchor.Scenario); s != "" {
		return s
	}
	if s := scenarioFromReferences(p.References); s != "" {
		return s
	}
	for _, ph := range p.Phases {
		if s := scenarioFromReferences(ph.References); s != "" {
			return s
		}
	}
	return ""
}

func scenarioFromReferences(refs []Reference) string {
	for _, ref := range refs {
		if ref.Kind != ReferenceCode {
			continue
		}
		if m := scenarioRefPattern.FindStringSubmatch(ref.Target); m != nil {
			return m[1]
		}
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
