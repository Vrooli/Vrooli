import path from 'path';
import { defineProject } from 'vitest/config';

// Custom plugin to handle ../types.js imports
const typeImportPlugin = {
    name: 'resolve-type-imports',
    resolveId(id: string, importer?: string) {
        // Handle ../types.js and ../../types.js
        if ((id === '../types.js' || id === '../../types.js') && importer) {
            // Check if this is coming from the server package
            if (importer && importer.includes('/server/src/')) {
                // Resolve to server's types.d.ts
                return path.resolve(__dirname, '../server/src/types.d.ts');
            }
            // Default to current directory types
            const srcDir = path.resolve(__dirname, 'src');
            return path.join(srcDir, 'types.d.ts');
        }
        return null;
    }
};

export default defineProject({
    plugins: [typeImportPlugin],
    resolve: {
        alias: {
            '@vrooli/shared': path.resolve(__dirname, '../shared/dist'),
            '@vrooli/server': path.resolve(__dirname, '../server/dist'),
        },
    },
    // Optimize dependencies for faster startup
    optimizeDeps: {
        include: ['@vrooli/shared', 'vitest'],
        exclude: ['@vrooli/server'], // Large server package excluded for faster startup
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
        // Reduce console output for performance unless verbose mode
        onConsoleLog: (log) => {
            if (process.env.TEST_VERBOSE === "true") return true;
            // Only show errors and warnings by default
            return log.includes("error") || log.includes("Error") || log.includes("⚠") || log.includes("FAIL");
        },
        coverage: {
            provider: 'v8',
            reporter: ['text', 'text-summary', 'json'],
            exclude: [
                'node_modules/**',
                'src/**/*.test.ts',
                'src/**/__test.*/**/*',
                'coverage/**',
                'dist/**',
                'test-dist/**',
                '**/*.d.ts',
                'src/db/migrations/**',
                'src/db/seeds/**',
            ],
        },
        // Increase timeout for complex execution tests
        testTimeout: 60000,
        hookTimeout: 300000, // 5 minutes for setup/teardown
    },
});