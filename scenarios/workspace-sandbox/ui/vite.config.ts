import { defineConfig, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";

// Mode-aware config. A regular `vite build` ships the lean prod artifact;
// `vite build --mode profile` produces a perf-build channel for performance
// tracing. The perf build is *still* a production bundle (minified, batched,
// no StrictMode double-renders) — it differs only in:
//
//   1. `react-dom/client` aliases to `react-dom/profiling` so React's
//      profiling instrumentation survives. <React.Profiler>'s onRender then
//      fires, letting `src/lib/profiler.ts` emit user_timing entries.
//   2. `esbuild.keepNames` preserves component/function names through
//      minification so CPU samples and React-track entries display real
//      names instead of mangled ones.
//
// Triggering the perf build:
//   - Direct:  `pnpm run build:profile`
//   - Via env: `VROOLI_BUILD_MODE=profile pnpm run build` (the `build`
//              script appends --mode profile when the env var is set).
export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === "profile";

  return {
    base: './',  // Required for tunnel/proxy contexts
    plugins: [react()],
    resolve: isProfile
      ? {
          alias: {
            "react-dom/client": "react-dom/profiling",
            // Internal references inside react-dom/client.js do
            // `require('react-dom')`, which would resolve back to the
            // stripped-prod bundle. Force them through the profiling entry too.
            "react-dom$": "react-dom/profiling",
          },
        }
      : undefined,
    esbuild: isProfile
      ? {
          keepNames: true,
        }
      : undefined,
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/test-setup.ts'],
      coverage: {
        provider: 'v8',
        reporter: ['json-summary', 'json', 'text'],
        reportOnFailure: true,
        thresholds: {
          lines: 0,
          functions: 0,
          branches: 0,
          statements: 0
        }
      }
    }
  };
});
