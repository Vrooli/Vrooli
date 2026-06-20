package autofix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// serviceJSON declares api+ui surfaces with a wrong name, no http health check,
// and no freshness checks — so SERVICE_NAME_MISMATCH, HEALTH_CHECK_MISSING and
// FRESHNESS_CHECK_MISSING are all auto-fixable. It carries an unknown field and a
// deliberate key order to prove edits are format-preserving.
const serviceJSON = `{
  "$schema": "../schema.json",
  "version": "1.0.0",
  "service": {
    "name": "wrong-name",
    "customField": "preserved"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "range": "20000-24999"
    }
  },
  "lifecycle": {
    "setup": {
      "steps": []
    },
    "health": {
      "endpoints": {
        "api": "/health"
      }
    }
  }
}
`

func writeScenario(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return root
}

func ruleSet(candidates []Candidate) map[string]bool {
	out := map[string]bool{}
	for _, c := range candidates {
		out[c.RuleID] = true
	}
	return out
}

// [REQ:SH-FIX-001] [REQ:SH-FIX-004]
func TestPreviewIsDryRunAndProducesFixes(t *testing.T) {
	root := writeScenario(t, serviceJSON)
	path := filepath.Join(root, ".vrooli", "service.json")
	before, _ := os.ReadFile(path)

	candidates, err := Preview(root, nil)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	got := ruleSet(candidates)
	for _, want := range []string{RuleServiceNameMismatch, RuleHealthCheckMissing, RuleFreshnessMissing} {
		if !got[want] {
			t.Fatalf("preview missing candidate %s; got %v", want, got)
		}
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("preview must not modify disk")
	}
}

// [REQ:SH-FIX-001]
func TestApplyComposesServiceJSONEditsAndIsIdempotent(t *testing.T) {
	root := writeScenario(t, serviceJSON)
	path := filepath.Join(root, ".vrooli", "service.json")

	applied, err := Apply(root, nil)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected applied candidates")
	}
	for _, c := range applied {
		if !c.Applied {
			t.Fatalf("candidate %s not marked applied", c.RuleID)
		}
	}

	raw, _ := os.ReadFile(path)
	out := string(raw)
	// All three service.json edits composed into one file (none clobbered).
	if !strings.Contains(out, `"name": "`+filepath.Base(root)+`"`) {
		t.Fatalf("name not fixed:\n%s", out)
	}
	if !strings.Contains(out, `"type": "http"`) {
		t.Fatalf("health check not added:\n%s", out)
	}
	if !strings.Contains(out, `"type": "binaries"`) || !strings.Contains(out, `"type": "ui-bundle"`) {
		t.Fatalf("freshness checks not added:\n%s", out)
	}
	// Unknown field preserved.
	if !strings.Contains(out, `"customField": "preserved"`) {
		t.Fatalf("unknown field dropped:\n%s", out)
	}
	// Result still parses as valid JSON.
	var sink map[string]any
	if err := json.Unmarshal(raw, &sink); err != nil {
		t.Fatalf("apply produced invalid json: %v", err)
	}

	// Idempotency: a second apply is a no-op.
	again, err := Apply(root, nil)
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("expected no further candidates on re-apply, got %d", len(again))
	}
	raw2, _ := os.ReadFile(path)
	if string(raw2) != out {
		t.Fatalf("re-apply changed the file")
	}
}

// [REQ:SH-FIX-003]
func TestApplyRuleFilterScopesEdits(t *testing.T) {
	root := writeScenario(t, serviceJSON)
	applied, err := Apply(root, []string{RuleServiceNameMismatch})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(applied) != 1 || applied[0].RuleID != RuleServiceNameMismatch {
		t.Fatalf("rule filter not honored: %+v", applied)
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
	if strings.Contains(string(raw), `"type": "http"`) {
		t.Fatalf("health check applied despite rule filter")
	}
}

// [REQ:SH-FIX-003]
func TestApplyCreatesMissingSurfaceDirs(t *testing.T) {
	root := writeScenario(t, serviceJSON)
	applied, err := Apply(root, []string{RuleSurfaceDirMissing})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(applied) != 2 {
		t.Fatalf("expected api+ui surface dirs, got %d: %+v", len(applied), applied)
	}
	for _, dir := range []string{"api", "ui"} {
		if fi, err := os.Stat(filepath.Join(root, dir)); err != nil || !fi.IsDir() {
			t.Fatalf("surface dir %s not created: %v", dir, err)
		}
	}
}

// [REQ:SH-FIX-003]
func TestApplyCreatesReadmeNotMakefile(t *testing.T) {
	root := writeScenario(t, serviceJSON)
	applied, err := Apply(root, []string{RuleRequiredFileMissing})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("expected exactly the README candidate, got %d", len(applied))
	}
	if _, err := os.Stat(filepath.Join(root, "README.md")); err != nil {
		t.Fatalf("README.md not created: %v", err)
	}
	// The Makefile is intentionally not auto-generated.
	if _, err := os.Stat(filepath.Join(root, "Makefile")); err == nil {
		t.Fatalf("Makefile must not be auto-generated")
	}
}

// [REQ:SH-FIX-003]
func TestFixClassFor(t *testing.T) {
	for _, code := range []string{RuleServiceNameMismatch, RuleHealthCheckMissing, RuleFreshnessMissing, RuleSurfaceDirMissing, RuleRequiredFileMissing} {
		if !FixClassFor(code).Autofixable() {
			t.Fatalf("%s should be autofixable", code)
		}
	}
	if FixClassFor("PROFILE_PORTS").Autofixable() {
		t.Fatal("PROFILE_PORTS should be detection_only")
	}
	if FixClassFor("SERVICE_JSON_MISSING").Autofixable() {
		t.Fatal("SERVICE_JSON_MISSING should be detection_only")
	}
}

// [REQ:SH-FIX-003]
func TestNoFixesWhenConformant(t *testing.T) {
	// A scenario whose name matches its dir, with no declared surfaces, has no
	// auto-fixable findings.
	root := t.TempDir()
	body := `{
  "service": {
    "name": "` + filepath.Base(root) + `"
  }
}
`
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// README is the only universal file fix; create it so nothing remains.
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	candidates, err := Preview(root, nil)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates, got %+v", candidates)
	}
}
