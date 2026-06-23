package autofix

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	_ "ui-health/internal/uiinterop/checks" // register standard rules for RunAll
)

func TestStandardFixClassFor(t *testing.T) {
	for _, code := range []string{RuleStandardTSConfigStrict, RuleStandardI18nLocaleParity} {
		if FixClassFor(code) != "autofix" {
			t.Fatalf("FixClassFor(%q) is not autofix", code)
		}
	}
	// The report-only standards must stay detection_only.
	for _, code := range []string{"standard_no_raw_hex", "standard_a11y_harness", "standard_pwa_manifest", "standard_eslint_stability"} {
		if FixClassFor(code) == "autofix" {
			t.Fatalf("%q must be detection_only, not autofix", code)
		}
	}
}

func TestTSConfigStrictFixFlipsAndIsIdempotent(t *testing.T) {
	root := interopScenario(t)
	writeFile(t, root, "ui/tsconfig.json", `{
  "compilerOptions": { "strict": false, "noEmit": true }
}`)
	abs := filepath.Join(root, "ui", "tsconfig.json")

	f := New(noopValidator{})

	if !f.CanFix(root, RuleStandardTSConfigStrict, abs) {
		t.Fatal("CanFix should be true for a tsconfig with strict:false")
	}

	// Preview must not write.
	if _, err := f.PreviewFixResponse("demo", root, []string{RuleStandardTSConfigStrict}); err != nil {
		t.Fatalf("preview: %v", err)
	}
	if strings.Contains(readFile(t, root, "ui/tsconfig.json"), `"strict": true`) {
		t.Fatal("preview must not modify the file")
	}

	resp, err := f.ApplyFixResponse("demo", root, []string{RuleStandardTSConfigStrict})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(resp.GetCandidates()) != 1 {
		t.Fatalf("apply candidates=%d, want 1", len(resp.GetCandidates()))
	}
	got := readFile(t, root, "ui/tsconfig.json")
	if !strings.Contains(got, `"strict": true`) || strings.Contains(got, `"strict": false`) {
		t.Fatalf("after fix, strict not flipped to true: %s", got)
	}

	// Idempotent.
	resp2, err := f.ApplyFixResponse("demo", root, []string{RuleStandardTSConfigStrict})
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if len(resp2.GetCandidates()) != 0 {
		t.Fatalf("re-apply candidates=%d, want 0 (idempotent)", len(resp2.GetCandidates()))
	}
	if f.CanFix(root, RuleStandardTSConfigStrict, abs) {
		t.Fatal("CanFix must be false once strict is true")
	}
}

// A missing strict flag is detected by the rule but is NOT safely insertable, so
// the fixer reports it as not fixable (no over-claim).
func TestTSConfigStrictFixSkipsMissingFlag(t *testing.T) {
	root := interopScenario(t)
	writeFile(t, root, "ui/tsconfig.json", `{ "compilerOptions": { "noEmit": true } }`)
	abs := filepath.Join(root, "ui", "tsconfig.json")

	f := New(noopValidator{})
	if f.CanFix(root, RuleStandardTSConfigStrict, abs) {
		t.Fatal("CanFix must be false when strict is absent (insertion is not a safe mechanical edit)")
	}
	resp, err := f.ApplyFixResponse("demo", root, []string{RuleStandardTSConfigStrict})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(resp.GetCandidates()) != 0 {
		t.Fatalf("apply candidates=%d, want 0 for a missing flag", len(resp.GetCandidates()))
	}
}

func TestI18nLocaleParityFixScaffoldsAndIsIdempotent(t *testing.T) {
	root := interopScenario(t)
	writeFile(t, root, "ui/src/i18n/locales/en.json", `{"greeting":"Hello","nav":{"home":"Home","away":"Away"}}`)
	writeFile(t, root, "ui/src/i18n/locales/ja.json", `{"greeting":"こんにちは","nav":{"home":"ホーム"}}`)
	abs := filepath.Join(root, "ui", "src", "i18n", "locales", "ja.json")

	f := New(noopValidator{})

	if !f.CanFix(root, RuleStandardI18nLocaleParity, abs) {
		t.Fatal("CanFix should be true for a locale missing keys")
	}

	resp, err := f.ApplyFixResponse("demo", root, []string{RuleStandardI18nLocaleParity})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(resp.GetCandidates()) != 1 {
		t.Fatalf("apply candidates=%d, want 1", len(resp.GetCandidates()))
	}

	// The missing nested key was scaffolded with the English placeholder.
	var ja map[string]any
	if err := json.Unmarshal([]byte(readFile(t, root, "ui/src/i18n/locales/ja.json")), &ja); err != nil {
		t.Fatalf("ja.json not valid JSON after fix: %v", err)
	}
	nav, ok := ja["nav"].(map[string]any)
	if !ok {
		t.Fatalf("nav missing/!object after fix: %v", ja["nav"])
	}
	if nav["away"] != "Away" {
		t.Fatalf("missing key nav.away not scaffolded with English placeholder; got %v", nav["away"])
	}
	// The pre-existing translation was preserved (not overwritten by en).
	if nav["home"] != "ホーム" {
		t.Fatalf("existing translation nav.home overwritten; got %v", nav["home"])
	}

	// Idempotent.
	resp2, err := f.ApplyFixResponse("demo", root, []string{RuleStandardI18nLocaleParity})
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if len(resp2.GetCandidates()) != 0 {
		t.Fatalf("re-apply candidates=%d, want 0 (idempotent)", len(resp2.GetCandidates()))
	}
	if f.CanFix(root, RuleStandardI18nLocaleParity, abs) {
		t.Fatal("CanFix must be false once parity is restored")
	}
}

// A locale whose only drift is orphan (extra) keys is detected by the rule but
// the fixer does not delete translations, so it reports not-fixable.
func TestI18nLocaleParityFixLeavesOrphanKeys(t *testing.T) {
	root := interopScenario(t)
	writeFile(t, root, "ui/src/i18n/locales/en.json", `{"greeting":"Hello"}`)
	writeFile(t, root, "ui/src/i18n/locales/ja.json", `{"greeting":"こんにちは","stale":"古い"}`)
	abs := filepath.Join(root, "ui", "src", "i18n", "locales", "ja.json")

	f := New(noopValidator{})
	if f.CanFix(root, RuleStandardI18nLocaleParity, abs) {
		t.Fatal("CanFix must be false when the only drift is orphan keys (deleting translations is not safe)")
	}
	resp, err := f.ApplyFixResponse("demo", root, []string{RuleStandardI18nLocaleParity})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(resp.GetCandidates()) != 0 {
		t.Fatalf("apply candidates=%d, want 0 (orphan-only is report-only)", len(resp.GetCandidates()))
	}
}
