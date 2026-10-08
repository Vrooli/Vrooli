import { act, fireEvent, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  useReleaseOnElementInteraction,
  useTerminalVoiceShortcut,
  useWindowKeyDown,
} from "./useKeyboardListeners";

const { emitShortcutIntent } = vi.hoisted(() => ({ emitShortcutIntent: vi.fn() }));
vi.mock("@vrooli/iframe-bridge", () => ({ emitShortcutIntent }));

describe("keyboard listener hooks", () => {
  beforeEach(() => emitShortcutIntent.mockClear());

  it("relays modified window shortcuts and invokes the local handler", () => {
    const handler = vi.fn();
    renderHook(() => useWindowKeyDown(true, handler));

    fireEvent.keyDown(window, { key: "k", ctrlKey: true, shiftKey: true });

    expect(handler).toHaveBeenCalledOnce();
    expect(emitShortcutIntent).toHaveBeenCalledWith(expect.objectContaining({
      action: "keyboard.shortcut",
      chord: "Ctrl+Shift+k",
      source: "keyboard",
    }));
  });

  it("does not install a window listener while inactive", () => {
    const handler = vi.fn();
    renderHook(() => useWindowKeyDown(false, handler));
    fireEvent.keyDown(window, { key: "k" });
    expect(handler).not.toHaveBeenCalled();
  });

  it("releases an element pin for input gestures and cleans up", () => {
    const element = document.createElement("div");
    const ref = { current: element };
    const release = vi.fn();
    const { unmount } = renderHook(() => useReleaseOnElementInteraction(ref, release));

    fireEvent.wheel(element);
    fireEvent.touchStart(element);
    fireEvent.pointerDown(element);
    fireEvent.keyDown(element, { key: "ArrowDown" });
    expect(release).toHaveBeenCalledTimes(4);

    unmount();
    fireEvent.wheel(element);
    expect(release).toHaveBeenCalledTimes(4);
  });

  it("captures the configured terminal voice shortcut", () => {
    const element = document.createElement("div");
    const ref = { current: element };
    const start = vi.fn();
    const stop = vi.fn();
    renderHook(() => useTerminalVoiceShortcut(ref, "Ctrl+Shift+V", start, stop));

    const down = new KeyboardEvent("keydown", { key: "V", ctrlKey: true, shiftKey: true, bubbles: true, cancelable: true });
    const up = new KeyboardEvent("keyup", { key: "V", ctrlKey: true, shiftKey: true, bubbles: true, cancelable: true });
    act(() => {
      element.dispatchEvent(down);
      element.dispatchEvent(up);
    });
    expect(start).toHaveBeenCalledOnce();
    expect(stop).toHaveBeenCalledOnce();
    expect(down.defaultPrevented).toBe(true);
  });

  it("ignores malformed shortcuts and missing elements", () => {
    const ref = { current: null };
    const start = vi.fn();
    const stop = vi.fn();
    renderHook(() => useTerminalVoiceShortcut(ref, "", start, stop));
    expect(start).not.toHaveBeenCalled();
  });
});
