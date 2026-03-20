// [REQ:REQ-P0-003] Keyboard navigation shortcuts
import { renderHook } from "@testing-library/react";
import { vi } from "vitest";
import { useGlobalKeyboardShortcuts } from "./useGlobalKeyboardShortcuts";

// Mock iframe-bridge to prevent side effects
vi.mock("@vrooli/iframe-bridge", () => ({
  emitShortcutIntent: vi.fn(),
}));

function fireKey(key: string, opts: Partial<KeyboardEventInit> = {}) {
  document.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, ...opts }));
}

describe("useGlobalKeyboardShortcuts", () => {
  it("calls onSwitchView with correct index for Alt+1", () => {
    const onSwitch = vi.fn();
    renderHook(() => useGlobalKeyboardShortcuts(["wizard", "dashboard", "glossary"], onSwitch));

    fireKey("1", { altKey: true });
    expect(onSwitch).toHaveBeenCalledWith(0);
  });

  it("calls onSwitchView with correct index for Alt+3", () => {
    const onSwitch = vi.fn();
    renderHook(() => useGlobalKeyboardShortcuts(["wizard", "dashboard", "glossary"], onSwitch));

    fireKey("3", { altKey: true });
    expect(onSwitch).toHaveBeenCalledWith(2);
  });

  it("ignores keys without Alt modifier", () => {
    const onSwitch = vi.fn();
    renderHook(() => useGlobalKeyboardShortcuts(["wizard", "dashboard"], onSwitch));

    fireKey("1");
    expect(onSwitch).not.toHaveBeenCalled();
  });

  it("ignores Alt+key when Ctrl is also held", () => {
    const onSwitch = vi.fn();
    renderHook(() => useGlobalKeyboardShortcuts(["wizard", "dashboard"], onSwitch));

    fireKey("1", { altKey: true, ctrlKey: true });
    expect(onSwitch).not.toHaveBeenCalled();
  });

  it("ignores Alt+key when Meta is also held", () => {
    const onSwitch = vi.fn();
    renderHook(() => useGlobalKeyboardShortcuts(["wizard", "dashboard"], onSwitch));

    fireKey("1", { altKey: true, metaKey: true });
    expect(onSwitch).not.toHaveBeenCalled();
  });

  it("ignores out-of-range keys", () => {
    const onSwitch = vi.fn();
    renderHook(() => useGlobalKeyboardShortcuts(["wizard", "dashboard"], onSwitch));

    fireKey("3", { altKey: true }); // only 2 views
    fireKey("0", { altKey: true }); // 0 maps to index -1
    expect(onSwitch).not.toHaveBeenCalled();
  });

  it("ignores non-numeric keys", () => {
    const onSwitch = vi.fn();
    renderHook(() => useGlobalKeyboardShortcuts(["wizard"], onSwitch));

    fireKey("a", { altKey: true });
    expect(onSwitch).not.toHaveBeenCalled();
  });

  it("ignores shortcuts when focus is in an input", () => {
    const onSwitch = vi.fn();
    renderHook(() => useGlobalKeyboardShortcuts(["wizard", "dashboard"], onSwitch));

    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();

    input.dispatchEvent(new KeyboardEvent("keydown", { key: "1", altKey: true, bubbles: true }));
    expect(onSwitch).not.toHaveBeenCalled();

    document.body.removeChild(input);
  });

  it("ignores shortcuts when focus is in a textarea", () => {
    const onSwitch = vi.fn();
    renderHook(() => useGlobalKeyboardShortcuts(["wizard"], onSwitch));

    const textarea = document.createElement("textarea");
    document.body.appendChild(textarea);
    textarea.focus();

    textarea.dispatchEvent(new KeyboardEvent("keydown", { key: "1", altKey: true, bubbles: true }));
    expect(onSwitch).not.toHaveBeenCalled();

    document.body.removeChild(textarea);
  });

  it("removes listener on unmount", () => {
    const onSwitch = vi.fn();
    const { unmount } = renderHook(() => useGlobalKeyboardShortcuts(["wizard"], onSwitch));

    unmount();
    fireKey("1", { altKey: true });
    expect(onSwitch).not.toHaveBeenCalled();
  });
});
