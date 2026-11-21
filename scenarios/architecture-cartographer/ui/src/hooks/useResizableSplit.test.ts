import { describe, expect, it } from "vitest";
import { act, renderHook } from "@testing-library/react";

import { useResizableSplit } from "./useResizableSplit";

describe("useResizableSplit", () => {
  it("starts at the initial percent clamped to [min, max]", () => {
    const { result } = renderHook(() => useResizableSplit({ initialPercent: 5, min: 20 }));
    expect(result.current.percent).toBe(20);
  });

  it("setPercent clamps to the min/max bounds", () => {
    const { result } = renderHook(() => useResizableSplit({ min: 30, max: 70 }));
    act(() => {
      result.current.setPercent(10);
    });
    expect(result.current.percent).toBe(30);
    act(() => {
      result.current.setPercent(90);
    });
    expect(result.current.percent).toBe(70);
  });

  it("isDragging flips true on beginDrag", () => {
    const { result } = renderHook(() => useResizableSplit());
    expect(result.current.isDragging).toBe(false);
    act(() => {
      result.current.beginDrag({ currentTarget: null } as unknown as PointerEvent);
    });
    expect(result.current.isDragging).toBe(true);
  });
});
