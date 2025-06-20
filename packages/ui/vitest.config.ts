import { defineConfig } from 'vitest/config';
import path from 'path';

export default defineConfig({
    resolve: {
        alias: [
            // Use pre-built shared package for faster loading
            { find: "@vrooli/shared/translations", replacement: path.resolve(__dirname, "../shared/dist/translations") },
            { find: "@vrooli/shared/utils", replacement: path.resolve(__dirname, "../shared/dist/utils") },
            { find: "@vrooli/shared/validation", replacement: path.resolve(__dirname, "../shared/dist/validation") },
            { find: "@vrooli/shared/validation/models", replacement: path.resolve(__dirname, "../shared/dist/validation/models") },
            { find: "@vrooli/shared/__test/fixtures", replacement: path.resolve(__dirname, "../shared/dist/__test/fixtures") },
            { find: "@vrooli/shared/validation/forms", replacement: path.resolve(__dirname, "../shared/dist/validation/forms") },
            { find: "@vrooli/shared/validation/utils", replacement: path.resolve(__dirname, "../shared/dist/validation/utils") },
            { find: "@vrooli/shared", replacement: path.resolve(__dirname, "../shared/dist") },
        ],
        extensions: ['.js', '.jsx', '.ts', '.tsx', '.json'],
    },
    esbuild: {
        // Use SWC for React transpilation (faster than default)
        jsx: 'automatic',
        target: 'es2022',
    },
    test: {
        globals: true,
        environment: 'happy-dom',
        include: ['src/**/*.test.{ts,tsx}'],
        exclude: [
            'node_modules', 
            'dist', 
            'src/**/*.stories.{ts,tsx}',
            // Exclude large test files that should be split
            '**/tasks.test.ts',
            '**/messages.test.ts'
        ],
        
        // Optimized for speed and reliability
        pool: 'threads',
        poolOptions: {
            threads: {
                maxThreads: 2,
                minThreads: 1,
                isolate: true,
            },
        },
        
        // Performance optimizations
        css: false,
        silent: false,
        reporter: 'default',
        
        // Reasonable timeouts
        testTimeout: 10000,
        hookTimeout: 10000,
        teardownTimeout: 1000,
        
        // Parallel execution for better performance
        maxConcurrency: 2,
        fileParallelism: false,
        
        // Better test isolation
        clearMocks: true,
        restoreMocks: true,
        mockReset: true,
        
        // Disable expensive features during testing
        typecheck: { enabled: false },
        
        // Coverage configuration (enabled only when explicitly requested)
        coverage: {
            provider: 'v8',
            reporter: ['text', 'text-summary', 'json'],
            exclude: [
                'node_modules/**',
                'src/**/*.test.{ts,tsx}',
                'src/**/*.stories.{ts,tsx}',
                'src/**/__test.*/**/*',
                'coverage/**',
                'dist/**',
                '**/*.d.ts',
                'src/generated/**',
            ],
        },
        watch: false,
        
        setupFiles: ['./src/__test/setup.vitest.ts'],
    },
});