import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// INTEROP-CRITICAL: base must be './' for tunnel/proxy embedding compatibility.
// Changing this breaks parent-scenario iframe routing.
export default defineConfig({
  base: './',
  plugins: [react()],
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
