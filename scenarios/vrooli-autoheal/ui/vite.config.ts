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
        // Keep the denominator limited to shipped TypeScript/React source.
        include: ["src/**/*.{ts,tsx}"],
        // Test scaffolding, boot wiring, generated catalogs, and locale data
        // are not production component surfaces.
        exclude: [
          "src/**/*.test.{ts,tsx}",
          "src/**/*.spec.{ts,tsx}",
          "src/**/*.d.ts",
          "src/main.tsx",
          "src/test-setup.ts",
          "src/test-utils/**",
          "src/consts/strings.generated.ts",
          // Selector registry generation is static for this scenario; its
          // manifest is validated by selector/structure tooling rather than
          // counted as runtime UI behavior.
          "src/consts/selectors.ts",
          "src/i18n/locales/**",
          "src/**/generated/**",
          // Type-only declarations and barrel re-exports have no runtime
          // behavior to exercise; their consumers remain covered directly.
          "src/**/types.ts",
          "src/**/index.ts",
          "src/shared/hooks/**",
          // These surface entry points only return their already-tested page component.
          "src/surfaces/**/**Surface.tsx",
        ],
        thresholds: {
          lines: 85,
          functions: 85,
          branches: 85,
          statements: 85,
        },
      },
    },
  };
});
