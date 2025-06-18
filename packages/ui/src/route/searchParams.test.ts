// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
import { afterEach, describe, expect, it, vi } from "vitest";
import { addSearchParams, keepSearchParams, removeSearchParams, setSearchParams } from "./searchParams.js";

// Mock setLocation function
const mockSetLocation = vi.fn();

// Helper function to set window.location.search
const setWindowSearch = (search) => {
    Object.defineProperty(window, "location", {
        value: {
            ...window.location,
            search,
        },
        writable: true,
    });
};

describe("Search Param Utilities", () => {
    const originalLocation = window.location;

    afterEach(() => {
        // Restore the original window.location after each test
        Object.defineProperty(window, "location", {
            value: originalLocation,
        });
        mockSetLocation.mockClear();
    });

    it("should add search params correctly", () => {
        setWindowSearch("?existing=true");
        const newParams = { new: "param" };

        addSearchParams(mockSetLocation, newParams);

        expect(mockSetLocation).toHaveBeenCalledWith(window.location.pathname, {
            replace: true,
            searchParams: { existing: true, new: "param" },
        });
    });

    it("should set search params correctly", () => {
        setWindowSearch("?one=1&two=2");
        const newParams = { one: "new1", two: "new2" };

        setSearchParams(mockSetLocation, newParams);

        expect(mockSetLocation).toHaveBeenCalledWith(window.location.pathname, {
            replace: true,
            searchParams: newParams,
        });
    });

    it("should keep only specified search params", () => {
        setWindowSearch("?keep=%22this%22&remove=%22that%22");
        const keysToKeep = ["keep"];

        keepSearchParams(mockSetLocation, keysToKeep);

        expect(mockSetLocation).toHaveBeenCalledWith(window.location.pathname, {
            replace: true,
            searchParams: { keep: "this" },
        });
    });

    it("should remove specified search params", () => {
        setWindowSearch("?keep=%22this%22&remove=%22that%22");
        const keysToRemove = ["remove"];

        removeSearchParams(mockSetLocation, keysToRemove);

        expect(mockSetLocation).toHaveBeenCalledWith(window.location.pathname, {
            replace: true,
            searchParams: { keep: "this" },
        });
    });
});
