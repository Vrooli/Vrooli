import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default defineConfig({
  base: './',  // Required for tunnel/proxy contexts
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    reporters: [new RequirementReporter({
      outputFile: 'coverage/vitest-requirements.json',
      emitStdout: true,
      verbose: true,
      conciseMode: true,
    })],
    coverage: {
      provider: 'v8',
      reporter: ['json-summary', 'json', 'text'],
      reportOnFailure: true,
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/**/*.test.{ts,tsx}', 'src/test-setup.ts', 'src/consts/selectors.ts'],
      thresholds: {
        lines: 0,
        functions: 0,
        branches: 0,
        statements: 0
      }
    }
  }
});
