// [REQ:P0-002e] Device silhouette geometry.
//
// One table describes each archetype's enclosure in fixed device units. Both
// the follower fitter (which needs the outer aspect and the screen aperture)
// and the silhouette components (which draw the enclosure) derive from it, so
// the bezel a follower's terminal is inset by is the same bezel that is drawn
// around it. They previously carried independent numbers.
//
// Units are arbitrary but proportional: an SVG viewBox of `width × height`
// with the default `preserveAspectRatio` renders these uniformly at any size.
// That is the whole fix for the stretched silhouette — the viewBox aspect and
// the element aspect are the same number, so nothing is squashed.
import type { DeviceArchetype } from "./deviceArchetype";

export interface DeviceGeometry {
  /** Outer panel width in device units. */
  width: number;
  /** Outer panel height in device units. The stand is drawn outside this. */
  height: number;
  /** Outer corner radius. */
  radius: number;
  /** Bezel on the left, right and top edges. */
  bezel: number;
  /** Bezel below the screen, which is thicker on every real device. */
  chin: number;
  /** Screen corner radius. */
  screenRadius: number;
  /** How the enclosure meets the desk, drawn below the panel bounds. */
  base: "none" | "wedge" | "stand";
  /** Extra device units the base occupies below the panel. */
  baseHeight: number;
}

export const DEVICE_GEOMETRY: Record<DeviceArchetype, DeviceGeometry> = {
  phone: { width: 196, height: 400, radius: 27, bezel: 9, chin: 9, screenRadius: 19, base: "none", baseHeight: 0 },
  tablet: { width: 286, height: 372, radius: 20, bezel: 14, chin: 14, screenRadius: 9, base: "none", baseHeight: 0 },
  laptop: { width: 392, height: 250, radius: 10, bezel: 9, chin: 20, screenRadius: 3, base: "wedge", baseHeight: 18 },
  monitor: { width: 392, height: 232, radius: 8, bezel: 8, chin: 24, screenRadius: 3, base: "stand", baseHeight: 44 },
  ultrawide: { width: 468, height: 190, radius: 8, bezel: 7, chin: 18, screenRadius: 3, base: "stand", baseHeight: 40 },
};

/** Screen rect in device units, for drawing. */
export function screenBox(geometry: DeviceGeometry): { x: number; y: number; width: number; height: number } {
  return {
    x: geometry.bezel,
    y: geometry.bezel,
    width: geometry.width - geometry.bezel * 2,
    height: geometry.height - geometry.bezel - geometry.chin,
  };
}


// ── Virtual keyboard plate ───────────────────────────────────────────────────
//
// The plate is sized from its own key geometry, not from whatever screen space
// the grid happened to leave over. Sizing it by leftovers made it collapse to
// three quarters of a phone screen whenever the grid was width-limited, which
// is most of the time.

export const KEYBOARD_COLUMNS = 10;
export const KEYBOARD_ROWS = 4;

/** A virtual key is taller than it is wide. Height ÷ width. */
export const KEYBOARD_KEY_ASPECT = 1.45;

/**
 * Padding above and below the key rows, expressed in key heights. It stands in
 * for the suggestion strip a real virtual keyboard carries.
 */
export const KEYBOARD_PAD_ROWS = 0.55;

/** Hard ceiling on the share of the screen the plate may claim. */
export const KEYBOARD_MAX_SCREEN_SHARE = 0.42;

const KEYBOARD_PAD_X_RATIO = 0.03;
const KEYBOARD_GAP_RATIO = 0.015;

/** Horizontal metrics, which depend only on the plate's width. */
export function keyboardColumns(width: number): { gap: number; padX: number; keyWidth: number } {
  const gap = width * KEYBOARD_GAP_RATIO;
  const padX = width * KEYBOARD_PAD_X_RATIO;
  const keyWidth = (width - padX * 2 - gap * (KEYBOARD_COLUMNS - 1)) / KEYBOARD_COLUMNS;
  return { gap, padX, keyWidth };
}

/**
 * The plate height a screen of this width wants, clamped to what the screen can
 * spare. Keys stay proportionate at their natural size and only ever get
 * flatter under the clamp — never taller and narrower, which is what read as
 * "stretched".
 */
export function keyboardPlateHeight(width: number, maxHeight: number): number {
  const { gap, keyWidth } = keyboardColumns(width);
  const keyHeight = keyWidth * KEYBOARD_KEY_ASPECT;
  const natural = KEYBOARD_ROWS * keyHeight + (KEYBOARD_ROWS - 1) * gap + KEYBOARD_PAD_ROWS * keyHeight;
  return Math.max(0, Math.min(natural, maxHeight));
}

/**
 * Invert {@link keyboardPlateHeight}: the key height that fills a plate of this
 * size. Feeding it an unclamped plate height returns the natural key height, so
 * the drawing and the fitter always agree.
 */
export function keyboardKeyHeight(width: number, height: number): number {
  const { gap } = keyboardColumns(width);
  return Math.max(0, (height - gap * (KEYBOARD_ROWS - 1)) / (KEYBOARD_ROWS + KEYBOARD_PAD_ROWS));
}
