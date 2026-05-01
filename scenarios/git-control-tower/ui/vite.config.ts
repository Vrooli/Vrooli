import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  // INTEROP-CRITICAL: Relative asset URLs are required when the UI is embedded behind scenario tunnels/proxies.
  base: "./", // Required for tunnel/proxy contexts
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        // INTEROP-CRITICAL: Keep the heavy mermaid bundle isolated so embedded hosts load consistently.
        manualChunks: {
          mermaid: ["mermaid"],
        },
      },
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./src/test-setup.ts"],
    coverage: {
      provider: "v8",
      reporter: ["json-summary", "json", "text"],
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
