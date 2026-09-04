import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => ({
  // ╔══════════════════════════════════════════════════════════════╗
  // ║  INTEROP-CRITICAL: Relative base for proxy/tunnel contexts  ║
  // ║                                                              ║
  // ║  When served through app-monitor's proxy at                  ║
  // ║  /apps/<name>/proxy/, absolute asset URLs (base: '/')        ║
  // ║  resolve to the domain root, breaking all JS/CSS loading.    ║
  // ║  Relative base ('./') makes assets resolve from the          ║
  // ║  current directory, which works in all three contexts.       ║
  // ║                                                              ║
  // ║  DO NOT change to '/' or remove this setting.                ║
  // ╚══════════════════════════════════════════════════════════════╝
  base: './',
  plugins: [react()],
  resolve: {
    alias: mode === "profile" ? [
      {
        find: "react-dom/client",
        replacement: new URL("./node_modules/react-dom/profiling.js", import.meta.url).pathname,
      },
    ] : [],
  },
  esbuild: {
    keepNames: true,
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./src/test-setup.ts"],
    minWorkers: 1,
    maxWorkers: 1,
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
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 85,
        statements: 85,
      },
    },
  },
}));
