package autofix

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"performance-health/internal/readiness"
)

// stubFacts is a readiness.FactsClient that reports a react-vite scenario rooted
// at the given path, so the autofix test can cross-check the detector.
type stubFacts struct{ root string }

func (s stubFacts) Describe(context.Context, string, string) (readiness.Facts, error) {
	return readiness.Facts{
		Scenario:    "bare",
		Surfaces:    []string{"ui"},
		UIFramework: "react-vite",
		RootPath:    s.root,
	}, nil
}

func testCtx() context.Context { return context.Background() }

func TestPreviewRequiresRoot(t *testing.T) {
	svc := NewService()
	if _, err := svc.Preview("", nil); err == nil {
		t.Fatal("expected error for empty root")
	}
}

// [REQ:PH-TIER-003] Preview reports the four perf-build fixes for a bare
// react-vite tree without writing anything to disk.
func TestPreviewBareReactViteIsDryRun(t *testing.T) {
	root := writeBareReactVite(t)
	before := snapshot(t, root)

	svc := NewService()
	candidates, err := svc.Preview(root, nil)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if len(candidates) != 4 {
		t.Fatalf("expected 4 fix candidates, got %d: %+v", len(candidates), ruleIDs(candidates))
	}
	for _, c := range candidates {
		if c.Applied {
			t.Fatalf("preview candidate %q must not be marked applied", c.RuleID)
		}
		if c.After == c.Before {
			t.Fatalf("candidate %q has no diff", c.RuleID)
		}
	}
	if after := snapshot(t, root); after != before {
		t.Fatal("Preview must not modify the tree")
	}
}

// [REQ:PH-TIER-003] Apply writes all four pieces and is idempotent: a second
// Apply finds nothing left to do and the readiness detector now reports the
// scenario fully instrumented.
func TestApplyWritesAndIsIdempotent(t *testing.T) {
	root := writeBareReactVite(t)

	svc := NewService()
	first, err := svc.Apply(root, nil)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(first) != 4 {
		t.Fatalf("first Apply should write 4 pieces, got %d", len(first))
	}
	for _, c := range first {
		if !c.Applied {
			t.Fatalf("candidate %q should be marked applied", c.RuleID)
		}
		if _, err := os.Stat(c.FilePath); err != nil {
			t.Fatalf("expected %s written: %v", c.FilePath, err)
		}
	}

	// Idempotent: a second apply over the fixed tree is a no-op.
	second, err := svc.Apply(root, nil)
	if err != nil {
		t.Fatalf("Apply (2nd): %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second Apply should be a no-op, got %d: %v", len(second), ruleIDs(second))
	}

	// The boundary edit must add the React + profiler imports, not just the
	// JSX wrapper (otherwise the file references undefined symbols).
	mainTSX := readFile(t, filepath.Join(root, "ui", "src", "main.tsx"))
	if !contains(mainTSX, `import React from "react"`) {
		t.Fatal("apply must add the React import for <React.Profiler>")
	}
	if !contains(mainTSX, `import { onProfilerRender } from "./lib/profiler"`) {
		t.Fatal("apply must add the onProfilerRender import")
	}

	// The build:profile script must keep shell operators verbatim (no HTML
	// escaping of && / $).
	pkg := readFile(t, filepath.Join(root, "ui", "package.json"))
	if contains(pkg, "\\u0026") {
		t.Fatal("package.json scripts must not HTML-escape shell operators (&&)")
	}
	if !contains(pkg, "--mode profile") {
		t.Fatal("package.json must gain a build:profile / mode-aware build")
	}

	// Cross-check against the readiness detector via the shared rule IDs.
	res, err := readiness.NewService(stubFacts{root}).Validate(testCtx(), "bare", "")
	if err != nil {
		t.Fatalf("Validate after apply: %v", err)
	}
	if res.Divergent || len(res.Findings) != 0 {
		t.Fatalf("scenario should be fully instrumented after apply, findings=%+v", res.Findings)
	}
}

// Format-preservation: a comment elsewhere in vite.config.ts survives the edit.
func TestApplyPreservesSurroundingFormatting(t *testing.T) {
	root := writeBareReactVite(t)
	vitePath := filepath.Join(root, "ui", "vite.config.ts")
	original := readFile(t, vitePath)

	svc := NewService()
	if _, err := svc.Apply(root, []string{readiness.RuleViteProfileMode}); err != nil {
		t.Fatalf("Apply vite: %v", err)
	}
	after := readFile(t, vitePath)
	// The original plugin line must still be present (we injected, not rewrote).
	if !contains(after, "plugins: [react()]") {
		t.Fatal("existing vite config content was not preserved")
	}
	if after == original {
		t.Fatal("vite config should have changed")
	}
	if !contains(after, "react-dom/profiling") || !contains(after, "keepNames") {
		t.Fatal("profile-mode block was not injected")
	}
}

// Targeted single-rule apply only touches that rule's file.
func TestApplySingleRule(t *testing.T) {
	root := writeBareReactVite(t)
	svc := NewService()
	got, err := svc.Apply(root, []string{readiness.RuleProfilerUtil})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(got) != 1 || got[0].RuleID != readiness.RuleProfilerUtil {
		t.Fatalf("expected only the profiler-util fix, got %v", ruleIDs(got))
	}
	if _, err := os.Stat(filepath.Join(root, "ui", "src", "lib", "profiler.ts")); err != nil {
		t.Fatalf("profiler.ts should exist: %v", err)
	}
}

// --- helpers (re-use the readiness test fixtures shape) ------------------

func writeBareReactVite(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	if err := os.MkdirAll(filepath.Join(ui, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(ui, "package.json"), `{
  "name": "bare-ui",
  "scripts": {
    "dev": "vite",
    "build": "tsc --noEmit && vite build"
  },
  "dependencies": { "react": "^19.0.0" },
  "devDependencies": { "vite": "^6.0.0" }
}
`)
	write(t, filepath.Join(ui, "vite.config.ts"), `import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(() => ({
  base: "./",
  plugins: [react()],
}));
`)
	write(t, filepath.Join(ui, "src", "main.tsx"), `import ReactDOM from "react-dom/client";
import App from "./App";

ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
`)
	return root
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func ruleIDs(candidates []Candidate) []string {
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.RuleID)
	}
	return out
}

func snapshot(t *testing.T, root string) string {
	t.Helper()
	var b []byte
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		b = append(b, path...)
		b = append(b, '\x00')
		b = append(b, data...)
		b = append(b, '\x00')
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A scenario declaring only `build` gets `build:profile` added, and `build`
// itself is left untouched — no shell conditional is introduced.
func TestInjectBuildProfileScriptLeavesBuildAlone(t *testing.T) {
	before := `{
  "name": "demo-ui",
  "scripts": {
    "build": "tsc --noEmit && vite build"
  }
}
`
	after, changed, err := injectBuildProfileScript(before)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if !contains(after, `"build": "tsc --noEmit && vite build"`) {
		t.Fatalf("build was rewritten:\n%s", after)
	}
	if !contains(after, `"build:profile": "tsc --noEmit && vite build --mode profile"`) {
		t.Fatalf("build:profile missing:\n%s", after)
	}
	for _, tok := range []string{"$(", "VROOLI_BUILD_MODE", "[ "} {
		if contains(after, tok) {
			t.Fatalf("shell token %q reintroduced:\n%s", tok, after)
		}
	}
	// Idempotent: a second pass is a no-op.
	if _, changed2, err := injectBuildProfileScript(after); err != nil || changed2 {
		t.Fatalf("second pass changed=%v err=%v", changed2, err)
	}
}
