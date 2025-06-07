import { defineConfig } from 'vitest/config';
import path from 'path';
import react from '@vitejs/plugin-react-swc';
import fs from 'fs';
import { iconsSpritesheet } from 'vite-plugin-icons-spritesheet';

// Custom plugin to resolve .js extensions to .ts/.tsx files
function resolveImportExtensions() {
    return {
        name: 'resolve-import-extensions',
        resolveId(source, importer) {
            if (source.endsWith('.js')) {
                const tsxPath = source.replace(/\.js$/, '.tsx');
                const tsPath = source.replace(/\.js$/, '.ts');
                const tsxAbsolutePath = path.resolve(path.dirname(importer), tsxPath);
                const tsAbsolutePath = path.resolve(path.dirname(importer), tsPath);
                if (fs.existsSync(tsxAbsolutePath)) {
                    return tsxAbsolutePath;
                }
                if (fs.existsSync(tsAbsolutePath)) {
                    return tsAbsolutePath;
                }
            }
            return null;
        },
    };
}

export default defineConfig({
  plugins: [
    react(),
    resolveImportExtensions(),
    iconsSpritesheet([
      {
        withTypes: true,
        inputDir: "src/assets/icons/common",
        outputDir: "public/sprites",
        typesOutputFile: "src/icons/types/commonIcons.ts",
        fileName: "common-sprite.svg",
      },
      {
        withTypes: true,
        inputDir: "src/assets/icons/routine",
        outputDir: "public/sprites",
        typesOutputFile: "src/icons/types/routineIcons.ts",
        fileName: "routine-sprite.svg",
      },
      {
        withTypes: true,
        inputDir: "src/assets/icons/service",
        outputDir: "public/sprites",
        typesOutputFile: "src/icons/types/serviceIcons.ts",
        fileName: "service-sprite.svg",
      },
      {
        withTypes: true,
        inputDir: "src/assets/icons/text",
        outputDir: "public/sprites",
        typesOutputFile: "src/icons/types/textIcons.ts",
        fileName: "text-sprite.svg",
      }
    ]),
  ],
  resolve: {
    alias: [
      { find: "@vrooli/shared", replacement: path.resolve(__dirname, "../shared/src") },
    ],
    extensionAlias: {
      '.js': ['.ts', '.tsx', '.js'],
      '.jsx': ['.tsx', '.jsx'],
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/__test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    exclude: ['node_modules', 'dist'],
    pool: 'forks',
    deps: {
      registerNodeLoader: true,
    },
    environmentOptions: {
      jsdom: {
        url: 'http://localhost',
      },
    },
    onConsoleLog: (log) => {
      // Suppress the defineProperty error message
      if (log.includes('Cannot read properties of undefined')) return false;
      return true;
    },
    coverage: {
      provider: 'v8',
      reporter: ['text', 'text-summary', 'html', 'lcov'],
      exclude: [
        'node_modules/**',
        'src/**/*.test.{ts,tsx}',
        'src/**/__test__/**/*',
        'src/__test/**/*',
        'coverage/**',
        'dist/**',
        '**/__test__/**/*',
        // Storybook files
        'src/**/*.stories.{ts,tsx}',
        // Generated icon types
        'src/icons/types/**',
        // Service worker
        'src/service-worker.ts',
        'src/serviceWorkerRegistration.ts',
      ],
      all: true,
      src: ['src'],
      lines: 80,
      functions: 80,
      branches: 80,
      statements: 80,
    },
  },
});