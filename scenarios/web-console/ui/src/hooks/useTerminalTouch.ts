import { useCallback, useEffect, useRef, useState } from "react";
import type { Terminal } from "@xterm/xterm";
import {
  TOUCH_MOVE_THRESHOLD_PX,
  TOUCH_LONG_PRESS_MS,
  TOUCH_TAP_MAX_MS,
  TOUCH_DOUBLE_TAP_MS,
  TOUCH_SCROLL_DECEL,
  TOUCH_SCROLL_MIN_VELOCITY,
} from "../consts/config";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface UseTerminalTouchOptions {
  terminal: Terminal | null;
  containerRef: React.RefObject<HTMLDivElement | null>;
  enabled?: boolean;
}

export interface UseTerminalTouchReturn {
  /** Whether a text selection is currently active (drives copy button). */
  hasSelection: boolean;
  /** Copy current selection to clipboard. Returns true on success. */
  copySelection: () => Promise<boolean>;
  /** Clear the active selection. */
  clearSelection: () => void;
}

/** Internal gesture states tracked via ref (no re-renders). */
type GestureState =
  | { type: "idle" }
  | {
      type: "pending";
      touchId: number;
      startX: number;
      startY: number;
      startTime: number;
      longPressTimer: ReturnType<typeof setTimeout>;
    }
  | {
      type: "scrolling";
      touchId: number;
      lastY: number;
      lastTime: number;
      velocity: number;
    }
  | {
      type: "selecting";
      touchId: number;
      anchorCol: number;
      anchorRow: number;
    };

/** Pending double-tap detection tracked separately. */
interface DoubleTapState {
  time: number;
  x: number;
  y: number;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Convert a touch point (client coords) to terminal cell coordinates
 * by dividing the .xterm-screen element dimensions by cols/rows.
 */
export function touchToCell(
  terminal: Terminal,
  container: HTMLElement,
  clientX: number,
  clientY: number,
): { col: number; row: number } {
  const screenEl = container.querySelector<HTMLElement>(".xterm-screen");
  if (!screenEl) return { col: 0, row: 0 };

  const rect = screenEl.getBoundingClientRect();
  const cellWidth = rect.width / terminal.cols;
  const cellHeight = rect.height / terminal.rows;

  const col = Math.floor((clientX - rect.left) / cellWidth);
  const row = Math.floor((clientY - rect.top) / cellHeight);

  return {
    col: Math.max(0, Math.min(col, terminal.cols - 1)),
    row: Math.max(0, Math.min(row, terminal.rows - 1)),
  };
}

/**
 * Find word boundaries around `col` in `text`.
 * Returns [startCol, endCol) or null if col is on whitespace.
 */
export function findWordBoundaries(
  text: string,
  col: number,
): [number, number] | null {
  if (col < 0 || col >= text.length) return null;
  const ch = text.charAt(col);
  if (/\s/.test(ch)) return null;

  const wordRe = /\S/;
  let start = col;
  while (start > 0 && wordRe.test(text.charAt(start - 1))) start--;
  let end = col;
  while (end < text.length - 1 && wordRe.test(text.charAt(end + 1))) end++;

  return [start, end + 1];
}

/**
 * Derive cell height from the .xterm-screen element and terminal rows.
 * Returns 0 if the element is missing.
 */
function getCellHeight(terminal: Terminal, container: HTMLElement): number {
  const screenEl = container.querySelector<HTMLElement>(".xterm-screen");
  if (!screenEl) return 0;
  return screenEl.getBoundingClientRect().height / terminal.rows;
}

/** Get the tracked touch from changedTouches matching the current gesture. */
function findGestureTouch(
  e: TouchEvent,
  g: GestureState,
): Touch | undefined {
  if (g.type === "idle") return undefined;
  const id = g.touchId;
  for (let i = 0; i < e.changedTouches.length; i++) {
    const t = e.changedTouches[i] as Touch | undefined;
    if (t && t.identifier === id) return t;
  }
  return undefined;
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useTerminalTouch({
  terminal,
  containerRef,
  enabled = true,
}: UseTerminalTouchOptions): UseTerminalTouchReturn {
  const [hasSelection, setHasSelection] = useState(false);
  const gestureRef = useRef<GestureState>({ type: "idle" });
  const doubleTapRef = useRef<DoubleTapState | null>(null);
  const momentumRafRef = useRef<number | null>(null);

  // ---- Clipboard helpers ----

  const copySelection = useCallback(async (): Promise<boolean> => {
    if (!terminal) return false;
    const text = terminal.getSelection();
    if (!text) return false;
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      return false;
    }
  }, [terminal]);

  const clearSelection = useCallback(() => {
    terminal?.clearSelection();
    setHasSelection(false);
  }, [terminal]);

  // ---- Cancel helpers ----

  const cancelMomentum = useCallback(() => {
    if (momentumRafRef.current !== null) {
      cancelAnimationFrame(momentumRafRef.current);
      momentumRafRef.current = null;
    }
  }, []);

  const resetGesture = useCallback(() => {
    const g = gestureRef.current;
    if (g.type === "pending") clearTimeout(g.longPressTimer);
    gestureRef.current = { type: "idle" };
  }, []);

  // ---- Main effect: attach touch listeners ----

  useEffect(() => {
    if (!terminal || !containerRef.current || !enabled) return;

    // Alias for use in closures — guaranteed non-null by the guard above.
    const term: Terminal = terminal;
    const container: HTMLDivElement = containerRef.current;

    // Set touch-action: none on the xterm canvas surface
    const screenEl = container.querySelector<HTMLElement>(".xterm-screen");
    if (screenEl) {
      screenEl.style.touchAction = "none";
    }

    const cellH = (): number => getCellHeight(term, container);

    // ---- Momentum scroll ----
    function startMomentum(velocity: number) {
      cancelMomentum();
      let v = velocity;
      const tick = () => {
        v *= TOUCH_SCROLL_DECEL;
        if (Math.abs(v) < TOUCH_SCROLL_MIN_VELOCITY) {
          momentumRafRef.current = null;
          return;
        }
        const ch = cellH();
        if (ch > 0) {
          const lines = Math.round(v / ch);
          if (lines !== 0) term.scrollLines(lines);
        }
        momentumRafRef.current = requestAnimationFrame(tick);
      };
      momentumRafRef.current = requestAnimationFrame(tick);
    }

    // ---- Word select (double-tap) ----
    function selectWordAt(col: number, row: number) {
      const bufferRow = row + term.buffer.active.viewportY;
      const line = term.buffer.active.getLine(bufferRow);
      if (!line) return;
      const text = line.translateToString();
      const bounds = findWordBoundaries(text, col);
      if (bounds) {
        term.select(bounds[0], row, bounds[1] - bounds[0]);
        setHasSelection(true);
      }
    }

    // ---- Touch handlers ----

    function onTouchStart(e: TouchEvent) {
      // Only track single-finger gestures
      if (e.touches.length !== 1) return;
      const touch = e.touches[0] as Touch | undefined;
      if (!touch) return;

      // Cancel any running momentum
      cancelMomentum();

      // Clear previous gesture if any
      resetGesture();

      const now = performance.now();

      // Check for double-tap
      const prev = doubleTapRef.current;
      if (
        prev &&
        now - prev.time < TOUCH_DOUBLE_TAP_MS &&
        Math.abs(touch.clientX - prev.x) < TOUCH_MOVE_THRESHOLD_PX * 2 &&
        Math.abs(touch.clientY - prev.y) < TOUCH_MOVE_THRESHOLD_PX * 2
      ) {
        doubleTapRef.current = null;
        const { col, row } = touchToCell(
          term,
          container,
          touch.clientX,
          touch.clientY,
        );
        selectWordAt(col, row);
        e.preventDefault();
        return;
      }

      // Start long-press timer
      const longPressTimer = setTimeout(() => {
        const g = gestureRef.current;
        if (g.type !== "pending") return;

        // Enter selection mode
        const { col, row } = touchToCell(term, container, g.startX, g.startY);

        // Haptic feedback if available
        if (navigator.vibrate) navigator.vibrate(30);

        term.select(col, row, 1);
        setHasSelection(true);

        gestureRef.current = {
          type: "selecting",
          touchId: g.touchId,
          anchorCol: col,
          anchorRow: row,
        };
      }, TOUCH_LONG_PRESS_MS);

      gestureRef.current = {
        type: "pending",
        touchId: touch.identifier,
        startX: touch.clientX,
        startY: touch.clientY,
        startTime: now,
        longPressTimer,
      };
    }

    function onTouchMove(e: TouchEvent) {
      const g = gestureRef.current;
      const touch = findGestureTouch(e, g);
      if (!touch) return;

      if (g.type === "pending") {
        const dx = touch.clientX - g.startX;
        const dy = touch.clientY - g.startY;
        const dist = Math.sqrt(dx * dx + dy * dy);

        if (dist > TOUCH_MOVE_THRESHOLD_PX) {
          // Moved beyond threshold — transition to scrolling
          clearTimeout(g.longPressTimer);
          gestureRef.current = {
            type: "scrolling",
            touchId: g.touchId,
            lastY: touch.clientY,
            lastTime: performance.now(),
            velocity: 0,
          };
          e.preventDefault();
        }
      } else if (g.type === "scrolling") {
        const deltaY = g.lastY - touch.clientY; // positive = finger up = scroll down
        const now = performance.now();
        const dt = now - g.lastTime;

        const ch = cellH();
        if (ch > 0) {
          const lines = Math.round(deltaY / ch);
          if (lines !== 0) {
            term.scrollLines(lines);
            // Track velocity for momentum (px/ms → scale to px/frame at ~16ms)
            g.velocity = dt > 0 ? (deltaY / dt) * 16 : 0;
            g.lastY = touch.clientY;
            g.lastTime = now;
          }
        }
        e.preventDefault();
      } else if (g.type === "selecting") {
        const { col, row } = touchToCell(
          term,
          container,
          touch.clientX,
          touch.clientY,
        );

        // Compute selection from anchor to current position
        const colDelta = col - g.anchorCol;
        const rowDelta = row - g.anchorRow;
        const length = rowDelta * term.cols + colDelta;

        if (length >= 0) {
          term.select(g.anchorCol, g.anchorRow, Math.max(1, length + 1));
        } else {
          // Selecting backwards: anchor at current position
          term.select(col, row, -length + 1);
        }
        setHasSelection(true);
        e.preventDefault();
      }
    }

    function onTouchEnd(e: TouchEvent) {
      const g = gestureRef.current;
      const touch = findGestureTouch(e, g);
      if (!touch) return;

      if (g.type === "pending") {
        clearTimeout(g.longPressTimer);
        const elapsed = performance.now() - g.startTime;

        if (elapsed < TOUCH_TAP_MAX_MS) {
          // Record for potential double-tap
          doubleTapRef.current = {
            time: performance.now(),
            x: touch.clientX,
            y: touch.clientY,
          };
          // Single-tap: focus terminal.
          // preventDefault suppresses the synthetic click the browser would
          // fire ~300ms later.  By that time the virtual keyboard has opened
          // and the terminal container has shrunk, so the click coordinates
          // may land outside the terminal and blur it.
          e.preventDefault();
          term.focus();
        }
      } else if (g.type === "scrolling") {
        // Start momentum scroll if there's velocity
        if (Math.abs(g.velocity) > TOUCH_SCROLL_MIN_VELOCITY) {
          startMomentum(g.velocity);
        }
      }
      // For "selecting", keep the selection in place on touchend

      gestureRef.current = { type: "idle" };
    }

    function onTouchCancel() {
      resetGesture();
    }

    // Attach in capture phase to intercept before xterm's bundled gesture system
    container.addEventListener("touchstart", onTouchStart, { capture: true });
    container.addEventListener("touchmove", onTouchMove, {
      capture: true,
      passive: false,
    });
    container.addEventListener("touchend", onTouchEnd, { capture: true });
    container.addEventListener("touchcancel", onTouchCancel, { capture: true });

    return () => {
      container.removeEventListener("touchstart", onTouchStart, {
        capture: true,
      });
      container.removeEventListener("touchmove", onTouchMove, {
        capture: true,
      });
      container.removeEventListener("touchend", onTouchEnd, { capture: true });
      container.removeEventListener("touchcancel", onTouchCancel, {
        capture: true,
      });

      // Clean up touch-action style
      if (screenEl) {
        screenEl.style.touchAction = "";
      }

      cancelMomentum();
      resetGesture();
    };
    // containerRef is a stable ref — its .current is read inside the effect.
    // cancelMomentum and resetGesture are stable useCallback([]) refs.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [terminal, enabled]);

  return { hasSelection, copySelection, clearSelection };
}
