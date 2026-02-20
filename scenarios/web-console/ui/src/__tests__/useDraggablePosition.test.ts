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
});
