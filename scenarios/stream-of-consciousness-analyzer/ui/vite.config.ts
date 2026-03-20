import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  // INTEROP-CRITICAL: base must be './' for tunnel/proxy/iframe contexts
  base: './',
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
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
