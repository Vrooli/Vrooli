import { defineConfig, type UserConfig } from "vite";
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
  const isProfile = mode === "profile";

  return {
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
    plugins: [react(), sourceLibraryResolver({ libraryRoot })],
    resolve: {
      alias: [
        ...libraryDependencyAliases,
        { find: "@", replacement: "/src" },
        ...(isProfile
          ? [
              { find: "react-dom/client", replacement: "react-dom/profiling" },
              { find: "react-dom$", replacement: "react-dom/profiling" },
            ]
          : []),
      ],
    },
    esbuild: isProfile
      ? {
          keepNames: true,
        }
      : undefined,
    build: {
      rollupOptions: {
        output: {
          manualChunks: {
            mermaid: ["mermaid"],
          },
        },
      },
    },
    test: {
      globals: true,
      environment: 'jsdom',
      passWithNoTests: true,
      setupFiles: ['./src/test/setup.ts'],
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
  };
});
