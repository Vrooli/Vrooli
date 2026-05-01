import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");

  return {
    // INTEROP-CRITICAL: Keep relative base so proxied deployments resolve assets
    // under nested /apps/<scenario>/proxy paths.
    base: "./",
    plugins: [react()],
    server: {
      port: env.VITE_DEV_SERVER_PORT ? Number(env.VITE_DEV_SERVER_PORT) : 5173,
      host: true
    },
    preview: {
      port: env.VITE_PREVIEW_PORT ? Number(env.VITE_PREVIEW_PORT) : 4173,
      host: true
    },
    test: {
      globals: true,
      environment: "jsdom",
      setupFiles: ["./src/test-utils/setup.ts"],
      include: ["tests/**/*.test.ts", "src/**/*.test.{ts,tsx}"],
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
  };
});
