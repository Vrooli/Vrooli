import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// INTEROP-CRITICAL: base must be './' for tunnel/proxy/iframe embedding contexts
export default defineConfig({
  base: './',
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
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
