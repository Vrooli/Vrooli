import { noNull } from "./noNull";

describe("noNull", () => {
    it("returns the first valid argument when all are valid", () => {
        expect(noNull(1, 2, 3)).toBe(1);
    });

    it("skips null and undefined to return the first valid value", () => {
        expect(noNull(null, undefined, "valid", "also valid")).toBe("valid");
    });

    it("returns undefined when all arguments are null", () => {
        expect(noNull(null, null)).toBeUndefined();
    });

    it("returns undefined when all arguments are undefined", () => {
        expect(noNull(undefined, undefined)).toBeUndefined();
    });

    it("returns undefined for a mix of null and undefined values", () => {
        expect(noNull(null, undefined, null)).toBeUndefined();
    });

    it("returns undefined when called with no arguments", () => {
        expect(noNull()).toBeUndefined();
    });

    it("works with complex objects", () => {
        const obj = { key: "value" };
        expect(noNull(null, obj)).toBe(obj);
    });

    it("correctly returns a function argument", () => {
        const func = () => "test";
        expect(noNull(undefined, func)).toBe(func);
    });
});
