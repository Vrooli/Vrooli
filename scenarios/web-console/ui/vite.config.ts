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
