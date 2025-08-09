import path from 'path';
import os from 'os';
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

// Calculate optimal number of forks based on system resources
const calculateMaxForks = () => {
    const cpuCount = os.cpus().length;
    const totalMemoryGB = os.totalmem() / (1024 * 1024 * 1024);
    
    // Each fork needs ~2GB of memory for our test suite
    const memoryBasedLimit = Math.floor(totalMemoryGB / 2);
    
    // Leave at least 1 CPU for the system
    const cpuBasedLimit = Math.max(1, cpuCount - 1);
    
    // Take the minimum and cap at 8
    return Math.min(8, memoryBasedLimit, cpuBasedLimit);
};

export default defineProject({
    plugins: [typeImportPlugin],
    // Let Node's module resolution handle @vrooli/shared imports using package.json exports
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
        // Parallel execution configuration
        pool: 'forks',
        poolOptions: {
            forks: {
                singleFork: false, // Enable parallel execution
                isolate: true,     // Isolate test files for better stability
                minForks: 1,       // Minimum number of forks
                maxForks: calculateMaxForks(),
                env: {
                    NODE_NO_WARNINGS: '1',
                    NODE_OPTIONS: '--max-old-space-size=2048' // Less memory per fork
                }
            },
        },
        // Randomize test order to detect interdependencies
        sequence: {
            shuffle: true,
        },
        // CRITICAL: Always set bail to 0 to run ALL tests and see complete picture
        // When bail=1, vitest stops after FIRST failure, making it appear like only
        // a few test files are "discovered" when actually all 189 files are found
        // but execution stops early. This creates confusing "test discovery issues"
        // that are actually just early bailout from the first failing test.
        bail: 0, // Run all tests regardless of failures - essential for proper test coverage analysis
        // Retry flaky tests
        retry: process.env.CI ? 2 : 0,
        deps: {
            // Use optimizer instead of deprecated inline
            optimizer: {
                ssr: {
                    include: ['bcrypt', 'rrule'],
                },
            },
        },
        // Suppress console logs unless LOG_LEVEL is set to 'debug'
        onConsoleLog: (log, type) => {
            // Allow console output only if LOG_LEVEL is explicitly set to debug
            if (process.env.LOG_LEVEL === 'debug') {
                return; // undefined means print the log
            }
            // Return false to suppress the log
            return false;
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
                'src/endpoints/generated/**',
            ],
        },
        // Lower timeouts for parallel mode (faster failure detection)
        testTimeout: 30000,     // 30 seconds per test
        hookTimeout: 120000,    // 2 minutes for setup/teardown
    },
});