import { defineConfig } from "vitest/config";
// import path from "path";

export default defineConfig({
    test: {
        globals: true,
        environment: "node", // Pure Node environment - no DOM
        globalSetup: "./src/setup/global-setup.ts",
        setupFiles: ["./vitest-sharp-mock-simple.ts", "./src/setup/test-setup.ts"],
        include: ["src/**/*.test.ts"],
        exclude: ["node_modules", "dist"],
        
        // Use single thread for database isolation
        pool: "forks",
        poolOptions: {
            forks: {
                singleFork: true,
                isolate: false,
            },
        },

        // Longer timeouts for integration tests
        testTimeout: 120000, // 2 minutes per test
        hookTimeout: 300000, // 5 minutes for setup/teardown
        teardownTimeout: 30000, // 30 seconds for cleanup

        // Sequential execution for database consistency
        maxConcurrency: 1,
        fileParallelism: false,

        // Better test isolation
        clearMocks: true,
        restoreMocks: true,
        mockReset: true,

        // Coverage settings
        coverage: {
            provider: "v8",
            reporter: ["text", "json", "html"],
            exclude: [
                "node_modules/",
                "src/setup/",
                "src/fixtures/",
                "dist/**",
                "test-dist/**",
                "**/*.d.ts",
                "**/*.config.*",
                "**/mockData",
            ],
        },
    },
});
