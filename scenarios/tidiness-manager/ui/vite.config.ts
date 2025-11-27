import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default defineConfig({
  base: '/',
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    reporters: ['default', new RequirementReporter()],
    coverage: {
      reporter: ['json-summary', 'text'],
      reportOnFailure: true,
      thresholds: {
        lines: 0,
        functions: 0,
        branches: 0,
        statements: 0,
      },
    },
  },
});
