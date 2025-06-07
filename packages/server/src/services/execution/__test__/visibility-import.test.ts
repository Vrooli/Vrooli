import { describe, it } from "mocha";
import { expect } from "chai";

describe("VisibilityType Import Test", () => {
    it("should import VisibilityType successfully", async () => {
        const { VisibilityType } = await import("@vrooli/shared");
        expect(VisibilityType.Own).to.equal("Own");
    });
});