import { describe, it, expect, beforeAll, afterAll, vi } from "vitest";
import { findSelection } from "./utils.js";

// Mock the console.warn to test warning outputs
const originalConsoleWarn = console.warn;
beforeAll(() => {
    console.warn = vi.fn();
});

afterAll(() => {
    console.warn = originalConsoleWarn;
});

describe("findSelection", () => {
    // Define a base ApiPartial object
    const baseApiPartial = {
        common: {},
        list: {},
        full: {},
        nav: {},
    };

    const selectionTests = [
        ["common", "common"],
        ["list", "list"],
        ["full", "full"],
        ["nav", "nav"],
    ] as const;

    selectionTests.forEach(([selection, expected]) => {
        it(`should return the specified selection ${selection} when it exists`, () => {
            const result = findSelection(baseApiPartial, selection);
            expect(result).toBe(expected);
        });
    });

    it("should fallback to the next best selection if the preferred one does not exist", () => {
        const apiPartial = {
            // "common" is missing
            list: {},
            full: {},
            nav: {},
        };
        const result = findSelection(apiPartial, "common");
        expect(result).toBe("list");
    });

    it("should throw an error if none of the selections exist", () => {
        const apiPartial = {
        };
        expect(() => findSelection(apiPartial, "common")).toThrow();
    });

    it("should log a warning if the specified selection does not exist but another valid selection is found", () => {
        const apiPartial = {
            // "common" is missing, but "list" exists
            list: {},
        };
        findSelection(apiPartial, "common");
        expect(console.warn).toHaveBeenCalledWith(expect.stringMatching(/Specified selection type 'common' for 'TestType' does not exist. Try using 'list' instead./));
    });

    const fallbackTests = [
        ["common", ["list", "full", "nav"]],
        ["list", ["common", "full", "nav"]],
        ["full", ["list", "common", "nav"]],
        ["nav", ["common", "list", "full"]],
    ] as const;

    fallbackTests.forEach(([selection, expectedOrder]) => {
        it(`should respect the fallback order for selection ${selection}`, () => {
            const apiPartial = {
            };
            // Ensure only the last preference exists
            apiPartial[expectedOrder[expectedOrder.length - 1]] = {};

            const result = findSelection(apiPartial, selection);
            expect(result).toBe(expectedOrder[expectedOrder.length - 1]);
        });
    });

    // Add more tests as necessary to cover edge cases or specific behaviors
});
