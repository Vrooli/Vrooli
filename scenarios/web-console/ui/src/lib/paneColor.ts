import type { CSSProperties } from "react";
import { HEADER_COLORS } from "../consts/config";

/**
 * Single source of truth for the pane header-color encoding.
 *
 * A pane carries one of three values in its `headerColor` string (and the
 * backend stores it verbatim as TEXT — see docs/plan §5/§8):
 *   - `"transparent"`        → no accent (inherits surface)
 *   - `"#rrggbb"`            → a single solid accent
 *   - `"#rrggbb|#rrggbb"`    → a two-color stripe ("candy-cane")
 *
 * The `|` delimiter never appears in a hex color, so single/transparent values
 * are byte-identical to the legacy encoding and round-trip untouched. Parsing
 * and serialization live ONLY here; render sites consume `paneColorStyle`.
 */

/** Matches a 3- or 6-digit hex color (with leading `#`). */
const HEX_COLOR = /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/;

/** Hard cap on the number of colors a pane may encode. */
export const MAX_PANE_COLORS = 2;

export interface ParsedPaneColor {
  /** 0–2 valid hex colors. Empty when transparent or malformed. */
  colors: string[];
  /** True when the pane has no accent color (transparent or unparseable). */
  isTransparent: boolean;
}

/** True when `value` is a syntactically valid hex color. */
export function isHexColor(value: string): boolean {
  return HEX_COLOR.test(value);
}

/**
 * Parse a 3- or 6-digit hex color into 0–255 RGB channels. Returns `null` for
 * anything that is not a valid hex color so callers can fall back safely.
 */
export function hexToRgb(value: string | null | undefined): { r: number; g: number; b: number } | null {
  if (!value || !HEX_COLOR.test(value)) return null;
  let hex = value.slice(1);
  if (hex.length === 3) {
    hex = hex
      .split("")
      .map((ch) => ch + ch)
      .join("");
  }
  const n = parseInt(hex, 16);
  return { r: (n >> 16) & 0xff, g: (n >> 8) & 0xff, b: n & 0xff };
}

/**
 * WCAG relative luminance (0 = black, 1 = white) of a hex color. Used to pick a
 * contrast-correct foreground for a tinted surface. Invalid input returns 0
 * (treated as dark) so the brighter foreground is chosen by default.
 */
export function relativeLuminance(value: string | null | undefined): number {
  const rgb = hexToRgb(value);
  if (!rgb) return 0;
  return relativeLuminanceRgb(rgb);
}

/** WCAG relative luminance (0 = black, 1 = white) of RGB channels. */
export function relativeLuminanceRgb(rgb: { r: number; g: number; b: number }): number {
  const channel = (c: number): number => {
    const s = c / 255;
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
  };
  return 0.2126 * channel(rgb.r) + 0.7152 * channel(rgb.g) + 0.0722 * channel(rgb.b);
}

/** WCAG contrast ratio between two RGB colors. */
export function contrastRatioRgb(
  a: { r: number; g: number; b: number },
  b: { r: number; g: number; b: number },
): number {
  const l1 = relativeLuminanceRgb(a);
  const l2 = relativeLuminanceRgb(b);
  const lighter = Math.max(l1, l2);
  const darker = Math.min(l1, l2);
  return (lighter + 0.05) / (darker + 0.05);
}

/** True when a color is light enough that dark foreground text reads better. */
export function isLightColor(value: string | null | undefined): boolean {
  return relativeLuminance(value) > 0.5;
}

/**
 * Parse a stored `headerColor` string into its component colors. Malformed
 * input (anything with no valid hex part) degrades safely to transparent.
 */
export function parsePaneColor(value: string | null | undefined): ParsedPaneColor {
  if (!value || value === "transparent") {
    return { colors: [], isTransparent: true };
  }
  const colors = value
    .split("|")
    .map((part) => part.trim())
    .filter((part) => HEX_COLOR.test(part))
    .slice(0, MAX_PANE_COLORS);
  if (colors.length === 0) {
    return { colors: [], isTransparent: true };
  }
  return { colors, isTransparent: false };
}

/**
 * Serialize 0–2 colors back into the stored encoding. Invalid entries are
 * dropped; an empty result becomes `"transparent"`.
 */
export function serializePaneColor(colors: readonly string[]): string {
  const valid = colors
    .filter((color) => HEX_COLOR.test(color))
    .slice(0, MAX_PANE_COLORS);
  if (valid.length === 0) return "transparent";
  return valid.join("|");
}

/**
 * Pick an auto-distinct color for a new group: the first palette entry not
 * already used by an existing group. When every palette color is taken, wrap
 * around by group count so new groups still differ from their predecessors.
 */
export function nextGroupColor(existingColors: readonly string[]): string {
  const used = new Set(existingColors);
  const free = HEADER_COLORS.find((color) => !used.has(color));
  if (free) return free;
  return HEADER_COLORS[existingColors.length % HEADER_COLORS.length] ?? HEADER_COLORS[0];
}

/**
 * The accent a pane should render: its own color, else its group's.
 *
 * Panes that join a group now have the group color seeded onto them (see the
 * store's withGroupAssigned), so this fallback is only load-bearing for panes
 * grouped before that existed — but it must live in one place. It used to be
 * written inline in the sidebar row and nowhere else, which is why the same
 * grouped session showed the group color in the sidebar and no color at all in
 * the tab strip and the grid pane header.
 */
export function paneAccentStyle(
  headerColor: string | null | undefined,
  groupColor: string | null | undefined,
  variant: "bar" | "header",
): CSSProperties | undefined {
  return (
    paneColorStyle(headerColor, variant)
    ?? (groupColor ? paneColorStyle(groupColor, variant) : undefined)
  );
}

/**
 * Width-aware CSS for a pane accent.
 *   - `"bar"`    — the thin sidebar/tab accent: two stacked horizontal bands,
 *     legible even at ~6px.
 *   - `"header"` — the wide terminal header: a 45° candy-cane repeat.
 *
 * A single color renders as a solid `backgroundColor` in either variant.
 * Returns `undefined` for transparent/malformed so callers keep their own
 * fallback (group color, surface background, or no accent).
 */
export function paneColorStyle(
  value: string | null | undefined,
  variant: "bar" | "header",
): CSSProperties | undefined {
  const { colors } = parsePaneColor(value);
  if (colors.length === 0) return undefined;
  const [c1, c2] = colors;
  if (colors.length === 1 || !c2) {
    return { backgroundColor: c1 };
  }
  if (variant === "bar") {
    return { backgroundImage: `linear-gradient(180deg, ${c1} 0 50%, ${c2} 50% 100%)` };
  }
  return {
    backgroundImage: `repeating-linear-gradient(45deg, ${c1} 0 10px, ${c2} 10px 20px)`,
  };
}
