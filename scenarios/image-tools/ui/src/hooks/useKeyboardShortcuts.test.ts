/**
 * useKeyboardShortcuts tests — the single app-level keydown owner. Branches:
 *   - focus in an editable target (INPUT/TEXTAREA/SELECT/contentEditable) or a
 *     non-HTMLElement target suppresses everything
 *   - a bare key (no Ctrl/Cmd) is ignored
 *   - Ctrl/Cmd+Z undoes; Ctrl/Cmd+Shift+Z and Ctrl/Cmd+Y redo
 *   - handlers are optional (optional-chaining must not throw when absent)
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";

import { useKeyboardShortcuts } from "./useKeyboardShortcuts";

/** Dispatch a keydown on `target` (defaults to document.body). */
function dispatchKey(
  init: KeyboardEventInit,
  target: EventTarget = document.body,
): KeyboardEvent {
  const event = new KeyboardEvent("keydown", { bubbles: true, cancelable: true, ...init });
  // jsdom KeyboardEvent ignores a `target` in the init; dispatch from the node.
  if (target instanceof Node) {
    target.dispatchEvent(event);
  } else {
    window.dispatchEvent(event);
  }
  return event;
}

afterEach(() => {
  document.body.innerHTML = "";
});

describe("useKeyboardShortcuts", () => {
  it("invokes onUndo for Ctrl+Z and prevents default", () => {
    const onUndo = vi.fn(() => true);
    renderHook(() => useKeyboardShortcuts({ onUndo }));

    const event = dispatchKey({ key: "z", ctrlKey: true });
    expect(onUndo).toHaveBeenCalledTimes(1);
    expect(event.defaultPrevented).toBe(true);
  });

  it("invokes onUndo for Cmd+Z (metaKey)", () => {
    const onUndo = vi.fn(() => true);
    renderHook(() => useKeyboardShortcuts({ onUndo }));

    dispatchKey({ key: "Z", metaKey: true });
    expect(onUndo).toHaveBeenCalledTimes(1);
  });

  it("invokes onRedo for Ctrl+Shift+Z", () => {
    const onRedo = vi.fn(() => true);
    renderHook(() => useKeyboardShortcuts({ onRedo }));

    const event = dispatchKey({ key: "z", ctrlKey: true, shiftKey: true });
    expect(onRedo).toHaveBeenCalledTimes(1);
    expect(event.defaultPrevented).toBe(true);
  });

  it("invokes onRedo for Ctrl+Y", () => {
    const onRedo = vi.fn(() => true);
    renderHook(() => useKeyboardShortcuts({ onRedo }));

    dispatchKey({ key: "y", ctrlKey: true });
    expect(onRedo).toHaveBeenCalledTimes(1);
  });

  it("does not invoke onRedo for a plain undo chord", () => {
    const onUndo = vi.fn(() => true);
    const onRedo = vi.fn(() => true);
    renderHook(() => useKeyboardShortcuts({ onUndo, onRedo }));

    dispatchKey({ key: "z", ctrlKey: true });
    expect(onUndo).toHaveBeenCalledTimes(1);
    expect(onRedo).not.toHaveBeenCalled();
  });

  it("ignores keys without a Ctrl/Cmd modifier", () => {
    const onUndo = vi.fn(() => true);
    const onRedo = vi.fn(() => true);
    renderHook(() => useKeyboardShortcuts({ onUndo, onRedo }));

    const event = dispatchKey({ key: "z" });
    expect(onUndo).not.toHaveBeenCalled();
    expect(onRedo).not.toHaveBeenCalled();
    expect(event.defaultPrevented).toBe(false);
  });

  it("does not throw when the matching handler is absent (optional chaining)", () => {
    renderHook(() => useKeyboardShortcuts({}));

    expect(() => dispatchKey({ key: "z", ctrlKey: true })).not.toThrow();
    expect(() => dispatchKey({ key: "y", ctrlKey: true })).not.toThrow();
  });

  it.each([
    ["INPUT", () => document.createElement("input")],
    ["TEXTAREA", () => document.createElement("textarea")],
    ["SELECT", () => document.createElement("select")],
  ])("suppresses shortcuts when focus is in a %s", (_name, make) => {
    const onUndo = vi.fn(() => true);
    renderHook(() => useKeyboardShortcuts({ onUndo }));

    const el = make();
    document.body.appendChild(el);
    dispatchKey({ key: "z", ctrlKey: true }, el);
    expect(onUndo).not.toHaveBeenCalled();
  });

  it("suppresses shortcuts inside a contentEditable host", () => {
    const onUndo = vi.fn(() => true);
    renderHook(() => useKeyboardShortcuts({ onUndo }));

    const el = document.createElement("div");
    el.setAttribute("contenteditable", "true");
    // jsdom reflects contentEditable to isContentEditable.
    Object.defineProperty(el, "isContentEditable", { value: true });
    document.body.appendChild(el);

    dispatchKey({ key: "z", ctrlKey: true }, el);
    expect(onUndo).not.toHaveBeenCalled();
  });

  it("does not suppress for a non-HTMLElement event target (e.g. window/document)", () => {
    const onUndo = vi.fn(() => true);
    renderHook(() => useKeyboardShortcuts({ onUndo }));

    // Dispatching on window yields a target that is not an HTMLElement, so
    // isEditableTarget returns false and the chord runs.
    const event = new KeyboardEvent("keydown", { key: "z", ctrlKey: true });
    window.dispatchEvent(event);
    expect(onUndo).toHaveBeenCalledTimes(1);
  });

  it("removes its window listener on unmount", () => {
    const onUndo = vi.fn(() => true);
    const { unmount } = renderHook(() => useKeyboardShortcuts({ onUndo }));

    unmount();
    dispatchKey({ key: "z", ctrlKey: true });
    expect(onUndo).not.toHaveBeenCalled();
  });
});
