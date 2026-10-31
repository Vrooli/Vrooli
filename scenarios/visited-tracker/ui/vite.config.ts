import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import RequirementReporter from "@vrooli/vitest-requirement-reporter";
import type { Reporter } from "vitest";

const requirementReporter = new RequirementReporter({
  outputFile: 'coverage/vitest-requirements.json',
  emitStdout: true,  // REQUIRED for phase integration
  verbose: true,
  conciseMode: true,  // Prevents HTML spam in test output
  artifactsDir: 'coverage/unit',
  autoClear: true,
}) as unknown as Reporter;

export default defineConfig({
  // INTEROP-CRITICAL: relative assets remain reachable through host proxies.
  base: './',  // Required for tunnel/proxy contexts
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    passWithNoTests: false,
    reporters: [requirementReporter],
    coverage: {
      provider: 'v8',
      reporter: ['json-summary', 'json', 'text'],
      reportOnFailure: true,
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/**/*.test.{ts,tsx}', 'src/**/*.spec.{ts,tsx}', 'src/**/*.d.ts',
        'src/main.tsx', 'src/test-setup.ts', 'src/test-utils/**',
        'src/consts/strings.generated.ts', 'src/i18n/locales/**', 'src/**/generated/**'],
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 85,
        statements: 85
      }
    }
  }
});
