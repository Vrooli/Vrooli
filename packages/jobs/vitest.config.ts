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
            '@vrooli/server': path.resolve(__dirname, '../server/dist'),
            '@vrooli/shared': path.resolve(__dirname, '../shared/dist'),
        },
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
        setupFiles: ['./src/__test/setup.ts'],
        include: [
            'src/**/*.test.ts',
            'src/**/__test.*/**/*.test.ts'
        ],
        exclude: ['node_modules', 'dist'],
        // Use threads with more aggressive settings
        pool: 'threads',
        poolOptions: {
            threads: {
                minThreads: 2,
                maxThreads: 8,
                isolate: false, // Share context for speed
            },
        },
        deps: {
            // Allow circular dependencies
            optimizer: {
                ssr: {
                    include: ['bcrypt'],
                },
            },
        },
        // Suppress console output for speed
        onConsoleLog: () => false,
        coverage: {
            provider: 'v8',
            reporter: ['text', 'text-summary'],
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
        testTimeout: 30000,
        hookTimeout: 10000,
    },
});