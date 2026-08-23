import { defineConfig, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";
import stringsCodegen from "./scripts/vite-plugin-strings-codegen.mjs";

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
// Cost in the perf build: ~5–15 % extra CPU per commit, ~10–20 KB extra gz.
// Trade only when auditing.
//
// Triggering the perf build:
//   - Direct:  `pnpm run build:profile` always uses --mode profile.
//   - Via env: `VROOLI_BUILD_MODE=profile vrooli scenario restart <name>`.
//              The lifecycle builder selects the `build:profile` script for
//              that channel, so the selection is argv the whole way down and
//              carries no shell conditional.
export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === "profile";
  return {
    // INTEROP-CRITICAL: relative base required for tunnel/iframe proxy contexts.
    base: './',
    plugins: [react(), stringsCodegen()],
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
      // The shared audio package is a workspace file. Inline it so Vitest
      // resolves its TypeScript graph through Vite instead of asking Node to
      // load extensionless ESM imports from the package's dist/ directory.
      server: { deps: { inline: [/@vrooli\/audio-capture-browser/] } },
      coverage: {
        provider: 'v8',
        reporter: ['text', 'json-summary', 'json'],
        reportOnFailure: true,
        // Scope coverage to the source tree. Without `include`, v8 walks every
        // file the bundler touches — config files, eslint plugins, codegen
        // scripts — and pollutes the denominator with files that have no
        // production reason to be tested.
        include: ['src/**/*.{ts,tsx}'],
        // Exclusions cover test scaffolding and codegen only; production
        // source under src/ is exhaustively included so removing a test
        // can never silently shrink the denominator.
        //
        //   1. Test-only files (tests, setup, helpers).
        //   2. Boot/codegen artefacts (main.tsx entry, type declarations,
        //      generated registries, JSON catalogs).
        //
        // If a scenario adds genuinely-untestable code, prefer narrow file
        // exclusions with a one-line rationale comment over loosening the
        // thresholds. The default position is: every new src/ file ships
        // with its own *.test.{ts,tsx} and lands inside the include set.
        exclude: [
          'src/**/*.test.{ts,tsx}',
          'src/**/*.spec.{ts,tsx}',
          'src/**/*.d.ts',
          'src/main.tsx',
          'src/test-setup.ts',
          'src/test-utils/**',
          'src/consts/strings.generated.ts',
          'src/i18n/locales/**',
          // Temporal-flow codegen. Everything under generated/ is
          // emitted by the flow-verifier scenario and verified by the
          // hand-authored thin-test at the feature root.
          'src/**/generated/**',
        ],
        // 85% is the floor every canonical-surface file (App.tsx +
        // button/input/textarea + consts + i18n + api/client + lib/utils +
        // hooks/{useGamepad,useSpatialNav,SpatialGroup}) clears with the
        // tests shipped in this template. Tightening beyond actual
        // coverage of a healthy template would make every new scenario
        // start red; loosening below it would make the gate vacuous.
        // When a scenario's surface stabilises above 90% for a full
        // release, raise these together.
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
