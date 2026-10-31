import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useDraggablePosition } from "../hooks/useDraggablePosition";
import type { PointerEvent as ReactPointerEvent } from "react";

// Stable default position — must NOT be an inline literal in renderHook
// to avoid infinite re-render loops (object identity changes each render).
const DEFAULT_POS = { x: 100, y: 12 };

function makeMockElement(): HTMLDivElement {
  const el = document.createElement("div");
  vi.spyOn(el, "getBoundingClientRect").mockReturnValue({
    left: 100,
    top: 12,
    width: 160,
    height: 36,
    right: 260,
    bottom: 48,
    x: 100,
    y: 12,
    toJSON: () => ({}),
  });
  el.setPointerCapture = vi.fn();
  el.releasePointerCapture = vi.fn();
  el.hasPointerCapture = vi.fn().mockReturnValue(false);
  return el;
}

function makePointerEvent(
  overrides: Partial<PointerEvent> = {},
): ReactPointerEvent {
  return {
    pointerId: 1,
    pointerType: "mouse",
    button: 0,
    clientX: 120,
    clientY: 24,
    preventDefault: vi.fn(),
    stopPropagation: vi.fn(),
    ...overrides,
  } as unknown as ReactPointerEvent;
}

describe("useDraggablePosition", () => {
  beforeEach(() => {
    vi.stubGlobal("innerWidth", 1024);
    vi.stubGlobal("innerHeight", 768);
    localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("updates DOM transform directly during drag (no React render lag)", () => {
    const el = makeMockElement();
    const { result } = renderHook(() =>
      useDraggablePosition({
        isActive: true,
        defaultPosition: DEFAULT_POS,
      }),
    );

    // Attach the mock element to the ref
    act(() => {
      (result.current.elementRef as { current: HTMLElement | null }).current = el;
    });

    // Simulate pointerDown at (120, 24) — offset within element is (20, 12)
    act(() => {
      result.current.pointerHandlers.onPointerDown(
        makePointerEvent({ clientX: 120, clientY: 24 }),
      );
    });

    // Move past the drag threshold (6px)
    act(() => {
      result.current.pointerHandlers.onPointerMove(
        makePointerEvent({ clientX: 140, clientY: 40 }),
      );
    });

    // The DOM element's style.transform should be set directly
    // newX = 140 - 20 = 120, newY = 40 - 12 = 28
    expect(el.style.transform).toBe("translate3d(120px, 28px, 0)");

    // Move again
    act(() => {
      result.current.pointerHandlers.onPointerMove(
        makePointerEvent({ clientX: 200, clientY: 60 }),
      );
    });

    // newX = 200 - 20 = 180, newY = 60 - 12 = 48
    expect(el.style.transform).toBe("translate3d(180px, 48px, 0)");
  });

  it("syncs final position to React state on pointerUp", () => {
    const el = makeMockElement();
    const { result } = renderHook(() =>
      useDraggablePosition({
        isActive: true,
        defaultPosition: DEFAULT_POS,
      }),
    );

    act(() => {
      (result.current.elementRef as { current: HTMLElement | null }).current = el;
    });

    // Drag: down → move past threshold → up
    act(() => {
      result.current.pointerHandlers.onPointerDown(
        makePointerEvent({ clientX: 120, clientY: 24 }),
      );
    });

    act(() => {
      result.current.pointerHandlers.onPointerMove(
        makePointerEvent({ clientX: 200, clientY: 60 }),
      );
    });

    act(() => {
      result.current.pointerHandlers.onPointerUp(
        makePointerEvent({ clientX: 200, clientY: 60 }),
      );
    });

    // After drag ends, React state should reflect the final position
    // newX = 200 - 20 = 180, newY = 60 - 12 = 48
    expect(result.current.position).toEqual({ x: 180, y: 48 });
  });

  it("does not update React state position during drag (only DOM)", () => {
    const el = makeMockElement();
    const { result } = renderHook(() =>
      useDraggablePosition({
        isActive: true,
        defaultPosition: DEFAULT_POS,
      }),
    );

    act(() => {
      (result.current.elementRef as { current: HTMLElement | null }).current = el;
    });

    const positionBeforeDrag = result.current.position;

    act(() => {
      result.current.pointerHandlers.onPointerDown(
        makePointerEvent({ clientX: 120, clientY: 24 }),
      );
    });

    act(() => {
      result.current.pointerHandlers.onPointerMove(
        makePointerEvent({ clientX: 200, clientY: 60 }),
      );
    });

    // During active drag, React state should NOT have been updated
    // (the DOM was updated directly instead)
    expect(result.current.position).toEqual(positionBeforeDrag);
    // But the DOM transform IS updated
    expect(el.style.transform).toContain("translate3d");
  });

  it("covers inactive, persisted, cancelled, and completed drag paths", () => {
    const el = makeMockElement();
    localStorage.setItem("drag", JSON.stringify({ x: 20, y: 30, savedAt: 1 }));
    const onDragStart = vi.fn();
    const onDragEnd = vi.fn();
    const { result, rerender } = renderHook(({ active }) => useDraggablePosition({
      isActive: active,
      storageKey: "drag",
      defaultPosition: () => null,
      onDragStart,
      onDragEnd,
    }), { initialProps: { active: true } });
    act(() => { result.current.elementRef.current = el; });
    expect(result.current.position).toEqual({ x: 20, y: 30 });

    act(() => {
      result.current.pointerHandlers.onPointerDown(makePointerEvent({ button: 2 }));
      result.current.pointerHandlers.onPointerMove(makePointerEvent({ pointerId: 99 }));
      result.current.pointerHandlers.onPointerUp(makePointerEvent({ pointerId: 99 }));
    });
    expect(onDragStart).not.toHaveBeenCalled();

    el.hasPointerCapture = vi.fn().mockReturnValue(true);
    act(() => { result.current.pointerHandlers.onPointerDown(makePointerEvent({ clientX: 120, clientY: 24 })); });
    act(() => { result.current.pointerHandlers.onPointerMove(makePointerEvent({ clientX: 150, clientY: 54 })); });
    act(() => { result.current.pointerHandlers.onPointerMove(makePointerEvent({ clientX: 180, clientY: 84 })); });
    act(() => { result.current.pointerHandlers.onPointerUp(makePointerEvent({ clientX: 180, clientY: 84 })); });
    expect(onDragStart).toHaveBeenCalledTimes(1);
    expect(onDragEnd).toHaveBeenCalledWith(expect.objectContaining({ position: expect.any(Object), velocity: expect.any(Object) }));
    const click = { preventDefault: vi.fn(), stopPropagation: vi.fn() } as unknown as Parameters<typeof result.current.handleClickCapture>[0];
    act(() => { result.current.handleClickCapture(click); });
    expect(click.preventDefault).toHaveBeenCalled();

    act(() => { result.current.moveTo({ x: 400, y: 500 }); result.current.resetPosition(); });
    expect(result.current.position).toEqual({ x: 160, y: 72 });
    // Reapplying the current coordinates is intentionally a no-op so resize
    // and docking reconciliation cannot create a render loop.
    const stablePosition = result.current.position;
    act(() => { result.current.moveTo(stablePosition); });
    expect(result.current.position).toBe(stablePosition);
    act(() => { rerender({ active: false }); });
    expect(result.current.floatingStyle).toBeUndefined();
  });

  it("uses fallback positions and survives storage and pointer-capture failures", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    localStorage.setItem("bad", "not-json");
    const { result } = renderHook(() => useDraggablePosition({ isActive: true, storageKey: "bad", defaultPosition: () => null }));
    expect(result.current.position).toEqual({ x: 12, y: 12 });
    const el = makeMockElement();
    el.setPointerCapture = vi.fn(() => { throw new Error("capture failed"); });
    el.hasPointerCapture = vi.fn().mockReturnValue(true);
    el.releasePointerCapture = vi.fn(() => { throw new Error("release failed"); });
    act(() => { result.current.elementRef.current = el; result.current.pointerHandlers.onPointerDown(makePointerEvent()); });
    act(() => { result.current.pointerHandlers.onPointerMove(makePointerEvent({ clientX: 150, clientY: 54 })); });
    act(() => { result.current.pointerHandlers.onPointerUp(makePointerEvent({ clientX: 150, clientY: 54 })); });
    expect(warn).toHaveBeenCalled();
    act(() => { result.current.moveTo({ x: 1, y: 1 }); });
    vi.spyOn(localStorage, "removeItem").mockImplementation(() => { throw new Error("quota"); });
    act(() => { result.current.resetPosition(); });
  });

  it("reclamps a persisted position when ResizeObserver measures the toolbar", () => {
    class MockResizeObserver {
      observe = vi.fn();
      disconnect = vi.fn();
      unobserve = vi.fn();

      constructor(private readonly onResize: ResizeObserverCallback) {
        observed = this;
      }

      notify() {
        this.onResize([], this as unknown as ResizeObserver);
      }
    }
    let observed: MockResizeObserver | undefined;
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
    localStorage.setItem("toolbar", JSON.stringify({ x: 1000, y: 30, savedAt: 1 }));

    const el = makeMockElement();
    const { result, rerender } = renderHook(({ active }) => useDraggablePosition({
      isActive: active,
      storageKey: "toolbar",
      defaultPosition: DEFAULT_POS,
    }), { initialProps: { active: true } });
    act(() => { result.current.elementRef.current = el; });
    act(() => { rerender({ active: false }); });
    act(() => { rerender({ active: true }); });

    const resizeObserver = observed;
    if (!resizeObserver) throw new Error("ResizeObserver was not constructed");
    expect(resizeObserver.observe).toHaveBeenCalledWith(el);
    expect(result.current.position).toEqual({ x: 852, y: 30 });
    act(() => { resizeObserver.notify(); });
    expect(resizeObserver.disconnect).not.toHaveBeenCalled();
  });
});
