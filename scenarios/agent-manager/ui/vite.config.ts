import { defineConfig, loadEnv, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { sourceLibraryResolver } from "../../../packages/react-component-library/tooling/resolve-specifier.mjs";

const rootDir = dirname(fileURLToPath(import.meta.url));
const libraryRoot = resolve(rootDir, "../../../scenarios/react-component-library/library");
const libraryDependencyAliases = ["clsx", "tailwind-merge", "lucide-react"].map((name) => ({
  find: name,
  replacement: resolve(rootDir, "node_modules", name),
}));

export default defineConfig(({ mode }): UserConfig => {
  const env = loadEnv(mode, process.cwd(), "");
  const isProfile = mode === "profile";

  return {
    // INTEROP-CRITICAL: Keep relative base so proxied deployments resolve assets
    // under nested /apps/<scenario>/proxy paths.
    base: "./",
    plugins: [react(), sourceLibraryResolver({ libraryRoot })],
    resolve: isProfile
      ? {
          alias: [
            ...libraryDependencyAliases,
            { find: "react-dom/client", replacement: "react-dom/profiling" },
            { find: "react-dom$", replacement: "react-dom/profiling" },
          ],
        }
      : { alias: libraryDependencyAliases },
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
          // Keep the native Vitest gate aligned with the scenario's declared
          // react-vite policy. All four dimensions are deliberate: a high
          // line total alone would not protect the investigation UI's error,
          // fallback, and operator-decision paths.
          lines: 85,
          functions: 85,
          branches: 85,
          statements: 85
        }
      }
    }
  };
});
