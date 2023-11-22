import { orDefault } from "./orDefault";

describe("orDefault", () => {
    // Arrays
    it("should return the default array if the existing array is empty", () => {
        const defaultArray = [1, 2, 3];
        expect(orDefault<any>([], defaultArray)).toEqual(defaultArray);
    });

    it("should return the existing array if it is non-empty", () => {
        const existingArray = [4, 5, 6];
        expect(orDefault<any>(existingArray, [1, 2, 3])).toEqual(existingArray);
    });

    // Objects
    it("should return the default object if the existing object is empty", () => {
        const defaultObj = { a: 1, b: 2 };
        expect(orDefault<any>({}, defaultObj)).toEqual(defaultObj);
    });

    it("should return the existing object if it is non-empty", () => {
        const existingObj = { x: 10, y: 20 };
        expect(orDefault<any>(existingObj, { a: 1, b: 2 })).toEqual(existingObj);
    });

    // Null and Undefined
    it("should return the default value if the existing value is null or undefined", () => {
        expect(orDefault(null, [1, 2, 3])).toEqual([1, 2, 3]);
        expect(orDefault(undefined, { a: 1, b: 2 })).toEqual({ a: 1, b: 2 });
    });
});

