package readiness

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type fakeFacts struct {
	facts Facts
	err   error
}

func (f fakeFacts) Describe(context.Context, string, string) (Facts, error) {
	return f.facts, f.err
}

// [REQ:PH-TIER-002] Readiness validates a scenario and surfaces its facts.
func TestValidateReturnsFacts(t *testing.T) {
	root := writeInstrumentedReactVite(t)
	svc := NewService(fakeFacts{facts: Facts{Scenario: "demo", Surfaces: []string{"api", "ui"}, UIFramework: "react", RootPath: root}})
	res, err := svc.Validate(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if res.Tier != Tier1 {
		t.Fatalf("tier = %v, want Tier1", res.Tier)
	}
	if res.UIFramework != "react" {
		t.Fatalf("ui framework = %q, want react", res.UIFramework)
	}
	if res.Divergent {
		t.Fatalf("fully-instrumented scenario should not be divergent")
	}
	if len(res.Findings) != 0 {
		t.Fatalf("instrumented scenario should emit no findings, got %d", len(res.Findings))
	}
}

func TestValidateRequiresScenario(t *testing.T) {
	svc := NewService(fakeFacts{})
	if _, err := svc.Validate(context.Background(), "", ""); err == nil {
		t.Fatal("expected error for empty scenario and path")
	}
}

func TestValidateRequiresFactsClient(t *testing.T) {
	svc := NewService(nil)
	if _, err := svc.Validate(context.Background(), "demo", ""); err == nil {
		t.Fatal("expected error when facts client is not wired")
	}
}

// [REQ:PH-TIER-002] No-UI scenario reports no browser-perf tier and no findings.
func TestValidateNoUISkipsBrowserPerf(t *testing.T) {
	svc := NewService(fakeFacts{facts: Facts{Scenario: "svc", Surfaces: []string{"api", "cli"}}})
	res, err := svc.Validate(context.Background(), "svc", "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if res.Tier != TierNone {
		t.Fatalf("tier = %v, want TierNone", res.Tier)
	}
	if len(res.Findings) != 0 || res.Divergent {
		t.Fatalf("no-UI scenario should produce no infra findings")
	}
}

// [REQ:PH-TIER-002] Non-React UI is held at Tier 0 with no perf-build findings.
func TestValidateNonReactTierZero(t *testing.T) {
	svc := NewService(fakeFacts{facts: Facts{Scenario: "vueapp", Surfaces: []string{"ui"}, UIFramework: "vue"}})
	res, err := svc.Validate(context.Background(), "vueapp", "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if res.Tier != Tier0 {
		t.Fatalf("tier = %v, want Tier0", res.Tier)
	}
	if len(res.Findings) != 0 || res.Divergent {
		t.Fatalf("non-React UI should not produce perf-build infra findings")
	}
}

// [REQ:PH-TIER-002] A React UI missing the perf-build infra is Tier-1 reachable
// but flagged divergent, with a finding per missing piece.
func TestValidateReactDivergentFlagsMissingInfra(t *testing.T) {
	root := writeBareReactVite(t)
	svc := NewService(fakeFacts{facts: Facts{Scenario: "bare", Surfaces: []string{"ui"}, UIFramework: "react-vite", RootPath: root}})
	res, err := svc.Validate(context.Background(), "bare", "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if res.Tier != Tier1 {
		t.Fatalf("tier = %v, want Tier1 (reachable)", res.Tier)
	}
	if !res.Divergent {
		t.Fatal("React scenario missing infra should be flagged divergent")
	}
	if got := len(res.Findings); got != 4 {
		t.Fatalf("expected 4 missing-infra findings, got %d: %+v", got, res.Findings)
	}
	if res.AutofixableCount() != 4 {
		t.Fatalf("all four findings should be autofixable, got %d", res.AutofixableCount())
	}
	want := map[string]bool{
		RuleViteProfileMode:    false,
		RuleBuildProfileScript: false,
		RuleProfilerUtil:       false,
		RuleProfilerBoundary:   false,
	}
	for _, f := range res.Findings {
		if _, ok := want[f.Code]; !ok {
			t.Fatalf("unexpected finding code %q", f.Code)
		}
		want[f.Code] = true
	}
	for code, seen := range want {
		if !seen {
			t.Fatalf("missing finding for rule %q", code)
		}
	}
}

// writeBareReactVite creates a react-vite scenario tree with NONE of the four
// perf-build infra pieces.
func writeBareReactVite(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	mustMkdir(t, filepath.Join(ui, "src"))
	mustWrite(t, filepath.Join(ui, "package.json"), `{
  "name": "bare-ui",
  "scripts": {
    "dev": "vite",
    "build": "tsc --noEmit && vite build"
  },
  "dependencies": { "react": "^19.0.0" },
  "devDependencies": { "vite": "^6.0.0" }
}
`)
	mustWrite(t, filepath.Join(ui, "vite.config.ts"), `import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(() => ({
  base: "./",
  plugins: [react()],
}));
`)
	mustWrite(t, filepath.Join(ui, "src", "main.tsx"), `import ReactDOM from "react-dom/client";
import App from "./App";

ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
`)
	return root
}

// writeInstrumentedReactVite creates a react-vite scenario tree with ALL four
// perf-build infra pieces present.
func writeInstrumentedReactVite(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	mustMkdir(t, filepath.Join(ui, "src", "lib"))
	mustWrite(t, filepath.Join(ui, "package.json"), `{
  "name": "good-ui",
  "scripts": {
    "build": "vite build",
    "build:profile": "vite build --mode profile"
  },
  "dependencies": { "react": "^19.0.0" },
  "devDependencies": { "vite": "^6.0.0" }
}
`)
	mustWrite(t, filepath.Join(ui, "vite.config.ts"), `import { defineConfig } from "vite";

export default defineConfig(({ mode }) => {
  const isProfile = mode === "profile";
  return {
    resolve: isProfile ? { alias: { "react-dom/client": "react-dom/profiling" } } : undefined,
    esbuild: isProfile ? { keepNames: true } : undefined,
  };
});
`)
	mustWrite(t, filepath.Join(ui, "src", "lib", "profiler.ts"), `import type { ProfilerOnRenderCallback } from "react";
export const onProfilerRender: ProfilerOnRenderCallback = (id, phase, actualDuration) => {
  performance.measure(`+"`⚛ ${id} (${phase})`"+`, { start: performance.now() - actualDuration, duration: actualDuration });
};
`)
	mustWrite(t, filepath.Join(ui, "src", "main.tsx"), `import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { onProfilerRender } from "./lib/profiler";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.Profiler id="App" onRender={onProfilerRender}>
    <App />
  </React.Profiler>
);
`)
	return root
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
