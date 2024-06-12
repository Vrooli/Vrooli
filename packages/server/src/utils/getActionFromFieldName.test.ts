import { getActionFromFieldName } from "./getActionFromFieldName";

describe("getActionFromFieldName", () => {
    it("should identify Connect action", () => {
        expect(getActionFromFieldName("userConnect")).toEqual("Connect");
    });

    it("should identify Create action", () => {
        expect(getActionFromFieldName("postCreate")).toEqual("Create");
    });

    it("should identify Delete action", () => {
        expect(getActionFromFieldName("commentDelete")).toEqual("Delete");
    });

    it("should identify Disconnect action", () => {
        expect(getActionFromFieldName("profileDisconnect")).toEqual("Disconnect");
    });

    it("should identify Update action", () => {
        expect(getActionFromFieldName("avatarUpdate")).toEqual("Update");
    });

    it("should return null for field names without action suffix", () => {
        expect(getActionFromFieldName("username")).toBeNull();
    });

    it("should handle case sensitivity correctly", () => {
        // Depending on your function's intended behavior regarding case sensitivity,
        // this test can be adjusted.
        expect(getActionFromFieldName("taskUPDATE")).toBeNull(); // or 'Update' if case-insensitive
    });

    it("should return null for empty strings", () => {
        expect(getActionFromFieldName("")).toBeNull();
    });
});
