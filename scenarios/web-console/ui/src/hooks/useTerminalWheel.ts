import { useEffect, useRef } from "react";
import type { Terminal } from "@xterm/xterm";
import {
  createWheelLineAccumulator,
  scrollTransportFor,
  terminalCellHeight,
  type ScrollSource,
} from "../lib/terminalScroll";

export interface UseTerminalWheelOptions {
  terminal: Terminal | null;
  containerRef: React.RefObject<HTMLDivElement | null>;
  enabled?: boolean;
  scrollBy: (lines: number, source: ScrollSource) => void;
}

/**
 * Takes ownership of the mouse wheel for the one case xterm.js handles badly.
 *
 * When a terminal has no scrollback and no mouse tracking, xterm's built-in
 * wheel listener translates each notch into an Up/Down cursor-key *data*
 * event — real stdin, indistinguishable from a keypress:
 *
 *     if (!mouseHandlers.wheel) {
 *       if (customHandler && customHandler(ev) === false) return false;
 *       if (!this.buffer.hasScrollback) { ...triggerDataEvent(ESC + '[' + 'A'|'B') }
 *     }
 *
 * That is a reasonable convenience for a full-screen program that reads arrow
 * keys as scrolling. It is actively wrong for our panes, because a tmux client
 * puts xterm in the alternate buffer for the entire session, so `hasScrollback`
 * is permanently false. Every wheel notch then reaches whatever runs inside the
 * pane as a genuine arrow key. In an interactive agent whose composer binds
 * Up/Down to message history, scrolling silently rewrites the operator's draft.
 *
 * The custom handler is consulted before that branch, so returning `false` is
 * the supported way to suppress it. We claim the wheel only when the built-in
 * behaviour would be wrong, and defer to xterm everywhere it is right:
 *
 * - `mouse-report`: the program asked for wheel events; it must keep getting
 *   them as mouse reports, so xterm's native encoding stays in charge.
 * - `local`: xterm has real scrollback and scrolls its own viewport, applying
 *   its own `scrollSensitivity`. Nothing to fix.
 * - `server-scroll`: the broken case. Convert to lines and route through the
 *   scroll seam, which asks the server to scroll the backend's history.
 */
export function useTerminalWheel({
  terminal,
  containerRef,
  enabled = true,
  scrollBy,
}: UseTerminalWheelOptions): void {
  const scrollByRef = useRef(scrollBy);
  scrollByRef.current = scrollBy;

  useEffect(() => {
    if (!terminal || !enabled) return;
    const term: Terminal = terminal;
    const accumulator = createWheelLineAccumulator();

    const handleWheel = (event: WheelEvent): boolean => {
      if (scrollTransportFor(term) !== "server-scroll") {
        // Someone else owns the wheel this frame. Drop the partial line so a
        // later gesture does not inherit a remainder from a different owner.
        accumulator.reset();
        return true;
      }
      const container = containerRef.current;
      const cellHeight = container ? terminalCellHeight(term, container) : 0;
      const lines = accumulator.consume(event, cellHeight, term.rows);
      if (lines !== 0) scrollByRef.current(lines, "wheel");
      // Suppress both the arrow-key emission and any native viewport scroll;
      // this transport does all of its scrolling through the seam above.
      event.preventDefault();
      return false;
    };

    term.attachCustomWheelEventHandler(handleWheel);
    return () => {
      // xterm has no detach call; restoring the permissive default is the
      // documented way to hand the wheel back.
      term.attachCustomWheelEventHandler(() => true);
    };
    // containerRef is a stable ref; scrollBy is read through a live ref.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [terminal, enabled]);
}
