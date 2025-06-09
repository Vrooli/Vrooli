import { defineConfig } from 'vitest/config';
import path from 'path';

export default defineConfig({
    resolve: {
        alias: [
            { find: "@vrooli/shared", replacement: path.resolve(__dirname, "../shared/src") },
        ],
    },
    test: {
        globals: true,
        environment: 'happy-dom',
        include: ['src/**/*.test.{ts,tsx}'],
        exclude: ['node_modules', 'dist'],
        
        // Optimized for speed
        pool: 'threads',
        poolOptions: {
            threads: {
                singleThread: true,
                isolate: false,
            },
        },
        
        css: false,
        silent: true,
        
        testTimeout: 2000,
        hookTimeout: 1000,
        teardownTimeout: 100,
        
        maxConcurrency: 1,
        fileParallelism: false,
        
        coverage: { enabled: false },
        typecheck: { enabled: false },
        watch: false,
        setupFiles: ['./src/__test/setup.vitest.ts'],
    },
});