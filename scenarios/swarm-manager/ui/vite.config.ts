import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  // ╔══════════════════════════════════════════════════════════════╗
  // ║  INTEROP-CRITICAL: Relative base for proxy/tunnel contexts  ║
  // ║                                                              ║
  // ║  When served through app-monitor's proxy at                  ║
  // ║  /apps/<name>/proxy/, absolute asset URLs (base: '/')        ║
  // ║  resolve to the domain root, breaking all JS/CSS loading.    ║
  // ║  Relative base ('./') makes assets resolve from the          ║
  // ║  current directory, which works in all three contexts.       ║
  // ║                                                              ║
  // ║  DO NOT change to '/' or remove this setting.                ║
  // ╚══════════════════════════════════════════════════════════════╝
  base: './',
  // Targets Chrome 67 to support older embedded browsers (e.g. Google TV).
  // Only transpiles syntax (optional chaining, class fields, etc.) — does NOT
  // polyfill missing runtime APIs. See main.tsx for runtime polyfills.
  build: {
    target: 'chrome67',
  },
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/setupTests.ts'],
    testTimeout: 30_000,
    hookTimeout: 30_000,
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
