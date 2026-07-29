import { defineConfig, loadEnv, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }): UserConfig => {
  const env = loadEnv(mode, process.cwd(), "");
  const isProfile = mode === "profile";

  return {
    // INTEROP-CRITICAL: Keep relative base so proxied deployments resolve assets
    // under nested /apps/<scenario>/proxy paths.
    base: "./",
    plugins: [react()],
    resolve: isProfile
      ? {
          alias: {
            "react-dom/client": "react-dom/profiling",
            "react-dom$": "react-dom/profiling",
          },
        }
      : undefined,
    esbuild: isProfile
      ? {
          keepNames: true,
        }
      : undefined,
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
      setupFiles: ["./src/test-setup.ts"],
      include: ["tests/**/*.test.ts", "src/**/*.test.{ts,tsx}"],
      coverage: {
        provider: "v8",
        reporter: ["json-summary", "json", "text"],
        reportOnFailure: true,
        include: ["src/**/*.{ts,tsx}"],
        exclude: [
          "src/**/*.test.{ts,tsx}",
          "src/**/*.spec.{ts,tsx}",
          "src/**/*.d.ts",
          "src/main.tsx",
          "src/test-setup.ts",
          "src/test-utils/**",
          "src/consts/strings.generated.ts",
          "src/i18n/locales/**",
          "src/**/generated/**",
        ],
        thresholds: {
          // Ratchet from Vitest 4's measured baseline; its V8 instrumentation
          // counts branch/function sites differently from Vitest 2. Every new
          // change must retain this meaningful investigation-UI coverage.
          lines: 30,
          functions: 25,
          branches: 25,
          statements: 30
        }
      }
    }
  };
});
