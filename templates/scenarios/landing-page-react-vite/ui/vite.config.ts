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
      ]
      // No `thresholds`: Unit Health owns the coverage gate through
      // `coverage.committed` in .vrooli/testing.json.
    }
  }
});
