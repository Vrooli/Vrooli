import { defineConfig, type UserConfig } from "vite";
import react from "@vitejs/plugin-react";

// INTEROP-CRITICAL: base and profile aliases must remain safe in embedded,
// tunneled, and desktop-hosted surfaces.
export default defineConfig(({ mode }): UserConfig => {
  const isProfile = mode === "profile";
  return {
    base: "./",
    plugins: [react()],
    resolve: isProfile
      ? { alias: { "react-dom/client": "react-dom/profiling", "react-dom$": "react-dom/profiling" } }
      : undefined,
    esbuild: isProfile ? { keepNames: true } : undefined,
    test: {
      globals: true,
      environment: "jsdom",
      setupFiles: ["./src/test-setup.ts"],
      coverage: {
        provider: "v8",
        reporter: ["text", "json-summary", "json"],
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
        thresholds: { lines: 85, functions: 85, branches: 85, statements: 85 },
      },
    },
  };
});
