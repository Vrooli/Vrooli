import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { act, useRef } from "react";
import { useResizablePanel } from "../useResizablePanel";
function Harness({ storageKey = "test-sidebar-width" }) {
    const containerRef = useRef(null);
    const targetRef = useRef(null);
    const { size, resizeHandleProps } = useResizablePanel({
        containerRef,
        targetRef,
        minSize: 240,
        maxSize: 520,
        defaultSize: 300,
        adjacentMinSize: 300,
        handleWidth: 12,
        storageKey,
    });
    return (_jsxs("div", { ref: containerRef, "data-testid": "container", style: { width: 900 }, children: [_jsx("aside", { ref: targetRef, "data-testid": "target", style: { width: size } }), _jsx("div", { "data-testid": "handle", ...resizeHandleProps })] }));
}
describe("useResizablePanel", () => {
    beforeEach(() => {
        window.localStorage.clear();
    });
    it("restores valid persisted width", () => {
        window.localStorage.setItem("test-sidebar-width", "360");
        render(_jsx(Harness, {}));
        expect(screen.getByTestId("target")).toHaveStyle({ width: "360px" });
        expect(screen.getByTestId("handle")).toHaveAttribute("role", "separator");
        expect(screen.getByTestId("handle")).toHaveAttribute("aria-orientation", "vertical");
    });
    it("writes live width during drag and persists final width", () => {
        render(_jsx(Harness, {}));
        const container = screen.getByTestId("container");
        container.getBoundingClientRect = () => ({
            x: 0,
            y: 0,
            left: 0,
            top: 0,
            right: 900,
            bottom: 600,
            width: 900,
            height: 600,
            toJSON: () => ({}),
        });
        act(() => {
            fireEvent.pointerDown(screen.getByTestId("handle"), { button: 0 });
        });
        act(() => {
            window.dispatchEvent(new PointerEvent("pointermove", { clientX: 420 }));
        });
        expect(screen.getByTestId("target")).toHaveStyle({ width: "420px" });
        act(() => {
            window.dispatchEvent(new PointerEvent("pointerup"));
        });
        expect(window.localStorage.getItem("test-sidebar-width")).toBe("420");
    });
});
