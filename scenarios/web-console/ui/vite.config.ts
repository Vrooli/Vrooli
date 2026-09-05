import { defineConfig, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";
import stringsCodegen from "./scripts/vite-plugin-strings-codegen.mjs";

// Mode-aware config. A regular `vite build` ships the lean prod artifact;
// `vite build --mode profile` produces the perf-build channel used for
// performance tracing. The perf build is *still* a production bundle
// (minified, batched, no StrictMode double-renders) — it differs only in:
//
//   1. `react-dom/client` aliases to `react-dom/profiling` so React's
//      profiling instrumentation survives. That makes <React.Profiler>'s
//      onRender fire, which lets `src/lib/profiler.ts` emit user_timing
//      entries performance-health can attribute per component.
//   2. `esbuild.keepNames` preserves component names through minification so
//      CPU samples read as real names instead of mangled ones.
//
// Cost in the perf build: ~5–15 % extra CPU per commit. Trade only when
// auditing — `pnpm run build:profile`, or the lifecycle perf-build channel.
export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === "profile";

  return {
    // INTEROP-CRITICAL: base must be './' for tunnel/proxy embedding compatibility.
    // Changing this breaks parent-scenario iframe routing.
    base: './',
    plugins: [react(), stringsCodegen()],
    resolve: isProfile
      ? {
          alias: {
            "react-dom/client": "react-dom/profiling",
            // Internal references inside react-dom/client.js do
            // `require('react-dom')`, which would resolve back to the
            // stripped-prod bundle. Force those through the profiling entry too.
            "react-dom$": "react-dom/profiling",
          },
        }
      : undefined,
    esbuild: isProfile ? { keepNames: true } : undefined,
    test: {
      globals: true,
      environment: 'jsdom',
      // Scenario-level test runs were intermittently crashing under the broader
      // `make test` harness. Keep worker isolation but bound the fork pool so
      // memory can be reclaimed between files without spawning an excessive
      // number of concurrent jsdom workers.
      pool: 'forks',
      poolOptions: {
        forks: {
          minForks: 1,
          maxForks: 2,
        },
      },
      setupFiles: ['./src/test-setup.ts'],
      // The shared audio package is a workspace file: link, so Vitest treats it
      // as an external dependency and lets Node resolve it — which lands on its
      // published dist/, whose emitted ESM uses extensionless relative imports
      // Node cannot resolve. Inlining routes the import back through Vite, whose
      // resolver probes extensions the same way the app bundle does. Without
      // this, every test that touches the audio integration fails to collect.
      server: { deps: { inline: [/@vrooli\/(audio-capture-browser|react-component-library)/] } },
      coverage: {
        provider: 'v8',
        reporter: ['text', 'json-summary', 'json'],
        include: ['src/**/*.{ts,tsx}'],
        exclude: [
          'src/**/*.test.{ts,tsx}',
          'src/**/*.spec.{ts,tsx}',
          'src/**/*.d.ts',
          'src/main.tsx',
          'src/test-setup.ts',
          'src/test-utils/**',
          'src/consts/strings.generated.ts',
          'src/i18n/locales/**',
          'src/**/generated/**',
          // These files are compatibility/type surfaces, not executable
          // web-console behavior. The implementation is owned by the shared
          // audio package or the consuming modules, so counting their one-line
          // re-export statements makes the aggregate floor measure import
          // topology instead of this UI's testable behavior.
          'src/types/**',
          'src/audio-integration/hooks/voice/wakeword/**',
          'src/audio-integration/hooks/voice/{activity,autoStopDecision,commandParser,commands,index,streamHealth}.ts',
          'src/audio-integration/hooks/tts/index.ts',
          'src/audio-integration/hooks/useServerVadStateStore.ts',
          'src/audio-integration/components/Button.tsx',
        ],
        reportOnFailure: true,
        thresholds: {
          lines: 85,
          functions: 85,
          branches: 85,
          statements: 85,
        }
      }
    }
  };
});
