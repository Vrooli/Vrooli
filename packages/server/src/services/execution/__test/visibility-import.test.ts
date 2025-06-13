import { expect, describe, it } from "vitest";

describe("VisibilityType Import Test", () => {
    it("should import VisibilityType successfully", async () => {
        const { VisibilityType } = await import("@vrooli/shared");
        expect(VisibilityType.Own).toBe("Own");
    });
});