import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { useMobileBackspaceRepeat } from "../hooks/useMobileBackspaceRepeat";

// ---------------------------------------------------------------------------
// Constants (mirrored from the hook for assertion clarity)
// ---------------------------------------------------------------------------

const DEL = "\x7f";
const PADDING = "\u200B".repeat(32);

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

/** Minimal mock of xterm's Terminal with a real textarea element. */
function createTerminalWithTextarea() {
  const textarea = document.createElement("textarea");
  const inputFn = vi.fn();

  const terminal = {
    textarea,
    input: inputFn,
  };

  return { terminal, textarea, inputFn };
}

/** Simulate a mobile touch device by setting navigator.maxTouchPoints. */
function simulateTouchDevice() {
  Object.defineProperty(navigator, "maxTouchPoints", {
    value: 1,
    configurable: true,
  });
}

/** Simulate a non-touch (desktop) device. */
function simulateDesktopDevice() {
  Object.defineProperty(navigator, "maxTouchPoints", {
    value: 0,
    configurable: true,
  });
  // Ensure ontouchstart is not on window
  if ("ontouchstart" in window) {
    delete (window as unknown as Record<string, unknown>).ontouchstart;
  }
}

/**
 * Fire a beforeinput event on the textarea.
 * jsdom doesn't support InputEvent with inputType natively, so we
 * construct it manually.
 */
function fireBeforeInput(
  textarea: HTMLTextAreaElement,
  inputType: string,
): { event: InputEvent; defaultPrevented: boolean } {
  const event = new InputEvent("beforeinput", {
    inputType,
    bubbles: true,
    cancelable: true,
  } as InputEventInit);

  textarea.dispatchEvent(event);
  return { event, defaultPrevented: event.defaultPrevented };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("useMobileBackspaceRepeat", () => {
  let originalMaxTouchPoints: number;

  beforeEach(() => {
    originalMaxTouchPoints = navigator.maxTouchPoints;
  });

  afterEach(() => {
    Object.defineProperty(navigator, "maxTouchPoints", {
      value: originalMaxTouchPoints,
      configurable: true,
    });
  });

  // ---- Core behavior ----

  it("seeds the textarea with padding on mount (touch device)", () => {
    simulateTouchDevice();
    const { terminal, textarea } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    expect(textarea.value).toBe(PADDING);
  });

  it("sends DEL on deleteContentBackward beforeinput event", () => {
    simulateTouchDevice();
    const { terminal, textarea, inputFn } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    fireBeforeInput(textarea, "deleteContentBackward");

    expect(inputFn).toHaveBeenCalledTimes(1);
    expect(inputFn).toHaveBeenCalledWith(DEL, true);
  });

  it("prevents default on deleteContentBackward to avoid confusing xterm", () => {
    simulateTouchDevice();
    const { terminal, textarea } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    const { defaultPrevented } = fireBeforeInput(textarea, "deleteContentBackward");
    expect(defaultPrevented).toBe(true);
  });

  it("replenishes padding after each backspace so repeat events keep firing", () => {
    simulateTouchDevice();
    const { terminal, textarea } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    // Simulate the browser consuming some padding
    textarea.value = "x";
    fireBeforeInput(textarea, "deleteContentBackward");

    expect(textarea.value).toBe(PADDING);
  });

  it("handles multiple rapid backspace events (simulating key repeat)", () => {
    simulateTouchDevice();
    const { terminal, textarea, inputFn } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    // Simulate holding backspace — 5 rapid events
    for (let i = 0; i < 5; i++) {
      fireBeforeInput(textarea, "deleteContentBackward");
    }

    expect(inputFn).toHaveBeenCalledTimes(5);
    // Padding should still be intact after all events
    expect(textarea.value).toBe(PADDING);
  });

  // ---- Selectivity: only intercepts deleteContentBackward ----

  it("does NOT intercept insertText events (normal typing)", () => {
    simulateTouchDevice();
    const { terminal, textarea, inputFn } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    const { defaultPrevented } = fireBeforeInput(textarea, "insertText");

    expect(defaultPrevented).toBe(false);
    expect(inputFn).not.toHaveBeenCalled();
  });

  it("does NOT intercept insertCompositionText events (IME input)", () => {
    simulateTouchDevice();
    const { terminal, textarea, inputFn } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    const { defaultPrevented } = fireBeforeInput(textarea, "insertCompositionText");

    expect(defaultPrevented).toBe(false);
    expect(inputFn).not.toHaveBeenCalled();
  });

  it("does NOT intercept deleteContentForward events (forward-delete)", () => {
    simulateTouchDevice();
    const { terminal, textarea, inputFn } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    const { defaultPrevented } = fireBeforeInput(textarea, "deleteContentForward");

    expect(defaultPrevented).toBe(false);
    expect(inputFn).not.toHaveBeenCalled();
  });

  // ---- Desktop: hook is inactive ----

  it("does NOT activate on desktop (non-touch) devices", () => {
    simulateDesktopDevice();
    const { terminal, textarea, inputFn } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    // Textarea should NOT be padded
    expect(textarea.value).toBe("");

    // Firing a backspace event should not be intercepted
    fireBeforeInput(textarea, "deleteContentBackward");
    expect(inputFn).not.toHaveBeenCalled();
  });

  // ---- Null terminal ----

  it("does nothing when terminal is null", () => {
    simulateTouchDevice();

    // Should not throw
    const { unmount } = renderHook(() => useMobileBackspaceRepeat(null));
    unmount();
  });

  // ---- Padding replenishment after xterm clears textarea ----

  it("replenishes padding after an input event clears the textarea (simulating xterm reset)", async () => {
    simulateTouchDevice();
    const { terminal, textarea } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    // Simulate xterm clearing the textarea after processing a typed character
    textarea.value = "";
    textarea.dispatchEvent(new Event("input", { bubbles: true }));

    // ensurePadding uses requestAnimationFrame — flush it
    await new Promise((resolve) => requestAnimationFrame(resolve));

    expect(textarea.value).toBe(PADDING);
  });

  it("replenishes padding on focus when textarea was cleared", async () => {
    simulateTouchDevice();
    const { terminal, textarea } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    // Simulate xterm clearing textarea, then user tapping terminal
    textarea.value = "";
    textarea.dispatchEvent(new FocusEvent("focus", { bubbles: true }));

    await new Promise((resolve) => requestAnimationFrame(resolve));

    expect(textarea.value).toBe(PADDING);
  });

  it("does NOT overwrite textarea when it already has content", async () => {
    simulateTouchDevice();
    const { terminal, textarea } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    // Simulate xterm writing its own content (e.g. during IME composition)
    textarea.value = "composing";
    textarea.dispatchEvent(new Event("input", { bubbles: true }));

    await new Promise((resolve) => requestAnimationFrame(resolve));

    // Should not overwrite non-empty content
    expect(textarea.value).toBe("composing");
  });

  it("survives the full cycle: type clears padding → replenished → backspace repeat works", async () => {
    simulateTouchDevice();
    const { terminal, textarea, inputFn } = createTerminalWithTextarea();

    renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    // 1. Simulate xterm clearing textarea after user typed a character
    textarea.value = "";
    textarea.dispatchEvent(new Event("input", { bubbles: true }));

    // 2. Wait for padding replenishment
    await new Promise((resolve) => requestAnimationFrame(resolve));
    expect(textarea.value).toBe(PADDING);

    // 3. Now hold backspace — repeat events should work
    for (let i = 0; i < 3; i++) {
      fireBeforeInput(textarea, "deleteContentBackward");
    }

    expect(inputFn).toHaveBeenCalledTimes(3);
    expect(textarea.value).toBe(PADDING);
  });

  // ---- Cleanup ----

  it("removes the event listener on unmount", () => {
    simulateTouchDevice();
    const { terminal, textarea, inputFn } = createTerminalWithTextarea();

    const { unmount } = renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    unmount();

    // After unmount, events should no longer be intercepted
    fireBeforeInput(textarea, "deleteContentBackward");
    expect(inputFn).not.toHaveBeenCalled();
  });

  it("removes input and focus listeners on unmount", async () => {
    simulateTouchDevice();
    const { terminal, textarea } = createTerminalWithTextarea();

    const { unmount } = renderHook(() =>
      useMobileBackspaceRepeat(terminal as unknown as import("@xterm/xterm").Terminal),
    );

    unmount();

    // Simulate xterm clearing textarea after unmount
    textarea.value = "";
    textarea.dispatchEvent(new Event("input", { bubbles: true }));
    textarea.dispatchEvent(new FocusEvent("focus", { bubbles: true }));

    await new Promise((resolve) => requestAnimationFrame(resolve));

    // Padding should NOT be replenished after unmount
    expect(textarea.value).toBe("");
  });
});
