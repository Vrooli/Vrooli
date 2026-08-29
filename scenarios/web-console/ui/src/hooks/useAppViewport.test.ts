import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const viewportState = vi.hoisted(() => ({
  current: {
    layoutWidth: 390,
    layoutHeight: 844,
    visibleWidth: 390,
    visibleHeight: 760,
    offsetLeft: 0,
    offsetTop: 0,
    scale: 1,
    keyboardInset: 0,
    keyboardVisible: false,
  },
}));

vi.mock("@vrooli/react-component-library/useViewportEnvironment/1.0.3", () => ({
  useViewportEnvironment: () => viewportState.current,
}));

import { useAppViewport } from "./useAppViewport";

const cssVar = (name: string) => document.documentElement.style.getPropertyValue(name) || undefined;

describe("useAppViewport", () => {
  beforeEach(() => {
    viewportState.current = {
      layoutWidth: 390,
      layoutHeight: 844,
      visibleWidth: 390,
      visibleHeight: 760,
      offsetLeft: 0,
      offsetTop: 0,
      scale: 1,
      keyboardInset: 0,
      keyboardVisible: false,
    };
  });

  afterEach(() => {
    document.documentElement.removeAttribute("style");
    vi.restoreAllMocks();
  });

  it("projects the normalized snapshot into only Web Console variables", () => {
    renderHook(() => useAppViewport());
    expect(cssVar("--wc-app-height")).toBe("760px");
    expect(cssVar("--wc-kb-height")).toBe("0px");
    expect(cssVar("--wc-safe-bottom")).toBe("env(safe-area-inset-bottom)");
    expect(cssVar("--rcl-viewport-height")).toBeUndefined();
    expect(cssVar("--rcl-keyboard-inset")).toBeUndefined();
  });

  it("projects established keyboard state and notifies followers once per transition", () => {
    const onKeyboardChange = vi.fn();
    const { rerender } = renderHook(() => useAppViewport({ onKeyboardChange }));
    expect(onKeyboardChange).toHaveBeenLastCalledWith(false);

    viewportState.current = { ...viewportState.current, visibleHeight: 520, keyboardInset: 324, keyboardVisible: true };
    rerender();
    expect(cssVar("--wc-app-height")).toBe("520px");
    expect(cssVar("--wc-kb-height")).toBe("324px");
    expect(cssVar("--wc-safe-bottom")).toBe("0px");
    expect(onKeyboardChange).toHaveBeenLastCalledWith(true);
    expect(onKeyboardChange).toHaveBeenCalledTimes(2);

    rerender();
    expect(onKeyboardChange).toHaveBeenCalledTimes(2);
  });

  it("does not scroll for ordinary resize snapshots", () => {
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    const { rerender } = renderHook(() => useAppViewport());
    viewportState.current = { ...viewportState.current, visibleHeight: 680 };
    rerender();
    expect(scrollTo).not.toHaveBeenCalled();
  });

  it("corrects a demonstrated visual viewport offset", () => {
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    viewportState.current = { ...viewportState.current, offsetTop: 24 };
    renderHook(() => useAppViewport());
    expect(scrollTo).toHaveBeenCalledOnce();
    expect(scrollTo).toHaveBeenCalledWith(0, 0);
  });

  it("removes its application variables on unmount", () => {
    const { unmount } = renderHook(() => useAppViewport());
    unmount();
    expect(cssVar("--wc-app-height")).toBeUndefined();
    expect(cssVar("--wc-kb-height")).toBeUndefined();
    expect(cssVar("--wc-safe-bottom")).toBeUndefined();
  });
});
