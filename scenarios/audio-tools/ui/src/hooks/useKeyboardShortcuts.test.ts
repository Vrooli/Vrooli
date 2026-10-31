import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook, cleanup } from "@testing-library/react";
import { useKeyboardShortcuts } from "./useKeyboardShortcuts";

afterEach(cleanup);

function fire(key: string, opts: Partial<KeyboardEventInit> = {}) {
  window.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, ...opts }));
}

describe("useKeyboardShortcuts", () => {
  it("calls handler when key matches", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: "a", handler }]));
    fire("a");
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("does not call handler for non-matching key", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: "a", handler }]));
    fire("b");
    expect(handler).not.toHaveBeenCalled();
  });

  it("matches keys case-insensitively", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: "A", handler }]));
    fire("a");
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("requires ctrlKey when ctrlOrMeta is true", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: ",", ctrlOrMeta: true, handler }]));
    fire(",");
    expect(handler).not.toHaveBeenCalled();
    fire(",", { ctrlKey: true });
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("fires on metaKey when ctrlOrMeta is true", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: ",", ctrlOrMeta: true, handler }]));
    fire(",", { metaKey: true });
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("blocks when ctrlOrMeta is false but event has ctrlKey", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: "a", handler }]));
    fire("a", { ctrlKey: true });
    expect(handler).not.toHaveBeenCalled();
  });

  it("blocks when ctrlOrMeta is false but event has metaKey", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: "a", handler }]));
    fire("a", { metaKey: true });
    expect(handler).not.toHaveBeenCalled();
  });

  it("requires shiftKey when shift is true", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: "s", shift: true, handler }]));
    fire("s");
    expect(handler).not.toHaveBeenCalled();
    fire("s", { shiftKey: true });
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("requires altKey when alt is true", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: "a", alt: true, handler }]));
    fire("a");
    expect(handler).not.toHaveBeenCalled();
    fire("a", { altKey: true });
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("does not fire on input elements by default", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: "a", handler }]));
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "a", bubbles: true }));
    document.body.removeChild(input);
    expect(handler).not.toHaveBeenCalled();
  });

  it("fires on input elements when allowInInputs is true", () => {
    const handler = vi.fn();
    renderHook(() =>
      useKeyboardShortcuts([{ key: "Escape", handler, allowInInputs: true }]),
    );
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
    document.body.removeChild(input);
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("does not fire on textarea elements by default", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: "a", handler }]));
    const ta = document.createElement("textarea");
    document.body.appendChild(ta);
    ta.dispatchEvent(new KeyboardEvent("keydown", { key: "a", bubbles: true }));
    document.body.removeChild(ta);
    expect(handler).not.toHaveBeenCalled();
  });

  it.skip("does not fire on contenteditable elements by default (jsdom limitation)", () => {
    const handler = vi.fn();
    renderHook(() => useKeyboardShortcuts([{ key: "a", handler }]));
    const div = document.createElement("div");
    div.contentEditable = "true";
    document.body.appendChild(div);
    div.dispatchEvent(new KeyboardEvent("keydown", { key: "a", bubbles: true }));
    document.body.removeChild(div);
    expect(handler).not.toHaveBeenCalled();
  });

  it("removes event listener on unmount", () => {
    const handler = vi.fn();
    const { unmount } = renderHook(() => useKeyboardShortcuts([{ key: "z", handler }]));
    unmount();
    fire("z");
    expect(handler).not.toHaveBeenCalled();
  });

  it("stops at the first matching shortcut (does not call subsequent handlers)", () => {
    const h1 = vi.fn();
    const h2 = vi.fn();
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "a", handler: h1 },
        { key: "a", handler: h2 },
      ]),
    );
    fire("a");
    expect(h1).toHaveBeenCalledTimes(1);
    expect(h2).not.toHaveBeenCalled();
  });

  it("picks up fresh shortcuts via ref (no re-registration)", () => {
    let callCount = 0;
    const handler = () => { callCount++; };
    const { rerender } = renderHook(
      ({ fn }: { fn: (e: KeyboardEvent) => void }) =>
        useKeyboardShortcuts([{ key: "q", handler: fn }]),
      { initialProps: { fn: handler } },
    );
    const handler2 = vi.fn();
    rerender({ fn: handler2 });
    fire("q");
    expect(handler2).toHaveBeenCalledTimes(1);
    expect(callCount).toBe(0);
  });
});
