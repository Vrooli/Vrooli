/**
 * Pure helpers for reading the *rendered* background color out of an xterm
 * buffer. Kept separate from the React hook so the color-resolution and
 * dominant-color logic can be unit-tested without a live terminal.
 *
 * Detection source is the xterm **buffer cell color API** (unlocked by
 * `allowProposedApi: true`) plus `OSC 11` default-background tracking — never
 * canvas pixels. See `hooks/terminal/useTerminalBackgroundDetector.ts`.
 */

/**
 * The subset of xterm's `IBufferCell` this module reads. Declaring it locally
 * keeps the helpers testable with synthetic cells and gives one place to assert
 * the proposed-API shape (see `terminalBackground.test.ts`).
 */
export interface BgCell {
  isBgDefault(): boolean;
  isBgPalette(): boolean;
  isBgRGB(): boolean;
  getBgColor(): number;
}

/** xterm.js default 16-color ANSI palette (indices 0–15). */
const ANSI_16 = [
  "#000000", "#cd3131", "#0dbc79", "#e5e510", "#2472c8", "#bc3fbc", "#11a8cd", "#e5e5e5",
  "#666666", "#f14c4c", "#23d18b", "#f5f543", "#3b8eea", "#d670d6", "#29b8db", "#e5e5e5",
] as const;

/** The six channel levels of the xterm 6×6×6 color cube (indices 16–231). */
const CUBE_LEVELS = [0, 95, 135, 175, 215, 255] as const;

function rgbHex(r: number, g: number, b: number): string {
  const h = (n: number) => Math.max(0, Math.min(255, n)).toString(16).padStart(2, "0");
  return `#${h(r)}${h(g)}${h(b)}`;
}

/**
 * Resolve a 256-color ANSI palette index to a hex string. Covers the 16 base
 * colors, the 6×6×6 cube, and the 24-step grayscale ramp. Returns `null` for
 * out-of-range indices.
 */
export function ansi256ToHex(index: number): string | null {
  if (!Number.isInteger(index) || index < 0 || index > 255) return null;
  if (index < 16) return ANSI_16[index] ?? null;
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
export function cellBackgroundHex(cell: BgCell, defaultBg: string | null): string | null {
  if (cell.isBgDefault()) return defaultBg;
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
export function dominantBackground(
  hexes: ReadonlyArray<string | null>,
  threshold: number,
): string | null {
  const counts = new Map<string, number>();
  let total = 0;
  for (const h of hexes) {
    if (!h) continue;
    const key = h.toLowerCase();
    counts.set(key, (counts.get(key) ?? 0) + 1);
    total++;
  }
  if (total === 0) return null;
  let best: string | null = null;
  let bestN = 0;
  for (const [hex, n] of counts) {
    if (n > bestN) {
      bestN = n;
      best = hex;
    }
  }
  if (best === null || bestN / total < threshold) return null;
  return best;
}

/**
 * Parse an `OSC 11` color payload (set-default-background) into a hex string.
 * Handles the common `rgb:RR/GG/BB` (1–4 hex digits per channel) and `#rgb` /
 * `#rrggbb` / `#rrrrggggbbbb` forms. Returns `null` for queries (`?`) or any
 * unrecognized value.
 */
export function parseOscColor(data: string | undefined | null): string | null {
  if (!data) return null;
  const s = data.trim();
  if (s.startsWith("rgb:")) {
    const parts = s.slice(4).split("/");
    if (parts.length !== 3) return null;
    const chans: number[] = [];
    for (const p of parts) {
      if (!/^[0-9a-fA-F]{1,4}$/.test(p)) return null;
      const max = (1 << (4 * p.length)) - 1;
      chans.push(Math.round((parseInt(p, 16) / max) * 255));
    }
    const [r = 0, g = 0, b = 0] = chans;
    return rgbHex(r, g, b);
  }
  if (s.startsWith("#")) {
    const hex = s.slice(1);
    if (/^[0-9a-fA-F]{6}$/.test(hex)) return `#${hex.toLowerCase()}`;
    if (/^[0-9a-fA-F]{3}$/.test(hex)) {
      return `#${hex.split("").map((c) => c + c).join("").toLowerCase()}`;
    }
    if (/^[0-9a-fA-F]{12}$/.test(hex)) {
      return `#${[0, 4, 8].map((i) => hex.slice(i, i + 2)).join("").toLowerCase()}`;
    }
  }
  return null;
}
