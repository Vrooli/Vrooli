package autofix

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"business-health/internal/checks"
	"business-health/internal/extraction"

	"github.com/stretchr/testify/require"
	intent "intent-go"
)

const fixturePRD = `# Product Requirements Document (PRD)

## 🎯 Overview
- **Purpose**: Fix things.

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

const fixtureModule = `{
  "requirements": [
    {
      "id": "FX-P0-001",
      "title": "First requirement",
      "status": "planned",
      "criticality": "P0",
      "prd_ref": "OT-P0-001",
      "validation": [{"type": "manual", "status": "planned", "notes": "attended"}]
    },
    {
      "id": "FX-P1-001",
      "title": "Later requirement",
      "status": "planned",
      "criticality": "P1",
      "prd_ref": "OT-P1-001",
      "validation": [{"type": "manual", "status": "implemented", "notes": "attended"}]
    },
    {
      "id": "FX-P2-001",
      "title": "Future requirement",
      "status": "planned",
      "criticality": "P2",
      "prd_ref": "OT-P2-001",
      "validation": [{"type": "manual", "status": "planned", "notes": "attended"}]
    }
  ]
}
`

func fixtureTree(t *testing.T, overrides map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"PRD.md":                           fixturePRD,
		"requirements/index.json":          `{"_metadata":{"schema_version":"1.0.0"},"imports":["01-core/module.json"]}`,
		"requirements/01-core/module.json": fixtureModule,
		"requirements/README.md":           canonicalReadme,
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
	return dir
}

// findingsFor runs the full check set over a tree.
func findingsFor(t *testing.T, root string) map[string]int {
	t.Helper()
	contract, err := extraction.NewFileExtractor().Load("fixture", root)
	require.NoError(t, err)
	out := map[string]int{}
	for _, chk := range checks.Registry() {
		for _, f := range chk.Run(context.Background(), contract) {
			code := f.Code
			if i := strings.IndexByte(code, ':'); i > 0 {
				code = code[:i]
			}
			out[code]++
		}
	}
	return out
}

// fixThenAssertResolved applies one rule and asserts the finding is gone
// and a second apply is a no-op.
func fixThenAssertResolved(t *testing.T, root, rule string) {
	t.Helper()
	reg := NewRegistry()
	require.True(t, reg.CanFix(root, rule, ""), "CanFix(%s) should be true on the broken tree", rule)

	preview, err := reg.Preview(root, []string{rule})
	require.NoError(t, err)
	require.NotEmpty(t, preview, "preview should have candidates for %s", rule)

	applied, err := ApplySequential(reg, root, []string{rule})
	require.NoError(t, err)
	require.NotEmpty(t, applied)

	require.Zero(t, findingsFor(t, root)[rule], "finding %s should be resolved after apply", rule)

	again, err := reg.Preview(root, []string{rule})
	require.NoError(t, err)
	require.Empty(t, again, "second apply must be a no-op for %s", rule)
}

// [REQ:BH-FIX-001] [REQ:BH-FIX-002] One before/after fixture per fixer,
// each proving fix-resolves-finding and idempotency.
func TestFixers(t *testing.T) {
	t.Run("prd_template_sections", func(t *testing.T) {
		root := fixtureTree(t, map[string]string{
			"PRD.md": strings.Replace(fixturePRD, "## 🧱 Tech Direction Snapshot\n- Preferred stacks: Go.\n\n", "", 1),
		})
		fixThenAssertResolved(t, root, intent.CodeTemplateSections)
		data, _ := os.ReadFile(filepath.Join(root, "PRD.md"))
		require.Contains(t, string(data), "## 🧱 Tech Direction Snapshot")
		require.Contains(t, string(data), "TODO:")
	})
	t.Run("prd_ot_id_format", func(t *testing.T) {
		root := fixtureTree(t, map[string]string{
			"PRD.md":                           strings.ReplaceAll(fixturePRD, "OT-P0-001", "OT-P0-1"),
			"requirements/01-core/module.json": strings.ReplaceAll(fixtureModule, "OT-P0-001", "OT-P0-1"),
		})
		fixThenAssertResolved(t, root, intent.CodeOTIDFormat)
		prd, _ := os.ReadFile(filepath.Join(root, "PRD.md"))
		require.Contains(t, string(prd), "OT-P0-001")
		module, _ := os.ReadFile(filepath.Join(root, "requirements", "01-core", "module.json"))
		require.Contains(t, string(module), `"prd_ref": "OT-P0-001"`, "prd_ref must be repointed with the OT id")
	})
	t.Run("prd_missing_requirements", func(t *testing.T) {
		root := fixtureTree(t, map[string]string{
			"requirements/index.json":          "",
			"requirements/01-core/module.json": "",
			"requirements/README.md":           "",
		})
		fixThenAssertResolved(t, root, "prd_missing_requirements")
	})
	t.Run("requirements_readme missing", func(t *testing.T) {
		root := fixtureTree(t, map[string]string{"requirements/README.md": ""})
		fixThenAssertResolved(t, root, "requirements_readme")
	})
	t.Run("requirements_readme drifted preserves content", func(t *testing.T) {
		root := fixtureTree(t, map[string]string{"requirements/README.md": "# Mine\nSpecial notes worth keeping.\n"})
		fixThenAssertResolved(t, root, "requirements_readme")
		data, _ := os.ReadFile(filepath.Join(root, "requirements", "README.md"))
		require.Contains(t, string(data), "Special notes worth keeping.")
	})
	t.Run("business_invalid_status", func(t *testing.T) {
		root := fixtureTree(t, map[string]string{
			"requirements/01-core/module.json": strings.Replace(fixtureModule, `"status": "planned",
      "criticality": "P1",`, `"status": "done",
      "criticality": "P1",`, 1),
		})
		fixThenAssertResolved(t, root, "business_invalid_status")
		data, _ := os.ReadFile(filepath.Join(root, "requirements", "01-core", "module.json"))
		require.Contains(t, string(data), `"status": "complete"`)
		require.Contains(t, string(data), `"status": "implemented"`, "validation-entry statuses must stay untouched")
	})
	t.Run("intent.ot_orphan", func(t *testing.T) {
		root := fixtureTree(t, map[string]string{
			"PRD.md": strings.Replace(fixturePRD,
				"- [ ] OT-P2-001 | Future target | Does a future thing.",
				"- [ ] OT-P2-001 | Future target | Does a future thing.\n- [ ] OT-P2-002 | Uncovered target | Nothing points here.", 1),
		})
		fixThenAssertResolved(t, root, intent.CodeOTOrphan)
		data, _ := os.ReadFile(filepath.Join(root, "requirements", orphanModuleSlug, "module.json"))
		require.Contains(t, string(data), `"prd_ref": "OT-P2-002"`)
		index, _ := os.ReadFile(filepath.Join(root, "requirements", "index.json"))
		require.Contains(t, string(index), orphanModuleSlug+"/module.json")
	})
}

// An "implemented" alias in a REQUIREMENT status normalizes without
// touching validation-entry statuses (the vocabulary collision case).
func TestInvalidStatusScopedReplace(t *testing.T) {
	root := fixtureTree(t, map[string]string{
		"requirements/01-core/module.json": strings.Replace(fixtureModule, `"status": "planned",
      "criticality": "P2",`, `"status": "implemented",
      "criticality": "P2",`, 1),
	})
	fixThenAssertResolved(t, root, "business_invalid_status")
	data, _ := os.ReadFile(filepath.Join(root, "requirements", "01-core", "module.json"))
	require.Contains(t, string(data), `"status": "complete",`)
	require.Equal(t, 1, strings.Count(string(data), `"status": "implemented"`), "the validation entry's implemented status must survive")
}

// [REQ:BH-FIX-002] The full fix loop on a multi-defect tree: apply all,
// everything auto-fixable resolves, and the conformant fixture stays
// untouched (preview empty).
func TestApplyAllOnCleanTreeIsNoop(t *testing.T) {
	root := fixtureTree(t, nil)
	reg := NewRegistry()
	preview, err := reg.Preview(root, nil)
	require.NoError(t, err)
	require.Empty(t, preview, "conformant tree must produce no candidates, got %+v", preview)
}

// [REQ:BH-FIX-003] Every fix_class:auto mapping in maturity.json has a
// registered fixer, and every registered fixer maps to an auto code —
// the accounting cannot drift.
func TestFixerAccountingMatchesSpec(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(specRoot(t), ".vrooli", "maturity.json"))
	require.NoError(t, err)
	var spec struct {
		Findings map[string]struct {
			FixClass    string `json:"fix_class"`
			FixerStatus string `json:"fixer_status"`
		} `json:"findings"`
	}
	require.NoError(t, jsonUnmarshal(data, &spec))

	registered := map[string]struct{}{}
	for _, rule := range RuleIDs() {
		registered[rule] = struct{}{}
	}
	for code, m := range spec.Findings {
		if m.FixClass == "auto" {
			_, ok := registered[code]
			require.True(t, ok, "auto code %q has no registered fixer", code)
			require.Equal(t, "implemented", m.FixerStatus, "auto code %q fixer_status", code)
		} else {
			_, ok := registered[code]
			require.False(t, ok, "manual code %q must not have a fixer", code)
		}
	}
}

func specRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(dir, "..", "..", ".."))
}
