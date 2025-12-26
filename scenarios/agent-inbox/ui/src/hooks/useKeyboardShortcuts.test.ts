/**
 * @file useKeyboardShortcuts.test.ts
 * Tests for the keyboard shortcuts hook.
 * [REQ:KEY-001, KEY-002, KEY-003, KEY-004, KEY-005]
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { useKeyboardShortcuts, formatShortcutKey, type KeyboardShortcut } from "./useKeyboardShortcuts";

describe("useKeyboardShortcuts", () => {
  let addEventListenerSpy: ReturnType<typeof vi.spyOn>;
  let removeEventListenerSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    addEventListenerSpy = vi.spyOn(document, "addEventListener");
    removeEventListenerSpy = vi.spyOn(document, "removeEventListener");
  });

  afterEach(() => {
    addEventListenerSpy.mockRestore();
    removeEventListenerSpy.mockRestore();
  });

  it("registers keydown event listener on mount", () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: "j", description: "Next", action: vi.fn() },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    expect(addEventListenerSpy).toHaveBeenCalledWith("keydown", expect.any(Function));
  });

  it("removes event listener on unmount", () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: "j", description: "Next", action: vi.fn() },
    ];

    const { unmount } = renderHook(() => useKeyboardShortcuts(shortcuts));
    unmount();

    expect(removeEventListenerSpy).toHaveBeenCalledWith("keydown", expect.any(Function));
  });

  it("calls action when key matches", () => {
    const action = vi.fn();
    const shortcuts: KeyboardShortcut[] = [
      { key: "j", description: "Next", action },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Simulate keypress
    const event = new KeyboardEvent("keydown", { key: "j" });
    document.dispatchEvent(event);

    expect(action).toHaveBeenCalledTimes(1);
  });

  it("calls action with ctrlKey modifier", () => {
    const action = vi.fn();
    const shortcuts: KeyboardShortcut[] = [
      { key: "n", ctrlKey: true, description: "New", action },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Without ctrl - should not trigger
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "n" }));
    expect(action).not.toHaveBeenCalled();

    // With ctrl - should trigger
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "n", ctrlKey: true }));
    expect(action).toHaveBeenCalledTimes(1);
  });

  it("calls action with metaKey (Cmd) modifier", () => {
    const action = vi.fn();
    const shortcuts: KeyboardShortcut[] = [
      { key: "n", ctrlKey: true, description: "New", action },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // With meta - should trigger (Cmd on Mac maps to ctrlKey in our logic)
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "n", metaKey: true }));
    expect(action).toHaveBeenCalledTimes(1);
  });

  it("prevents default when shortcut matches", () => {
    const shortcuts: KeyboardShortcut[] = [
      { key: "k", description: "Previous", action: vi.fn() },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    const event = new KeyboardEvent("keydown", { key: "k" });
    const preventDefaultSpy = vi.spyOn(event, "preventDefault");
    document.dispatchEvent(event);

    expect(preventDefaultSpy).toHaveBeenCalled();
  });

  it("does not call action when typing in input", () => {
    const action = vi.fn();
    const shortcuts: KeyboardShortcut[] = [
      { key: "j", description: "Next", action },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Simulate keypress inside input element
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();

    const event = new KeyboardEvent("keydown", { key: "j", bubbles: true });
    Object.defineProperty(event, "target", { value: input, writable: false });
    document.dispatchEvent(event);

    expect(action).not.toHaveBeenCalled();

    document.body.removeChild(input);
  });

  it("allows Escape even when typing in input", () => {
    const action = vi.fn();
    const shortcuts: KeyboardShortcut[] = [
      { key: "Escape", description: "Close", action },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Simulate Escape keypress inside input element
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();

    const event = new KeyboardEvent("keydown", { key: "Escape", bubbles: true });
    Object.defineProperty(event, "target", { value: input, writable: false });
    document.dispatchEvent(event);

    expect(action).toHaveBeenCalledTimes(1);

    document.body.removeChild(input);
  });

  it("does not call action when disabled", () => {
    const action = vi.fn();
    const shortcuts: KeyboardShortcut[] = [
      { key: "j", description: "Next", action },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts, { disabled: true }));

    document.dispatchEvent(new KeyboardEvent("keydown", { key: "j" }));

    expect(action).not.toHaveBeenCalled();
  });

  it("handles shiftKey modifier", () => {
    const action = vi.fn();
    const shortcuts: KeyboardShortcut[] = [
      { key: "Enter", shiftKey: true, description: "New line", action },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Without shift - should not trigger
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" }));
    expect(action).not.toHaveBeenCalled();

    // With shift - should trigger
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", shiftKey: true }));
    expect(action).toHaveBeenCalledTimes(1);
  });

  it("matches keys case-insensitively", () => {
    const action = vi.fn();
    const shortcuts: KeyboardShortcut[] = [
      { key: "J", description: "Next", action },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    // Lowercase j should still match uppercase J shortcut
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "j" }));
    expect(action).toHaveBeenCalledTimes(1);
  });

  it("handles / key for search shortcut [KEY-005]", () => {
    const action = vi.fn();
    const shortcuts: KeyboardShortcut[] = [
      { key: "/", description: "Focus search", action },
    ];

    renderHook(() => useKeyboardShortcuts(shortcuts));

    document.dispatchEvent(new KeyboardEvent("keydown", { key: "/" }));
    expect(action).toHaveBeenCalledTimes(1);
  });
});

describe("formatShortcutKey", () => {
  // Mock navigator.platform for consistent tests
  const originalPlatform = Object.getOwnPropertyDescriptor(navigator, "platform");

  afterEach(() => {
    if (originalPlatform) {
      Object.defineProperty(navigator, "platform", originalPlatform);
    }
  });

  it("formats single key correctly", () => {
    const result = formatShortcutKey({ key: "j", description: "Next", action: vi.fn() });
    expect(result).toBe("J");
  });

  it("formats Escape as Esc", () => {
    const result = formatShortcutKey({ key: "Escape", description: "Close", action: vi.fn() });
    expect(result).toBe("Esc");
  });

  it("formats space key as Space", () => {
    const result = formatShortcutKey({ key: " ", description: "Toggle", action: vi.fn() });
    expect(result).toBe("Space");
  });

  it("includes Shift modifier", () => {
    const result = formatShortcutKey({ key: "Enter", shiftKey: true, description: "New line", action: vi.fn() });
    expect(result).toBe("Shift + Enter");
  });

  it("includes Ctrl modifier on non-Mac", () => {
    Object.defineProperty(navigator, "platform", { value: "Win32", configurable: true });
    const result = formatShortcutKey({ key: "n", ctrlKey: true, description: "New", action: vi.fn() });
    expect(result).toBe("Ctrl + N");
  });
});
