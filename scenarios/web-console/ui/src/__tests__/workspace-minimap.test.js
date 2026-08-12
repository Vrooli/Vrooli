import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach, beforeAll } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
// Polyfill ResizeObserver for jsdom
beforeAll(() => {
    if (typeof globalThis.ResizeObserver === "undefined") {
        globalThis.ResizeObserver = class {
            observe() { }
            unobserve() { }
            disconnect() { }
        };
    }
});
// Mock store state
const mockStoreState = {
    isMinimapVisible: true,
    setMinimapVisible: vi.fn(),
};
vi.mock("../stores/useWorkspaceStore", () => ({
    useWorkspaceStore: (selector) => selector(mockStoreState),
}));
import WorkspaceMinimap from "../components/WorkspaceMinimap";
function makeScrollRef(overrides) {
    const el = document.createElement("div");
    const vals = { scrollTop: 0, scrollHeight: 2000, clientHeight: 500, ...overrides };
    Object.defineProperty(el, "scrollTop", { value: vals.scrollTop, writable: true, configurable: true });
    Object.defineProperty(el, "scrollHeight", { value: vals.scrollHeight, writable: true, configurable: true });
    Object.defineProperty(el, "clientHeight", { value: vals.clientHeight, writable: true, configurable: true });
    el.scrollTo = vi.fn();
    return { current: el };
}
describe("WorkspaceMinimap", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockStoreState.isMinimapVisible = true;
    });
    it("returns null when isMinimapVisible is false", () => {
        mockStoreState.isMinimapVisible = false;
        const ref = makeScrollRef();
        const { container } = render(_jsx(WorkspaceMinimap, { scrollRef: ref, rowCount: 3 }));
        expect(container.innerHTML).toBe("");
    });
    it("returns null when no scrollable overflow", () => {
        const ref = makeScrollRef({ scrollHeight: 500 });
        const { container } = render(_jsx(WorkspaceMinimap, { scrollRef: ref, rowCount: 2 }));
        expect(container.innerHTML).toBe("");
    });
    it("renders rail with ARIA attributes when visible and scrollable", () => {
        const ref = makeScrollRef();
        render(_jsx(WorkspaceMinimap, { scrollRef: ref, rowCount: 3 }));
        const rail = screen.getByRole("slider");
        expect(rail).toBeTruthy();
        expect(rail.getAttribute("aria-label")).toBe("workspaceMinimap.scrollPosition");
        expect(rail.getAttribute("aria-valuemin")).toBe("0");
        expect(rail.getAttribute("aria-valuemax")).toBe("100");
    });
    it("renders row markers matching rowCount", () => {
        const ref = makeScrollRef();
        render(_jsx(WorkspaceMinimap, { scrollRef: ref, rowCount: 4 }));
        const markers = document.querySelectorAll(".wc-minimap-marker");
        expect(markers.length).toBe(4);
    });
    it("click on rail calls scrollTo", () => {
        const ref = makeScrollRef();
        render(_jsx(WorkspaceMinimap, { scrollRef: ref, rowCount: 3 }));
        const rail = screen.getByRole("slider");
        fireEvent.pointerDown(rail, { clientY: 50 });
        expect(ref.current.scrollTo).toHaveBeenCalled();
    });
    it("keyboard ArrowDown triggers scrollTo", () => {
        const ref = makeScrollRef();
        render(_jsx(WorkspaceMinimap, { scrollRef: ref, rowCount: 3 }));
        const rail = screen.getByRole("slider");
        fireEvent.keyDown(rail, { key: "ArrowDown" });
        expect(ref.current.scrollTo).toHaveBeenCalled();
    });
    it("keyboard Home triggers scrollTo with top=0", () => {
        const ref = makeScrollRef();
        render(_jsx(WorkspaceMinimap, { scrollRef: ref, rowCount: 3 }));
        const rail = screen.getByRole("slider");
        fireEvent.keyDown(rail, { key: "Home" });
        expect(ref.current.scrollTo).toHaveBeenCalledWith({ top: 0, behavior: "auto" });
    });
    it("keyboard End triggers scrollTo to bottom", () => {
        const ref = makeScrollRef();
        render(_jsx(WorkspaceMinimap, { scrollRef: ref, rowCount: 3 }));
        const rail = screen.getByRole("slider");
        fireEvent.keyDown(rail, { key: "End" });
        expect(ref.current.scrollTo).toHaveBeenCalledWith({
            top: 1500, // scrollHeight(2000) - clientHeight(500)
            behavior: "auto",
        });
    });
});
