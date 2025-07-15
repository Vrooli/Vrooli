import { defineConfig } from "vitest/config";

export default defineConfig({
    test: {
        globals: true,
        environment: "node",
        include: ["src/**/*.test.ts", "src/**/*.spec.ts"],
        coverage: {
            reporter: ["text", "json", "html"],
            exclude: [
                "node_modules/",
                "src/**/*.d.ts",
                "src/**/__test/**",
                "src/**/*.test.ts",
                "src/**/*.spec.ts",
            ],
        },
        setupFiles: ["./src/test-setup.ts"],
    },
});