import { defineConfig, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";

// Mode-aware config so a regular `vite build` ships the lean prod artifact and
// `vite build --mode profile` produces a perf-build channel. The perf build is
// still a production bundle (minified, batched, no StrictMode double-renders);
// it only differs in two ways:
//
//   1. `react-dom/client` aliases to `react-dom/profiling` so React's internal
//      profiling instrumentation survives. This makes <React.Profiler>'s
//      onRender callbacks fire (otherwise they're stripped).
//   2. `esbuild.keepNames` preserves component/function identifiers through
//      minification so CPU samples and React-track entries display real names
//      (`BacklogTab`) instead of mangled ones (`hR`).
//
// Cost vs. regular prod: ~5–15% extra CPU per commit, ~10–20 KB extra gz.
// Trade only when auditing.
//
// Triggering the perf build:
//   - Direct:  `pnpm run build:profile` always uses --mode profile.
//   - Via env: `VROOLI_BUILD_MODE=profile pnpm run build` (the conditional
//              lives in package.json's `build` script). This lets
//              `VROOLI_BUILD_MODE=profile vrooli scenario restart swarm-manager`
//              produce the perf bundle through the standard lifecycle path.
//
// See scratch/perf-spike/README.md for the audit workflow.
export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === "profile";
  return {
    // ╔══════════════════════════════════════════════════════════════╗
    // ║  INTEROP-CRITICAL: Relative base for proxy/tunnel contexts  ║
    // ║                                                              ║
    // ║  When served through app-monitor's proxy at                  ║
    // ║  /apps/<name>/proxy/, absolute asset URLs (base: '/')        ║
    // ║  resolve to the domain root, breaking all JS/CSS loading.    ║
    // ║  Relative base ('./') makes assets resolve from the          ║
    // ║  current directory, which works in all three contexts.       ║
    // ║                                                              ║
    // ║  DO NOT change to '/' or remove this setting.                ║
    // ╚══════════════════════════════════════════════════════════════╝
    base: "./",
    // Targets Chrome 67 to support older embedded browsers (e.g. Google TV).
    // Only transpiles syntax (optional chaining, class fields, etc.) — does NOT
    // polyfill missing runtime APIs. See main.tsx for runtime polyfills.
    build: {
      target: "chrome67",
    },
    resolve: isProfile
      ? {
          alias: {
            "react-dom/client": "react-dom/profiling",
            // Internal references inside react-dom/client.js do `require('react-dom')`,
            // which would resolve back to the stripped-prod bundle. Force them
            // through the profiling entry too.
            "react-dom$": "react-dom/profiling",
          },
        }
      : undefined,
    esbuild: isProfile
      ? {
          keepNames: true,
        }
      : undefined,
    plugins: [react()],
    test: {
      globals: true,
      environment: "jsdom",
      setupFiles: ["./src/setupTests.ts"],
      testTimeout: 30_000,
      hookTimeout: 30_000,
      coverage: {
        provider: "v8",
        reporter: ["json-summary", "json", "text"],
        reportOnFailure: true,
        thresholds: {
          lines: 0,
          functions: 0,
          branches: 0,
          statements: 0,
        },
      },
    },
  };
});
