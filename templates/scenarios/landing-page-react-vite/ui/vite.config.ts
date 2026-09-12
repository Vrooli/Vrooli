import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  base: './',  // Required for tunnel/proxy contexts
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json-summary', 'json'],
      reportOnFailure: true,
      // Scope coverage to the source tree so the denominator stays on
      // production source files rather than config/codegen/test scaffolding.
      include: ['src/**/*.{ts,tsx}'],
      // Exclusions cover test scaffolding and codegen only; production source
      // under src/ is exhaustively included so removing a test can never
      // silently shrink the denominator.
      exclude: [
        'src/**/*.test.{ts,tsx}',
        'src/**/*.spec.{ts,tsx}',
        'src/**/*.d.ts',
        'src/main.tsx',
        'src/test-setup.ts',
        'src/test-utils/**',
        'src/consts/strings.generated.ts',
        'src/i18n/locales/**',
        'src/**/generated/**'
      ],
      // Policy floor for the react_vite_ui class. New surfaces ship with their
      // own *.test.{ts,tsx}; prefer narrow file exclusions with a rationale
      // over loosening these thresholds.
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 85,
        statements: 85
      }
    }
  }
});
