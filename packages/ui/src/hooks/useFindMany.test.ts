/* eslint-disable @typescript-eslint/ban-ts-comment */
import { describe, it, expect, afterEach, vi } from "vitest";

// Unmock @vrooli/shared for this test
vi.unmock("@vrooli/shared");

import { CommentSortBy, ResourceSortBy, stringifySearchParams } from "@vrooli/shared";
import { searchTypeToParams } from "../utils/search/objectToSearch.js";
import { getUrlSearchParams, parseData, readyToSearch, updateSearchUrl, updateSortBy } from "./useFindMany.js";

// Helper function to set window.location.search
function setWindowSearch(search) {
    Object.defineProperty(window, "location", {
        value: {
            ...window.location,
            search,
            pathname: "/",
        },
        writable: true,
    });
}

// Mock setLocation function
const mockSetLocation = vi.fn((path, { replace, searchParams }) => {
    const search = stringifySearchParams(searchParams);
    setWindowSearch(`?${search}`);
});

describe("parseData", () => {
    // Test 1: No Data
    it("should return an empty array if data is null undefined", () => {
        // @ts-ignore: Testing runtime scenario
        expect(parseData(null)).toEqual([]);
        expect(parseData(undefined)).toEqual([]);
    });

    it("should return an empty array if data is not an object", () => {
        // @ts-ignore: Testing runtime scenario
        expect(parseData("string")).toEqual([]);
        // @ts-ignore: Testing runtime scenario
        expect(parseData(123)).toEqual([]);
    });

    // Test 2: Custom Resolver
    it("should use custom resolver if provided", () => {
        const customResolver = vi.fn().mockReturnValue([{ id: 1 }]);
        const data = { items: [{ id: 1 }] };
        expect(parseData(data, customResolver)).toEqual([{ id: 1 }]);
        expect(customResolver).toHaveBeenCalledWith(data);
    });

    // Test 3: Invalid Custom Resolver
    it("should handle custom resolver returning non-array values", () => {
        const customResolver = vi.fn().mockReturnValue(null);
        const data = { items: [{ id: 1 }] };
        expect(parseData(data, customResolver)).toBe(null);
    });

    // Test 4: Paginated Data
    it("should correctly parse paginated data", () => {
        const paginatedData = { edges: [{ node: { id: 1 } }, { node: { id: 2 } }] };
        expect(parseData(paginatedData)).toEqual([{ id: 1 }, { id: 2 }]);
    });

    // Test 5: Non-Paginated Data
    it("should return empty array if edges are not present", () => {
        const nonPaginatedData = { items: [{ id: 1 }] };
        expect(parseData(nonPaginatedData)).toEqual([]);
    });

    // Test 6: Error Handling
    it("should not throw error with unexpected input types", () => {
        expect(() => parseData([1, 2, 3])).not.toThrow();
    });
});

describe("updateSortBy", () => {
    const searchParams = {
        defaultSortBy: "DateCreatedAsc",
        sortByOptions: { DateCreatedAsc: true, NameDesc: true },
    };

    // Test 1: Valid SortBy
    it("returns sortBy when it is valid", () => {
        const sortBy = "NameDesc";
        const result = updateSortBy(searchParams, sortBy);
        expect(result).toBe(sortBy);
    });

    // Test 2: Invalid SortBy
    it("returns defaultSortBy when sortBy is not valid", () => {
        const sortBy = "unknown";
        const result = updateSortBy(searchParams, sortBy);
        expect(result).toBe(searchParams.defaultSortBy);
    });

    // Test 3: Default Fallback
    it("returns an empty string if defaultSortBy is undefined and sortBy is not valid", () => {
        const localSearchParams = { sortByOptions: { NameDesc: true }, defaultSortBy: undefined };
        const sortBy = "unknown";
        const result = updateSortBy(localSearchParams, sortBy);
        expect(result).toBe("");
    });

    // Test 4: Non-String SortBy
    it("handles non-string sortBy gracefully", () => {
        const sortBy = 12345;
        // @ts-ignore: Testing runtime scenario
        const result = updateSortBy(searchParams, sortBy);
        expect(result).toBeUndefined();
    });

    // Test 5: Missing sortByOptions
    it("returns defaultSortBy if sortByOptions is missing", () => {
        const localSearchParams = { defaultSortBy: "DateCreatedAsc" };
        const sortBy = "NameDesc";
        // @ts-ignore: Testing runtime scenario
        const result = updateSortBy(localSearchParams, sortBy);
        expect(result).toBe("DateCreatedAsc");
    });
});

describe("readyToSearch", () => {
    const baseParams = {
        canSearch: true,
        loading: false,
        hasMore: true,
        findManyEndpoint: "http://api.example.com/search",
        sortBy: "date",  // Only one sortBy needed, correctly initialized here
        advancedSearchParams: null,
        searchString: "",
        timeFrame: undefined,
        where: {},
    };

    // Test 1: All conditions met
    it("returns true when all conditions are met", () => {
        expect(readyToSearch(baseParams)).toBe(true);
    });

    // Test 2: canSearch is false
    it("returns false when canSearch is false", () => {
        const params = { ...baseParams, canSearch: false };
        expect(readyToSearch(params)).toBe(false);
    });

    // Test 3: loading is true
    it("returns false when loading is true", () => {
        const params = { ...baseParams, loading: true };
        expect(readyToSearch(params)).toBe(false);
    });

    // Test 4: hasMore is false
    it("returns false when hasMore is false", () => {
        const params = { ...baseParams, hasMore: false };
        expect(readyToSearch(params)).toBe(false);
    });

    // Test 5: findManyEndpoint is empty
    it("returns false when findManyEndpoint is empty", () => {
        const params = { ...baseParams, findManyEndpoint: "" };
        expect(readyToSearch(params)).toBe(false);
    });

    // Test 6: sortBy is empty
    it("returns false when sortBy is empty", () => {
        const params = { ...baseParams, sortBy: "" };
        expect(readyToSearch(params)).toBe(false);
    });

    // Test 7: Handling partial searchParams
    it("returns false when params are partially undefined", () => {
        const params = {
            canSearch: true,
            loading: false,
            hasMore: true,
            // Omit findManyEndpoint and sortBy to simulate missing values
        };
        expect(readyToSearch(params)).toBe(false);
    });
});

describe("getUrlSearchParams", () => {
    // Save the original window.location
    const originalLocation = window.location;

    afterEach(() => {
        // Restore the original window.location after each test
        window.location = originalLocation;
    });

    it("returns default values when not controlling URL", () => {
        setWindowSearch(stringifySearchParams({
            search: "query",
            sort: ResourceSortBy.DateUpdatedAsc,
            time: { after: new Date("2022-01-01") },
        }));
        const result = getUrlSearchParams(false, "Resource");
        expect(result).toEqual({
            searchString: "",
            sortBy: "",
            timeFrame: undefined,
        });
    });

    it("extracts searchString and sortBy from URL when controlling URL", () => {
        setWindowSearch(stringifySearchParams({
            search: "query",
            sort: CommentSortBy.DateUpdatedDesc,
        }));
        const result = getUrlSearchParams(true, "Comment");
        expect(result.searchString).toBe("query");
        expect(result.sortBy).toBe(CommentSortBy.DateUpdatedDesc);
    });

    it("includes timeFrame with only 'after' when 'after' is present", () => {
        setWindowSearch(stringifySearchParams({
            time: {
                after: new Date("2022-01-01").toISOString(),
            },
        }));
        const result = getUrlSearchParams(true, "Issue");
        expect(result.timeFrame).toEqual({ after: new Date("2022-01-01") });
    });

    it("includes timeFrame with only 'before' when 'before' is present", () => {
        setWindowSearch(stringifySearchParams({
            time: {
                before: new Date("2022-12-31").toISOString(),
            },
        }));
        const result = getUrlSearchParams(true, "User");
        expect(result.timeFrame).toEqual({ before: new Date("2022-12-31") });
    });

    it("includes timeFrame with both 'after' and 'before' when both are present", () => {
        setWindowSearch(stringifySearchParams({
            time: {
                after: new Date("2022-01-01").toISOString(),
                before: new Date("2022-12-31").toISOString(),
            },
        }));
        const result = getUrlSearchParams(true, "Team");
        expect(result.timeFrame).toEqual({
            after: new Date("2022-01-01"),
            before: new Date("2022-12-31"),
        });
    });

    it("works when other miscellaneous parameters are present", () => {
        setWindowSearch(stringifySearchParams({
            search: "query",
            sort: CommentSortBy.DateUpdatedAsc,
            other1: "param",
            other2: 1,
            other3: [{ time: "value" }],
            time: {
                after: new Date("2022-01-01").toISOString(),
                before: new Date("2022-12-31").toISOString(),
            },
        }));
        const result = getUrlSearchParams(true, "Comment");
        expect(result).toEqual({
            searchString: "query",
            sortBy: CommentSortBy.DateUpdatedAsc,
            timeFrame: {
                after: new Date("2022-01-01"),
                before: new Date("2022-12-31"),
            },
        });
    });

    it("returns default values when URL is malformed", () => {
        setWindowSearch("?search=&sort=");
        const result = getUrlSearchParams(true, "Resource");
        expect(result).toEqual({
            searchString: "",
            sortBy: searchTypeToParams.Resource().defaultSortBy,
            timeFrame: undefined,
        });
    });

    it("handles invalid searchType gracefully", () => {
        setWindowSearch(stringifySearchParams({
            search: "query",
            sort: "invalidSort",
        }));
        console.warn = vi.fn(); // Mock console.warn to check if it gets called

        const result = getUrlSearchParams(true, "invalidType");
        expect(console.warn).toHaveBeenCalledWith("Invalid search type provided: invalidType");
        expect(result).toEqual({
            searchString: "query",
            sortBy: "",
        });
    });
});

describe("updateSearchUrl", () => {

    const originalLocation = window.location;

    afterEach(() => {
        // Restore the original window.location after each test
        Object.defineProperty(window, "location", {
            value: originalLocation,
        });
        mockSetLocation.mockClear();
    });

    it("does not update URL when controlsUrl is false", () => {
        setWindowSearch("");
        updateSearchUrl(false, {
            searchString: "test",
            sortBy: "date",
            timeFrame: undefined,
        }, mockSetLocation);

        expect(mockSetLocation).not.toHaveBeenCalled();
    });

    it("updates URL with searchString and sortBy when controlsUrl is true", () => {
        setWindowSearch("");
        updateSearchUrl(true, {
            searchString: "query",
            sortBy: "name",
            timeFrame: undefined,
        }, mockSetLocation);

        expect(mockSetLocation).toHaveBeenCalledWith("/", {
            replace: true,
            searchParams: {
                search: "query",
                sort: "name",
                time: undefined,
            },
        });
    });

    it("includes timeFrame in URL when 'after' is provided", () => {
        setWindowSearch("");
        const afterDate = new Date("2022-01-01");
        updateSearchUrl(true, {
            searchString: "",
            sortBy: "date",
            timeFrame: { after: afterDate, before: undefined },
        }, mockSetLocation);

        expect(mockSetLocation).toHaveBeenCalledWith("/", {
            replace: true,
            searchParams: {
                search: undefined,
                sort: "date",
                time: { after: afterDate.toISOString(), before: "" },
            },
        });
    });

    it("includes timeFrame in URL when 'before' is provided", () => {
        setWindowSearch("");
        const beforeDate = new Date("2022-12-31");
        updateSearchUrl(true, {
            searchString: "",
            sortBy: "date",
            timeFrame: { after: undefined, before: beforeDate },
        }, mockSetLocation);

        expect(mockSetLocation).toHaveBeenCalledWith("/", {
            replace: true,
            searchParams: {
                search: undefined,
                sort: "date",
                time: { after: "", before: beforeDate.toISOString() },
            },
        });
    });

    it("includes full timeFrame in URL when both 'after' and 'before' are provided", () => {
        setWindowSearch("");
        const afterDate = new Date("2022-01-01");
        const beforeDate = new Date("2022-12-31");
        updateSearchUrl(true, {
            searchString: "",
            sortBy: "date",
            timeFrame: { after: afterDate, before: beforeDate },
        }, mockSetLocation);

        expect(mockSetLocation).toHaveBeenCalledWith("/", {
            replace: true,
            searchParams: {
                search: undefined,
                sort: "date",
                time: { after: afterDate.toISOString(), before: beforeDate.toISOString() },
            },
        });
    });
});

