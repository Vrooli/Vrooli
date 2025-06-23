import { describe, it, expect } from "vitest";

describe("Simple Test", () => {
    it("should pass basic test", () => {
        expect(1 + 1).toBe(2);
    });

    it("should import from shared", async () => {
        const { generatePK } = await import("@vrooli/shared");
        const id = generatePK();
        expect(id).toBeDefined();
        expect(typeof id).toBe("bigint");
        expect(id.toString()).toMatch(/^\d+$/);
    });
});
