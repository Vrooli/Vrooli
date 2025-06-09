import react from '@vitejs/plugin-react-swc';
import { defineConfig } from 'vitest/config';
import path from 'path';

export default defineConfig({
    plugins: [
        react({
            // SWC options for maximum speed
            devTarget: 'es2020',
            plugins: [],
            tsDecorators: false,
        }),
    ],
    resolve: {
        alias: [
            { find: "@vrooli/shared", replacement: path.resolve(__dirname, "../shared/src") },
        ],
    },
    // Performance optimizations
    optimizeDeps: {
        include: ['react', 'react-dom', '@mui/material', '@emotion/react', '@emotion/styled'],
        exclude: ['@vrooli/shared'],
    },
    test: {
        globals: true,
        environment: 'jsdom',
        setupFiles: ['./src/__test/setup.vitest.ts'],
        include: ['src/**/*.test.{ts,tsx}'],
        exclude: ['node_modules', 'dist', '**/*.optimized.ts'],
        
        // Use threads with aggressive parallelization
        pool: 'threads',
        poolOptions: {
            threads: {
                minThreads: 4,
                maxThreads: 12,
                isolate: false, // Share context for speed
                useAtomics: true, // Enable atomics for better thread coordination
            },
        },
        
        // Optimize deps loading
        deps: {
            optimizer: {
                web: {
                    enabled: true,
                    include: ['@mui/**', '@emotion/**', 'react', 'react-dom'],
                },
            },
        },
        
        environmentOptions: {
            jsdom: {
                url: 'http://localhost',
                // Reduce jsdom features we don't need
                resources: 'usable',
                runScripts: 'dangerously',
                pretendToBeVisual: false,
            },
        },
        
        // Performance settings
        onConsoleLog: () => false,
        css: false,
        reporters: process.env.CI ? ['default'] : ['basic'],
        
        // Shorter timeouts to catch slow tests
        testTimeout: 10000,
        hookTimeout: 5000,
        
        // Cache for faster reruns
        cache: {
            dir: '.vitest-cache',
        },
        
        // Bail early in CI
        bail: process.env.CI ? 1 : 0,
        
        // Limit concurrent test files
        maxConcurrency: 20,
        
        coverage: {
            enabled: false, // Disable by default
            provider: 'v8',
            reporter: ['text-summary'],
            exclude: [
                'node_modules/**',
                'src/**/*.test.{ts,tsx}',
                'coverage/**',
                'dist/**',
                'src/**/*.stories.{ts,tsx}',
                'src/__test/**',
                'src/__mocks__/**',
            ],
        },
    },
});