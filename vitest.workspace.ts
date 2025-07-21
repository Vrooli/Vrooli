import { defineWorkspace } from "vitest/config";

export default defineWorkspace([
    // Server package - Node environment
    {
        extends: "./packages/server/vitest.config.ts",
        test: {
            name: "server",
            root: "./packages/server",
            env: {
                NODE_NO_WARNINGS: "1",
            },
        },
    },
    // UI package - Happy DOM environment for component tests
    {
        extends: "./packages/ui/vitest.config.ts",
        test: {
            name: "ui",
            root: "./packages/ui",
        },
    },
    // Jobs package - Node environment
    {
        extends: "./packages/jobs/vitest.config.ts",
        test: {
            name: "jobs",
            root: "./packages/jobs",
        },
    },
    // Shared package - Node environment
    {
        extends: "./packages/shared/vitest.config.ts",
        test: {
            name: "shared",
            root: "./packages/shared",
        },
    },
    // Integration package - Node environment with testcontainers
    {
        extends: "./packages/integration/vitest.config.ts",
        test: {
            name: "integration",
            root: "./packages/integration",
        },
    },
    // CLI package - Node environment
    {
        extends: "./packages/cli/vitest.config.ts",
        test: {
            name: "cli",
            root: "./packages/cli",
        },
    },
]);
