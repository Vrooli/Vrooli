// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-06-24
import { describe, expect, it } from "vitest";
import { getSetupStatus } from "./setup.js";

describe("test setup module", () => {
    it("should export getSetupStatus function", () => {
        expect(typeof getSetupStatus).toBe("function");
    });

    it("should return setup status object with expected properties", () => {
        const status = getSetupStatus();
        
        expect(status).toHaveProperty("modelMap");
        expect(status).toHaveProperty("dbProvider");
        expect(status).toHaveProperty("mocks");
        expect(status).toHaveProperty("idGenerator");
        
        expect(typeof status.modelMap).toBe("boolean");
        expect(typeof status.dbProvider).toBe("boolean");
        expect(typeof status.mocks).toBe("boolean");
        expect(typeof status.idGenerator).toBe("boolean");
    });

    it("should indicate components are initialized", () => {
        const status = getSetupStatus();
        
        // These should be true after beforeAll has run
        expect(status.modelMap).toBe(true);
        expect(status.dbProvider).toBe(true);
        expect(status.mocks).toBe(true);
    });
});
