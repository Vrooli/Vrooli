// INTEROP-CRITICAL: Vite config for vrooli-onboarding UI
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const isProfile = mode === "profile";

  return {
    base: './',  // Required for tunnel/proxy contexts
    plugins: [react()],
    resolve: {
      alias: isProfile ? {
        "react-dom/client": "react-dom/profiling",
        "react-dom$": "react-dom/profiling",
      } : undefined,
    },
    esbuild: isProfile ? { keepNames: true } : undefined,
    test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.test.{ts,tsx}',
        'src/**/*_test.{ts,tsx}',
        'src/**/*.spec.{ts,tsx}',
        'src/**/*.d.ts',
        'src/main.tsx',
        'src/test-setup.ts',
        'src/test-utils.tsx',
        'src/test-utils/**',
        'src/consts/strings.generated.ts',
        'src/i18n/locales/**',
        'src/**/generated/**',
      ],
      reporter: ['json-summary', 'json', 'text'],
      reportOnFailure: true,
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 85,
        statements: 85,
      }
    }
    },
  };
});
