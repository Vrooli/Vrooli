export const FONT_SIZE_MIN = 8;
export const FONT_SIZE_MAX = 24;
export const FONT_SIZE_STEP = 1;
const FONT_SIZE_DEFAULT = 14;

/** Clamp font size to valid range, rounding to integer. Returns 14 for NaN. */
export function clampFontSize(size: number): number {
  if (!Number.isFinite(size)) return FONT_SIZE_DEFAULT;
  return Math.min(FONT_SIZE_MAX, Math.max(FONT_SIZE_MIN, Math.round(size)));
}
