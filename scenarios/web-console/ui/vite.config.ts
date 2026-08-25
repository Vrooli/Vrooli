import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import stringsCodegen from "./scripts/vite-plugin-strings-codegen.mjs";

// INTEROP-CRITICAL: base must be './' for tunnel/proxy embedding compatibility.
// Changing this breaks parent-scenario iframe routing.
export default defineConfig({
  base: './',
  plugins: [react(), stringsCodegen()],
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
    server: { deps: { inline: [/@vrooli\/audio-capture-browser/] } },
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
        statements: 85,
      }
    }
  }
});
