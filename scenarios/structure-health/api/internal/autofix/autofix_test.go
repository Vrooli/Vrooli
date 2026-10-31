package autofix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// serviceJSON declares api+ui surfaces with a wrong name and no http health
// check. It carries an unknown field and deliberate key order to prove edits
// are format-preserving.
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

// serviceJSONBandBinaryServe declares a canonical API port off its band with
// the wrong env var and otherwise unrelated component commands. A deliberate
// key order plus unknown field prove focused edits are format-preserving.
const serviceJSONBandBinaryServe = `{
  "service": {
    "name": "demo",
    "customField": "preserved"
  },
  "ports": {
    "api": {
      "env_var": "PORT",
      "range": "10000-12000"
    },
    "ui": {
      "env_var": "UI_PORT",
      "range": "20000-24999"
    }
  },
  "components": {
    "api": {
      "role": "api",
      "build": {"kind": "go_module", "dir": "api"},
      "run": {"argv": ["./wrong"], "cwd": "api", "env": {"DB": "1"}, "port": "api"}
    },
    "ui": {
      "role": "ui",
      "build": {"kind": "pnpm_vite", "dir": "ui"},
      "run": {"argv": ["pnpm", "dev"], "cwd": "ui", "port": "ui"}
    }
  }
}
`

const serviceJSONMalformedHealth = `{
  "service": {
    "name": "demo"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    }
  },
  "lifecycle": {
    "health": {
      "checks": [
        {
          "type": "http",
          "target": "http://localhost:${API_PORT}/api/v1/health",
          "critical": false
        }
      ]
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
	for _, want := range []string{RuleServiceNameMismatch, RuleHealthCheckMissing} {
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
	// Both service.json edits composed into one file (neither clobbered).
	if !strings.Contains(out, `"name": "`+filepath.Base(root)+`"`) {
		t.Fatalf("name not fixed:\n%s", out)
	}
	if !strings.Contains(out, `"type": "http"`) {
		t.Fatalf("health check not added:\n%s", out)
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
func TestApplyNormalizesMalformedHealthCheck(t *testing.T) {
	root := writeScenario(t, serviceJSONMalformedHealth)
	applied, err := Apply(root, []string{RuleHealthCheckMalformed})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(applied) != 1 || applied[0].RuleID != RuleHealthCheckMalformed {
		t.Fatalf("expected malformed-health candidate, got %+v", applied)
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
	out := string(raw)
	for _, want := range []string{
		`"name": "api_endpoint"`,
		`"target": "http://localhost:${API_PORT}/health"`,
		`"critical": true`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("health fix missing %q:\n%s", want, out)
		}
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

var newFixerRules = []string{RulePortBand}

// [REQ:SH-FIX-002]
func TestNewFixersPreviewProducesCandidates(t *testing.T) {
	root := writeScenario(t, serviceJSONBandBinaryServe)
	candidates, err := Preview(root, newFixerRules)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	got := ruleSet(candidates)
	for _, want := range newFixerRules {
		if !got[want] {
			t.Fatalf("preview missing candidate %s; got %v", want, got)
		}
	}
}

// [REQ:SH-FIX-001] [REQ:SH-FIX-002]
func TestNewFixersApplyComposeAndAreIdempotent(t *testing.T) {
	root := writeScenario(t, serviceJSONBandBinaryServe)
	path := filepath.Join(root, ".vrooli", "service.json")

	applied, err := Apply(root, newFixerRules)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(applied) != len(newFixerRules) {
		t.Fatalf("expected %d applied candidates, got %d: %+v", len(newFixerRules), len(applied), applied)
	}

	raw, _ := os.ReadFile(path)
	out := string(raw)
	for _, want := range []string{
		`"env_var": "API_PORT"`,
		`"range": "15000-19999"`,
		`"customField": "preserved"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("apply result missing %q:\n%s", want, out)
		}
	}
	for _, gone := range []string{`"range": "10000-12000"`, `"env_var": "PORT"`} {
		if strings.Contains(out, gone) {
			t.Fatalf("apply result still contains %q:\n%s", gone, out)
		}
	}
	var sink map[string]any
	if err := json.Unmarshal(raw, &sink); err != nil {
		t.Fatalf("apply produced invalid json: %v", err)
	}
	again, err := Apply(root, newFixerRules)
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("expected no further candidates on re-apply, got %+v", again)
	}
	raw2, _ := os.ReadFile(path)
	if string(raw2) != out {
		t.Fatalf("re-apply changed the file")
	}
}

// [REQ:SH-FIX-003]
func TestPortBandFilterLeavesComponentsUntouched(t *testing.T) {
	root := writeScenario(t, serviceJSONBandBinaryServe)
	applied, err := Apply(root, []string{RulePortBand})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(applied) != 1 || applied[0].RuleID != RulePortBand {
		t.Fatalf("rule filter not honored: %+v", applied)
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
	out := string(raw)
	if !strings.Contains(out, `"range": "15000-19999"`) {
		t.Fatalf("port band not fixed:\n%s", out)
	}
	// Port correction preserves unrelated manifest values. Compare decoded
	// subtrees so the assertion does not depend on array line wrapping.
	var before, after map[string]any
	if err := json.Unmarshal([]byte(serviceJSONBandBinaryServe), &before); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if err := json.Unmarshal(raw, &after); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	for _, key := range []string{"service", "components"} {
		if !reflect.DeepEqual(after[key], before[key]) {
			t.Fatalf("%s changed during port correction:\nbefore=%v\nafter=%v", key, before[key], after[key])
		}
	}
}

// [REQ:SH-FIX-003]
func TestFixClassFor(t *testing.T) {
	for _, code := range []string{RuleServiceNameMismatch, RuleHealthCheckMissing, RuleHealthCheckMalformed, RuleSurfaceDirMissing, RuleRequiredFileMissing, RulePortBand, RuleProjectConfigSurface} {
		if !FixClassFor(code).Autofixable() {
			t.Fatalf("%s should be autofixable", code)
		}
	}
	// The coarse profile-pack codes stay detection_only — their findings bundle
	// many non-fixable violations; the focused universal codes above carry the
	// precise autofix signal instead.
	if FixClassFor("PROFILE_PORTS").Autofixable() {
		t.Fatal("PROFILE_PORTS should be detection_only")
	}
	if FixClassFor("SERVICE_JSON_MISSING").Autofixable() {
		t.Fatal("SERVICE_JSON_MISSING should be detection_only")
	}
}

func TestProjectConfigSurfacePreviewApplyIsDeterministic(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	contract := `{"layout":{"project_config_dir":".vrooli","project_config_allowlist":["repo-contract.json"]}}` + "\n"
	path := filepath.Join(configDir, "repo-contract.json")
	if err := os.WriteFile(path, []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "operator-state.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	candidates, err := Preview(root, []string{RuleProjectConfigSurface})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(candidates) != 1 || !strings.Contains(candidates[0].After, `"operator-state.json"`) {
		t.Fatalf("unexpected project candidate: %+v", candidates)
	}
	if applied, err := Apply(root, []string{RuleProjectConfigSurface}); err != nil || len(applied) != 1 {
		t.Fatalf("apply = %+v, err = %v", applied, err)
	}
	if again, err := Apply(root, []string{RuleProjectConfigSurface}); err != nil || len(again) != 0 {
		t.Fatalf("re-apply = %+v, err = %v", again, err)
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
