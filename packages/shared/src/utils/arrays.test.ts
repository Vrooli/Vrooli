import { describe, it, expect } from "vitest";
import { arraysEqual, difference, flatten, uniqBy } from "./arrays.js";

function numberComparator(a: number, b: number) { return a === b; }
function stringComparator(a: string, b: string) { return a === b; }
function objectComparator(a: { id: number }, b: { id: number }) { return a.id === b.id; }
function caseInsensitiveComparator(a: string, b: string) { return a.toLowerCase() === b.toLowerCase(); }

describe("arraysEqual", () => {
    it("should return true for equal number arrays", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2, 3];
        expect(arraysEqual(arr1, arr2, numberComparator)).toBe(true);
    });

    it("should return false for different number arrays", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2, 4];
        expect(arraysEqual(arr1, arr2, numberComparator)).toBe(false);
    });

    it("should return true for equal string arrays", () => {
        const arr1 = ["a", "b", "c"];
        const arr2 = ["a", "b", "c"];
        expect(arraysEqual(arr1, arr2, stringComparator)).toBe(true);
    });

    it("should return false for different string arrays", () => {
        const arr1 = ["a", "b", "c"];
        const arr2 = ["a", "b", "d"];
        expect(arraysEqual(arr1, arr2, stringComparator)).toBe(false);
    });

    it("should return true for equal object arrays", () => {
        const arr1 = [{ id: 1 }, { id: 2 }];
        const arr2 = [{ id: 1 }, { id: 2 }];
        expect(arraysEqual(arr1, arr2, objectComparator)).toBe(true);
    });

    it("should return false for different object arrays", () => {
        const arr1 = [{ id: 1 }, { id: 2 }];
        const arr2 = [{ id: 1 }, { id: 3 }];
        expect(arraysEqual(arr1, arr2, objectComparator)).toBe(false);
    });

    it("should return false for arrays of different lengths", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2];
        expect(arraysEqual(arr1, arr2, numberComparator)).toBe(false);
    });

    it("should return true for custom comparison logic (case insensitive strings)", () => {
        const arr1 = ["A", "b", "C"];
        const arr2 = ["a", "B", "c"];
        expect(arraysEqual(arr1, arr2, caseInsensitiveComparator)).toBe(true);
    });

    it("should return false for custom comparison logic with unequal elements", () => {
        const arr1 = ["A", "b", "C"];
        const arr2 = ["a", "B", "d"];
        expect(arraysEqual(arr1, arr2, caseInsensitiveComparator)).toBe(false);
    });

    it("should handle empty arrays correctly", () => {
        const arr1: number[] = [];
        const arr2: number[] = [];
        expect(arraysEqual(arr1, arr2, numberComparator)).toBe(true);
    });

    it("should return false when one array is empty and the other is not", () => {
        const arr1: number[] = [];
        const arr2 = [1];
        expect(arraysEqual(arr1, arr2, numberComparator)).toBe(false);
    });
});

describe("difference", () => {
    it("should return items in first array not in second", () => {
        const arr1 = [1, 2, 3, 4, 5];
        const arr2 = [3, 4, 5, 6, 7];
        expect(difference(arr1, arr2)).toEqual([1, 2]);
    });

    it("should return empty array when all items exist in second array", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2, 3, 4, 5];
        expect(difference(arr1, arr2)).toEqual([]);
    });

    it("should return first array when second array is empty", () => {
        const arr1 = [1, 2, 3];
        const arr2: number[] = [];
        expect(difference(arr1, arr2)).toEqual([1, 2, 3]);
    });

    it("should return empty array when first array is empty", () => {
        const arr1: number[] = [];
        const arr2 = [1, 2, 3];
        expect(difference(arr1, arr2)).toEqual([]);
    });

    it("should handle arrays with duplicate values", () => {
        const arr1 = [1, 2, 2, 3, 3, 3];
        const arr2 = [2, 3];
        expect(difference(arr1, arr2)).toEqual([1]);
    });

    it("should work with string arrays", () => {
        const arr1 = ["apple", "banana", "cherry", "date"];
        const arr2 = ["banana", "date", "elderberry"];
        expect(difference(arr1, arr2)).toEqual(["apple", "cherry"]);
    });

    it("should work with object arrays", () => {
        const obj1 = { id: 1 };
        const obj2 = { id: 2 };
        const obj3 = { id: 3 };
        const arr1 = [obj1, obj2, obj3];
        const arr2 = [obj2];
        expect(difference(arr1, arr2)).toEqual([obj1, obj3]);
    });

    it("should handle null and undefined values", () => {
        const arr1 = [1, null, 2, undefined, 3];
        const arr2 = [null, 3];
        expect(difference(arr1, arr2)).toEqual([1, 2, undefined]);
    });

    it("should handle arrays with identical elements", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2, 3];
        expect(difference(arr1, arr2)).toEqual([]);
    });

    it("should handle large arrays efficiently", () => {
        const arr1 = Array.from({ length: 1000 }, (_, i) => i);
        const arr2 = Array.from({ length: 500 }, (_, i) => i * 2);
        
        const result = difference(arr1, arr2);
        
        // Should contain odd numbers and even numbers not in arr2
        expect(result).toContain(1); // Odd number
        expect(result).toContain(3); // Odd number
        expect(result).not.toContain(0); // In arr2
        expect(result).not.toContain(2); // In arr2
    });

    it("should maintain order of elements from first array", () => {
        const arr1 = [3, 1, 4, 1, 5, 9, 2, 6];
        const arr2 = [1, 5, 9];
        const result = difference(arr1, arr2);
        
        expect(result).toEqual([3, 4, 2, 6]);
        // Should maintain original order: 3, 4, 2, 6
    });
});

describe("flatten", () => {
    it("should flatten a nested array one level deep", () => {
        const nested = [[1, 2], [3, 4], [5, 6]];
        expect(flatten(nested)).toEqual([1, 2, 3, 4, 5, 6]);
    });

    it("should handle empty arrays", () => {
        const nested = [[], [], []];
        expect(flatten(nested)).toEqual([]);
    });

    it("should handle mixed nested and non-nested elements", () => {
        const mixed = [1, [2, 3], 4, [5]];
        expect(flatten(mixed)).toEqual([1, 2, 3, 4, 5]);
    });

    it("should only flatten one level deep", () => {
        const deeplyNested = [[1, [2, 3]], [4, [5, [6]]]];
        expect(flatten(deeplyNested)).toEqual([1, [2, 3], 4, [5, [6]]]);
    });

    it("should work with string arrays", () => {
        const nested = [["a", "b"], ["c", "d"], ["e"]];
        expect(flatten(nested)).toEqual(["a", "b", "c", "d", "e"]);
    });

    it("should handle single-element arrays", () => {
        const nested = [[1], [2], [3]];
        expect(flatten(nested)).toEqual([1, 2, 3]);
    });

    it("should return empty array for empty input", () => {
        expect(flatten([])).toEqual([]);
    });

    it("should handle arrays with different types", () => {
        const mixed = [[1, "two"], [true, null], [undefined, { key: "value" }]];
        expect(flatten(mixed)).toEqual([1, "two", true, null, undefined, { key: "value" }]);
    });

    it("should handle sparse arrays by skipping holes", () => {
        const sparse = [, [1, 2], , [3, 4], ,]; // Array with holes
        const result = flatten(sparse);
        // Array.flat() skips sparse array elements (holes)
        expect(result).toEqual([1, 2, 3, 4]);
    });

    it("should maintain object references", () => {
        const obj1 = { id: 1 };
        const obj2 = { id: 2 };
        const nested = [[obj1], [obj2]];
        const result = flatten(nested);
        
        expect(result[0]).toBe(obj1); // Same reference
        expect(result[1]).toBe(obj2); // Same reference
    });

    it("should handle very large nested arrays", () => {
        const large = Array.from({ length: 1000 }, (_, i) => [i, i + 1000]);
        const result = flatten(large);
        
        expect(result).toHaveLength(2000);
        expect(result[0]).toBe(0);
        expect(result[1]).toBe(1000);
        expect(result[result.length - 2]).toBe(999);
        expect(result[result.length - 1]).toBe(1999);
    });

    it("should preserve array prototype methods", () => {
        const nested = [[1, 2], [3, 4]];
        const result = flatten(nested);
        
        expect(Array.isArray(result)).toBe(true);
        expect(result.length).toBe(4);
        expect(typeof result.map).toBe("function");
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
        expect(unique).toEqual([
            { id: 1, name: "Alice" },
            { id: 2, name: "Bob" },
            { id: 3, name: "Charlie" },
        ]);
    });

    it("should find unique items by transformation", () => {
        const strings = ["apple", "Apple", "banana", "BANANA", "cherry"];
        const unique = uniqBy(strings, item => item.toLowerCase());
        expect(unique).toEqual(["apple", "banana", "cherry"]);
    });

    it("should handle empty array", () => {
        expect(uniqBy([], item => item)).toEqual([]);
    });

    it("should handle array with all unique items", () => {
        const numbers = [1, 2, 3, 4, 5];
        expect(uniqBy(numbers, item => item)).toEqual([1, 2, 3, 4, 5]);
    });

    it("should handle array with all duplicate items", () => {
        const numbers = [1, 1, 1, 1];
        expect(uniqBy(numbers, item => item)).toEqual([1]);
    });

    it("should work with complex iteratee functions", () => {
        const items = [
            { x: 1, y: 2 },
            { x: 2, y: 1 },
            { x: 1, y: 2 },
            { x: 3, y: 3 },
        ];
        const unique = uniqBy(items, item => `${item.x},${item.y}`);
        expect(unique).toEqual([
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
        expect(unique).toEqual([{ id: 1, value: "first" }]);
    });

    it("should handle null and undefined values", () => {
        const items = [null, undefined, null, 1, undefined, 2, 1];
        const unique = uniqBy(items, item => item);
        expect(unique).toEqual([null, undefined, 1, 2]);
    });

    it("should handle empty iteratee results", () => {
        const items = [
            { name: "" },
            { name: "test" },
            { name: "" },
            { name: "other" },
        ];
        const unique = uniqBy(items, item => item.name);
        expect(unique).toHaveLength(3);
        expect(unique[0].name).toBe("");
        expect(unique[1].name).toBe("test");
        expect(unique[2].name).toBe("other");
    });

    it("should handle complex object comparisons", () => {
        const items = [
            { user: { id: 1, name: "Alice" }, role: "admin" },
            { user: { id: 2, name: "Bob" }, role: "user" },
            { user: { id: 1, name: "Alice Updated" }, role: "admin" },
            { user: { id: 3, name: "Charlie" }, role: "user" },
        ];
        
        const unique = uniqBy(items, item => `${item.user.id}-${item.role}`);
        expect(unique).toHaveLength(3);
        expect(unique[0].user.name).toBe("Alice"); // First occurrence preserved
        expect(unique[1].user.name).toBe("Bob");
        expect(unique[2].user.name).toBe("Charlie");
    });

    it("should work with large datasets efficiently", () => {
        const large = Array.from({ length: 10000 }, (_, i) => ({
            id: Math.floor(i / 2), // Create duplicates
            value: `item-${i}`,
        }));
        
        const unique = uniqBy(large, item => item.id);
        expect(unique).toHaveLength(5000); // Half should be unique
        expect(unique[0].value).toBe("item-0"); // First occurrence preserved
    });

    it("should handle iteratee that throws for some values", () => {
        const items = [
            { value: "safe1" },
            null,
            { value: "safe2" },
            undefined,
            { value: "safe1" }, // duplicate
        ];
        
        const unique = uniqBy(items, item => {
            if (!item || !item.value) return "null-or-undefined";
            return item.value;
        });
        
        expect(unique).toHaveLength(3);
        expect(unique.map(item => item?.value || "falsy")).toEqual(["safe1", "falsy", "safe2"]);
    });
});
