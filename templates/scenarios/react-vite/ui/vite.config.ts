import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import stringsCodegen from "./scripts/vite-plugin-strings-codegen.mjs";

export default defineConfig({
  base: './',  // Required for tunnel/proxy contexts
  plugins: [react(), stringsCodegen()],
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
