package autofix

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ui-health/internal/services/manifestvalidation"

	_ "ui-health/internal/uiinterop/checks" // register interop rules for RunAll
)

// noopValidator satisfies the Validator seam for interop-only tests; the interop
// fixers derive their candidates from the interop engine, not the manifest
// validator, so it returns an empty report.
type noopValidator struct{}

func (noopValidator) ValidateScenario(_ context.Context, _ string) (manifestvalidation.Report, error) {
	return manifestvalidation.Report{}, nil
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// interopScenario builds a minimal UI scenario whose package.json declares the
// deps the interop rules gate on (React, Vite, iframe-bridge).
func interopScenario(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "ui/package.json", `{
  "dependencies": { "react": "18", "react-dom": "18", "@vrooli/iframe-bridge": "1" },
  "devDependencies": { "vite": "5" }
}`)
	return root
}

func TestInteropFixClassFor(t *testing.T) {
	for _, code := range []string{RuleInteropHScreen, RuleInteropProtectiveComments} {
		if FixClassFor(code) != "autofix" {
			t.Fatalf("FixClassFor(%q) is not autofix", code)
		}
	}
	// A detection-only interop rule must not be autofixable.
	if FixClassFor("interop_banned_scroll") == "autofix" {
		t.Fatal("interop_banned_scroll must be detection_only, not autofix")
	}
}

func TestHScreenFixRewritesAndIsIdempotent(t *testing.T) {
	root := interopScenario(t)
	writeFile(t, root, "ui/src/App.tsx", `export const App = () => <div className="h-screen w-screen min-h-screen">x</div>;`)
	abs := filepath.Join(root, "ui", "src", "App.tsx")

	f := New(noopValidator{})

	// CanFix is true while the file still has viewport units.
	if !f.CanFix(root, RuleInteropHScreen, abs) {
		t.Fatal("CanFix should be true for a file containing h-screen")
	}

	// Preview must not write.
	if _, err := f.PreviewFixResponse("demo", root, []string{RuleInteropHScreen}); err != nil {
		t.Fatalf("preview: %v", err)
	}
	if strings.Contains(readFile(t, root, "ui/src/App.tsx"), "h-full") {
		t.Fatal("preview must not modify the file")
	}

	// Apply rewrites the tokens.
	resp, err := f.ApplyFixResponse("demo", root, []string{RuleInteropHScreen})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(resp.GetCandidates()) != 1 {
		t.Fatalf("apply candidates=%d, want 1", len(resp.GetCandidates()))
	}
	got := readFile(t, root, "ui/src/App.tsx")
	for _, banned := range []string{"h-screen", "w-screen"} {
		if strings.Contains(got, banned) {
			t.Fatalf("after fix, file still contains %q: %s", banned, got)
		}
	}
	if !strings.Contains(got, "h-full") || !strings.Contains(got, "w-full") || !strings.Contains(got, "min-h-full") {
		t.Fatalf("after fix, expected container-relative classes: %s", got)
	}

	// Idempotent: re-apply yields no candidates and CanFix is now false.
	resp2, err := f.ApplyFixResponse("demo", root, []string{RuleInteropHScreen})
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if len(resp2.GetCandidates()) != 0 {
		t.Fatalf("re-apply candidates=%d, want 0 (idempotent)", len(resp2.GetCandidates()))
	}
	if f.CanFix(root, RuleInteropHScreen, abs) {
		t.Fatal("CanFix must be false once viewport units are gone")
	}
}

func TestProtectiveCommentFixInsertsBanner(t *testing.T) {
	root := interopScenario(t)
	writeFile(t, root, "ui/vite.config.ts", `export default defineConfig({ base: "/" });`)
	writeFile(t, root, "ui/src/main.tsx", `import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
initIframeBridgeChild();`)

	f := New(noopValidator{})

	resp, err := f.ApplyFixResponse("demo", root, []string{RuleInteropProtectiveComments})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(resp.GetCandidates()) != 2 {
		t.Fatalf("apply candidates=%d, want 2 (vite config + main entry)", len(resp.GetCandidates()))
	}
	for _, rel := range []string{"ui/vite.config.ts", "ui/src/main.tsx"} {
		if !strings.Contains(readFile(t, root, rel), "INTEROP-CRITICAL") {
			t.Fatalf("%s missing INTEROP-CRITICAL after fix", rel)
		}
	}

	// Idempotent.
	resp2, err := f.ApplyFixResponse("demo", root, []string{RuleInteropProtectiveComments})
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if len(resp2.GetCandidates()) != 0 {
		t.Fatalf("re-apply candidates=%d, want 0 (idempotent)", len(resp2.GetCandidates()))
	}
}
