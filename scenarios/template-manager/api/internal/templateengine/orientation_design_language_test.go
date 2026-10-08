package templateengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func TestDesignLanguageRejectsMarkerOnlyCompletion(t *testing.T) {
	const stockShell = `export function AppShell(){return <LibraryShell navigation="sidebar"/>}`
	const chosenShell = `export function AppShell(){return <LibraryShell navigation="rail"/>}`
	const stockKit = `:root { --color-background: #fff; --font-body: system-ui; }`
	const chosenKit = `:root { --color-background: #f9f5ed; --font-body: system-ui; }`
	for _, tc := range []struct {
		name, home, shell, kit string
		pass                   bool
		message                string
	}{
		{name: "untouched", home: "// PLACEHOLDER:home-surface", shell: stockShell, kit: stockKit, message: "home surface still contains"},
		{name: "marker removed", home: "export const DashboardPage=()=> <main/>", shell: stockShell, kit: stockKit, message: "shell configuration is unchanged"},
		{name: "shell changed but stock kit", home: "export const DashboardPage=()=> <main/>", shell: chosenShell, kit: stockKit, message: "design kit color and font values still equal stock"},
		{name: "comments do not choose kit", home: "export const DashboardPage=()=> <main/>", shell: chosenShell, kit: stockKit + "/* chosen brand */", message: "design kit color and font values still equal stock"},
		{name: "deleting values does not choose kit", home: "export const DashboardPage=()=> <main/>", shell: chosenShell, kit: `:root { --color-background: #f9f5ed; }`, message: "required stock values are missing"},
		{name: "adapted", home: "export const DashboardPage=()=> <main>Inbox</main>", shell: chosenShell, kit: chosenKit, pass: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scenario, template := t.TempDir(), t.TempDir()
			write := func(root, path, body string) {
				t.Helper()
				target := filepath.Join(root, path)
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, []byte(body), 0644); err != nil {
					t.Fatal(err)
				}
			}
			write(template, "ui/src/layout/AppShell.tsx", stockShell)
			write(template, "ui/src/design-tokens.css", stockKit)
			write(scenario, "DESIGN.md", "Chosen product design")
			write(scenario, "ui/src/pages/DashboardPage.tsx", tc.home)
			write(scenario, "ui/src/layout/AppShell.tsx", tc.shell)
			write(scenario, "ui/src/design-tokens.css", tc.kit)
			// Deliberately use the old manifest: new generated checks must not be a
			// prerequisite for enforcement on the already-generated fleet.
			step := templatecontracts.TemplateOrientationStep{ID: "design-language", Checks: []templatecontracts.TemplateOrientationCheck{{Kind: "file_exists", Path: "DESIGN.md"}}}
			result := evaluateOrientationStep(HandlerDeps[struct{}]{}, struct{}{}, orientationEval{scenarioRoot: scenario, templateSourceRoot: template}, step)
			if result.Complete != tc.pass {
				t.Fatalf("complete=%v, want %v: %+v", result.Complete, tc.pass, result.Checks)
			}
			if len(result.Checks) != 4 {
				t.Fatalf("engine-derived checks missing: %+v", result.Checks)
			}
			messages := ""
			for _, check := range result.Checks {
				messages += check.Message + "\n"
			}
			if tc.message != "" && !strings.Contains(messages, tc.message) {
				t.Fatalf("missing diagnostic %q: %s", tc.message, messages)
			}
		})
	}
}

// Opt-in real-source verification keeps ordinary unit tests isolated from the
// changing fleet. The scenario under review must satisfy all source conditions.
func TestDesignLanguageCurrentProduct(t *testing.T) {
	root := os.Getenv("TEMPLATE_DESIGN_SOURCE_ROOT")
	scenario := os.Getenv("TEMPLATE_DESIGN_SCENARIO")
	if root == "" || scenario == "" {
		t.Skip("set TEMPLATE_DESIGN_SOURCE_ROOT and TEMPLATE_DESIGN_SCENARIO for current-source verification")
	}
	checks := evaluateDesignLanguageSource(orientationEval{scenarioRoot: filepath.Join(root, "scenarios", scenario), templateSourceRoot: filepath.Join(root, "templates/scenarios/react-vite")})
	if len(checks) != 3 {
		t.Fatalf("missing source conditions: %+v", checks)
	}
	for _, check := range checks {
		t.Logf("%s: passed=%v %s", check.Kind, check.Passed, check.Message)
		if !check.Passed {
			t.Errorf("%s: %s", check.Label, check.Message)
		}
	}
}
