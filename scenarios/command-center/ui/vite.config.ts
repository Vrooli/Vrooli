import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  // INTEROP-CRITICAL: relative assets remain valid behind proxy/tunnel paths.
  base: './',  // Required for tunnel/proxy contexts
  plugins: [react()],
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
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 85,
        statements: 85
      }
    }
  }
});
