import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { usePressGesture } from "../hooks/usePressGesture";

describe("usePressGesture", () => {
  const onTap = vi.fn();
  const onLongPress = vi.fn();
  const onMoveThreshold = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  const makePointerEvent = (overrides?: Partial<React.PointerEvent<HTMLElement>>) => ({
    pointerType: "touch",
    pointerId: 1,
    button: 0,
    clientX: 10,
    clientY: 20,
    currentTarget: document.createElement("button"),
    ...overrides,
  } as unknown as React.PointerEvent<HTMLElement>);

  it("classifies a short touch as a tap", () => {
    const { result } = renderHook(() => usePressGesture({
      onTap,
      onLongPress,
      onMoveThreshold,
    }));

    act(() => {
      result.current.getGestureHandlers("pane-1").onPointerDown(makePointerEvent());
      window.dispatchEvent(new PointerEvent("pointerup", {
        pointerId: 1,
        pointerType: "touch",
        clientX: 10,
        clientY: 20,
      }));
    });

    expect(onTap).toHaveBeenCalledWith("pane-1", { x: 10, y: 20 });
    expect(onLongPress).not.toHaveBeenCalled();
    expect(onMoveThreshold).not.toHaveBeenCalled();
  });

  it("opens long press on release after the hold threshold", () => {
    const { result } = renderHook(() => usePressGesture({
      longPressMs: 500,
      onTap,
      onLongPress,
    }));

    act(() => {
      result.current.getGestureHandlers("pane-1").onPointerDown(makePointerEvent());
      vi.advanceTimersByTime(500);
    });
    expect(onLongPress).not.toHaveBeenCalled();

    act(() => {
      window.dispatchEvent(new PointerEvent("pointerup", {
        pointerId: 1,
        pointerType: "touch",
        clientX: 11,
        clientY: 21,
      }));
    });

    expect(onLongPress).toHaveBeenCalledWith("pane-1", { x: 11, y: 21 });
    expect(onTap).not.toHaveBeenCalled();
  });

  it("cancels tap and long press after movement crosses the threshold", () => {
    const { result } = renderHook(() => usePressGesture({
      moveThresholdPx: 8,
      onTap,
      onLongPress,
      onMoveThreshold,
    }));

    act(() => {
      result.current.getGestureHandlers("pane-1").onPointerDown(makePointerEvent());
      window.dispatchEvent(new PointerEvent("pointermove", {
        pointerId: 1,
        pointerType: "touch",
        clientX: 10,
        clientY: 40,
      }));
      vi.advanceTimersByTime(500);
      window.dispatchEvent(new PointerEvent("pointerup", {
        pointerId: 1,
        pointerType: "touch",
        clientX: 10,
        clientY: 40,
      }));
    });

    expect(onMoveThreshold).toHaveBeenCalledOnce();
    expect(onTap).not.toHaveBeenCalled();
    expect(onLongPress).not.toHaveBeenCalled();
  });

  it("suppresses the follow-up synthetic click after a moved touch, then expires", () => {
    const { result } = renderHook(() => usePressGesture({
      moveThresholdPx: 8,
      onTap,
      onLongPress,
      onMoveThreshold,
    }));

    act(() => {
      result.current.getGestureHandlers("pane-1").onPointerDown(makePointerEvent());
      window.dispatchEvent(new PointerEvent("pointermove", {
        pointerId: 1,
        pointerType: "touch",
        clientX: 10,
        clientY: 40,
      }));
    });

    expect(result.current.shouldSuppressClick("pane-1")).toBe(true);
    expect(result.current.shouldSuppressClick("pane-1")).toBe(false);

    act(() => {
      result.current.getGestureHandlers("pane-1").onPointerDown(makePointerEvent());
      window.dispatchEvent(new PointerEvent("pointermove", {
        pointerId: 1,
        pointerType: "touch",
        clientX: 10,
        clientY: 40,
      }));
      vi.advanceTimersByTime(751);
    });

    expect(result.current.shouldSuppressClick("pane-1")).toBe(false);
  });

  it("ignores mouse pointerdown so normal mouse click and drag logic can own it", () => {
    const { result } = renderHook(() => usePressGesture({
      onTap,
      onLongPress,
      onMoveThreshold,
    }));

    act(() => {
      result.current.getGestureHandlers("pane-1").onPointerDown(makePointerEvent({
        pointerType: "mouse",
      }));
      window.dispatchEvent(new PointerEvent("pointerup", {
        pointerId: 1,
        pointerType: "mouse",
        clientX: 10,
        clientY: 20,
      }));
    });

    expect(onTap).not.toHaveBeenCalled();
    expect(onLongPress).not.toHaveBeenCalled();
    expect(onMoveThreshold).not.toHaveBeenCalled();
  });
});
