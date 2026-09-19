import { describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";

import { useKeyboardShortcut } from "./useKeyboardShortcut";

const dispatch = (target: EventTarget, init: KeyboardEventInit) => {
  target.dispatchEvent(new KeyboardEvent("keydown", init));
};

describe("useKeyboardShortcut", () => {
  it("fires the handler when the chord matches", () => {
    const target = new EventTarget();
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcut({ chord: "k", handler, target }));
    dispatch(target, { key: "k" });
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("does not fire when modifier mismatches", () => {
    const target = new EventTarget();
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcut({ chord: "mod+k", handler, target }));
    dispatch(target, { key: "k" });
    expect(handler).not.toHaveBeenCalled();
  });

  it("fires the handler with mod=ctrl on non-mac", () => {
    const target = new EventTarget();
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcut({ chord: "mod+k", handler, target }));
    dispatch(target, { key: "k", ctrlKey: true });
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("respects enabled=false", () => {
    const target = new EventTarget();
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcut({ chord: "k", handler, enabled: false, target }));
    dispatch(target, { key: "k" });
    expect(handler).not.toHaveBeenCalled();
  });

  it("removes its listener on unmount", () => {
    const target = new EventTarget();
    const handler = vi.fn();
    const { unmount } = renderHook(() =>
      useKeyboardShortcut({ chord: "k", handler, target }),
    );
    unmount();
    dispatch(target, { key: "k" });
    expect(handler).not.toHaveBeenCalled();
  });
});
