import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
// import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default defineConfig({
  base: './',
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    // reporters: [new RequirementReporter({
    //   outputFile: 'coverage/vitest-requirements.json',
    //   emitStdout: true,
    //   verbose: true,
    //   conciseMode: true,
    // })],
    coverage: {
      exclude: ['src/**/*.test.{ts,tsx}', 'src/**/*.spec.{ts,tsx}', 'src/**/*.d.ts', 'src/main.tsx', 'src/test-setup.ts', 'src/test-utils/**', 'src/consts/strings.generated.ts', 'src/i18n/locales/**', 'src/**/generated/**'],
      include: ['src/**/*.{ts,tsx}'],
      provider: 'v8',
      reporter: ['json-summary', 'text', 'json'],
      reportOnFailure: true,
    },
  },
});
