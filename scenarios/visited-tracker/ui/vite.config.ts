import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import RequirementReporter from "@vrooli/vitest-requirement-reporter";

export default defineConfig({
  base: './',  // Required for tunnel/proxy contexts
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    passWithNoTests: true,
    reporters: [
      new RequirementReporter({
        outputFile: 'coverage/vitest-requirements.json',
        emitStdout: true,  // REQUIRED for phase integration
        verbose: true,
        conciseMode: true,  // Prevents HTML spam in test output
        artifactsDir: 'coverage/unit',
        autoClear: true,
      }),
    ],
    coverage: {
      provider: 'v8',
      reporter: ['json-summary', 'json', 'text'],
      reportOnFailure: true,
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/**/*.test.{ts,tsx}', 'src/test-setup.ts', 'src/vite-env.d.ts'],
      thresholds: {
        lines: 60,
        functions: 60,
        branches: 50,
        statements: 60
      }
    }
  }
});
