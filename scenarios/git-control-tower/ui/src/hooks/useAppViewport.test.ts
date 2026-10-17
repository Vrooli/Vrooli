import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useAppViewport } from "./useAppViewport";

const state = vi.hoisted(() => ({ viewport: {
  frameHeight: undefined as number | undefined,
  layoutWidth: 390, layoutHeight: 844, visibleWidth: 390, visibleHeight: 844,
  keyboardInset: 0, keyboardVisible: false, reachesScreenBottom: true,
  offsetLeft: 0, offsetTop: 0, scale: 1,
} }));
vi.mock("@vrooli/react-component-library/useViewportEnvironment/1", () => ({
  useViewportEnvironment: () => state.viewport,
}));
afterEach(() => {
  state.viewport = { ...state.viewport, keyboardVisible: false, reachesScreenBottom: true, visibleHeight: 844, frameHeight: undefined };
});
const value = (name: string) => document.documentElement.style.getPropertyValue(name);

describe("mobile viewport ownership", () => {
  it("publishes one effective bottom inset for both navigation and library overlays", () => {
    const { rerender, unmount } = renderHook(useAppViewport);
    expect(value("--gct-app-height")).toBe("844px");
    expect(value("--rcl-safe-bottom")).toBe("env(safe-area-inset-bottom, 0px)");
    state.viewport = { ...state.viewport, reachesScreenBottom: false };
    rerender();
    expect(value("--rcl-safe-bottom")).toBe("0px");
    expect(value("--gct-app-height")).toBe("844px");
    unmount();
    expect(value("--rcl-safe-bottom")).toBe("");
  });

  it("uses the client-measured drawable canvas instead of the shortened installed viewport", () => {
    state.viewport = { ...state.viewport, layoutHeight: 793, visibleHeight: 793, frameHeight: 852 };
    renderHook(useAppViewport);
    expect(value("--gct-app-height")).toBe("852px");
    expect(value("--rcl-safe-bottom")).toBe("env(safe-area-inset-bottom, 0px)");
    state.viewport = { ...state.viewport, layoutHeight: 844 };
  });

  it("uses the keyboard viewport only while the keyboard is established", () => {
    const { rerender } = renderHook(useAppViewport);
    state.viewport = { ...state.viewport, keyboardVisible: true, visibleHeight: 510, reachesScreenBottom: false };
    rerender();
    expect(value("--gct-app-height")).toBe("510px");
    expect(value("--rcl-safe-bottom")).toBe("0px");
    state.viewport = { ...state.viewport, keyboardVisible: false };
    rerender();
    expect(value("--gct-app-height")).toBe("844px");
  });
});
