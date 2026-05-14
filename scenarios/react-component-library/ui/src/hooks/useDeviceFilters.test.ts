import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { act, renderHook } from "@testing-library/react";

import {
  DEVICE_FILTERS_STORAGE_KEY,
  filterCSS,
  useDeviceFilters,
} from "./useDeviceFilters";

describe("useDeviceFilters", () => {
  beforeEach(() => window.localStorage.clear());
  afterEach(() => window.localStorage.clear());

  it("defaults to system / none / 0px", () => {
    const { result } = renderHook(() => useDeviceFilters());
    expect(result.current.colorScheme).toBe("system");
    expect(result.current.visionFilter).toBe("none");
    expect(result.current.blurPx).toBe(0);
    expect(result.current.filterCSS).toBe("");
  });

  it("persists state across remounts", () => {
    const { result, unmount } = renderHook(() => useDeviceFilters());
    act(() => result.current.setColorScheme("dark"));
    act(() => result.current.setVisionFilter("tritanopia"));
    act(() => result.current.setBlurPx(5));
    unmount();
    const remount = renderHook(() => useDeviceFilters());
    expect(remount.result.current.colorScheme).toBe("dark");
    expect(remount.result.current.visionFilter).toBe("tritanopia");
    expect(remount.result.current.blurPx).toBe(5);
  });

  it("clamps blur to [0, 10]", () => {
    const { result } = renderHook(() => useDeviceFilters());
    act(() => result.current.setBlurPx(99));
    expect(result.current.blurPx).toBe(10);
    act(() => result.current.setBlurPx(-3));
    expect(result.current.blurPx).toBe(0);
  });

  it("reset returns to defaults and reset clears subsequent reads", () => {
    const { result } = renderHook(() => useDeviceFilters());
    act(() => result.current.setColorScheme("light"));
    act(() => result.current.setVisionFilter("grayscale"));
    act(() => result.current.setBlurPx(7));
    act(() => result.current.reset());
    expect(result.current.colorScheme).toBe("system");
    expect(result.current.visionFilter).toBe("none");
    expect(result.current.blurPx).toBe(0);
  });

  it("sanitizes bogus persisted shape to defaults", () => {
    window.localStorage.setItem(
      DEVICE_FILTERS_STORAGE_KEY,
      JSON.stringify({ colorScheme: "neon", visionFilter: "deepfried", blurPx: "huge" }),
    );
    const { result } = renderHook(() => useDeviceFilters());
    expect(result.current.colorScheme).toBe("system");
    expect(result.current.visionFilter).toBe("none");
    expect(result.current.blurPx).toBe(0);
  });

  it("filterCSS chains url() and blur() when both are active", () => {
    expect(filterCSS("none", 0)).toBe("");
    expect(filterCSS("protanopia", 0)).toBe("url(#rcl-vision-protanopia)");
    expect(filterCSS("none", 3)).toBe("blur(3px)");
    expect(filterCSS("tritanopia", 2)).toBe("url(#rcl-vision-tritanopia) blur(2px)");
  });
});
