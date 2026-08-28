import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTerminalTouch, touchToCell, findWordBoundaries } from "../hooks/useTerminalTouch";
import { createMockTerminal, type MockTerminal } from "../test-utils/mocks";
import type { Terminal } from "@xterm/xterm";
import {
  TOUCH_LONG_PRESS_MS,
  TOUCH_DOUBLE_TAP_MS,
  TOUCH_MOVE_THRESHOLD_PX,
} from "../consts/config";

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

/** Build a fake .xterm-screen element inside the container for coordinate mapping. */
function setupScreenElement(
  container: HTMLDivElement,
  width = 800,
  height = 480,
) {
  const screen = document.createElement("div");
  screen.classList.add("xterm-screen");
  container.appendChild(screen);

  // Mock getBoundingClientRect to return known dimensions
  screen.getBoundingClientRect = () => ({
    left: 0,
    top: 0,
    right: width,
    bottom: height,
    width,
    height,
    x: 0,
    y: 0,
    toJSON: () => ({}),
  });

  return screen;
}

/**
 * Create and dispatch a TouchEvent on a container.
 * jsdom doesn't fully support TouchEvent/TouchList constructors,
 * so we construct a plain Event and manually attach touch properties.
 */
function fireTouchEvent(
  container: HTMLElement,
  type: "touchstart" | "touchmove" | "touchend" | "touchcancel",
  opts: { clientX?: number; clientY?: number; identifier?: number } = {},
) {
  const identifier = opts.identifier ?? 0;
  const clientX = opts.clientX ?? 100;
  const clientY = opts.clientY ?? 100;

  const touchObj = {
    identifier,
    clientX,
    clientY,
    pageX: clientX,
    pageY: clientY,
    screenX: clientX,
    screenY: clientY,
    target: container,
    radiusX: 0,
    radiusY: 0,
    rotationAngle: 0,
    force: 0,
  } as unknown as Touch;

  const touchList =
    type === "touchend" || type === "touchcancel" ? [] : [touchObj];

  // Use a plain Event and attach touch properties manually since
  // jsdom's TouchEvent constructor doesn't populate TouchLists.
  const event = new Event(type, { bubbles: true, cancelable: true }) as Event & {
    touches: Touch[];
    changedTouches: Touch[];
    targetTouches: Touch[];
  };
  Object.defineProperty(event, "touches", { value: touchList });
  Object.defineProperty(event, "changedTouches", { value: [touchObj] });
  Object.defineProperty(event, "targetTouches", { value: touchList });

  act(() => {
    container.dispatchEvent(event);
  });
  return event;
}

function fireMultiTouchEvent(
  container: HTMLElement,
  type: "touchstart" | "touchmove" | "touchend",
  touches: Array<{ identifier: number; clientX: number; clientY: number }>,
) {
  const touchObjects = touches.map((touch) => ({
    ...touch,
    pageX: touch.clientX,
    pageY: touch.clientY,
    screenX: touch.clientX,
    screenY: touch.clientY,
    target: container,
    radiusX: 0,
    radiusY: 0,
    rotationAngle: 0,
    force: 0,
  } as unknown as Touch));
  const event = new Event(type, { bubbles: true, cancelable: true }) as Event & {
    touches: Touch[];
    changedTouches: Touch[];
    targetTouches: Touch[];
  };
  Object.defineProperty(event, "touches", { value: type === "touchend" ? [] : touchObjects });
  Object.defineProperty(event, "changedTouches", { value: touchObjects });
  Object.defineProperty(event, "targetTouches", { value: type === "touchend" ? [] : touchObjects });
  act(() => container.dispatchEvent(event));
  return event;
}

function fireTouchEventAtTarget(
  target: HTMLElement,
  container: HTMLElement,
  type: "touchstart" | "touchmove" | "touchend",
  opts: { clientX?: number; clientY?: number; identifier?: number } = {},
) {
  const identifier = opts.identifier ?? 0;
  const clientX = opts.clientX ?? 100;
  const clientY = opts.clientY ?? 100;
  const touch = {
    identifier,
    clientX,
    clientY,
    pageX: clientX,
    pageY: clientY,
    screenX: clientX,
    screenY: clientY,
    target,
    radiusX: 0,
    radiusY: 0,
    rotationAngle: 0,
    force: 0,
  } as unknown as Touch;
  const event = new Event(type, { bubbles: true, cancelable: true }) as Event & {
    touches: Touch[];
    changedTouches: Touch[];
    targetTouches: Touch[];
  };
  Object.defineProperty(event, "touches", { value: type === "touchend" ? [] : [touch] });
  Object.defineProperty(event, "changedTouches", { value: [touch] });
  Object.defineProperty(event, "targetTouches", { value: type === "touchend" ? [] : [touch] });
  act(() => target.dispatchEvent(event));
  return event;
}

function makeHookArgs(
  terminal: MockTerminal | null,
  container: HTMLDivElement,
  enabled = true,
) {
  return {
    terminal: terminal as unknown as Terminal | null,
    containerRef: { current: container },
    enabled,
  };
}

// ---------------------------------------------------------------------------
// Pure helper tests
// ---------------------------------------------------------------------------

describe("touchToCell", () => {
  it("maps client coords to terminal cell positions", () => {
    const container = document.createElement("div");
    setupScreenElement(container, 800, 480);
    const terminal = createMockTerminal(); // 80 cols, 24 rows

    // Cell size = 800/80 = 10px wide, 480/24 = 20px tall
    const result = touchToCell(
      terminal as unknown as Terminal,
      container,
      55, // col = floor(55/10) = 5
      45, // row = floor(45/20) = 2
    );

    expect(result).toEqual({ col: 5, row: 2 });
  });

  it("clamps to valid range", () => {
    const container = document.createElement("div");
    setupScreenElement(container, 800, 480);
    const terminal = createMockTerminal();

    const result = touchToCell(
      terminal as unknown as Terminal,
      container,
      -10,
      9999,
    );

    expect(result.col).toBe(0);
    expect(result.row).toBe(23); // terminal.rows - 1
  });

  it("returns (0,0) when .xterm-screen is missing", () => {
    const container = document.createElement("div"); // no .xterm-screen child
    const terminal = createMockTerminal();

    const result = touchToCell(
      terminal as unknown as Terminal,
      container,
      100,
      100,
    );

    expect(result).toEqual({ col: 0, row: 0 });
  });
});

describe("findWordBoundaries", () => {
  it("finds word around a middle character", () => {
    expect(findWordBoundaries("hello world", 2)).toEqual([0, 5]);
  });

  it("finds word at start", () => {
    expect(findWordBoundaries("hello world", 0)).toEqual([0, 5]);
  });

  it("finds word at end", () => {
    expect(findWordBoundaries("hello world", 8)).toEqual([6, 11]);
  });

  it("returns null for whitespace", () => {
    expect(findWordBoundaries("hello world", 5)).toBeNull();
  });

  it("returns null for out-of-range col", () => {
    expect(findWordBoundaries("hello", -1)).toBeNull();
    expect(findWordBoundaries("hello", 99)).toBeNull();
  });

  it("handles single character word", () => {
    expect(findWordBoundaries("a b c", 2)).toEqual([2, 3]);
  });
});

// ---------------------------------------------------------------------------
// Hook tests
// ---------------------------------------------------------------------------

describe("useTerminalTouch", () => {
  let terminal: MockTerminal;
  let container: HTMLDivElement;
  let screen: HTMLDivElement;

  beforeEach(() => {
    terminal = createMockTerminal();
    container = document.createElement("div");
    document.body.appendChild(container);
    screen = setupScreenElement(container) as unknown as HTMLDivElement;

    vi.useFakeTimers({ shouldAdvanceTime: true, toFake: ["setTimeout", "clearTimeout", "Date", "performance"] });

    // Mock the browser clipboard capability.
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
      writable: true,
      configurable: true,
    });

    // Mock navigator.vibrate
    Object.defineProperty(navigator, "vibrate", {
      value: vi.fn().mockReturnValue(true),
      writable: true,
      configurable: true,
    });

    // Mock requestAnimationFrame for momentum tests
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((cb) => {
      return setTimeout(() => cb(performance.now()), 16) as unknown as number;
    });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation((id) => {
      clearTimeout(id);
    });
  });

  afterEach(() => {
    document.body.removeChild(container);
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  // ---- Setup / Teardown ----

  it("sets touch-action: none on .xterm-screen on mount", () => {
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );
    expect(screen.style.touchAction).toBe("none");
  });

  it("removes touch-action on unmount", () => {
    const { unmount } = renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );
    unmount();
    expect(screen.style.touchAction).toBe("");
  });

  it("does nothing when terminal is null", () => {
    const { result } = renderHook(() =>
      useTerminalTouch(makeHookArgs(null, container)),
    );
    // Should not throw and return defaults
    expect(result.current.hasSelection).toBe(false);
    fireTouchEvent(container, "touchstart");
    fireTouchEvent(container, "touchend");
    // No errors
  });

  it("does nothing when enabled is false", () => {
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container, false)),
    );
    expect(screen.style.touchAction).not.toBe("none");
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 100 });
    fireTouchEvent(container, "touchend", { clientX: 100, clientY: 100 });
    expect(terminal.focus).not.toHaveBeenCalled();
  });

  // ---- Single tap ----

  it("single tap focuses the terminal", () => {
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 100 });
    vi.advanceTimersByTime(50); // quick tap
    fireTouchEvent(container, "touchend", { clientX: 100, clientY: 100 });

    // Wait for double-tap window to expire
    vi.advanceTimersByTime(TOUCH_DOUBLE_TAP_MS + 50);

    expect(terminal.focus).toHaveBeenCalled();
  });

  // ---- Vertical scroll ----

  it("vertical swipe down scrolls terminal buffer down", () => {
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    // Cell height = 480/24 = 20px
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 200 });
    vi.advanceTimersByTime(10);
    // First move beyond threshold → transitions pending to scrolling (sets lastY=160)
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 160 });
    vi.advanceTimersByTime(16);
    // Second move: lastY(160) - 120 = 40px up → 2 lines down
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 120 });

    expect(terminal.scrollLines).toHaveBeenCalledWith(2);
  });

  it("vertical swipe up scrolls terminal buffer up", () => {
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 100 });
    vi.advanceTimersByTime(10);
    // Move finger down beyond threshold → transitions to scrolling
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 140 });
    vi.advanceTimersByTime(16);
    // Second move generates delta: lastY(140) - 180 = -40 → -2 lines
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 180 });

    expect(terminal.scrollLines).toHaveBeenCalledWith(-2);
  });

  it("small movement below threshold does not trigger scroll", () => {
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 100 });
    vi.advanceTimersByTime(10);
    // Move by less than TOUCH_MOVE_THRESHOLD_PX
    fireTouchEvent(container, "touchmove", {
      clientX: 100,
      clientY: 100 + TOUCH_MOVE_THRESHOLD_PX - 1,
    });

    expect(terminal.scrollLines).not.toHaveBeenCalled();
  });

  it("momentum scroll continues after touchend", () => {
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    // Fast swipe
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 300 });
    vi.advanceTimersByTime(10);
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 200 });
    vi.advanceTimersByTime(16);
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 100 });

    terminal.scrollLines.mockClear();
    fireTouchEvent(container, "touchend", { clientX: 100, clientY: 100 });

    // Advance a few RAF frames
    vi.advanceTimersByTime(16);
    vi.advanceTimersByTime(16);
    vi.advanceTimersByTime(16);

    // Should have continued scrolling via momentum
    expect(terminal.scrollLines.mock.calls.length).toBeGreaterThan(0);
  });

  it("previews a pinch font-size change and commits it once on release", () => {
    const onFontSizeCommit = vi.fn();
    const onFontSizePreview = vi.fn();
    (terminal as unknown as { options: { fontSize: number } }).options = { fontSize: 14 };

    renderHook(() => useTerminalTouch({
      ...makeHookArgs(terminal, container),
      fontSize: 14,
      onFontSizeCommit,
      onFontSizePreview,
    }));

    fireMultiTouchEvent(container, "touchstart", [
      { identifier: 1, clientX: 100, clientY: 100 },
    ]);
    fireMultiTouchEvent(container, "touchstart", [
      { identifier: 1, clientX: 100, clientY: 100 },
      { identifier: 2, clientX: 200, clientY: 100 },
    ]);
    fireMultiTouchEvent(container, "touchmove", [
      { identifier: 1, clientX: 80, clientY: 100 },
      { identifier: 2, clientX: 220, clientY: 100 },
    ]);

    expect((terminal as unknown as { options: { fontSize: number } }).options.fontSize).toBeGreaterThan(14);
    expect(onFontSizePreview.mock.calls.at(-1)?.[0]).toBeGreaterThan(14);
    expect(onFontSizeCommit).not.toHaveBeenCalled();

    fireMultiTouchEvent(container, "touchend", []);
    expect(onFontSizeCommit).toHaveBeenCalledTimes(1);
    expect(onFontSizeCommit.mock.calls[0]?.[0]).toBeGreaterThan(14);
    expect(onFontSizePreview).toHaveBeenLastCalledWith(null);
  });

  it("new touchstart cancels momentum", () => {
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    // Fast swipe to start momentum
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 300 });
    vi.advanceTimersByTime(10);
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 100 });
    fireTouchEvent(container, "touchend", { clientX: 100, clientY: 100 });

    // Let one momentum frame happen
    vi.advanceTimersByTime(16);
    terminal.scrollLines.mockClear();

    // New touch should cancel momentum
    fireTouchEvent(container, "touchstart", { clientX: 200, clientY: 200 });

    // Advance more frames — no more momentum calls
    vi.advanceTimersByTime(100);
    expect(terminal.scrollLines).not.toHaveBeenCalled();
  });

  // ---- Long-press selection ----

  it("long press enters selection mode", () => {
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    fireTouchEvent(container, "touchstart", { clientX: 50, clientY: 40 });

    // Wait for long-press threshold
    act(() => {
      vi.advanceTimersByTime(TOUCH_LONG_PRESS_MS + 10);
    });

    expect(terminal.select).toHaveBeenCalled();
    expect(navigator.vibrate).toHaveBeenCalledWith(30);
  });

  it("long press then drag extends selection", () => {
    const { result } = renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    act(() => {
      fireTouchEvent(container, "touchstart", { clientX: 50, clientY: 40 });
      vi.advanceTimersByTime(TOUCH_LONG_PRESS_MS + 10);
    });

    // Long-press should have already called select once
    expect(terminal.select).toHaveBeenCalled();
    terminal.select.mockClear();

    // Drag to the right — should extend selection
    act(() => {
      fireTouchEvent(container, "touchmove", { clientX: 150, clientY: 40 });
    });

    expect(terminal.select).toHaveBeenCalled();
    expect(result.current.hasSelection).toBe(true);
  });

  it("clears the temporary selection on touchend after long-press without drag", () => {
    const { result } = renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    fireTouchEvent(container, "touchstart", { clientX: 50, clientY: 40 });
    act(() => {
      vi.advanceTimersByTime(TOUCH_LONG_PRESS_MS + 10);
    });

    fireTouchEvent(container, "touchend", { clientX: 50, clientY: 40 });

    // A long-press without drag opens the context menu path, so the
    // temporary single-character selection is cleared on release.
    expect(result.current.hasSelection).toBe(false);
  });

  // ---- Double-tap word selection ----

  it("double tap selects word under cursor", () => {
    terminal.buffer.active.getLine = vi.fn().mockReturnValue({
      translateToString: vi.fn().mockReturnValue("hello world test"),
    });

    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    // Cell size: 800/80 = 10px wide, 480/24 = 20px tall
    // Touch at clientX=75 → col=7 → within "world" (chars 6-10)
    const x = 75;
    const y = 20;

    // First tap
    fireTouchEvent(container, "touchstart", { clientX: x, clientY: y });
    vi.advanceTimersByTime(50);
    fireTouchEvent(container, "touchend", { clientX: x, clientY: y });

    // Second tap within double-tap window
    vi.advanceTimersByTime(100);
    fireTouchEvent(container, "touchstart", { clientX: x, clientY: y });

    // Should have selected "world" → select(6, row, 5)
    expect(terminal.select).toHaveBeenCalledWith(6, 1, 5);
  });

  it("double tap on whitespace does not select", () => {
    terminal.buffer.active.getLine = vi.fn().mockReturnValue({
      translateToString: vi.fn().mockReturnValue("hello   world"),
    });

    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    // clientX=55 → col=5 → space character
    const x = 55;
    const y = 20;

    fireTouchEvent(container, "touchstart", { clientX: x, clientY: y });
    vi.advanceTimersByTime(50);
    fireTouchEvent(container, "touchend", { clientX: x, clientY: y });

    vi.advanceTimersByTime(100);
    fireTouchEvent(container, "touchstart", { clientX: x, clientY: y });

    expect(terminal.select).not.toHaveBeenCalled();
  });

  it("taps too far apart in time are treated as single taps", () => {
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 100 });
    vi.advanceTimersByTime(50);
    fireTouchEvent(container, "touchend", { clientX: 100, clientY: 100 });

    // Wait longer than TOUCH_DOUBLE_TAP_MS
    vi.advanceTimersByTime(TOUCH_DOUBLE_TAP_MS + 100);

    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 100 });
    vi.advanceTimersByTime(50);
    fireTouchEvent(container, "touchend", { clientX: 100, clientY: 100 });

    // No word selection, just focus calls
    expect(terminal.select).not.toHaveBeenCalled();
    expect(terminal.focus).toHaveBeenCalled();
  });

  // ---- Copy / Clear selection ----

  it("copySelection writes to clipboard", async () => {
    terminal.getSelection = vi.fn().mockReturnValue("selected text");

    const { result } = renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    let success = false;
    await act(async () => {
      success = await result.current.copySelection();
    });

    expect(success).toBe(true);
    expect(navigator["clipboard"].writeText).toHaveBeenCalledWith("selected text");
  });

  it("copySelection returns false when no selection", async () => {
    terminal.getSelection = vi.fn().mockReturnValue("");

    const { result } = renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    let success = true;
    await act(async () => {
      success = await result.current.copySelection();
    });

    expect(success).toBe(false);
  });

  it("copySelection returns false without a terminal and tolerates a missing screen", async () => {
    const withoutTerminal = renderHook(() =>
      useTerminalTouch(makeHookArgs(null, container)),
    );
    await expect(withoutTerminal.result.current.copySelection()).resolves.toBe(false);
    withoutTerminal.unmount();

    const { result } = renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );
    screen.remove();
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 200 });
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 180 });
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 160 });
    expect(terminal.scrollLines).not.toHaveBeenCalled();
    expect(result.current.hasSelection).toBe(false);
  });

  it("clearSelection calls terminal.clearSelection and resets state", () => {
    const { result } = renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    act(() => {
      result.current.clearSelection();
    });

    expect(terminal.clearSelection).toHaveBeenCalled();
    expect(result.current.hasSelection).toBe(false);
  });

  // ---- Edge cases ----

  it("touchcancel resets gesture to idle", () => {
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 100 });
    fireTouchEvent(container, "touchcancel");

    // Subsequent tap should still work normally
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 100 });
    vi.advanceTimersByTime(50);
    fireTouchEvent(container, "touchend", { clientX: 100, clientY: 100 });

    vi.advanceTimersByTime(TOUCH_DOUBLE_TAP_MS + 50);
    expect(terminal.focus).toHaveBeenCalled();
  });

  it("ignores moves and ends for touches that are not being tracked", () => {
    renderHook(() => useTerminalTouch(makeHookArgs(terminal, container)));

    fireTouchEvent(container, "touchmove", { identifier: 99 });
    fireTouchEvent(container, "touchstart", { identifier: 1, clientY: 100 });
    fireTouchEvent(container, "touchmove", { identifier: 0, clientY: 50 });
    fireTouchEvent(container, "touchend", { identifier: 0, clientY: 50 });
    expect(terminal.focus).not.toHaveBeenCalled();
  });

  it("does not intercept touches that begin inside the context menu or backdrop", () => {
    const backdrop = document.createElement("div");
    backdrop.dataset.testid = "ctx-backdrop";
    container.appendChild(backdrop);
    renderHook(() => useTerminalTouch(makeHookArgs(terminal, container)));

    fireTouchEventAtTarget(backdrop, container, "touchstart", { clientX: 50, clientY: 50 });
    fireTouchEventAtTarget(backdrop, container, "touchend", { clientX: 50, clientY: 50 });
    expect(terminal.focus).not.toHaveBeenCalled();
  });

  it("handles invalid pinch tracking and zero-distance gestures", () => {
    const onFontSizeCommit = vi.fn();
    (terminal as unknown as { options: { fontSize: number } }).options = { fontSize: 14 };
    renderHook(() => useTerminalTouch({
      ...makeHookArgs(terminal, container),
      fontSize: 14,
      onFontSizeCommit,
    }));

    fireMultiTouchEvent(container, "touchstart", [
      { identifier: 1, clientX: 100, clientY: 100 },
      { identifier: 2, clientX: 100, clientY: 100 },
    ]);
    fireMultiTouchEvent(container, "touchmove", [
      { identifier: 1, clientX: 90, clientY: 100 },
      { identifier: 3, clientX: 110, clientY: 100 },
    ]);
    fireMultiTouchEvent(container, "touchend", []);

    expect(onFontSizeCommit).not.toHaveBeenCalled();
    expect((terminal as unknown as { options: { fontSize: number } }).options.fontSize).toBe(14);
  });

  it("ignores a multi-touch start when pinch handling is unavailable", () => {
    renderHook(() => useTerminalTouch(makeHookArgs(terminal, container)));
    fireMultiTouchEvent(container, "touchstart", [
      { identifier: 1, clientX: 100, clientY: 100 },
      { identifier: 2, clientX: 200, clientY: 100 },
    ]);
    expect(terminal.focus).not.toHaveBeenCalled();
  });

  it("ignores a touchstart whose touch list contains no touch object", () => {
    renderHook(() => useTerminalTouch(makeHookArgs(terminal, container)));
    const event = new Event("touchstart", { bubbles: true, cancelable: true }) as Event & {
      touches: Touch[];
      changedTouches: Touch[];
      targetTouches: Touch[];
    };
    Object.defineProperty(event, "touches", { value: [undefined] });
    Object.defineProperty(event, "changedTouches", { value: [] });
    Object.defineProperty(event, "targetTouches", { value: [] });
    act(() => container.dispatchEvent(event));
    expect(terminal.focus).not.toHaveBeenCalled();
  });

  it("cancels the already-cancelled long-press timer when resetting scrolling", () => {
    const { unmount } = renderHook(() => useTerminalTouch(makeHookArgs(terminal, container)));
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 200 });
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 180 });
    vi.advanceTimersByTime(16);
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 130 });
    // Unmount invokes resetGesture while the scrolling state's timer is
    // already null, covering the cleanup path after clearTimeout.
    unmount();
    expect(terminal.scrollLines).toHaveBeenCalled();
  });

  it("routes tracked scrolling through the injected control sender in mouse mode", () => {
    const sendControl = vi.fn().mockReturnValue(true);
    (terminal as unknown as { modes: { mouseTrackingMode: string } }).modes = { mouseTrackingMode: "sgr" };
    const { unmount: unmountWithSender } = renderHook(() => useTerminalTouch({
      ...makeHookArgs(terminal, container),
      sendControl,
    }));
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 200 });
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 160 });
    vi.advanceTimersByTime(16);
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 120 });
    vi.advanceTimersByTime(16);
    expect(sendControl).toHaveBeenCalled();
    unmountWithSender();

    const noSenderTerminal = createMockTerminal();
    (noSenderTerminal as unknown as { modes: { mouseTrackingMode: string } }).modes = { mouseTrackingMode: "sgr" };
    const { unmount } = renderHook(() => useTerminalTouch(makeHookArgs(noSenderTerminal, container)));
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 200 });
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 160 });
    vi.advanceTimersByTime(16);
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 120 });
    vi.advanceTimersByTime(16);
    unmount();
  });

  it("scrolls a slow drag the same distance as a fast one", () => {
    // The regression: `lastY` advanced to the finger on every move, but a
    // sample whose delta rounded to zero rows was discarded. A drag emitting
    // 3px per event at a 20px row height therefore discarded 100% of its
    // travel — the pane did not scroll at all, however far the finger went.
    const scrollBy = vi.fn();
    const { unmount } = renderHook(() => useTerminalTouch({
      ...makeHookArgs(terminal, container),
      scrollBy,
    }));

    // 800px tall / 24 rows = 33.33px per row. Drag 300px in 3px steps.
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 500 });
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 490 });
    for (let y = 487; y >= 200; y -= 3) {
      fireTouchEvent(container, "touchmove", { clientX: 100, clientY: y });
      vi.advanceTimersByTime(16);
    }

    const scrolled = scrollBy.mock.calls.reduce((sum, call) => sum + (call[0] as number), 0);
    // 300px of travel over a 33.33px row is 9 rows; allow one row of rounding.
    expect(scrolled).toBeGreaterThanOrEqual(8);
    unmount();
  });

  it("keeps scrolling when a drag slows down without lifting", () => {
    // Crossing back below the per-sample threshold used to stop the scroll
    // dead mid-gesture, because each small sample was discarded on its own.
    const scrollBy = vi.fn();
    const { unmount } = renderHook(() => useTerminalTouch({
      ...makeHookArgs(terminal, container),
      scrollBy,
    }));

    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 500 });
    // The first move only crosses the tap/drag threshold and enters the
    // scrolling state; scrolling starts from the move after it.
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 490 });
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 440 }); // fast
    vi.advanceTimersByTime(16);
    const afterFast = scrollBy.mock.calls.length;
    expect(afterFast).toBeGreaterThan(0);

    for (let y = 437; y >= 380; y -= 3) { // slow, same gesture
      fireTouchEvent(container, "touchmove", { clientX: 100, clientY: y });
      vi.advanceTimersByTime(16);
    }
    expect(scrollBy.mock.calls.length).toBeGreaterThan(afterFast);
    unmount();
  });

  it("asks the server to scroll a pane that has neither scrollback nor mouse tracking", () => {
    // Every tmux-backed pane sits in the alternate screen buffer, so
    // scrollLines() cannot move it. With no mouse tracking either, the only
    // history is the backend's, and a touch drag that cannot reach it is a
    // drag that silently does nothing.
    const sendScroll = vi.fn().mockReturnValue(true);
    const altTerminal = createMockTerminal();
    const alternate = {};
    (altTerminal as unknown as { modes: { mouseTrackingMode: string } }).modes = { mouseTrackingMode: "none" };
    (altTerminal as unknown as { buffer: unknown }).buffer = { active: alternate, normal: {}, alternate };

    const { unmount } = renderHook(() => useTerminalTouch({
      ...makeHookArgs(altTerminal, container),
      sendScroll,
    }));
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 200 });
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 160 });
    vi.advanceTimersByTime(16);
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 120 });
    vi.advanceTimersByTime(16);

    expect(altTerminal.scrollLines).not.toHaveBeenCalled();
    expect(sendScroll).toHaveBeenCalled();
    unmount();
  });

  it("stops momentum cleanly when the injected clock advances past the decay threshold", () => {
    let clock = 0;
    renderHook(() => useTerminalTouch({
      ...makeHookArgs(terminal, container),
      now: () => clock,
    }));
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 300 });
    clock = 1;
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 100 });
    vi.advanceTimersByTime(16);
    clock = 2;
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 0 });
    fireTouchEvent(container, "touchend", { clientX: 100, clientY: 0 });
	clock = 10 * 1000;
    vi.advanceTimersByTime(16);
    expect(terminal.scrollLines).toHaveBeenCalled();
  });

  it("ignores a double tap when the buffer line has disappeared", () => {
    terminal.buffer.active.getLine = vi.fn().mockReturnValue(undefined);
    renderHook(() => useTerminalTouch(makeHookArgs(terminal, container)));
    fireTouchEvent(container, "touchstart", { clientX: 75, clientY: 20 });
    vi.advanceTimersByTime(50);
    fireTouchEvent(container, "touchend", { clientX: 75, clientY: 20 });
    vi.advanceTimersByTime(100);
    fireTouchEvent(container, "touchstart", { clientX: 75, clientY: 20 });
    expect(terminal.select).not.toHaveBeenCalled();
  });

  it("ignores a stale long-press callback after the gesture has returned idle", () => {
    const clearTimeoutSpy = vi.spyOn(globalThis, "clearTimeout").mockImplementation(() => undefined);
    renderHook(() => useTerminalTouch(makeHookArgs(terminal, container)));
    fireTouchEvent(container, "touchstart", { clientX: 50, clientY: 40 });
    fireTouchEvent(container, "touchend", { clientX: 50, clientY: 40 });
    act(() => vi.advanceTimersByTime(TOUCH_LONG_PRESS_MS + 10));
    clearTimeoutSpy.mockRestore();
    expect(terminal.select).not.toHaveBeenCalled();
  });

  it("covers backward selection, zero-time samples, and textarea focus intent", () => {
    const textarea = { inputMode: "none" };
    (terminal as unknown as { textarea: typeof textarea }).textarea = textarea;
    renderHook(() => useTerminalTouch(makeHookArgs(terminal, container)));

    fireTouchEvent(container, "touchstart", { clientX: 50, clientY: 40 });
    act(() => vi.advanceTimersByTime(TOUCH_LONG_PRESS_MS + 10));
    fireTouchEvent(container, "touchmove", { clientX: 0, clientY: 40 });
    expect(terminal.select).toHaveBeenLastCalledWith(0, 2, 6);
    fireTouchEvent(container, "touchend", { clientX: 0, clientY: 40 });

    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 200 });
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 180 });
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 160 });
    expect(textarea.inputMode).toBe("none");

    fireTouchEvent(container, "touchend", { clientX: 100, clientY: 160 });
    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 100 });
    fireTouchEvent(container, "touchend", { clientX: 100, clientY: 100 });
    expect(textarea.inputMode).toBe("");
  });

  it("opens the context menu on a double tap when a callback is provided", () => {
    const onContextMenu = vi.fn();
    terminal.buffer.active.getLine = vi.fn().mockReturnValue({
      translateToString: vi.fn().mockReturnValue("hello world"),
    });
    renderHook(() => useTerminalTouch({
      ...makeHookArgs(terminal, container),
      onContextMenu,
    }));

    fireTouchEvent(container, "touchstart", { clientX: 75, clientY: 20 });
    vi.advanceTimersByTime(50);
    fireTouchEvent(container, "touchend", { clientX: 75, clientY: 20 });
    vi.advanceTimersByTime(100);
    fireTouchEvent(container, "touchstart", { clientX: 75, clientY: 20 });

    expect(onContextMenu).toHaveBeenCalledWith(75, 20);
  });

  it("movement during pending scrolls, then long-press timer reclaims for selection", () => {
    // On mobile, natural hand tremor during a hold easily exceeds the 8px
    // movement threshold. The long-press timer is intentionally kept alive
    // during scrolling so that after 500ms the gesture transitions to
    // "selecting" — if the user genuinely wanted to scroll, they would
    // have already lifted their finger by then.
    renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 200 });
    vi.advanceTimersByTime(100); // before long-press threshold

    // Move beyond TOUCH_MOVE_THRESHOLD_PX (8px) to transition to scrolling,
    // but keep total cumulative distance at exactly TOUCH_SCROLL_CANCEL_PX (30px)
    // so the long-press timer is NOT cancelled (cancel check is strict >).
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 190 });

    // Second move: 20px delta gives exactly 1 line (cellH=20px, round(20/20)=1)
    // Cumulative = 10 + 20 = 30px — not greater than 30, so timer survives.
    vi.advanceTimersByTime(16);
    fireTouchEvent(container, "touchmove", { clientX: 100, clientY: 170 });

    // Scrolling should have occurred while in scrolling state
    expect(terminal.scrollLines).toHaveBeenCalled();

    // Now wait past long-press — SHOULD enter selection mode because the
    // timer was kept alive across the pending→scrolling transition.
    act(() => {
      vi.advanceTimersByTime(TOUCH_LONG_PRESS_MS);
    });

    expect(terminal.select).toHaveBeenCalled();
  });

  // ---- Context menu (long-press / right-click) ----

  it("long press without drag fires onContextMenu", () => {
    const onContextMenu = vi.fn();

    renderHook(() =>
      useTerminalTouch({
        ...makeHookArgs(terminal, container),
        onContextMenu,
      }),
    );

    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 100 });

    // Wait for long-press threshold to enter selecting mode
    act(() => {
      vi.advanceTimersByTime(TOUCH_LONG_PRESS_MS + 10);
    });

    // Touch end without drag → should fire context menu
    fireTouchEvent(container, "touchend", { clientX: 100, clientY: 100 });

    expect(onContextMenu).toHaveBeenCalledWith(100, 100);
    // Selection should be cleared (single char selection removed)
    expect(terminal.clearSelection).toHaveBeenCalled();
  });

  it("long press with drag fires onContextMenu at release point", () => {
    const onContextMenu = vi.fn();

    renderHook(() =>
      useTerminalTouch({
        ...makeHookArgs(terminal, container),
        onContextMenu,
      }),
    );

    fireTouchEvent(container, "touchstart", { clientX: 100, clientY: 100 });

    act(() => {
      vi.advanceTimersByTime(TOUCH_LONG_PRESS_MS + 10);
    });

    // Drag to extend selection
    fireTouchEvent(container, "touchmove", { clientX: 200, clientY: 100 });
    fireTouchEvent(container, "touchend", { clientX: 200, clientY: 100 });

    // Context menu should open at the release point so the user can
    // Copy/Speak the selected text.
    expect(onContextMenu).toHaveBeenCalledWith(200, 100);
  });

  it("desktop right-click fires onContextMenu", () => {
    const onContextMenu = vi.fn();

    renderHook(() =>
      useTerminalTouch({
        ...makeHookArgs(terminal, container),
        onContextMenu,
      }),
    );

    const event = new MouseEvent("contextmenu", {
      bubbles: true,
      cancelable: true,
      clientX: 150,
      clientY: 250,
    });

    container.dispatchEvent(event);

    expect(onContextMenu).toHaveBeenCalledWith(150, 250);
    expect(event.defaultPrevented).toBe(true);
  });

  it("cleanup removes event listeners on unmount", () => {
    const removeSpy = vi.spyOn(container, "removeEventListener");

    const { unmount } = renderHook(() =>
      useTerminalTouch(makeHookArgs(terminal, container)),
    );

    unmount();

    const removedTypes = removeSpy.mock.calls.map((c) => c[0]);
    expect(removedTypes).toContain("touchstart");
    expect(removedTypes).toContain("touchmove");
    expect(removedTypes).toContain("touchend");
    expect(removedTypes).toContain("touchcancel");
  });
});
