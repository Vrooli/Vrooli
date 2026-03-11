import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  // INTEROP-CRITICAL: base './' is required for tunnel/proxy contexts
  // Do not change this without verifying iframe embedding still works
  base: './',
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-utils/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
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
