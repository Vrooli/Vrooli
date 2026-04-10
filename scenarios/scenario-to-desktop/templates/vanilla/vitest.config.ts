/**
 * Vitest Configuration
 *
 * DOC: docs/internal/SEAMS.md#test-infrastructure
 *
 * Configures Vitest for testing Electron main process modules.
 * Key considerations:
 * - ESM + TypeScript support
 * - Electron module mocking
 * - Coverage reporting for extracted modules
 */

import { defineConfig } from "vitest/config";

export default defineConfig({
    test: {
        // Use Node environment (not jsdom) since we're testing Electron main process
        environment: "node",

        // Test file patterns
        include: [
            "**/__tests__/**/*.test.ts",
            "**/*.test.ts",
        ],
        exclude: [
            "**/node_modules/**",
            "**/dist/**",
        ],

        // Enable globals (describe, it, expect, vi) without imports
        globals: true,

        // Coverage configuration
        coverage: {
            provider: "v8",
            reporter: ["text", "html", "lcov"],
            include: [
                "telemetry/**/*.ts",
                "runtime/**/*.ts",
                "storage/**/*.ts",
                "ipc/**/*.ts",
                "auth/**/*.ts",
                "bundle/**/*.ts",
                "window-state/**/*.ts",
                "splash/**/*.ts",
            ],
            exclude: [
                "**/__tests__/**",
                "**/test-utils/**",
                "**/index.ts", // Barrel exports
                "**/*.d.ts",
            ],
        },

        // Timeout for async tests
        testTimeout: 10000,

        // Reporter configuration
        reporters: ["verbose"],

        // Setup file for global mocks and utilities
        setupFiles: ["./test-utils/setup.ts"],
    },

    // Resolve configuration
    resolve: {
        // Handle Electron imports in tests
        alias: {
            electron: "./test-utils/electron-mocks.ts",
        },
    },

    // ESBuild configuration for TypeScript
    esbuild: {
        target: "node18",
    },
});
