import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    include: ['src/**/*.test.ts', 'src/**/*.test.tsx'],
    exclude: ['**/node_modules/**', '**/dist/**', '**/coverage/**'],
    setupFiles: ['./src/test-utils/setup.ts'],
    environment: 'jsdom',
    globals: true,
    passWithNoTests: true,
    clearMocks: true,
    restoreMocks: true,
  },
})
