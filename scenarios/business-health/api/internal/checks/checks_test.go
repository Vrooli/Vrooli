package checks

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"business-health/internal/extraction"

	"github.com/stretchr/testify/require"
	maturity "github.com/vrooli/maturity-go/assessment"
	intent "intent-go"
)

// conformantPRD is a fully canonical PRD fixture; per-code tests mutate it.
const conformantPRD = `# Product Requirements Document (PRD)

## 🎯 Overview
- **Purpose**: Validate things.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | First target | Does the first thing.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Later target | Does a later thing.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Future target | Does a future thing.

## 🧱 Tech Direction Snapshot
- Preferred stacks: Go.

## 🤝 Dependencies & Launch Plan
- None.

## 🎨 UX & Branding
- Accessibility: WCAG AA.
`

const conformantIndex = `{"_metadata":{"description":"x","auto_sync_enabled":false,"schema_version":"1.0.0"},"imports":["01-core/module.json"]}`

const conformantModule = `{"requirements":[
  {"id":"FX-001","title":"First requirement","status":"planned","criticality":"P0","prd_ref":"OT-P0-001",
   "validation":[{"type":"test","ref":"api/present_test.go","status":"planned","phase":"unit"}]},
  {"id":"FX-002","title":"Later requirement","status":"planned","criticality":"P1","prd_ref":"OT-P1-001",
   "validation":[{"type":"manual","status":"planned","notes":"attended check"}]},
  {"id":"FX-003","title":"Future requirement","status":"planned","criticality":"P2","prd_ref":"OT-P2-001",
   "validation":[{"type":"test","ref":"api/present_test.go","status":"planned","phase":"unit"}]}
]}`

const conformantReadme = `# Requirements
Every operational target links here; auto-sync earns statuses from tagged tests; each requirement carries validation entries.
`

// fixture builds a scenario tree; overrides replace (or with empty value
// delete) the default conformant files.
func fixture(t *testing.T, overrides map[string]string) extraction.Contract {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"PRD.md":                           conformantPRD,
		"requirements/index.json":          conformantIndex,
		"requirements/01-core/module.json": conformantModule,
		"requirements/README.md":           conformantReadme,
		"api/present_test.go":              "package api\n",
	}
	for rel, content := range overrides {
		if content == "" {
			delete(files, rel)
			continue
		}
		files[rel] = content
	}
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	contract, err := extraction.NewFileExtractor().Load("fixture", dir)
	require.NoError(t, err)
	return contract
}

func runAll(t *testing.T, c extraction.Contract) []intent.Finding {
	t.Helper()
	var out []intent.Finding
	for _, chk := range Registry() {
		out = append(out, chk.Run(context.Background(), c)...)
	}
	return out
}

// baseCodes maps findings to their base code (":CLAIM" suffix stripped).
func baseCodes(findings []intent.Finding) map[string][]intent.Finding {
	out := map[string][]intent.Finding{}
	for _, f := range findings {
		code := f.Code
		if i := strings.IndexByte(code, ':'); i > 0 {
			code = code[:i]
		}
		out[code] = append(out[code], f)
	}
	return out
}

// [REQ:BH-VAL-001] [REQ:BH-VAL-002] [REQ:BH-VAL-003] [REQ:BH-VAL-004]
// The conformant fixture is findings-clean — the negative case for every code.
func TestConformantFixtureIsClean(t *testing.T) {
	findings := runAll(t, fixture(t, nil))
	require.Empty(t, findings, "conformant fixture should be clean, got %+v", findings)
}

// [REQ:BH-VAL-001] Template trio + presence + OT format, one mutation each.
func TestPRDContractCodes(t *testing.T) {
	cases := []struct {
		name      string
		overrides map[string]string
		wantCode  string
		severity  string
	}{
		{"missing PRD", map[string]string{"PRD.md": ""}, "prd_missing_prd", "error"},
		{"missing required section", map[string]string{
			"PRD.md": strings.Replace(conformantPRD, "## 🧱 Tech Direction Snapshot\n- Preferred stacks: Go.\n\n", "", 1),
		}, "prd_template_sections", "error"},
		{"unexpected section", map[string]string{
			"PRD.md": conformantPRD + "\n## Roadmap\nStuff.\n",
		}, "prd_template_unexpected_sections", "info"},
		{"placeholder content", map[string]string{
			"PRD.md": strings.Replace(conformantPRD, "- **Purpose**: Validate things.", "- Something else.", 1),
		}, "prd_template_content", "error"},
		{"malformed OT id", map[string]string{
			"PRD.md":                           strings.ReplaceAll(conformantPRD, "OT-P0-001", "OT-P0-1"),
			"requirements/01-core/module.json": strings.ReplaceAll(conformantModule, "OT-P0-001", "OT-P0-1"),
		}, "prd_ot_id_format", "warning"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := baseCodes(runAll(t, fixture(t, tc.overrides)))
			group, ok := found[tc.wantCode]
			require.True(t, ok, "expected %s in %v", tc.wantCode, keys(found))
			require.Equal(t, tc.severity, group[0].Severity)
		})
	}
}

// [REQ:BH-VAL-003] Registry structural + quality codes, one mutation each.
func TestRegistryCodes(t *testing.T) {
	cases := []struct {
		name      string
		overrides map[string]string
		wantCode  string
		severity  string
	}{
		{"missing registry", map[string]string{
			"requirements/index.json":          "",
			"requirements/01-core/module.json": "",
			"requirements/README.md":           "",
		}, "prd_missing_requirements", "error"},
		{"unparseable module", map[string]string{
			"requirements/01-core/module.json": "{not json",
		}, "business_registry_unparseable", "error"},
		{"duplicate id", map[string]string{
			"requirements/01-core/module.json": strings.Replace(conformantModule, `"id":"FX-002"`, `"id":"fx-001"`, 1),
		}, "business_duplicate_req_id", "error"},
		{"missing id", map[string]string{
			"requirements/01-core/module.json": strings.Replace(conformantModule, `"id":"FX-002",`, `"id":"",`, 1),
		}, "business_req_missing_id", "error"},
		{"missing title", map[string]string{
			"requirements/01-core/module.json": strings.Replace(conformantModule, `"title":"Later requirement",`, `"title":"",`, 1),
		}, "business_req_missing_title", "warning"},
		{"invalid status", map[string]string{
			"requirements/01-core/module.json": strings.Replace(conformantModule, `"id":"FX-002","title":"Later requirement","status":"planned"`, `"id":"FX-002","title":"Later requirement","status":"done"`, 1),
		}, "business_invalid_status", "warning"},
		{"orphaned children ref", map[string]string{
			"requirements/01-core/module.json": strings.Replace(conformantModule, `"criticality":"P1",`, `"criticality":"P1","children":["FX-999"],`, 1),
		}, "business_orphaned_ref", "error"},
		{"cycle", map[string]string{
			"requirements/01-core/module.json": strings.Replace(
				strings.Replace(conformantModule, `"id":"FX-001","title":"First requirement","status":"planned","criticality":"P0",`, `"id":"FX-001","title":"First requirement","status":"planned","criticality":"P0","depends_on":["FX-002"],`, 1),
				`"id":"FX-002","title":"Later requirement","status":"planned","criticality":"P1",`, `"id":"FX-002","title":"Later requirement","status":"planned","criticality":"P1","depends_on":["FX-001"],`, 1),
		}, "business_import_cycle", "error"},
		{"P1 without validation", map[string]string{
			"requirements/01-core/module.json": strings.Replace(conformantModule, `"validation":[{"type":"manual","status":"planned","notes":"attended check"}]`, `"validation":[]`, 1),
		}, "business_req_no_validation", "warning"},
		{"starter template", map[string]string{
			"requirements/01-core/module.json": strings.Replace(conformantModule, `"prd_ref":"OT-P2-001",`, `"prd_ref":"OT-P2-001","tags":["template-starter"],`, 1),
		}, "business_starter_template", "warning"},
		{"readme drift", map[string]string{
			"requirements/README.md": "# Requirements\nNothing useful.\n",
		}, "requirements_readme", "warning"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := baseCodes(runAll(t, fixture(t, tc.overrides)))
			group, ok := found[tc.wantCode]
			require.True(t, ok, "expected %s in %v", tc.wantCode, keys(found))
			require.Equal(t, tc.severity, group[0].Severity)
		})
	}
}

// [REQ:BH-VAL-003] P0 requirements without validation escalate to error.
func TestP0NoValidationEscalates(t *testing.T) {
	c := fixture(t, map[string]string{
		"requirements/01-core/module.json": strings.Replace(conformantModule,
			`"validation":[{"type":"test","ref":"api/present_test.go","status":"planned","phase":"unit"}]},
  {"id":"FX-002"`, `"validation":[]},
  {"id":"FX-002"`, 1),
	})
	found := baseCodes(runAll(t, c))
	group := found["business_req_no_validation"]
	require.Len(t, group, 1)
	require.Equal(t, "error", group[0].Severity)
	require.Contains(t, group[0].Code, ":FX-001")
}

// [REQ:BH-VAL-002] Linkage codes in both directions.
func TestLinkageCodes(t *testing.T) {
	t.Run("dangling prd_ref", func(t *testing.T) {
		c := fixture(t, map[string]string{
			"requirements/01-core/module.json": strings.Replace(conformantModule, `"prd_ref":"OT-P2-001"`, `"prd_ref":"OT-P2-099"`, 1),
		})
		found := baseCodes(runAll(t, c))
		group, ok := found[intent.CodePRDRefUnmatched]
		require.True(t, ok, "expected prd_ref_unmatched in %v", keys(found))
		require.Equal(t, "warning", group[0].Severity)
		require.Contains(t, group[0].Code, ":FX-003")
	})
	t.Run("orphaned OT warning for P1", func(t *testing.T) {
		c := fixture(t, map[string]string{
			"PRD.md": strings.Replace(conformantPRD,
				"- [ ] OT-P1-001 | Later target | Does a later thing.",
				"- [ ] OT-P1-001 | Later target | Does a later thing.\n- [ ] OT-P1-002 | Uncovered target | Nothing points here.", 1),
		})
		found := baseCodes(runAll(t, c))
		group, ok := found[intent.CodeOTOrphan]
		require.True(t, ok, "expected ot_orphan in %v", keys(found))
		require.Equal(t, "warning", group[0].Severity)
	})
	t.Run("orphaned P0 OT escalates to error", func(t *testing.T) {
		c := fixture(t, map[string]string{
			"PRD.md": strings.Replace(conformantPRD,
				"- [ ] OT-P0-001 | First target | Does the first thing.",
				"- [ ] OT-P0-001 | First target | Does the first thing.\n- [ ] OT-P0-002 | Uncovered launch target | Nothing points here.", 1),
		})
		found := baseCodes(runAll(t, c))
		group, ok := found[intent.CodeOTOrphan]
		require.True(t, ok)
		require.Equal(t, "error", group[0].Severity)
	})
}

// [REQ:BH-VAL-004] Ref existence via the mini-format: broken code refs are
// errors; manual refs are never path-checked.
func TestRefExistence(t *testing.T) {
	c := fixture(t, map[string]string{
		"requirements/01-core/module.json": strings.ReplaceAll(conformantModule, "api/present_test.go", "api/gone_test.go"),
	})
	found := baseCodes(runAll(t, c))
	group, ok := found[intent.CodeRefMissing]
	require.True(t, ok, "expected ref_missing in %v", keys(found))
	require.Equal(t, "error", group[0].Severity)
	require.Len(t, group, 2, "both code-typed refs broke; the manual validation must not be path-checked")
}

// [REQ:BH-PRV-003] The engine caps severity at ERROR (advisory posture).
func TestEngineSeverityCap(t *testing.T) {
	engine := New("", stubExtractor{}, stubCheck{findings: []intent.Finding{{Code: "x", Severity: "blocker"}}})
	rep, err := engine.ValidateScenario(context.Background(), "any", t.TempDir())
	require.NoError(t, err)
	require.Equal(t, "SEVERITY_ERROR", rep.Findings[0].Severity)
}

// [REQ:BH-VAL-006] Every finding code the checks emit is declared in
// .vrooli/maturity.json — the vocabulary can't drift from the spec.
func TestEmittedCodesAreInVocabulary(t *testing.T) {
	spec := loadSpec(t)
	mutations := []map[string]string{
		{"PRD.md": ""},
		{"PRD.md": "# PRD\n\n## Whatever\nx\n"},
		{"requirements/index.json": "", "requirements/01-core/module.json": "", "requirements/README.md": ""},
		{"requirements/01-core/module.json": "{broken"},
		{"requirements/01-core/module.json": strings.ReplaceAll(conformantModule, "api/present_test.go", "api/gone_test.go")},
		{"requirements/README.md": "# empty\n"},
	}
	for _, overrides := range mutations {
		for _, f := range runAll(t, fixture(t, overrides)) {
			code := f.Code
			if i := strings.IndexByte(code, ':'); i > 0 {
				code = code[:i]
			}
			_, ok := spec[code]
			require.True(t, ok, "emitted code %q is not in .vrooli/maturity.json", code)
		}
	}
}

type stubExtractor struct{}

func (stubExtractor) Load(scenario, dir string) (extraction.Contract, error) {
	return extraction.Contract{Scenario: scenario, ScenarioDir: dir, PRDPresent: true, RegistryPresent: true}, nil
}

type stubCheck struct{ findings []intent.Finding }

func (stubCheck) Name() string { return "stub" }
func (s stubCheck) Run(context.Context, extraction.Contract) []intent.Finding {
	return s.findings
}

func loadSpec(t *testing.T) map[string]struct{} {
	t.Helper()
	spec, err := maturity.LoadSpecFromScenario(scenarioRootFromTest(t))
	require.NoError(t, err)
	out := make(map[string]struct{}, len(spec.Findings))
	for code := range spec.Findings {
		out[code] = struct{}{}
	}
	return out
}

func scenarioRootFromTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(dir, "..", "..", ".."))
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
