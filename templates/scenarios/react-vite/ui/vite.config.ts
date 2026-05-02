import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import stringsCodegen from "./scripts/vite-plugin-strings-codegen.mjs";

export default defineConfig({
  base: './',  // Required for tunnel/proxy contexts
  plugins: [react(), stringsCodegen()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
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
      ],
      // 85% is the floor every canonical-surface file (App.tsx +
      // button/input/textarea + consts + i18n + lib/api + lib/utils +
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
});
