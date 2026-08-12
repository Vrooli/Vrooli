import { useEffect, useRef } from "react";
import { ambientBackground, cellBackgroundHex, parseOscColor } from "../../lib/terminalBackground";
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
 * Minimum weighted share a single color must hold across the *perimeter*
 * samples (corners + edge bands) to be treated as the terminal's ambient
 * background. Above 0.5 so an even split reports `null` rather than flipping
 * on scan order. The perimeter is what app chrome visually borders, so a large
 * center-only content block (e.g. a coding-agent user message) is ignored.
 */
export const BG_DETECT_PERIMETER_DOMINANCE = 0.6;
/**
 * Minimum share of the *whole* usable grid a single color must hold to win as
 * a fallback, so a true full-screen TUI whose new background also fills the
 * interior still retints the chrome. Higher than the perimeter threshold
 * because this pass has no perimeter bias to lean on.
 */
export const BG_DETECT_FULLSCREEN_DOMINANCE = 0.75;
/**
 * Detects the focused pane's *rendered* terminal background by sampling the
 * xterm buffer cell colors, and tracks `OSC 11` default-background changes.
 *
 * Performance: detection is debounced, sampling is bounded, it pauses while the
 * document is hidden, and the result is delivered through `onColor` — which the
 * caller wires straight to the imperative chrome applier, never React state.
 * See `lib/chromeTheme.ts` and the plan's performance contract.
 */
export function useTerminalBackgroundDetector(term, { enabled, defaultBackground, onColor }) {
    const onColorRef = useRef(onColor);
    onColorRef.current = onColor;
    const defaultBgRef = useRef(defaultBackground);
    defaultBgRef.current = defaultBackground;
    // OSC 11 default-background override. A program can change the terminal's
    // default bg without per-cell SGR; DEFAULT-mode cells then resolve to this.
    const osc11Ref = useRef(null);
    useEffect(() => {
        if (!term || !enabled) {
            onColorRef.current(null);
            return;
        }
        let timer = null;
        let disposed = false;
        const sample = () => {
            timer = null;
            if (disposed)
                return;
            if (typeof document !== "undefined" && document.visibilityState === "hidden")
                return;
            const buf = term.buffer.active;
            // Reuse one cell object across reads to avoid per-cell allocation.
            const reuse = buf.getNullCell?.();
            const top = buf.viewportY;
            const defaultBg = osc11Ref.current ?? defaultBgRef.current;
            // Build the visible grid (status row included); the ambient selector
            // excludes the bottom status row(s) itself via `statusRows`.
            const grid = [];
            for (let r = 0; r < term.rows; r++) {
                const line = buf.getLine(top + r);
                if (!line)
                    continue;
                const cells = [];
                for (let c = 0; c < term.cols; c++) {
                    const cell = line.getCell(c, reuse);
                    cells.push(cell ? cellBackgroundHex(cell, defaultBg) : null);
                }
                grid.push(cells);
            }
            onColorRef.current(ambientBackground(grid, {
                statusRows: BG_DETECT_STATUS_ROWS,
                perimeterThreshold: BG_DETECT_PERIMETER_DOMINANCE,
                fullScreenThreshold: BG_DETECT_FULLSCREEN_DOMINANCE,
            }));
        };
        const schedule = () => {
            if (timer !== null)
                return;
            timer = setTimeout(sample, BG_DETECT_DEBOUNCE_MS);
        };
        const renderDisposable = term.onRender(schedule);
        // Track OSC 11 (set default bg). Returning false lets xterm apply it too,
        // so rendering is unaffected.
        const oscSetDisposable = term.parser.registerOscHandler(11, (data) => {
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
        const onVisibility = () => {
            if (typeof document !== "undefined" && document.visibilityState === "visible")
                schedule();
        };
        if (typeof document !== "undefined") {
            document.addEventListener("visibilitychange", onVisibility);
        }
        // Sample once now so the chrome reflects the current screen immediately.
        schedule();
        return () => {
            disposed = true;
            if (timer !== null)
                clearTimeout(timer);
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
