import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    globals: true,
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      include: [
        'src/gamepadInput.ts',
        'src/spatialNav.ts',
        'src/spatialNavStyles.ts',
        'src/spatialNavBridge.ts',
      ],
      exclude: [
        'src/**/*.test.ts',
        'src/__tests__/**',
        'src/**/*.d.ts',
        'src/**/index.ts',
        // iframeBridgeChild.ts is pre-existing code with its own test story;
        // spatial nav coverage thresholds apply only to the new files above.
        'src/iframeBridgeChild.ts',
      ],
      thresholds: {
        lines: 90,
        functions: 90,
        // Branch coverage for host-relay code paths (emitShortcutIntent) is
        // inherently limited in jsdom because the relay is a no-op when
        // window.parent === window.  80% accommodates this.
        branches: 80,
        statements: 90,
      },
    },
  },
})
