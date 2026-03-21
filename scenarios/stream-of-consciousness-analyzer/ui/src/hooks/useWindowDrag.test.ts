import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { useWindowDrag } from "./useWindowDrag";

function fireMouseEvent(type: string, clientX: number, clientY: number) {
  window.dispatchEvent(new MouseEvent(type, { clientX, clientY, bubbles: true }));
}

function makeMouse(clientX: number, clientY: number): MouseEvent {
  return new MouseEvent("mousedown", { clientX, clientY });
}

describe("useWindowDrag", () => {
  it("calls onMove during drag with displacement", () => {
    const onMove = vi.fn();
    const { result } = renderHook(() => useWindowDrag({ onMove }));

    act(() => result.current.startDrag(makeMouse(100, 200)));
    expect(result.current.isDragging.current).toBe(true);

    act(() => fireMouseEvent("mousemove", 130, 250));
    expect(onMove).toHaveBeenCalledWith(30, 50);
  });

  it("calls onEnd on mouseup with final displacement", () => {
    const onEnd = vi.fn();
    const { result } = renderHook(() => useWindowDrag({ onEnd }));

    act(() => result.current.startDrag(makeMouse(50, 50)));
    act(() => fireMouseEvent("mouseup", 80, 100));

    expect(onEnd).toHaveBeenCalledWith(30, 50);
    expect(result.current.isDragging.current).toBe(false);
  });

  it("applies scale factor to displacement", () => {
    const onEnd = vi.fn();
    const { result } = renderHook(() => useWindowDrag({ onEnd, scale: 0.5 }));

    act(() => result.current.startDrag(makeMouse(0, 0)));
    act(() => fireMouseEvent("mouseup", 100, 200));

    expect(onEnd).toHaveBeenCalledWith(50, 100);
  });

  it("cleans up listeners on unmount", () => {
    const onMove = vi.fn();
    const { result, unmount } = renderHook(() => useWindowDrag({ onMove }));

    act(() => result.current.startDrag(makeMouse(0, 0)));
    unmount();

    // After unmount, mousemove should not trigger onMove
    fireMouseEvent("mousemove", 100, 100);
    expect(onMove).not.toHaveBeenCalled();
  });

  it("removes listeners after drag completes", () => {
    const onMove = vi.fn();
    const { result } = renderHook(() => useWindowDrag({ onMove }));

    act(() => result.current.startDrag(makeMouse(0, 0)));
    act(() => fireMouseEvent("mouseup", 10, 10));

    onMove.mockClear();
    fireMouseEvent("mousemove", 50, 50);
    expect(onMove).not.toHaveBeenCalled();
  });
});
