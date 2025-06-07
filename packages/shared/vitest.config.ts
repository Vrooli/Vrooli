import { defineConfig } from 'vitest/config';
import path from 'path';

// Custom plugin to handle ../types.js imports
const typeImportPlugin = {
  name: 'resolve-type-imports',
  resolveId(id: string, importer?: string) {
    // Handle ../types.js and ../../types.js
    if ((id === '../types.js' || id === '../../types.js') && importer) {
      const srcDir = path.resolve(__dirname, 'src');
      // Always resolve to src/types.d.ts
      return path.join(srcDir, 'types.d.ts');
    }
    return null;
  }
};

export default defineConfig({
  plugins: [typeImportPlugin],
  resolve: {
    alias: {
      '@vrooli/shared': path.resolve(__dirname, './src'),
    },
    extensionAlias: {
      '.js': ['.ts', '.js'],
      '.jsx': ['.tsx', '.jsx'],
    },
  },
  server: {
    deps: {
      // This helps with circular dependencies
      external: [],
      inline: ['@vrooli/shared'],
    },
  },
  optimizeDeps: {
    // Force Vite to pre-bundle dependencies to help with circular deps
    include: ['yup'],
  },
  esbuild: {
    target: 'node18',
  },
  test: {
    globals: true,
    environment: 'node',
    setupFiles: ['./src/__test/setup.ts'],
    include: ['src/**/*.test.ts'],
    exclude: ['node_modules', 'dist'],
    // Use threads instead of forks for better module handling
    pool: 'threads',
    poolOptions: {
      threads: {
        singleThread: true,
      },
    },
    deps: {
      // Allow circular dependencies
      registerNodeLoader: true,
    },
    coverage: {
      provider: 'v8',
      reporter: ['text', 'text-summary', 'html', 'lcov'],
      exclude: [
        'node_modules/**',
        'src/**/*.test.ts',
        'src/**/__test__/**/*',
        'src/__test/**/*',
        'coverage/**',
        'dist/**',
        '**/__test__/**/*',
        // Add specific files that should be ignored from coverage
        'src/consts/numbers.ts', // Already has c8 ignore
        'src/validation/utils/regex.ts', // Already has c8 ignore
      ],
      all: true,
      src: ['src'],
      // Show coverage summary in terminal
      lines: 80,
      functions: 80,
      branches: 80,
      statements: 80,
    },
  },
});