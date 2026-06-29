import { useEffect, useRef } from "react";
import type { Terminal, IBufferCell } from "@xterm/xterm";
import { cellBackgroundHex, dominantBackground, parseOscColor } from "../../lib/terminalBackground";

/**
 * Debounce window for re-sampling after a render. `onRender` fires frequently
 * while a TUI repaints; we coalesce bursts so the chrome follows the screen
 * without strobing.
 */
export const BG_DETECT_DEBOUNCE_MS = 200;

/**
 * Number of bottom rows excluded from the sample. A tmux / shell status line
 * (e.g. the green `[wc-…:grok*]` band) sits on the last row; excluding it keeps
 * a thin colored status bar from hijacking the whole chrome color.
 */
export const BG_DETECT_STATUS_ROWS = 1;

/**
 * Minimum share of non-empty sampled cells a single color must hold to win.
 * Below this the sample is ambiguous and we report `null` (→ theme fallback).
 */
export const BG_DETECT_DOMINANCE = 0.5;

interface BackgroundDetectorOptions {
  /** Run detection only when true (focused pane, tab-like mode, adaptive on). */
  enabled: boolean;
  /** Configured theme background — the fallback for DEFAULT-mode cells. */
  defaultBackground: string;
  /** Called (debounced) with the detected hex, or `null` when ambiguous. */
  onColor: (hex: string | null) => void;
}

/**
 * Detects the focused pane's *rendered* terminal background by sampling the
 * xterm buffer cell colors, and tracks `OSC 11` default-background changes.
 *
 * Performance: detection is debounced, sampling is bounded, it pauses while the
 * document is hidden, and the result is delivered through `onColor` — which the
 * caller wires straight to the imperative chrome applier, never React state.
 * See `lib/chromeTheme.ts` and the plan's performance contract.
 */
export function useTerminalBackgroundDetector(
  term: Terminal | null,
  { enabled, defaultBackground, onColor }: BackgroundDetectorOptions,
): void {
  const onColorRef = useRef(onColor);
  onColorRef.current = onColor;
  const defaultBgRef = useRef(defaultBackground);
  defaultBgRef.current = defaultBackground;
  // OSC 11 default-background override. A program can change the terminal's
  // default bg without per-cell SGR; DEFAULT-mode cells then resolve to this.
  const osc11Ref = useRef<string | null>(null);

  useEffect(() => {
    if (!term || !enabled) {
      onColorRef.current(null);
      return;
    }

    let timer: ReturnType<typeof setTimeout> | null = null;
    let disposed = false;

    const sample = (): void => {
      timer = null;
      if (disposed) return;
      if (typeof document !== "undefined" && document.visibilityState === "hidden") return;
      const buf = term.buffer.active;
      // Reuse one cell object across reads to avoid per-cell allocation.
      const reuse: IBufferCell | undefined = buf.getNullCell?.();
      const top = buf.viewportY;
      const maxRow = Math.max(1, term.rows - BG_DETECT_STATUS_ROWS);
      const defaultBg = osc11Ref.current ?? defaultBgRef.current;
      const hexes: (string | null)[] = [];
      for (let r = 0; r < maxRow; r++) {
        const line = buf.getLine(top + r);
        if (!line) continue;
        for (let c = 0; c < term.cols; c++) {
          const cell = line.getCell(c, reuse);
          if (!cell) continue;
          hexes.push(cellBackgroundHex(cell, defaultBg));
        }
      }
      onColorRef.current(dominantBackground(hexes, BG_DETECT_DOMINANCE));
    };

    const schedule = (): void => {
      if (timer !== null) return;
      timer = setTimeout(sample, BG_DETECT_DEBOUNCE_MS);
    };

    const renderDisposable = term.onRender(schedule);
    // Track OSC 11 (set default bg). Returning false lets xterm apply it too,
    // so rendering is unaffected.
    const oscSetDisposable = term.parser.registerOscHandler(11, (data: string) => {
      const hex = parseOscColor(data);
      if (hex) {
        osc11Ref.current = hex;
        schedule();
      }
      return false;
    });
    // OSC 111 resets the default bg back to the theme value.
    const oscResetDisposable = term.parser.registerOscHandler(111, () => {
      osc11Ref.current = null;
      schedule();
      return false;
    });

    const onVisibility = (): void => {
      if (typeof document !== "undefined" && document.visibilityState === "visible") schedule();
    };
    if (typeof document !== "undefined") {
      document.addEventListener("visibilitychange", onVisibility);
    }

    // Sample once now so the chrome reflects the current screen immediately.
    schedule();

    return () => {
      disposed = true;
      if (timer !== null) clearTimeout(timer);
      renderDisposable.dispose();
      oscSetDisposable.dispose();
      oscResetDisposable.dispose();
      if (typeof document !== "undefined") {
        document.removeEventListener("visibilitychange", onVisibility);
      }
      // Clear this pane's contribution when detection stops.
      onColorRef.current(null);
    };
  }, [term, enabled]);
}
