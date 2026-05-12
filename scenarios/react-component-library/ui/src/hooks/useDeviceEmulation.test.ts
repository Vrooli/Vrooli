import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { act, renderHook } from "@testing-library/react";

import {
  DEVICE_EMULATION_STORAGE_KEY,
  DEVICE_PRESETS,
  ZOOM_MAX,
  ZOOM_MIN,
  useDeviceEmulation,
} from "./useDeviceEmulation";

describe("useDeviceEmulation", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });
  afterEach(() => {
    window.localStorage.clear();
  });

  it("defaults to Desktop 1280 with zoom 1 and no rotation", () => {
    const { result } = renderHook(() => useDeviceEmulation());
    expect(result.current.presetId).toBe("desktop-1280");
    expect(result.current.zoom).toBe(1);
    expect(result.current.isRotated).toBe(false);
    expect(result.current.displayWidth).toBe(1280);
    expect(result.current.displayHeight).toBe(800);
  });

  it("exposes every preset listed in the spec", () => {
    const { result } = renderHook(() => useDeviceEmulation());
    const ids = result.current.presets.map((p) => p.id);
    expect(ids).toEqual(
      expect.arrayContaining([
        "iphone-14",
        "iphone-se",
        "ipad",
        "ipad-pro",
        "desktop-1280",
        "desktop-1440",
        "desktop-1920",
      ]),
    );
  });

  it("changes display dimensions when preset is selected", () => {
    const { result } = renderHook(() => useDeviceEmulation());
    act(() => result.current.setPreset("iphone-se"));
    expect(result.current.displayWidth).toBe(375);
    expect(result.current.displayHeight).toBe(667);
  });

  it("clamps zoom into [0.1, 2.0]", () => {
    const { result } = renderHook(() => useDeviceEmulation());
    act(() => result.current.setZoom(5));
    expect(result.current.zoom).toBe(ZOOM_MAX);
    act(() => result.current.setZoom(-1));
    expect(result.current.zoom).toBe(ZOOM_MIN);
  });

  it("zoomIn / zoomOut step by 0.1 and clamp", () => {
    const { result } = renderHook(() => useDeviceEmulation());
    act(() => result.current.setZoom(1.95));
    act(() => result.current.zoomIn());
    expect(result.current.zoom).toBe(ZOOM_MAX);
    act(() => result.current.setZoom(0.15));
    act(() => result.current.zoomOut());
    expect(result.current.zoom).toBe(ZOOM_MIN);
  });

  it("resetZoom returns to 1.0 without touching preset or rotation", () => {
    const { result } = renderHook(() => useDeviceEmulation());
    act(() => result.current.setPreset("ipad"));
    act(() => result.current.rotate());
    act(() => result.current.setZoom(1.5));
    act(() => result.current.resetZoom());
    expect(result.current.zoom).toBe(1);
    expect(result.current.presetId).toBe("ipad");
    expect(result.current.isRotated).toBe(true);
  });

  it("rotate swaps width and height", () => {
    const { result } = renderHook(() => useDeviceEmulation());
    act(() => result.current.setPreset("iphone-14"));
    expect(result.current.displayWidth).toBe(390);
    expect(result.current.displayHeight).toBe(844);
    act(() => result.current.rotate());
    expect(result.current.displayWidth).toBe(844);
    expect(result.current.displayHeight).toBe(390);
  });

  it("reset returns to defaults", () => {
    const { result } = renderHook(() => useDeviceEmulation());
    act(() => result.current.setPreset("ipad-pro"));
    act(() => result.current.rotate());
    act(() => result.current.setZoom(0.5));
    act(() => result.current.reset());
    expect(result.current.presetId).toBe("desktop-1280");
    expect(result.current.isRotated).toBe(false);
    expect(result.current.zoom).toBe(1);
  });

  it("scaledWidth / scaledHeight reflect zoom multiplier", () => {
    const { result } = renderHook(() => useDeviceEmulation());
    act(() => result.current.setPreset("iphone-se"));
    act(() => result.current.setZoom(0.5));
    expect(result.current.scaledWidth).toBe(187.5);
    expect(result.current.scaledHeight).toBe(333.5);
  });

  it("persists state to localStorage under the namespaced key", () => {
    const { result, unmount } = renderHook(() => useDeviceEmulation());
    act(() => result.current.setPreset("ipad"));
    act(() => result.current.setZoom(0.75));
    act(() => result.current.rotate());
    unmount();

    const raw = window.localStorage.getItem(DEVICE_EMULATION_STORAGE_KEY);
    expect(raw).not.toBeNull();
    const parsed = JSON.parse(raw!) as Record<string, unknown>;
    expect(parsed.presetId).toBe("ipad");
    expect(parsed.zoom).toBe(0.75);
    expect(parsed.isRotated).toBe(true);
  });

  it("restores persisted state on a fresh mount", () => {
    window.localStorage.setItem(
      DEVICE_EMULATION_STORAGE_KEY,
      JSON.stringify({ presetId: "iphone-14", zoom: 1.5, isRotated: true }),
    );
    const { result } = renderHook(() => useDeviceEmulation());
    expect(result.current.presetId).toBe("iphone-14");
    expect(result.current.zoom).toBe(1.5);
    expect(result.current.isRotated).toBe(true);
  });

  it("tolerates schema drift — unknown presetId falls back to default", () => {
    window.localStorage.setItem(
      DEVICE_EMULATION_STORAGE_KEY,
      JSON.stringify({ presetId: "future-foldable", zoom: 1, isRotated: false }),
    );
    const { result } = renderHook(() => useDeviceEmulation());
    expect(DEVICE_PRESETS.some((p) => p.id === result.current.presetId)).toBe(true);
    expect(result.current.presetId).toBe("desktop-1280");
  });

  it("tolerates corrupt payload", () => {
    window.localStorage.setItem(DEVICE_EMULATION_STORAGE_KEY, "{not json");
    const { result } = renderHook(() => useDeviceEmulation());
    expect(result.current.presetId).toBe("desktop-1280");
    expect(result.current.zoom).toBe(1);
  });
});
