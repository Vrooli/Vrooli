/**
 * Unit tests for useSpatialNav.
 *
 * The hook wraps `initSpatialNav` from `@vrooli/iframe-bridge/spatial`.
 * Tests pin two contracts:
 *
 *   1. mount calls initSpatialNav exactly once with the passed options
 *   2. unmount calls controller.dispose() exactly once
 *
 * Same hoisting-safe `vi.hoisted` + per-test reset pattern as
 * useGamepad.test.ts.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";

const mockState = vi.hoisted(() => {
  const controller = {
    registerGroup: vi.fn().mockReturnValue(vi.fn()),
    pushScope: vi.fn(),
    popScope: vi.fn(),
    dispose: vi.fn(),
  };
  const initSpatialNav = vi.fn().mockImplementation(() => controller);
  return { controller, initSpatialNav };
});

vi.mock("@vrooli/iframe-bridge/spatial", () => ({
  initSpatialNav: mockState.initSpatialNav,
}));

import { useSpatialNav } from "./useSpatialNav";

describe("useSpatialNav", () => {
  beforeEach(() => {
    mockState.controller.dispose.mockReset();
    mockState.controller.registerGroup.mockReset().mockReturnValue(vi.fn());
    mockState.controller.pushScope.mockReset();
    mockState.controller.popScope.mockReset();
    mockState.initSpatialNav.mockClear();
  });

  afterEach(() => {
    mockState.controller.dispose.mockReset();
    mockState.initSpatialNav.mockClear();
  });

  it("calls initSpatialNav once on mount with the passed options", () => {
    // Empty options object — the SpatialNavBridgeOptions type has no
    // required fields. We assert the call shape (count + arg identity)
    // rather than relying on specific option values.
    const opts = {};
    renderHook(() => useSpatialNav(opts));

    expect(mockState.initSpatialNav).toHaveBeenCalledTimes(1);
    expect(mockState.initSpatialNav).toHaveBeenCalledWith(opts);
  });

  it("calls controller.dispose on unmount", () => {
    const { unmount } = renderHook(() => useSpatialNav());

    expect(mockState.controller.dispose).not.toHaveBeenCalled();
    unmount();
    expect(mockState.controller.dispose).toHaveBeenCalledTimes(1);
  });
});
