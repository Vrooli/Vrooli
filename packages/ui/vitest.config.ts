import path from 'path';
import { defineProject } from 'vitest/config';

export default defineProject({
    // Enable caching for faster test runs
    cacheDir: '.vitest',
    // Enable dependency optimization for faster module resolution
    optimizeDeps: {
        include: [
            '@vrooli/shared',
            'react',
            'react-dom',
            '@mui/material',
            '@mui/material/**',
            '@mui/lab',
            '@mui/styles',
            '@testing-library/react',
            'react-i18next',
            'i18next',
            'zustand',
            '@emotion/react',
            '@emotion/styled',
            'socket.io-client',
        ],
        exclude: [
            'fsevents',
        ],
    },
    resolve: {
        alias: [
            // UI internal aliases
            { find: "@", replacement: path.resolve(__dirname, "./src") },
            { find: "@local", replacement: path.resolve(__dirname, "./src") },
            // Use pre-built shared package for testing (much faster test collection)
            { find: "@vrooli/shared/test-fixtures", replacement: path.resolve(__dirname, "../shared/test-dist/__test/fixtures") },
            { find: "@vrooli/shared", replacement: path.resolve(__dirname, "../shared/test-dist") },
        ],
        extensions: ['.js', '.jsx', '.ts', '.tsx', '.json'],
    },
    esbuild: {
        jsx: 'automatic',
        target: 'es2022',
    },
    test: {
        // Speed up dependency resolution
        deps: {
            registerNodeLoader: true,
            // Only optimize packages that are safe to optimize
            optimizer: {
                web: {
                    enabled: false, // Disable web optimization to avoid MUI issues
                },
            },
        },
        globals: true,
        environment: 'happy-dom', // Pure DOM environment for component testing
        environmentOptions: {
            happyDOM: {
                settings: {
                    disableJavaScriptEvaluation: false,
                    disableJavaScriptFileLoading: true,
                    disableCSSFileLoading: true,
                    disableIframePageLoading: true,
                    disableComputedStyleRendering: true,
                    enableFileSystemHttpRequests: false,
                },
                // Add HTML5 doctype to prevent @hello-pangea/dnd warnings
                innerHTML: '<!doctype html><html><head><title>Test</title></head><body></body></html>',
            },
        },
        setupFiles: ['./src/__test/setup.vitest.ts'],
        include: ['src/**/*.test.{ts,tsx}'],
        exclude: [
            'node_modules',
            'dist',
            'src/**/*.stories.{ts,tsx}',
            // Exclude integration test patterns
            'src/**/*.integration.test.{ts,tsx}',
            'src/**/*.roundtrip.test.{ts,tsx}',
            'src/**/*.scenario.test.{ts,tsx}',
        ],

        // Performance optimizations
        pool: 'threads',
        poolOptions: {
            threads: {
                singleThread: false,
                isolate: true,
                useAtomics: true,
                minThreads: 1,
                maxThreads: 4, // Limit parallelization to prevent resource exhaustion
            }
        },
        testTimeout: 10000, // Default timeout for unit tests
        hookTimeout: 15000, // Hook timeout for setup/teardown

        // Better test isolation
        clearMocks: true,
        restoreMocks: true,
        mockReset: true,
        
        // Reduce console output for failed tests
        outputTruncateLength: 300, // Further reduce error output
        outputDiffMaxLines: 30, // Reduce diff output
        outputDiffMaxSize: 5000, // Reduce diff size
        
        // Disable stack traces for faster output
        outputStackLines: 0,

        // Coverage configuration
        coverage: {
            provider: 'v8',
            reporter: ['text', 'text-summary', 'json', 'html'],
            exclude: [
                'node_modules/**',
                'src/**/*.test.{ts,tsx}',
                'src/**/*.stories.{ts,tsx}',
                'src/**/__test.*/**/*',
                'src/**/__mocks__/**',
                'coverage/**',
                'dist/**',
                'test-dist/**',
                '**/*.d.ts',
                'src/generated/**',
            ],
        },
    },
});