// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
import { renderHook } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { act } from "react";
import { useObjectContextMenu } from "./useObjectContextMenu.js";

describe("useObjectContextMenu", () => {
    it("initial state is correct", () => {
        const { result } = renderHook(() => useObjectContextMenu());
        expect(result.current.anchorEl).toBeNull();
        expect(result.current.object).toBeNull();
    });

    it("handleContextMenu sets anchorEl and object", () => {
        const { result } = renderHook(() => useObjectContextMenu());
        const dummyTarget = document.createElement("div");
        const dummyObject = { id: "1", name: "Test Object" };

        act(() => {
            result.current.handleContextMenu(dummyTarget, dummyObject);
        });

        expect(result.current.anchorEl).toEqual(dummyTarget);
        expect(result.current.object).toEqual(dummyObject);
    });

    it("handleContextMenu does nothing if object is null", () => {
        const { result } = renderHook(() => useObjectContextMenu());
        const dummyTarget = document.createElement("div");

        act(() => {
            result.current.handleContextMenu(dummyTarget, null);
        });

        expect(result.current.anchorEl).toBeNull();
        expect(result.current.object).toBeNull();
    });

    it("closeContextMenu resets anchorEl but keeps object", () => {
        const { result } = renderHook(() => useObjectContextMenu());
        const dummyTarget = document.createElement("div");
        const dummyObject = { id: "1", name: "Test Object" };

        // Set some initial state
        act(() => {
            result.current.handleContextMenu(dummyTarget, dummyObject);
        });

        // Close the context menu
        act(() => {
            result.current.closeContextMenu();
        });

        expect(result.current.anchorEl).toBeNull();
        expect(result.current.object).toEqual(dummyObject); // object remains the same
    });
});
