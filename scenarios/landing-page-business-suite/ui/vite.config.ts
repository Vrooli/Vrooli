import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  base: './',  // Required for tunnel/proxy contexts
  resolve: {
    alias: {
      "@proto-lprv": path.resolve(__dirname, "../../../packages/proto/gen/typescript/landing-page-react-vite/v1"),
    },
  },
  server: {
    fs: {
      allow: [path.resolve(__dirname, "../../../packages")],
    },
  },
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
