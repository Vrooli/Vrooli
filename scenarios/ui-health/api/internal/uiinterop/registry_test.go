package uiinterop_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"ui-health/internal/uiinterop"
	_ "ui-health/internal/uiinterop/checks"
)

var expectedRegisteredRuleIDs = []string{
	"interop_api_base_dep",
	"interop_banned_scroll",
	"interop_bridge_app_id",
	"interop_bridge_init",
	"interop_capture_enabled",
	"interop_focus_visible_styles",
	"interop_h_screen",
	"interop_hardcoded_localhost",
	"interop_helmet_frame_ancestors",
	"interop_iframe_bridge_dep",
	"interop_iframe_guard",
	"interop_no_custom_server",
	"interop_no_scattered_keydown",
	"interop_protective_comments",
	"interop_proxy_base_preserved",
	"interop_relative_base",
	"interop_resolve_api_base_single",
	"interop_router_basename",
	"interop_secure_tunnel",
	"interop_shortcut_relay",
	"interop_spatial_nav_init",
	"interop_standard_server",
	"pwa_launch_scope",
	"pwa_manifest_install_fields",
	"pwa_optional_platform_fields",
	"pwa_service_worker_offline",
	"standard_a11y_harness",
	"standard_component_location",
	"standard_component_version_staleness",
	"standard_eslint_stability",
	"standard_i18n_locale_parity",
	"standard_no_raw_hex",
	"standard_pwa_manifest",
	"standard_raw_primitive_overuse",
	"standard_tsconfig_strict",
	"standard_unused_custom_component",
}

func TestParseRuleDoc(t *testing.T) {
	src := `package checks

/*
Rule: Demo rule
ID: interop_demo
Description: first line
  continuation line
Why: keeps embeds healthy
Category: interop
Severity: high
Slot: D
SlotFile: ui/src/main.tsx
TechStack: React, Vite
Recommendation: Fix it.
Standard: Project UI
GoodExample:
  good()
BadExample:
  bad()
*/
`
	def, ok := uiinterop.ParseRuleDoc(src)
	if !ok {
		t.Fatal("ParseRuleDoc returned ok=false")
	}
	if def.ID != "interop_demo" || def.Name != "Demo rule" || def.Description != "first line continuation line" {
		t.Fatalf("unexpected parsed def: %+v", def)
	}
	if !reflect.DeepEqual(def.TechStack, []string{"React", "Vite"}) {
		t.Fatalf("TechStack = %#v", def.TechStack)
	}
	if def.GoodExample != "good()" || def.BadExample != "bad()" {
		t.Fatalf("examples not dedented: good=%q bad=%q", def.GoodExample, def.BadExample)
	}
}

func TestParseRuleDocRequiresMarkers(t *testing.T) {
	for _, src := range []string{
		`package checks`,
		`/* Rule: Missing ID */`,
		`/* ID: missing_rule */`,
	} {
		if def, ok := uiinterop.ParseRuleDoc(src); ok {
			t.Fatalf("ParseRuleDoc(%q) = %+v, true; want false", src, def)
		}
	}
}

func TestRegisteredRuleIDSetIsComplete(t *testing.T) {
	var got []string
	for _, rule := range uiinterop.All() {
		got = append(got, rule.Def.ID)
	}
	sort.Strings(got)
	want := append([]string(nil), expectedRegisteredRuleIDs...)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("registered rule IDs drifted\n got: %v\nwant: %v", got, want)
	}
}

func TestRegisteredRulesHaveParsedMetadata(t *testing.T) {
	for _, rule := range uiinterop.All() {
		if rule.Def.Name == "" || rule.Def.Category == "" || rule.Def.Severity == "" || rule.Def.Recommendation == "" {
			t.Fatalf("rule %s is missing parsed metadata: %+v", rule.Def.ID, rule.Def)
		}
	}
}

func TestForTechStackFiltersRules(t *testing.T) {
	var hasReactOnly bool
	for _, rule := range uiinterop.ForTechStack([]string{"React"}) {
		if rule.Def.ID == "standard_pwa_manifest" {
			hasReactOnly = true
		}
		if rule.Def.ID == "interop_proxy_base_preserved" {
			t.Fatal("api-base-specific rule should not match a React-only stack")
		}
	}
	if !hasReactOnly {
		t.Fatal("React stack did not include standard_pwa_manifest")
	}
}

func TestSortDefsBySeverity(t *testing.T) {
	defs := []uiinterop.RuleDef{
		{ID: "low-b", Severity: "low"},
		{ID: "high-b", Severity: "high"},
		{ID: "critical", Severity: "critical"},
		{ID: "high-a", Severity: "high"},
	}
	uiinterop.SortDefsBySeverity(defs)
	got := []string{defs[0].ID, defs[1].ID, defs[2].ID, defs[3].ID}
	want := []string{"critical", "high-a", "high-b", "low-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorted IDs = %v, want %v", got, want)
	}
}

func TestWalkUISourceSetSplitsProductionAndTests(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "ui/src/App.tsx", "export function App() { return <main /> }")
	writeFile(t, root, "ui/src/App.test.tsx", "test('app', () => {})")
	writeFile(t, root, "ui/src/__fixtures__/Fixture.tsx", "ignored")
	writeFile(t, root, "ui/dist/bundle.js", "ignored")

	set := uiinterop.WalkUISourceSet(root, "ui")
	if len(set.Production) != 1 || set.Production[0].RelPath != "ui/src/App.tsx" {
		t.Fatalf("production sources = %+v", set.Production)
	}
	if len(set.Tests) != 1 || set.Tests[0].RelPath != "ui/src/App.test.tsx" {
		t.Fatalf("test sources = %+v", set.Tests)
	}
}

func TestRunAllUsesPrewalkedSourceSet(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "ui/package.json", `{"dependencies":{"react":"latest","vite":"latest"}}`)
	writeFile(t, root, "ui/src/App.tsx", `export function App() { return <main className="h-screen" /> }`)

	results := uiinterop.RunAll(root, "demo")
	for _, result := range results {
		if result.RuleID == "interop_h_screen" {
			if result.Passed || len(result.Violations) == 0 {
				t.Fatalf("interop_h_screen should flag the prewalked source file: %+v", result)
			}
			return
		}
	}
	t.Fatal("interop_h_screen did not run")
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", rel, err)
	}
}
