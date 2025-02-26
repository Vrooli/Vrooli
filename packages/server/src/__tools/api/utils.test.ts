import { expect } from "chai";
import { findSelection } from "./utils.js";

// Mock the console.warn to test warning outputs
const originalConsoleWarn = console.warn;
before(() => {
    console.warn = jest.fn();
});

after(() => {
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

    it.each([
        ["common", "common"],
        ["list", "list"],
        ["full", "full"],
        ["nav", "nav"],
    ] as const)("should return the specified selection %s when it exists", (selection, expected) => {
        const result = findSelection(baseApiPartial, selection);
        expect(result).to.deep.equal(expected);
    });

    it("should fallback to the next best selection if the preferred one does not exist", () => {
        const apiPartial = {
            // "common" is missing
            list: {},
            full: {},
            nav: {},
        };
        const result = findSelection(apiPartial, "common");
        expect(result).to.deep.equal("list");
    });

    it("should throw an error if none of the selections exist", () => {
        const apiPartial = {
        };
        expect(() => findSelection(apiPartial, "common")).to.throw();
    });

    it("should log a warning if the specified selection does not exist but another valid selection is found", () => {
        const apiPartial = {
            // "common" is missing, but "list" exists
            list: {},
        };
        findSelection(apiPartial, "common");
        expect(console.warn).toHaveBeenCalledWith(expect.stringContaining("Specified selection type 'common' for 'TestType' does not exist. Try using 'list' instead."));
    });

    it.each([
        ["common", ["list", "full", "nav"]],
        ["list", ["common", "full", "nav"]],
        ["full", ["list", "common", "nav"]],
        ["nav", ["common", "list", "full"]],
    ] as const)("should respect the fallback order for selection %s", (selection, expectedOrder) => {
        const apiPartial = {
        };
        // Ensure only the last preference exists
        apiPartial[expectedOrder[expectedOrder.length - 1]] = {};

        const result = findSelection(apiPartial, selection);
        expect(result).to.deep.equal(expectedOrder[expectedOrder.length - 1]);
    });

    // Add more tests as necessary to cover edge cases or specific behaviors
});
