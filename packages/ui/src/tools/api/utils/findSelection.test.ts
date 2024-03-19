import { findSelection } from "./findSelection"; // Adjust the import path as necessary

// Mock the console.warn to test warning outputs
const originalConsoleWarn = console.warn;
beforeAll(() => {
    console.warn = jest.fn();
});

afterAll(() => {
    console.warn = originalConsoleWarn;
});

describe("findSelection", () => {
    // Define a base GqlPartial object
    const baseGqlPartial = {
        __typename: "TestType",
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
        const result = findSelection(baseGqlPartial, selection);
        expect(result).toEqual(expected);
    });

    it("should fallback to the next best selection if the preferred one does not exist", () => {
        const gqlPartial = {
            __typename: "TestType",
            // "common" is missing
            list: {},
            full: {},
            nav: {},
        };
        const result = findSelection(gqlPartial, "common");
        expect(result).toEqual("list");
    });

    it("should throw an error if none of the selections exist", () => {
        const gqlPartial = {
            __typename: "TestType",
        };
        expect(() => findSelection(gqlPartial, "common")).toThrow();
    });

    it("should log a warning if the specified selection does not exist but another valid selection is found", () => {
        const gqlPartial = {
            __typename: "TestType",
            // "common" is missing, but "list" exists
            list: {},
        };
        findSelection(gqlPartial, "common");
        expect(console.warn).toHaveBeenCalledWith(expect.stringContaining("Specified selection type 'common' for 'TestType' does not exist. Try using 'list' instead."));
    });

    it.each([
        ["common", ["list", "full", "nav"]],
        ["list", ["common", "full", "nav"]],
        ["full", ["list", "common", "nav"]],
        ["nav", ["common", "list", "full"]],
    ] as const)("should respect the fallback order for selection %s", (selection, expectedOrder) => {
        const gqlPartial = {
            __typename: "TestType",
        };
        // Ensure only the last preference exists
        gqlPartial[expectedOrder[expectedOrder.length - 1]] = {};

        const result = findSelection(gqlPartial, selection);
        expect(result).toEqual(expectedOrder[expectedOrder.length - 1]);
    });

    // Add more tests as necessary to cover edge cases or specific behaviors
});
