import path from 'path';
import { defineConfig } from 'vitest/config';

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
            '@vrooli/shared': path.resolve(__dirname, '../shared/dist'),
        },
    },
    // Disable optimizeDeps for faster startup
    optimizeDeps: {
        disabled: true,
    },
    esbuild: {
        target: 'node18',
        format: 'esm',
    },
    test: {
        server: {
            deps: {
                // This helps with circular dependencies
                external: [],
                inline: ['@vrooli/shared'],
            },
        },
        globals: true,
        environment: 'node',
        globalSetup: './vitest.global-setup.ts',
        setupFiles: ['./vitest-sharp-mock-simple.ts', './src/__test/setup.ts'],
        include: [
            'src/**/*.test.ts',
            'src/**/__test.*/**/*.test.ts'
        ],
        exclude: ['node_modules', 'dist'],
        // Use single thread with longer timeout
        pool: 'forks',
        poolOptions: {
            forks: {
                singleFork: true,
                isolate: false, // Don't isolate to reduce overhead
            },
        },
        deps: {
            // Use optimizer instead of deprecated inline
            optimizer: {
                ssr: {
                    include: ['bcrypt'],
                },
            },
        },
        // Allow console output to debug setup issues
        onConsoleLog: () => {},
        coverage: {
            provider: 'v8',
            reporter: ['text', 'text-summary', 'json'],
            exclude: [
                'node_modules/**',
                'src/**/*.test.ts',
                'src/**/__test.*/**/*',
                'coverage/**',
                'dist/**',
                '**/*.d.ts',
                'src/db/migrations/**',
                'src/db/seeds/**',
                'src/endpoints/generated/**',
            ],
        },
        // Increase timeout for complex execution tests
        testTimeout: 60000,
        hookTimeout: 300000, // 5 minutes for setup/teardown
    },
});