package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile writes content to root/rel, creating parents.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunInteropFindingsFlagsMissingBridgeDeps(t *testing.T) {
	root := t.TempDir()
	// A ui/ surface whose package.json is missing the interop deps triggers the
	// dependency-presence interop rules.
	writeFile(t, root, "ui/package.json", `{"name":"demo","dependencies":{"react":"^18.0.0"}}`)

	finds := runInteropFindings(root, "demo")
	if len(finds) == 0 {
		t.Fatal("expected interop findings for a ui/ surface missing bridge deps, got none")
	}
	seen := map[string]bool{}
	for _, f := range finds {
		seen[f.Code] = true
	}
	for _, want := range []string{"interop_api_base_dep", "interop_iframe_bridge_dep"} {
		if !seen[want] {
			t.Errorf("expected interop finding %q in %v", want, seen)
		}
	}
}

func TestRunInteropFindingsSkipsNonUIScenario(t *testing.T) {
	root := t.TempDir() // no ui/ directory at all
	if finds := runInteropFindings(root, "demo"); len(finds) != 0 {
		t.Fatalf("expected no interop findings for a scenario with no ui/ surface, got %d: %v", len(finds), finds)
	}
}

// These three scenario fixtures exercise the ordinary validation adapter, not
// just a direct rule call. Shell forks must remain errors in the unified report.
func TestRunInteropFindingsShellOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, source, ejection string
		wantError              bool
	}{
		{name: "library", source: `import {AppShell as LibraryShell} from '@vrooli/react-component-library/AppShell/2'; export function AppShell(){return <LibraryShell/>}`},
		{name: "fork", source: `export function AppShell(){return <nav><a href="/">Home</a></nav>}`, wantError: true},
		{name: "ejected", source: `export function AppShell(){return <nav><a href="/">Home</a></nav>}`, ejection: "```shell-ejection\n{\"archetype\":\"navigated-console\",\"reason\":\"Fixture product gap\",\"files\":[\"ui/src/features/Console.tsx\"]}\n```"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "ui/package.json", `{"dependencies":{"react":"^18.0.0"}}`)
			writeFile(t, root, "ui/manifest.json", `{"shell":{"archetype":"navigated-console","asset":"AppShell","entry":"ui/src/features/Console.tsx","export":"AppShell"}}`)
			writeFile(t, root, "ui/src/features/Console.tsx", tc.source)
			if tc.ejection != "" {
				writeFile(t, root, "docs/reference/component-library-gaps.md", tc.ejection)
			}
			found := false
			for _, finding := range runInteropFindings(root, tc.name) {
				if finding.Code != "standard_shell_ownership" {
					continue
				}
				found = true
				if finding.Severity != "error" {
					t.Fatalf("shell violation downgraded: %+v", finding)
				}
				if !strings.Contains(finding.Location, "ui/src/features/Console.tsx") || !strings.Contains(finding.Message, "AppShell") && !strings.Contains(finding.Message, "<nav>") {
					t.Fatalf("file or element lost: %+v", finding)
				}
			}
			if found != tc.wantError {
				t.Fatalf("shell error=%v, want %v", found, tc.wantError)
			}
		})
	}
}
