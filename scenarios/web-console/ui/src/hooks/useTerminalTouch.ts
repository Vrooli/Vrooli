import { useCallback, useEffect, useRef, useState } from "react";
import type { Terminal } from "@xterm/xterm";
import {
  TOUCH_MOVE_THRESHOLD_PX,
  TOUCH_LONG_PRESS_MS,
  TOUCH_TAP_MAX_MS,
  TOUCH_DOUBLE_TAP_MS,
  TOUCH_SCROLL_CANCEL_PX,
  TOUCH_SCROLL_DECEL,
  TOUCH_SCROLL_MIN_VELOCITY,
} from "../consts/config";
import { mouseWheelSequence } from "../lib/terminalKeys";
import { terminalIsInMouseTrackingMode } from "../components/terminal/inputGate";
import { writeText } from "../lib/clipboard";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface UseTerminalTouchOptions {
  terminal: Terminal | null;
  containerRef: React.RefObject<HTMLDivElement | null>;
  enabled?: boolean;
  /** Fired on right-click (desktop) or long-press-release without drag (mobile). */
  onContextMenu?: (x: number, y: number) => void;
  /**
   * Send synthetic terminal control bytes. Controls are best-effort and
   * deliberately bypass the reliable stdin gate.
   */
  sendControl?: (data: string) => boolean;
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
      startX: number;
      startY: number;
      lastY: number;
      lastTime: number;
      velocity: number;
      /** Cumulative distance scrolled (px). Once this exceeds
       *  TOUCH_SCROLL_CANCEL_PX the long-press timer is cancelled — the
       *  gesture is clearly an intentional scroll, not tremor. */
      cumulativeDist: number;
      /** The long-press timer is kept alive for small movements (hand tremor)
       *  but cancelled once cumulativeDist proves real scroll intent. */
      longPressTimer: ReturnType<typeof setTimeout> | null;
    }
  | {
      type: "selecting";
      touchId: number;
      anchorCol: number;
      anchorRow: number;
      hasDragged: boolean;
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
  onContextMenu,
  sendControl,
}: UseTerminalTouchOptions): UseTerminalTouchReturn {
  const [hasSelection, setHasSelection] = useState(false);
  const gestureRef = useRef<GestureState>({ type: "idle" });
  const doubleTapRef = useRef<DoubleTapState | null>(null);
  const momentumRafRef = useRef<number | null>(null);
  const onContextMenuRef = useRef(onContextMenu);
  onContextMenuRef.current = onContextMenu;
  const sendControlRef = useRef(sendControl);
  sendControlRef.current = sendControl;

  // ---- Clipboard helpers ----

  const copySelection = useCallback(async (): Promise<boolean> => {
    if (!terminal) return false;
    const text = terminal.getSelection();
    if (!text) return false;
    const result = await writeText(text);
    return result.ok;
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
    if (g.type === "scrolling" && g.longPressTimer !== null) clearTimeout(g.longPressTimer);
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

    // ---- Scroll dispatch (xterm buffer vs application mouse mode) ----
    // When the terminal application has enabled mouse tracking (e.g. tmux
    // with `mouse on`), scroll gestures must be sent as mouse wheel escape
    // sequences so the application can handle them. Otherwise, scroll
    // xterm.js's own buffer as before.
    function scrollTerminal(lines: number) {
      if (lines === 0) return;
      if (terminalIsInMouseTrackingMode(term) && sendControlRef.current) {
        // Send one wheel event per line, at screen center. tmux treats
        // each event as a single scroll-line regardless of position.
        const col = Math.floor(term.cols / 2);
        const row = Math.floor(term.rows / 2);
        const up = lines < 0;
        const count = Math.abs(lines);
        const seq = mouseWheelSequence(up, col, row);
        for (let i = 0; i < count; i++) {
          sendControlRef.current(seq);
        }
      } else {
        term.scrollLines(lines);
      }
    }

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
          if (lines !== 0) scrollTerminal(lines);
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

      // Don't intercept touches on the context menu or its backdrop — let
      // their native click handlers fire so buttons work and the backdrop
      // dismisses the menu.
      const target = e.target as HTMLElement | null;
      if (target?.closest("[data-testid='terminal-context-menu'], [data-testid='ctx-backdrop']")) {
        return;
      }

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
        // Open context menu at the double-tap location so the user can
        // immediately Copy/Speak the selected word.
        onContextMenuRef.current?.(touch.clientX, touch.clientY);
        e.preventDefault();
        return;
      }

      // Start long-press timer.
      // This timer fires after TOUCH_LONG_PRESS_MS and accepts both
      // "pending" and "scrolling" states — hand tremor during a hold
      // easily exceeds the 8px movement threshold.  However, once the
      // finger travels beyond TOUCH_SCROLL_CANCEL_PX the timer is
      // cancelled (in onTouchMove) because that much movement is clearly
      // an intentional scroll, not tremor.
      const longPressTimer = setTimeout(() => {
        const g = gestureRef.current;
        if (g.type !== "pending" && g.type !== "scrolling") return;

        // Enter selection mode at the *original* touch-down position,
        // not the current finger position (which may have drifted).
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
          // If the finger already moved (was scrolling), mark as dragged
          // so onTouchEnd keeps the selection instead of showing only the
          // context menu.
          hasDragged: g.type === "scrolling",
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
          // Moved beyond threshold — transition to scrolling. Keep the
          // long-press timer alive for now (could be hand tremor); it will
          // be cancelled once cumulativeDist exceeds TOUCH_SCROLL_CANCEL_PX.
          gestureRef.current = {
            type: "scrolling",
            touchId: g.touchId,
            startX: g.startX,
            startY: g.startY,
            lastY: touch.clientY,
            lastTime: performance.now(),
            velocity: 0,
            cumulativeDist: dist,
            longPressTimer: g.longPressTimer,
          };
          e.preventDefault();
        }
      } else if (g.type === "scrolling") {
        const deltaY = g.lastY - touch.clientY; // positive = finger up = scroll down
        const now = performance.now();
        const dt = now - g.lastTime;

        // Track cumulative scroll distance and cancel long-press timer
        // once the finger has clearly moved beyond tremor range.
        g.cumulativeDist += Math.abs(deltaY);
        if (g.longPressTimer !== null && g.cumulativeDist > TOUCH_SCROLL_CANCEL_PX) {
          clearTimeout(g.longPressTimer);
          g.longPressTimer = null;
        }

        const ch = cellH();
        if (ch > 0) {
          const lines = Math.round(deltaY / ch);
          if (lines !== 0) {
            scrollTerminal(lines);
            // Track velocity for momentum (px/ms → scale to px/frame at ~16ms)
            g.velocity = dt > 0 ? (deltaY / dt) * 16 : 0;
            g.lastY = touch.clientY;
            g.lastTime = now;
          }
        }
        e.preventDefault();
      } else if (g.type === "selecting") {
        g.hasDragged = true;
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
          // Signal typing intent to the browser by temporarily allowing the
          // virtual keyboard. xterm's textarea defaults to inputMode="none"
          // (set in TerminalPane.tsx) to suppress the keyboard on programmatic
          // focus. A direct tap is the strongest signal that the user wants
          // to type, so we flip to inputMode="" before focusing. The blur
          // handler in TerminalPane.tsx will reset it back to "none" when
          // the terminal later loses focus.
          if (term.textarea) {
            term.textarea.inputMode = "";
          }
          term.focus();
        }
      } else if (g.type === "scrolling") {
        // Clean up the long-press timer if it's still alive
        if (g.longPressTimer !== null) clearTimeout(g.longPressTimer);
        // Start momentum scroll if there's velocity
        if (Math.abs(g.velocity) > TOUCH_SCROLL_MIN_VELOCITY) {
          startMomentum(g.velocity);
        }
      } else if (g.type === "selecting") {
        if (!g.hasDragged) {
          // Long-press without drag: show context menu instead of keeping
          // the single-character selection.
          term.clearSelection();
          setHasSelection(false);
          onContextMenuRef.current?.(touch.clientX, touch.clientY);
        }
        if (g.hasDragged) {
          // Long-press + drag completed a selection — open context menu at
          // the release point so the user can Copy/Speak the selected text.
          onContextMenuRef.current?.(touch.clientX, touch.clientY);
        }
      }

      gestureRef.current = { type: "idle" };
    }

    function onTouchCancel() {
      resetGesture();
    }

    // Right-click context menu (desktop)
    function onContextMenuEvent(e: MouseEvent) {
      e.preventDefault();
      onContextMenuRef.current?.(e.clientX, e.clientY);
    }

    // Desktop mouse-drag selection: after mouseup, check if xterm has a
    // selection and auto-open the context menu at the release point.
    function onMouseUp(e: MouseEvent) {
      // Small delay lets xterm finalize the selection before we read it.
      requestAnimationFrame(() => {
        const sel = term.getSelection();
        if (sel) {
          setHasSelection(true);
          onContextMenuRef.current?.(e.clientX, e.clientY);
        }
      });
    }

    // Sync hasSelection with xterm's native selection state (covers desktop
    // mouse selection, programmatic select/clear, and Ctrl+Shift+A).
    const selectionDisposable = term.onSelectionChange(() => {
      const sel = term.getSelection();
      setHasSelection(!!sel);
    });

    // Attach in capture phase to intercept before xterm's bundled gesture system
    container.addEventListener("touchstart", onTouchStart, { capture: true });
    container.addEventListener("touchmove", onTouchMove, {
      capture: true,
      passive: false,
    });
    container.addEventListener("touchend", onTouchEnd, { capture: true });
    container.addEventListener("touchcancel", onTouchCancel, { capture: true });
    container.addEventListener("contextmenu", onContextMenuEvent);
    container.addEventListener("mouseup", onMouseUp);

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
      container.removeEventListener("contextmenu", onContextMenuEvent);
      container.removeEventListener("mouseup", onMouseUp);
      selectionDisposable.dispose();

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
