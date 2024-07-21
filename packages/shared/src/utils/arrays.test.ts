import { arraysEqual } from "./arrays";

describe("arraysEqual", () => {
    it("should return true for equal number arrays", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2, 3];
        const numberComparator = (a: number, b: number) => a === b;

        expect(arraysEqual(arr1, arr2, numberComparator)).toBe(true);
    });

    it("should return false for different number arrays", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2, 4];
        const numberComparator = (a: number, b: number) => a === b;

        expect(arraysEqual(arr1, arr2, numberComparator)).toBe(false);
    });

    it("should return true for equal string arrays", () => {
        const arr1 = ["a", "b", "c"];
        const arr2 = ["a", "b", "c"];
        const stringComparator = (a: string, b: string) => a === b;

        expect(arraysEqual(arr1, arr2, stringComparator)).toBe(true);
    });

    it("should return false for different string arrays", () => {
        const arr1 = ["a", "b", "c"];
        const arr2 = ["a", "b", "d"];
        const stringComparator = (a: string, b: string) => a === b;

        expect(arraysEqual(arr1, arr2, stringComparator)).toBe(false);
    });

    it("should return true for equal object arrays", () => {
        const arr1 = [{ id: 1 }, { id: 2 }];
        const arr2 = [{ id: 1 }, { id: 2 }];
        const objectComparator = (a: { id: number }, b: { id: number }) => a.id === b.id;

        expect(arraysEqual(arr1, arr2, objectComparator)).toBe(true);
    });

    it("should return false for different object arrays", () => {
        const arr1 = [{ id: 1 }, { id: 2 }];
        const arr2 = [{ id: 1 }, { id: 3 }];
        const objectComparator = (a: { id: number }, b: { id: number }) => a.id === b.id;

        expect(arraysEqual(arr1, arr2, objectComparator)).toBe(false);
    });

    it("should return false for arrays of different lengths", () => {
        const arr1 = [1, 2, 3];
        const arr2 = [1, 2];
        const numberComparator = (a: number, b: number) => a === b;

        expect(arraysEqual(arr1, arr2, numberComparator)).toBe(false);
    });

    it("should return true for custom comparison logic (case insensitive strings)", () => {
        const arr1 = ["A", "b", "C"];
        const arr2 = ["a", "B", "c"];
        const caseInsensitiveComparator = (a: string, b: string) => a.toLowerCase() === b.toLowerCase();

        expect(arraysEqual(arr1, arr2, caseInsensitiveComparator)).toBe(true);
    });

    it("should return false for custom comparison logic with unequal elements", () => {
        const arr1 = ["A", "b", "C"];
        const arr2 = ["a", "B", "d"];
        const caseInsensitiveComparator = (a: string, b: string) => a.toLowerCase() === b.toLowerCase();

        expect(arraysEqual(arr1, arr2, caseInsensitiveComparator)).toBe(false);
    });

    it("should handle empty arrays correctly", () => {
        const arr1: number[] = [];
        const arr2: number[] = [];
        const numberComparator = (a: number, b: number) => a === b;

        expect(arraysEqual(arr1, arr2, numberComparator)).toBe(true);
    });

    it("should return false when one array is empty and the other is not", () => {
        const arr1: number[] = [];
        const arr2 = [1];
        const numberComparator = (a: number, b: number) => a === b;

        expect(arraysEqual(arr1, arr2, numberComparator)).toBe(false);
    });
});
