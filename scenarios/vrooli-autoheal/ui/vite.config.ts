import { defineConfig, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";
import RequirementReporter from "@vrooli/vitest-requirement-reporter";

// INTEROP-CRITICAL: base must stay relative so tunnel, proxy, and iframe-hosted
// deployments resolve built assets correctly regardless of parent route prefix.
export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === "profile";

  return {
    base: "./", // Required for tunnel/proxy contexts
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
    test: {
      globals: true,
      environment: "jsdom",
      setupFiles: ["./src/test-setup.ts"],
      reporters: [
        "default",
        new RequirementReporter({
          outputFile: "test/artifacts/vitest-requirements.json",
        }),
      ],
      coverage: {
        provider: "v8",
        reporter: ["json-summary", "json", "text"],
        reportOnFailure: true,
        thresholds: {
          lines: 0,
          functions: 0,
          branches: 0,
          statements: 0,
        },
      },
    },
  };
});
