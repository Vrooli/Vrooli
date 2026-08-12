/**
 * Pure helpers for reading the *rendered* background color out of an xterm
 * buffer. Kept separate from the React hook so the color-resolution and
 * dominant-color logic can be unit-tested without a live terminal.
 *
 * Detection source is the xterm **buffer cell color API** (unlocked by
 * `allowProposedApi: true`) plus `OSC 11` default-background tracking — never
 * canvas pixels. See `hooks/terminal/useTerminalBackgroundDetector.ts`.
 */
/** xterm.js default 16-color ANSI palette (indices 0–15). */
const ANSI_16 = [
    "#000000", "#cd3131", "#0dbc79", "#e5e510", "#2472c8", "#bc3fbc", "#11a8cd", "#e5e5e5",
    "#666666", "#f14c4c", "#23d18b", "#f5f543", "#3b8eea", "#d670d6", "#29b8db", "#e5e5e5",
];
/** The six channel levels of the xterm 6×6×6 color cube (indices 16–231). */
const CUBE_LEVELS = [0, 95, 135, 175, 215, 255];
function rgbHex(r, g, b) {
    const h = (n) => Math.max(0, Math.min(255, n)).toString(16).padStart(2, "0");
    return `#${h(r)}${h(g)}${h(b)}`;
}
/**
 * Resolve a 256-color ANSI palette index to a hex string. Covers the 16 base
 * colors, the 6×6×6 cube, and the 24-step grayscale ramp. Returns `null` for
 * out-of-range indices.
 */
export function ansi256ToHex(index) {
    if (!Number.isInteger(index) || index < 0 || index > 255)
        return null;
    if (index < 16)
        return ANSI_16[index] ?? null;
    if (index < 232) {
        const n = index - 16;
        const r = CUBE_LEVELS[Math.floor(n / 36) % 6] ?? 0;
        const g = CUBE_LEVELS[Math.floor(n / 6) % 6] ?? 0;
        const b = CUBE_LEVELS[n % 6] ?? 0;
        return rgbHex(r, g, b);
    }
    const v = 8 + (index - 232) * 10;
    return rgbHex(v, v, v);
}
/**
 * Resolve a single cell's background to a hex color.
 *   - DEFAULT mode → `defaultBg` (the OSC-11 default or the theme background),
 *     so a program that paints with the terminal default still reports a color.
 *   - PALETTE mode (0–255) → resolved against the ANSI palette.
 *   - RGB (truecolor) mode → the packed `0xRRGGBB` value, used directly.
 */
export function cellBackgroundHex(cell, defaultBg) {
    if (cell.isBgDefault())
        return defaultBg;
    const value = cell.getBgColor();
    if (cell.isBgRGB()) {
        return rgbHex((value >> 16) & 0xff, (value >> 8) & 0xff, value & 0xff);
    }
    if (cell.isBgPalette()) {
        return ansi256ToHex(value);
    }
    return defaultBg;
}
/**
 * Pick the dominant color from a sampled list of cell backgrounds. Returns
 * `null` when there is no confident winner (empty sample, or the top color
 * holds less than `threshold` of the non-empty cells) so the caller can fall
 * back to the configured theme background.
 */
export function dominantBackground(hexes, threshold) {
    const counts = new Map();
    let total = 0;
    for (const h of hexes) {
        if (!h)
            continue;
        const key = h.toLowerCase();
        counts.set(key, (counts.get(key) ?? 0) + 1);
        total++;
    }
    if (total === 0)
        return null;
    let best = null;
    let bestN = 0;
    for (const [hex, n] of counts) {
        if (n > bestN) {
            bestN = n;
            best = hex;
        }
    }
    if (best === null || bestN / total < threshold)
        return null;
    return best;
}
/**
 * Pick the color holding the largest share of total *weight* across the
 * samples. Null hexes and non-positive weights are ignored. Returns `null`
 * when no color reaches `threshold` of the total weight (an ambiguous sample),
 * so the caller can fall back to a flat pass or the configured theme.
 */
export function dominantWeightedBackground(samples, threshold) {
    const weights = new Map();
    let total = 0;
    for (const { hex, weight } of samples) {
        if (!hex || weight <= 0)
            continue;
        const key = hex.toLowerCase();
        weights.set(key, (weights.get(key) ?? 0) + weight);
        total += weight;
    }
    if (total === 0)
        return null;
    let best = null;
    let bestW = 0;
    for (const [hex, w] of weights) {
        if (w > bestW) {
            bestW = w;
            best = hex;
        }
    }
    if (best === null || bestW / total < threshold)
        return null;
    return best;
}
/**
 * Defaults chosen so the perimeter (where app chrome visually meets the
 * terminal) decides the ambient color, while a color that fills the *entire*
 * usable screen still wins via the full-screen fallback. Thresholds sit above
 * 0.5 so an even split returns `null` rather than flipping on scan order.
 */
export const AMBIENT_BACKGROUND_DEFAULTS = {
    statusRows: 1,
    cornerRows: 3,
    cornerCols: 8,
    edgeRows: 2,
    edgeCols: 4,
    cornerWeight: 3,
    edgeWeight: 2,
    perimeterThreshold: 0.6,
    fullScreenThreshold: 0.75,
};
/**
 * Select the terminal's *ambient* background from a rectangular-ish grid of
 * resolved cell colors (rows × columns, ragged rows tolerated).
 *
 * The app chrome sits visually adjacent to the terminal's perimeter, so the
 * heuristic is perimeter-biased rather than a flat full-buffer histogram:
 *   1. The bottom `statusRows` are excluded (status-line protection).
 *   2. Perimeter pass — corner and edge-band cells are weighted (corners more
 *      than edges); if one color holds `perimeterThreshold` of that weight it
 *      wins. A large center-only content block (e.g. a coding-agent user
 *      message) is ignored here, so it can no longer hijack the chrome.
 *   3. Full-screen fallback — otherwise, if one color fills `fullScreenThreshold`
 *      of the whole usable grid it wins, so a real full-screen TUI still retints.
 *   4. Otherwise `null`, leaving the caller's theme fallback in charge.
 *
 * O(rows × cols), no allocations beyond two bounded sample passes, no blending.
 */
export function ambientBackground(hexGrid, options) {
    const o = { ...AMBIENT_BACKGROUND_DEFAULTS, ...options };
    const rowCount = hexGrid.length;
    if (rowCount === 0)
        return null;
    // Exclude the bottom status row(s). If that would leave nothing to sample
    // (a 1–2 row terminal), fall back to the whole grid rather than returning
    // nothing.
    let usableRows = rowCount - Math.max(0, o.statusRows);
    if (usableRows < 1)
        usableRows = rowCount;
    // Widest usable row sets the column extent; grids may be ragged.
    let cols = 0;
    for (let r = 0; r < usableRows; r++) {
        const len = hexGrid[r]?.length ?? 0;
        if (len > cols)
            cols = len;
    }
    if (cols === 0)
        return null;
    const cornerRows = Math.min(o.cornerRows, usableRows);
    const cornerCols = Math.min(o.cornerCols, cols);
    const edgeRows = Math.min(o.edgeRows, usableRows);
    const edgeCols = Math.min(o.edgeCols, cols);
    const perimeter = [];
    const all = [];
    for (let r = 0; r < usableRows; r++) {
        const row = hexGrid[r];
        if (!row)
            continue;
        const nearTopBottomEdge = r < edgeRows || r >= usableRows - edgeRows;
        const nearTopBottomCorner = r < cornerRows || r >= usableRows - cornerRows;
        for (let c = 0; c < row.length; c++) {
            const hex = row[c] ?? null;
            all.push(hex);
            const nearLeftRightEdge = c < edgeCols || c >= cols - edgeCols;
            const nearLeftRightCorner = c < cornerCols || c >= cols - cornerCols;
            if (nearTopBottomCorner && nearLeftRightCorner) {
                perimeter.push({ hex, weight: o.cornerWeight });
            }
            else if (nearTopBottomEdge || nearLeftRightEdge) {
                perimeter.push({ hex, weight: o.edgeWeight });
            }
        }
    }
    const perimeterWinner = dominantWeightedBackground(perimeter, o.perimeterThreshold);
    if (perimeterWinner)
        return perimeterWinner;
    return dominantBackground(all, o.fullScreenThreshold);
}
/**
 * Parse an `OSC 11` color payload (set-default-background) into a hex string.
 * Handles the common `rgb:RR/GG/BB` (1–4 hex digits per channel) and `#rgb` /
 * `#rrggbb` / `#rrrrggggbbbb` forms. Returns `null` for queries (`?`) or any
 * unrecognized value.
 */
export function parseOscColor(data) {
    if (!data)
        return null;
    const s = data.trim();
    if (s.startsWith("rgb:")) {
        const parts = s.slice(4).split("/");
        if (parts.length !== 3)
            return null;
        const chans = [];
        for (const p of parts) {
            if (!/^[0-9a-fA-F]{1,4}$/.test(p))
                return null;
            const max = (1 << (4 * p.length)) - 1;
            chans.push(Math.round((parseInt(p, 16) / max) * 255));
        }
        const [r = 0, g = 0, b = 0] = chans;
        return rgbHex(r, g, b);
    }
    if (s.startsWith("#")) {
        const hex = s.slice(1);
        if (/^[0-9a-fA-F]{6}$/.test(hex))
            return `#${hex.toLowerCase()}`;
        if (/^[0-9a-fA-F]{3}$/.test(hex)) {
            return `#${hex.split("").map((c) => c + c).join("").toLowerCase()}`;
        }
        if (/^[0-9a-fA-F]{12}$/.test(hex)) {
            return `#${[0, 4, 8].map((i) => hex.slice(i, i + 2)).join("").toLowerCase()}`;
        }
    }
    return null;
}
