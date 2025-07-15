// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-07-13
import { describe, it, expect } from "vitest";
import type { CliOptions } from "./types.js";

describe("Utils Types", () => {
    describe("CliOptions", () => {
        it("should have optional debug property", () => {
            const options: CliOptions = { debug: true };
            expect(options.debug).toBe(true);
        });

        it("should have optional json property", () => {
            const options: CliOptions = { json: true };
            expect(options.json).toBe(true);
        });

        it("should have optional profile property", () => {
            const options: CliOptions = { profile: "test" };
            expect(options.profile).toBe("test");
        });

        it("should allow all properties together", () => {
            const options: CliOptions = {
                debug: true,
                json: false,
                profile: "production",
            };
            expect(options.debug).toBe(true);
            expect(options.json).toBe(false);
            expect(options.profile).toBe("production");
        });

        it("should allow empty options", () => {
            const options: CliOptions = {};
            expect(options.debug).toBeUndefined();
            expect(options.json).toBeUndefined();
            expect(options.profile).toBeUndefined();
        });
    });

    it("should allow importing axios types", async () => {
        // Test that re-exported types can be imported without errors
        const typeImports = await import("./types.js");
        expect(typeImports).toBeDefined();
    });
});
