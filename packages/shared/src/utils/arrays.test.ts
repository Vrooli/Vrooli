import { expect } from "chai";
import { arraysEqual, difference, flatten, uniqBy } from "./arrays.js";

function numberComparator(a: number, b: number) { return a === b; }
function stringComparator(a: string, b: string) { return a === b; }
function objectComparator(a: { id: number }, b: { id: number }) { return a.id === b.id; }
function caseInsensitiveComparator(a: string, b: string) { return a.toLowerCase() === b.toLowerCase(); }

describe("arraysEqual", () => {
    it("should return true for equal number arrays", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2, 3];
        expect(arraysEqual(arr1, arr2, numberComparator)).to.equal(true);
    });

    it("should return false for different number arrays", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2, 4];
        expect(arraysEqual(arr1, arr2, numberComparator)).to.equal(false);
    });

    it("should return true for equal string arrays", () => {
        const arr1 = ["a", "b", "c"];
        const arr2 = ["a", "b", "c"];
        expect(arraysEqual(arr1, arr2, stringComparator)).to.equal(true);
    });

    it("should return false for different string arrays", () => {
        const arr1 = ["a", "b", "c"];
        const arr2 = ["a", "b", "d"];
        expect(arraysEqual(arr1, arr2, stringComparator)).to.equal(false);
    });

    it("should return true for equal object arrays", () => {
        const arr1 = [{ id: 1 }, { id: 2 }];
        const arr2 = [{ id: 1 }, { id: 2 }];
        expect(arraysEqual(arr1, arr2, objectComparator)).to.equal(true);
    });

    it("should return false for different object arrays", () => {
        const arr1 = [{ id: 1 }, { id: 2 }];
        const arr2 = [{ id: 1 }, { id: 3 }];
        expect(arraysEqual(arr1, arr2, objectComparator)).to.equal(false);
    });

    it("should return false for arrays of different lengths", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2];
        expect(arraysEqual(arr1, arr2, numberComparator)).to.equal(false);
    });

    it("should return true for custom comparison logic (case insensitive strings)", () => {
        const arr1 = ["A", "b", "C"];
        const arr2 = ["a", "B", "c"];
        expect(arraysEqual(arr1, arr2, caseInsensitiveComparator)).to.equal(true);
    });

    it("should return false for custom comparison logic with unequal elements", () => {
        const arr1 = ["A", "b", "C"];
        const arr2 = ["a", "B", "d"];
        expect(arraysEqual(arr1, arr2, caseInsensitiveComparator)).to.equal(false);
    });

    it("should handle empty arrays correctly", () => {
        const arr1: number[] = [];
        const arr2: number[] = [];
        expect(arraysEqual(arr1, arr2, numberComparator)).to.equal(true);
    });

    it("should return false when one array is empty and the other is not", () => {
        const arr1: number[] = [];
        const arr2 = [1];
        expect(arraysEqual(arr1, arr2, numberComparator)).to.equal(false);
    });
});

describe("difference", () => {
    it("should return items in first array not in second", () => {
        const arr1 = [1, 2, 3, 4, 5];
        const arr2 = [3, 4, 5, 6, 7];
        expect(difference(arr1, arr2)).to.deep.equal([1, 2]);
    });

    it("should return empty array when all items exist in second array", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2, 3, 4, 5];
        expect(difference(arr1, arr2)).to.deep.equal([]);
    });

    it("should return first array when second array is empty", () => {
        const arr1 = [1, 2, 3];
        const arr2: number[] = [];
        expect(difference(arr1, arr2)).to.deep.equal([1, 2, 3]);
    });

    it("should return empty array when first array is empty", () => {
        const arr1: number[] = [];
        const arr2 = [1, 2, 3];
        expect(difference(arr1, arr2)).to.deep.equal([]);
    });

    it("should handle arrays with duplicate values", () => {
        const arr1 = [1, 2, 2, 3, 3, 3];
        const arr2 = [2, 3];
        expect(difference(arr1, arr2)).to.deep.equal([1]);
    });

    it("should work with string arrays", () => {
        const arr1 = ["apple", "banana", "cherry", "date"];
        const arr2 = ["banana", "date", "elderberry"];
        expect(difference(arr1, arr2)).to.deep.equal(["apple", "cherry"]);
    });

    it("should work with object arrays", () => {
        const obj1 = { id: 1 };
        const obj2 = { id: 2 };
        const obj3 = { id: 3 };
        const arr1 = [obj1, obj2, obj3];
        const arr2 = [obj2];
        expect(difference(arr1, arr2)).to.deep.equal([obj1, obj3]);
    });

    it("should handle null and undefined values", () => {
        const arr1 = [1, null, 2, undefined, 3];
        const arr2 = [null, 3];
        expect(difference(arr1, arr2)).to.deep.equal([1, 2, undefined]);
    });
});

describe("flatten", () => {
    it("should flatten a nested array one level deep", () => {
        const nested = [[1, 2], [3, 4], [5, 6]];
        expect(flatten(nested)).to.deep.equal([1, 2, 3, 4, 5, 6]);
    });

    it("should handle empty arrays", () => {
        const nested = [[], [], []];
        expect(flatten(nested)).to.deep.equal([]);
    });

    it("should handle mixed nested and non-nested elements", () => {
        const mixed = [1, [2, 3], 4, [5]];
        expect(flatten(mixed)).to.deep.equal([1, 2, 3, 4, 5]);
    });

    it("should only flatten one level deep", () => {
        const deeplyNested = [[1, [2, 3]], [4, [5, [6]]]];
        expect(flatten(deeplyNested)).to.deep.equal([1, [2, 3], 4, [5, [6]]]);
    });

    it("should work with string arrays", () => {
        const nested = [["a", "b"], ["c", "d"], ["e"]];
        expect(flatten(nested)).to.deep.equal(["a", "b", "c", "d", "e"]);
    });

    it("should handle single-element arrays", () => {
        const nested = [[1], [2], [3]];
        expect(flatten(nested)).to.deep.equal([1, 2, 3]);
    });

    it("should return empty array for empty input", () => {
        expect(flatten([])).to.deep.equal([]);
    });

    it("should handle arrays with different types", () => {
        const mixed = [[1, "two"], [true, null], [undefined, { key: "value" }]];
        expect(flatten(mixed)).to.deep.equal([1, "two", true, null, undefined, { key: "value" }]);
    });
});

describe("uniqBy", () => {
    it("should find unique items by property", () => {
        const users = [
            { id: 1, name: "Alice" },
            { id: 2, name: "Bob" },
            { id: 1, name: "Alice2" },
            { id: 3, name: "Charlie" },
        ];
        const unique = uniqBy(users, item => item.id);
        expect(unique).to.deep.equal([
            { id: 1, name: "Alice" },
            { id: 2, name: "Bob" },
            { id: 3, name: "Charlie" },
        ]);
    });

    it("should find unique items by transformation", () => {
        const strings = ["apple", "Apple", "banana", "BANANA", "cherry"];
        const unique = uniqBy(strings, item => item.toLowerCase());
        expect(unique).to.deep.equal(["apple", "banana", "cherry"]);
    });

    it("should handle empty array", () => {
        expect(uniqBy([], item => item)).to.deep.equal([]);
    });

    it("should handle array with all unique items", () => {
        const numbers = [1, 2, 3, 4, 5];
        expect(uniqBy(numbers, item => item)).to.deep.equal([1, 2, 3, 4, 5]);
    });

    it("should handle array with all duplicate items", () => {
        const numbers = [1, 1, 1, 1];
        expect(uniqBy(numbers, item => item)).to.deep.equal([1]);
    });

    it("should work with complex iteratee functions", () => {
        const items = [
            { x: 1, y: 2 },
            { x: 2, y: 1 },
            { x: 1, y: 2 },
            { x: 3, y: 3 },
        ];
        const unique = uniqBy(items, item => `${item.x},${item.y}`);
        expect(unique).to.deep.equal([
            { x: 1, y: 2 },
            { x: 2, y: 1 },
            { x: 3, y: 3 },
        ]);
    });

    it("should preserve the first occurrence", () => {
        const items = [
            { id: 1, value: "first" },
            { id: 1, value: "second" },
            { id: 1, value: "third" },
        ];
        const unique = uniqBy(items, item => item.id);
        expect(unique).to.deep.equal([{ id: 1, value: "first" }]);
    });

    it("should handle null and undefined values", () => {
        const items = [null, undefined, null, 1, undefined, 2, 1];
        const unique = uniqBy(items, item => item);
        expect(unique).to.deep.equal([null, undefined, 1, 2]);
    });
});
