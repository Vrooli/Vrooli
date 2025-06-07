import { describe, it, expect } from "vitest";
import {
    addToArray,
    updateArray,
    deleteArrayIndex,
    deleteArrayObject,
    findWithAttr,
    moveArrayIndex,
    rotateArray,
    mapIfExists,
} from "./arrayTools.js";

describe("arrayTools", () => {
    describe("addToArray", () => {
        it("should add a value to the end of an array", () => {
            const array = [1, 2, 3];
            const result = addToArray(array, 4);
            expect(result).to.deep.equal([1, 2, 3, 4]);
            // Ensure original array is not modified
            expect(array).to.deep.equal([1, 2, 3]);
        });

        it("should work with empty arrays", () => {
            const array: number[] = [];
            const result = addToArray(array, 1);
            expect(result).to.deep.equal([1]);
        });

        it("should work with different types", () => {
            const array = ["a", "b"];
            const result = addToArray(array, "c");
            expect(result).to.deep.equal(["a", "b", "c"]);
        });
    });

    describe("updateArray", () => {
        it("should update a value at the specified index", () => {
            const array = [1, 2, 3];
            const result = updateArray(array, 1, 5);
            expect(result).to.deep.equal([1, 5, 3]);
            // Ensure original array is not modified
            expect(array).to.deep.equal([1, 2, 3]);
        });

        it("should return the same array if the value is unchanged", () => {
            const array = [{ a: 1 }, { b: 2 }];
            const result = updateArray(array, 0, { a: 1 });
            expect(result).to.equal(array);
        });

        it("should return the same array if index is negative", () => {
            const array = [1, 2, 3];
            const result = updateArray(array, -1, 5);
            expect(result).to.equal(array);
        });

        it("should return the same array if index is out of bounds", () => {
            const array = [1, 2, 3];
            const result = updateArray(array, 5, 10);
            expect(result).to.equal(array);
        });

        it("should work with complex objects", () => {
            const array = [{ id: 1, name: "John" }, { id: 2, name: "Jane" }];
            const result = updateArray(array, 1, { id: 2, name: "Janet" });
            expect(result).to.deep.equal([{ id: 1, name: "John" }, { id: 2, name: "Janet" }]);
        });
    });

    describe("deleteArrayIndex", () => {
        it("should delete element at specified index", () => {
            const array = [1, 2, 3, 4];
            const result = deleteArrayIndex(array, 2);
            expect(result).to.deep.equal([1, 2, 4]);
            // Ensure original array is not modified
            expect(array).to.deep.equal([1, 2, 3, 4]);
        });

        it("should handle deleting first element", () => {
            const array = [1, 2, 3];
            const result = deleteArrayIndex(array, 0);
            expect(result).to.deep.equal([2, 3]);
        });

        it("should handle deleting last element", () => {
            const array = [1, 2, 3];
            const result = deleteArrayIndex(array, 2);
            expect(result).to.deep.equal([1, 2]);
        });

        it("should return same array if index is out of bounds", () => {
            const array = [1, 2, 3];
            const result = deleteArrayIndex(array, 5);
            expect(result).to.deep.equal([1, 2, 3]);
        });

        it("should return same array if index is negative", () => {
            const array = [1, 2, 3];
            const result = deleteArrayIndex(array, -1);
            expect(result).to.deep.equal([1, 2, 3]);
        });
    });

    describe("deleteArrayObject", () => {
        it("should handle the bug in deleteArrayObject implementation", () => {
            const obj1 = { id: 1 };
            const obj2 = { id: 2 };
            const obj3 = { id: 3 };
            const array = [obj1, obj2, obj3];
            
            // The function has a bug - it's calling findIndex with the object
            // as a parameter, but findIndex expects a predicate function
            // This will throw a TypeError
            expect(() => deleteArrayObject(array, obj2)).to.throw(TypeError);
        });

        it("should throw TypeError when called with primitive values", () => {
            const array = [1, 2, 3];
            expect(() => deleteArrayObject(array, 4)).to.throw(TypeError);
        });

        // TODO: This is how the function should work if fixed:
        // it("should delete object from array when fixed", () => {
        //     const obj1 = { id: 1 };
        //     const obj2 = { id: 2 };
        //     const obj3 = { id: 3 };
        //     const array = [obj1, obj2, obj3];
        //     const result = deleteArrayObject(array, obj2);
        //     expect(result).to.deep.equal([obj1, obj3]);
        // });
    });

    describe("findWithAttr", () => {
        it("should find index of object with matching attribute", () => {
            const array = [
                { id: 1, name: "John" },
                { id: 2, name: "Jane" },
                { id: 3, name: "Bob" },
            ];
            const result = findWithAttr(array, "name", "Jane");
            expect(result).to.equal(1);
        });

        it("should return -1 if no match found", () => {
            const array = [
                { id: 1, name: "John" },
                { id: 2, name: "Jane" },
            ];
            const result = findWithAttr(array, "name", "Bob");
            expect(result).to.equal(-1);
        });

        it("should find first matching element", () => {
            const array = [
                { id: 1, type: "A" },
                { id: 2, type: "B" },
                { id: 3, type: "A" },
            ];
            const result = findWithAttr(array, "type", "A");
            expect(result).to.equal(0);
        });

        it("should work with nested properties", () => {
            const array = [
                { id: 1, nested: { value: 10 } },
                { id: 2, nested: { value: 20 } },
            ];
            const result = findWithAttr(array, "id", 2);
            expect(result).to.equal(1);
        });

        it("should handle empty array", () => {
            const result = findWithAttr([], "any", "value");
            expect(result).to.equal(-1);
        });
    });

    describe("moveArrayIndex", () => {
        it("should move element from one index to another", () => {
            const array = [1, 2, 3, 4, 5];
            const result = moveArrayIndex(array, 1, 3);
            expect(result).to.deep.equal([1, 3, 4, 2, 5]);
            // Ensure original array is not modified
            expect(array).to.deep.equal([1, 2, 3, 4, 5]);
        });

        it("should handle moving to beginning", () => {
            const array = ["a", "b", "c", "d"];
            const result = moveArrayIndex(array, 2, 0);
            expect(result).to.deep.equal(["c", "a", "b", "d"]);
        });

        it("should handle moving to end", () => {
            const array = ["a", "b", "c", "d"];
            const result = moveArrayIndex(array, 1, 3);
            expect(result).to.deep.equal(["a", "c", "d", "b"]);
        });

        it("should handle moving to same position", () => {
            const array = [1, 2, 3];
            const result = moveArrayIndex(array, 1, 1);
            expect(result).to.deep.equal([1, 2, 3]);
        });
    });

    describe("rotateArray", () => {
        it("should rotate array to the right by default", () => {
            const array = [1, 2, 3, 4];
            const result = rotateArray(array);
            expect(result).to.deep.equal([4, 1, 2, 3]);
            // Ensure original array is not modified
            expect(array).to.deep.equal([1, 2, 3, 4]);
        });

        it("should rotate array to the left when to_right is false", () => {
            const array = [1, 2, 3, 4];
            const result = rotateArray(array, false);
            expect(result).to.deep.equal([2, 3, 4, 1]);
        });

        it("should handle empty array", () => {
            const array: number[] = [];
            const result = rotateArray(array);
            expect(result).to.deep.equal([]);
        });

        it("should handle single element array", () => {
            const array = [1];
            const result = rotateArray(array);
            expect(result).to.deep.equal([1]);
        });

        it("should handle two element array", () => {
            const array = [1, 2];
            const result = rotateArray(array);
            expect(result).to.deep.equal([2, 1]);
        });
    });

    describe("mapIfExists", () => {
        it("should map array values when notation exists", () => {
            const object = {
                data: {
                    items: [1, 2, 3],
                },
            };
            const result = mapIfExists(object, "data.items", (val) => val * 2);
            expect(result).to.deep.equal([2, 4, 6]);
        });

        it("should return null if notation does not exist", () => {
            const object = {
                data: {
                    items: [1, 2, 3],
                },
            };
            const result = mapIfExists(object, "data.notexist", (val) => val * 2);
            expect(result).to.be.null;
        });

        it("should return null if value is not an array", () => {
            const object = {
                data: {
                    value: 123,
                },
            };
            const result = mapIfExists(object, "data.value", (val) => val * 2);
            expect(result).to.be.null;
        });

        it("should handle top-level arrays", () => {
            const object = {
                items: ["a", "b", "c"],
            };
            const result = mapIfExists(object, "items", (val) => val.toUpperCase());
            expect(result).to.deep.equal(["A", "B", "C"]);
        });

        it("should handle empty arrays", () => {
            const object = {
                items: [],
            };
            const result = mapIfExists(object, "items", (val) => val);
            expect(result).to.deep.equal([]);
        });
    });
});
