import path from 'path';
import { defineProject } from 'vitest/config';

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

export default defineProject({
    plugins: [typeImportPlugin],
    resolve: {
        alias: {
            '@vrooli/shared': path.resolve(__dirname, './src'),
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
        setupFiles: ['./src/__test/setup.ts'],
        include: ['src/**/*.test.ts'],
        exclude: ['node_modules', 'dist'],
        // Use threads with conservative settings for stability
        pool: 'threads',
        poolOptions: {
            threads: {
                minThreads: 1,
                maxThreads: 4,
                isolate: true, // Better isolation to prevent issues
            },
        },
        // Suppress console output for speed
        onConsoleLog: () => false,
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
                '**/__test.*/**/*',
                // Add specific files that should be ignored from coverage
                'src/consts/numbers.ts', // Already has c8 ignore
                'src/validation/utils/regex.ts', // Already has c8 ignore
                // Pure type files with no executable code  
                'src/execution/index.ts', // Only re-exports
                'src/execution/types/context.ts', // Only interfaces and types
                'src/execution/types/index.ts', // Only re-exports
                // Enum-heavy type files (low testing value)
                'src/execution/types/resilience.ts', // 18 enums
                'src/execution/types/security.ts', // 17 enums  
                'src/execution/types/resources.ts', // 4 enums
                'src/execution/types/routine.ts', // 2 enums
                'src/execution/types/monitoring.ts', // 1 enum
                'src/execution/types/strategies.ts', // 1 enum
                'src/execution/types/core.ts', // 1 enum
                'src/execution/types/llm.ts', // 1 enum
            ],
        },
        // Increased timeouts for stability
        testTimeout: 60000,
        hookTimeout: 30000,
    },
});