import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}', 'scripts/world-build-provenance.test.ts'],
    exclude: ['node_modules', 'dist'],
    // Use forks pool for better native module (gl) compatibility
    pool: 'forks',
    poolOptions: {
      forks: {
        singleFork: false,
      },
    },
    coverage: {
      reportOnFailure: true,
      include: ['src/**/*.{ts,tsx}'],
      provider: 'v8',
      // Unit Health consumes coverage-summary.json or lcov.info.
      reporter: ['text', 'json', 'html', 'json-summary', 'lcov'],
      exclude: [
        'node_modules/',
        'src/test/',
        '**/*.d.ts',
        '**/*.config.*',
        '**/types/**', 'src/**/*.test.{ts,tsx}', 'src/**/*.spec.{ts,tsx}', 'src/**/*.d.ts', 'src/main.tsx', 'src/test-setup.ts', 'src/test-utils/**', 'src/consts/strings.generated.ts', 'src/i18n/locales/**', 'src/**/generated/**',
      ],
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
