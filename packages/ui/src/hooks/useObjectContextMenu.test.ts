import { act, renderHook } from "@testing-library/react";
import { useObjectContextMenu } from "./useObjectContextMenu";

describe("useObjectContextMenu", () => {
    test("initial state is correct", () => {
        const { result } = renderHook(() => useObjectContextMenu());
        expect(result.current.anchorEl).toBeNull();
        expect(result.current.object).toBeNull();
    });

    test("handleContextMenu sets anchorEl and object", () => {
        const { result } = renderHook(() => useObjectContextMenu());
        const dummyTarget = document.createElement("div");
        const dummyObject = { id: "1", name: "Test Object" };

        act(() => {
            result.current.handleContextMenu(dummyTarget, dummyObject);
        });

        expect(result.current.anchorEl).toEqual(dummyTarget);
        expect(result.current.object).toEqual(dummyObject);
    });

    test("handleContextMenu does nothing if object is null", () => {
        const { result } = renderHook(() => useObjectContextMenu());
        const dummyTarget = document.createElement("div");

        act(() => {
            result.current.handleContextMenu(dummyTarget, null);
        });

        expect(result.current.anchorEl).toBeNull();
        expect(result.current.object).toBeNull();
    });

    test("closeContextMenu resets anchorEl but keeps object", () => {
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
