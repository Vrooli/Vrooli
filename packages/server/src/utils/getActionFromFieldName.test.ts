import { expect } from "chai";
import { getActionFromFieldName } from "./getActionFromFieldName.js";

describe("getActionFromFieldName", () => {
    it("should identify Connect action", () => {
        expect(getActionFromFieldName("userConnect")).to.deep.equal("Connect");
    });

    it("should identify Create action", () => {
        expect(getActionFromFieldName("postCreate")).to.deep.equal("Create");
    });

    it("should identify Delete action", () => {
        expect(getActionFromFieldName("commentDelete")).to.deep.equal("Delete");
    });

    it("should identify Disconnect action", () => {
        expect(getActionFromFieldName("profileDisconnect")).to.deep.equal("Disconnect");
    });

    it("should identify Update action", () => {
        expect(getActionFromFieldName("avatarUpdate")).to.deep.equal("Update");
    });

    it("should return null for field names without action suffix", () => {
        expect(getActionFromFieldName("username")).to.be.null;
    });

    it("should handle case sensitivity correctly", () => {
        // Depending on your function's intended behavior regarding case sensitivity,
        // this test can be adjusted.
        expect(getActionFromFieldName("taskUPDATE")).to.be.null; // or 'Update' if case-insensitive
    });

    it("should return null for empty strings", () => {
        expect(getActionFromFieldName("")).to.be.null;
    });
});
