import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { resolve } from "path";

const UI_PORT = process.env.UI_PORT;

// UI_PORT is only required when running dev server, not for build
if (!UI_PORT && process.argv.includes("serve")) {
  throw new Error("UI_PORT environment variable is required. Run the scenario through the Vrooli lifecycle so it is provided automatically.");
}

export default defineConfig({
  base: "./",
  plugins: [react()],
  resolve: {
    alias: {
      "@": resolve(__dirname, "./src")
    }
  },
  server: {
    port: UI_PORT ? parseInt(UI_PORT, 10) : 3000,
    strictPort: true,
    host: true
  },
  build: {
    outDir: "dist",
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ["react", "react-dom"],
          query: ["@tanstack/react-query"]
        }
      }
    }
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: "./src/test-utils/setupTests.ts",
  },
});
