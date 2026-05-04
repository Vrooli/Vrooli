import { defineConfig, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";

// Mode-aware config. A regular `vite build` ships the lean prod artifact;
// `vite build --mode profile` produces a perf-build channel for performance
// tracing. The perf build is *still* a production bundle (minified, batched,
// no StrictMode double-renders) — it differs only in:
//
//   1. `react-dom/client` aliases to `react-dom/profiling` so React's
//      profiling instrumentation survives. This makes <React.Profiler>'s
//      onRender callbacks fire, which lets `src/lib/profiler.ts` emit
//      user_timing entries that show up in Chrome DevTools' Performance panel.
//   2. `esbuild.keepNames` preserves component/function names through
//      minification so CPU samples and React-track entries display real names
//      instead of mangled ones.
//
// Triggering the perf build:
//   - Direct:  `pnpm run build:profile` always uses --mode profile.
//   - Via env: `VROOLI_BUILD_MODE=profile pnpm run build`.
export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === "profile";

  return {
    // INTEROP-CRITICAL: Relative asset URLs are required when the UI is embedded behind scenario tunnels/proxies.
    base: "./", // Required for tunnel/proxy contexts
    plugins: [react()],
    resolve: isProfile
      ? {
          alias: {
            "react-dom/client": "react-dom/profiling",
            "react-dom$": "react-dom/profiling",
          },
        }
      : undefined,
    esbuild: isProfile
      ? {
          keepNames: true,
        }
      : undefined,
    build: {
      rollupOptions: {
        output: {
          // INTEROP-CRITICAL: Keep the heavy mermaid bundle isolated so embedded hosts load consistently.
          manualChunks: {
            mermaid: ["mermaid"],
          },
        },
      },
    },
    test: {
      globals: true,
      environment: "jsdom",
      setupFiles: ["./src/test-setup.ts"],
      coverage: {
        provider: "v8",
        reporter: ["json-summary", "json", "text"],
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
