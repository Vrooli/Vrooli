// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-07-13
import { describe, expect, it } from "vitest";
import { ApiClient, ConfigManager } from "./index.js";

describe("Utils Index Exports", () => {
    it("should export ApiClient", () => {
        expect(ApiClient).toBeDefined();
        expect(typeof ApiClient).toBe("function");
    });

    it("should export ConfigManager", () => {
        expect(ConfigManager).toBeDefined();
        expect(typeof ConfigManager).toBe("function");
    });

    it("should allow importing types", async () => {
        // Test that types can be imported without errors
        const typeImports = await import("./index.js");
        expect(typeImports.ApiClient).toBeDefined();
        expect(typeImports.ConfigManager).toBeDefined();
    });
});
