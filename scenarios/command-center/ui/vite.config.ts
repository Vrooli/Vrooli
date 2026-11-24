import { realpathSync } from "node:fs";
import { defineConfig, searchForWorkspaceRoot } from "vite";
import react from "@vitejs/plugin-react";
import { sourceLibraryResolver } from "../../../packages/react-component-library/tooling/resolve-specifier.mjs";
import { authoredRoot } from "../../../packages/react-component-library/tooling/catalog-source.mjs";

const commandCenterWorkspace = searchForWorkspaceRoot(process.cwd());
const componentLibrarySource = realpathSync(authoredRoot);

export default defineConfig(({ mode }) => ({
  // INTEROP-CRITICAL: relative assets remain valid behind proxy/tunnel paths.
  base: './',  // Required for tunnel/proxy contexts
  plugins: [sourceLibraryResolver({ devOnly: true }), react()],
  // Baseline Modes can serve this scenario from a frozen source copy. The
  // component library remains an explicitly shared source tree, reached by a
  // symlink from that copy, so Vite must allow the resolved source root too.
  server: {
    fs: {
      allow: [commandCenterWorkspace, componentLibrarySource],
    },
  },
  // Profile mode (performance-health capture): the profiling react-dom makes
  // <React.Profiler> emit commit durations, and kept names attribute them.
  resolve: mode === "profile"
    ? {
        alias: [
          { find: "react-dom/client", replacement: "react-dom/profiling" },
          { find: /^react-dom$/, replacement: "react-dom/profiling" },
        ],
      }
    : undefined,
  esbuild: mode === "profile" ? { keepNames: true } : undefined,
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("node_modules/world-atlas") || id.includes("node_modules/topojson") || id.includes("node_modules/d3-geo")) return "geo-runtime";
          if (id.includes("node_modules/react") || id.includes("node_modules/react-dom") || id.includes("node_modules/react-router")) return "react-runtime";
          return undefined;
        },
      },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    include: ['src/**/*.test.{ts,tsx}', 'src/**/*.spec.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['json-summary', 'json', 'text'],
      reportOnFailure: true,
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.test.{ts,tsx}',
        'src/**/*.spec.{ts,tsx}',
        'src/**/*.d.ts',
        'src/main.tsx',
        'src/App.tsx',
        'src/test-setup.ts',
        'src/test/**',
        'src/test-utils/**',
        'src/consts/strings.generated.ts',
        'src/i18n/locales/**',
        'src/**/generated/**',
        'src/consts/**',
        // Canvas scenes and the input controller are exercised by the BAS suite against a real browser.
        'src/scenes/**',
        'src/components/AmbientCanvas.tsx',
        'src/components/AmbientShell.tsx',
        'src/components/BoardController.tsx',
        'src/pages/**',
        'src/lib/api.ts',
      ],
    }
  }
}));
