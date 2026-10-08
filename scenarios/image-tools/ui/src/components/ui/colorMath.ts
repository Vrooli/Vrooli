/**
 * Pure hex-color helpers for ColorField. Kept out of the component file so the
 * `.tsx` surface exports only the component (react-refresh fast-refresh
 * cleanliness) and so the math is unit-testable in isolation.
 *
 * Contract: a color is "" (backend default), a 6-digit `#rrggbb` at full
 * opacity, or an 8-digit `#rrggbbaa` with alpha.
 */

const FALLBACK_HEX = "#000000";

/** Split a hex value into its 6-digit base color and 0–100 alpha percent. */
export function parseColor(value: string): { base: string; alpha: number } {
  if (/^#[0-9a-fA-F]{8}$/.test(value)) {
    const base = value.slice(0, 7).toLowerCase();
    const alpha = Math.round((parseInt(value.slice(7), 16) / 255) * 100);
    return { base, alpha };
  }
  if (/^#[0-9a-fA-F]{6}$/.test(value)) {
    return { base: value.toLowerCase(), alpha: 100 };
  }
  if (/^#[0-9a-fA-F]{3}$/.test(value)) {
    const [, r, g, b] = value;
    return { base: `#${r}${r}${g}${g}${b}${b}`.toLowerCase(), alpha: 100 };
  }
  return { base: FALLBACK_HEX, alpha: 100 };
}

/** Recombine a base color + alpha percent into the emitted hex string. */
export function composeColor(base: string, alpha: number): string {
  if (alpha >= 100) {
    return base;
  }
  const a = Math.max(0, Math.min(255, Math.round((alpha / 100) * 255)));
  return `${base}${a.toString(16).padStart(2, "0")}`;
}
