/**
 * Minimal Test
 * 
 * Basic test to verify the execution framework can import and instantiate
 */

import { describe, it, expect } from "vitest";

describe("Minimal Execution Test", () => {
    it("should pass a basic test", () => {
        expect(true).toBe(true);
    });

    it("should be able to import basic modules", async () => {
        // Test that basic imports work
        const { DbProvider } = await import("../../db/provider.js");
        expect(DbProvider).toBeDefined();
        expect(typeof DbProvider.get).toBe("function");
    });

    it("should be able to import execution modules", async () => {
        // Test that execution modules can be imported
        const { RoutineFactory } = await import("./factories/routine/RoutineFactory.js");
        expect(RoutineFactory).toBeDefined();
        expect(typeof RoutineFactory).toBe("function");
    });
});
