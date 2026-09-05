import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useListReorder } from "./useListReorder";

/** A pointer event shaped enough for the hook's grip handler. */
const grip = (x = 0, y = 0) => ({ button: 0, clientX: x, clientY: y, preventDefault: vi.fn() }) as unknown as React.PointerEvent;

/** Dispatch a real window pointer event, which is what the hook listens to. */
function pointer(type: string, x: number, y: number) {
  const event = new Event(type, { bubbles: true }) as Event & { clientX: number; clientY: number };
  event.clientX = x;
  event.clientY = y;
  window.dispatchEvent(event);
}

/** A stand-in element occupying a known rectangle. */
function nodeAt(top: number): HTMLElement {
  const node = document.createElement("div");
  node.getBoundingClientRect = () => ({ left: 0, right: 100, top, bottom: top + 50, width: 100, height: 50, x: 0, y: top, toJSON: () => ({}) });
  return node;
}

describe("useListReorder", () => {
  const source = ["a", "b", "c"];
  let onCommit: ReturnType<typeof vi.fn>;

  beforeEach(() => { onCommit = vi.fn(); });

  it("moves an item with the keyboard and reports the new order", () => {
    const { result } = renderHook(() => useListReorder({ source, onCommit }));
    act(() => { expect(result.current.moveFocused(0, 1)).toBe(true); });
    expect(onCommit).toHaveBeenCalledWith(["b", "a", "c"]);
    expect(result.current.items).toEqual(["b", "a", "c"]);
  });

  it("refuses a keyboard move that would leave the list", () => {
    const { result } = renderHook(() => useListReorder({ source, onCommit }));
    act(() => { expect(result.current.moveFocused(0, -1)).toBe(false); });
    act(() => { expect(result.current.moveFocused(2, 1)).toBe(false); });
    expect(onCommit).not.toHaveBeenCalled();
  });

  it("does nothing at all while disabled", () => {
    const { result } = renderHook(() => useListReorder({ source, onCommit, enabled: false }));
    act(() => { expect(result.current.moveFocused(0, 1)).toBe(false); });
    act(() => { result.current.onGripPointerDown(0, grip()); });
    expect(result.current.draggingIndex).toBeNull();
    expect(onCommit).not.toHaveBeenCalled();
  });

  // A tap on the grip is a tap, not a reorder. Without the threshold a
  // one-pixel tremor during a press silently reorders the operator's list.
  it("treats a press that never travels as a press, not a drag", () => {
    const { result } = renderHook(() => useListReorder({ source, onCommit }));
    act(() => { result.current.registerItem(0, nodeAt(0)); result.current.registerItem(1, nodeAt(50)); });
    act(() => { result.current.onGripPointerDown(0, grip(10, 10)); });
    act(() => { pointer("pointermove", 12, 11); });
    act(() => { pointer("pointerup", 12, 11); });
    expect(onCommit).not.toHaveBeenCalled();
  });

  it("commits a drag that crosses the threshold and lands on another card", () => {
    const { result } = renderHook(() => useListReorder({ source, onCommit }));
    act(() => {
      result.current.registerItem(0, nodeAt(0));
      result.current.registerItem(1, nodeAt(50));
      result.current.registerItem(2, nodeAt(100));
    });
    act(() => { result.current.onGripPointerDown(0, grip(10, 10)); });
    act(() => { pointer("pointermove", 10, 120); });
    act(() => { pointer("pointerup", 10, 120); });
    expect(onCommit).toHaveBeenCalledWith(["b", "c", "a"]);
  });

  it("cancels cleanly when the pointer is taken away mid-drag", () => {
    const { result } = renderHook(() => useListReorder({ source, onCommit }));
    act(() => { result.current.registerItem(0, nodeAt(0)); result.current.registerItem(1, nodeAt(50)); });
    act(() => { result.current.onGripPointerDown(0, grip(10, 10)); });
    act(() => { pointer("pointermove", 10, 60); });
    act(() => { pointer("pointercancel", 10, 60); });
    expect(result.current.draggingIndex).toBeNull();
    expect(result.current.active).toBe(false);
  });

  // A catalog refresh mid-drag moves the ground under a position computed
  // against a list that no longer exists.
  it("drops a pending order when the source list changes", () => {
    const { result, rerender } = renderHook(({ list }) => useListReorder({ source: list, onCommit }), {
      initialProps: { list: source },
    });
    act(() => { result.current.moveFocused(0, 1); });
    expect(result.current.items).toEqual(["b", "a", "c"]);
    rerender({ list: ["a", "b", "c", "d"] });
    expect(result.current.items).toEqual(["a", "b", "c", "d"]);
  });

  it("ignores a non-primary button", () => {
    const { result } = renderHook(() => useListReorder({ source, onCommit }));
    act(() => { result.current.onGripPointerDown(0, { button: 2, clientX: 0, clientY: 0, preventDefault: vi.fn() } as unknown as React.PointerEvent); });
    expect(result.current.draggingIndex).toBeNull();
  });

  it("returns to the source order on reset", () => {
    const { result } = renderHook(() => useListReorder({ source, onCommit }));
    act(() => { result.current.moveFocused(2, -1); });
    expect(result.current.items).toEqual(["a", "c", "b"]);
    act(() => { result.current.reset(); });
    expect(result.current.items).toEqual(["a", "b", "c"]);
  });
});
