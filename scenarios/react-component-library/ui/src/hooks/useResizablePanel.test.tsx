import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { useResizablePanel } from "./useResizablePanel";

describe("useResizablePanel", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("loads, clamps, persists, and commits pointer-resized dimensions", () => {
    window.localStorage.setItem("sidebar", "999");
    const container = document.createElement("div");
    const target = document.createElement("aside");
    vi.spyOn(container, "getBoundingClientRect").mockReturnValue({
      width: 700,
      right: 700,
      left: 0,
      top: 0,
      bottom: 0,
      height: 0,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    });
    const containerRef = { current: container };
    const targetRef = { current: target };
    const onSizeCommit = vi.fn();
    const { result } = renderHook(() =>
      useResizablePanel({
        containerRef,
        targetRef,
        minSize: 240,
        maxSize: 520,
        defaultSize: 320,
        storageKey: "sidebar",
        adjacentMinSize: 100,
        handleWidth: 20,
        onSizeCommit,
      }),
    );
    expect(result.current.size).toBe(520);

    act(() => {
      result.current.resizeHandleProps.onPointerDown({
        button: 0,
        preventDefault: vi.fn(),
      } as never);
    });
    const move = new Event("pointermove");
    Object.defineProperty(move, "clientX", { value: 450 });
    act(() => window.dispatchEvent(move));
    expect(target.style.width).toBe("450px");
    act(() => window.dispatchEvent(new Event("pointerup")));
    expect(result.current.isResizing).toBe(false);
    expect(result.current.size).toBe(450);
    expect(onSizeCommit).toHaveBeenCalledWith(450);
    expect(window.localStorage.getItem("sidebar")).toBe("450");
  });
});
