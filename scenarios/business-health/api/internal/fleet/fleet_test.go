package fleet

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"business-health/internal/autofix"
	"business-health/internal/checks"
	"business-health/internal/extraction"

	"github.com/stretchr/testify/require"
)

func fixedNow() time.Time { return time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC) }

// write builds a file tree under root.
func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
}

const cleanPRD = `# Product Requirements Document (PRD)

## 🎯 Overview
- **Purpose**: Clean.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Target | Does a thing.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Target | Does a thing.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Target | Does a thing.

## 🧱 Tech Direction Snapshot
- Preferred stacks: Go.

## 🤝 Dependencies & Launch Plan
- None.

## 🎨 UX & Branding
- Accessibility: WCAG AA.
`

const cleanModule = `{"requirements":[
  {"id":"A-P0-001","title":"T","status":"planned","criticality":"P0","prd_ref":"OT-P0-001","validation":[{"type":"manual","status":"planned","notes":"n"}]},
  {"id":"A-P1-001","title":"T","status":"planned","criticality":"P1","prd_ref":"OT-P1-001","validation":[{"type":"manual","status":"planned","notes":"n"}]},
  {"id":"A-P2-001","title":"T","status":"planned","criticality":"P2","prd_ref":"OT-P2-001","validation":[{"type":"manual","status":"planned","notes":"n"}]}
]}`

const cleanReadme = "# Requirements\noperational target linkage; auto-sync earns statuses; every requirement carries validation.\n"

// [REQ:BH-FLT-001] The sweep discovers scenarios, grades them through the
// check engine, ranks worst-first, and stamps as-of.
func TestSweep(t *testing.T) {
	root := t.TempDir()
	// Scenario "clean": fully conformant, current template version.
	write(t, root, map[string]string{
		"templates/scenarios/react-vite/template.json":  `{"version":"1.3.0"}`,
		"scenarios/clean/.vrooli/service.json":          `{"generation":{"template":{"id":"react-vite","version":"1.3.0"}}}`,
		"scenarios/clean/PRD.md":                        cleanPRD,
		"scenarios/clean/requirements/index.json":       `{"imports":["01-a/module.json"]}`,
		"scenarios/clean/requirements/01-a/module.json": cleanModule,
		"scenarios/clean/requirements/README.md":        cleanReadme,
		// Scenario "messy": starter tag, missing PRD, template laggard.
		"scenarios/messy/.vrooli/service.json":          `{"generation":{"template":{"id":"react-vite","version":"1.1.0"}}}`,
		"scenarios/messy/requirements/index.json":       `{"imports":["01-a/module.json"]}`,
		"scenarios/messy/requirements/01-a/module.json": `{"requirements":[{"id":"M-001","title":"T","status":"planned","prd_ref":"OT-P0-001","tags":["template-starter"]}]}`,
		"scenarios/messy/requirements/README.md":        cleanReadme,
		// Not a scenario: no service.json.
		"scenarios/junk/notes.txt": "x",
	})
	engine := checks.New(root, extraction.NewFileExtractor(), checks.Registry()...)
	s := NewSweeper(root, engine, autofix.RuleIDs(), fixedNow)

	res, err := s.Scan(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, fixedNow(), res.AsOf)
	require.Equal(t, 2, res.ScenarioCount, "junk/ has no service.json and must be skipped")
	require.Equal(t, 1, res.StarterRegistryCount)
	require.Equal(t, 1, res.TemplateLaggardCount)

	require.Equal(t, "messy", res.Entries[0].Scenario, "worst first")
	messy := res.Entries[0]
	require.False(t, messy.Passed)
	require.True(t, messy.StarterRegistry)
	require.True(t, messy.TemplateLaggard)
	require.Positive(t, messy.DebtScore)

	clean := res.Entries[1]
	require.Equal(t, "clean", clean.Scenario)
	require.True(t, clean.Passed)
	require.Zero(t, clean.TotalFindings)
	require.Zero(t, clean.DebtScore)
	require.False(t, clean.TemplateLaggard)
}

// A subset scan grades only the requested slugs and unknown slugs land in
// errors, not silence.
func TestSweepSubsetAndErrors(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		"scenarios/clean/.vrooli/service.json":          `{}`,
		"scenarios/clean/PRD.md":                        cleanPRD,
		"scenarios/clean/requirements/index.json":       `{"imports":["01-a/module.json"]}`,
		"scenarios/clean/requirements/01-a/module.json": cleanModule,
		"scenarios/clean/requirements/README.md":        cleanReadme,
	})
	engine := checks.New(root, extraction.NewFileExtractor(), checks.Registry()...)
	s := NewSweeper(root, engine, autofix.RuleIDs(), fixedNow)

	res, err := s.Scan(context.Background(), []string{"clean", "ghost"})
	require.NoError(t, err)
	require.Equal(t, 1, res.ScenarioCount)
	require.Len(t, res.Errors, 1)
	require.Equal(t, "ghost", res.Errors[0].Scenario)
}
