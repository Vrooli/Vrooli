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
    setupFiles: ['./src/test-utils/setup.ts'],
    // The shared audio package is a workspace file: link, so Vitest treats it
    // as an external dependency and lets Node resolve it — which lands on its
    // published dist/, whose emitted ESM uses extensionless relative imports
    // Node cannot resolve. Inlining routes the import back through Vite, whose
    // resolver probes extensions the same way the app bundle does. Without
    // this, every test that touches the audio integration fails to collect.
    server: { deps: { inline: [/@vrooli\/audio-capture-browser/] } },
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
});
