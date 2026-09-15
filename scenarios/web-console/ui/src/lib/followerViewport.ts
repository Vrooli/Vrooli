// [REQ:P0-002e] Multi-Device Follower Presentation
import { DEVICE_ARCHETYPES, type DeviceArchetype } from "./deviceArchetype";
import { DEVICE_GEOMETRY, KEYBOARD_MAX_SCREEN_SHARE, keyboardPlateHeight, screenBox } from "./deviceGeometry";

export const MIN_LEGIBLE_FONT_PX = 9;

/**
 * Below this frame height the silhouette has no room to read as a device, so
 * the presentation degrades to the caption strip instead.
 */
export const MIN_SILHOUETTE_HEIGHT_PX = 140;

/**
 * Surplus is the share of the pane the fitted frame does not occupy. Below
 * {@link FULL_CHROME_MAX_SURPLUS} the frame has ample room for a silhouette;
 * above {@link STRIP_MIN_SURPLUS} it has so little that only a caption fits.
 */
export const FULL_CHROME_MAX_SURPLUS = 0.6;
export const STRIP_MIN_SURPLUS = 0.95;

/** Gap between the lowest point of the silhouette and its caption. */
export const CAPTION_GAP_PX = 8;

/** Height of the caption chip itself. */
export const CAPTION_HEIGHT_PX = 36;

/** Height kept free below the silhouette for the label and Take over control. */
export const CONTROLS_LANE_PX = CAPTION_GAP_PX + CAPTION_HEIGHT_PX;

/**
 * Cell aspect assumed when xterm has not been measured yet. Deliberately
 * conservative: it is only used for the first frame before a real measurement
 * arrives, and a wrong guess must not pick a wildly wrong silhouette.
 */
export const FALLBACK_CELL_ASPECT = 0.5;

export type FollowerRect = { x: number; y: number; width: number; height: number; fontSize: number; scale: number };
export type ChromeTier = "full" | "hairline" | "strip";
export type ScreenAperture = { x: number; y: number; width: number; height: number };

/**
 * The device's screen opening in pane pixels, plus the corner radius it is
 * drawn with. Everything presented as being "on the leader's screen" is clipped
 * to this shape, so nothing overhangs the bezel.
 */
export type ApertureRect = { x: number; y: number; width: number; height: number; radius: number };

/**
 * Outer silhouette aspect (width ÷ height) per archetype, derived from the one
 * geometry table the silhouettes are drawn from.
 *
 * This is the change that stops a phone morphing into a laptop: the frame's
 * proportions are a property of the *device*, not of the grid it is currently
 * showing. The grid is letterboxed inside the screen aperture, and whatever
 * room it does not use is drawn as the leader's state — a keyboard, typically.
 */
export const FRAME_ASPECT: Record<DeviceArchetype, number> = Object.fromEntries(
  DEVICE_ARCHETYPES.map((archetype) => {
    const geometry = DEVICE_GEOMETRY[archetype];
    return [archetype, geometry.width / geometry.height];
  }),
) as Record<DeviceArchetype, number>;

/** Height kept free for the follower label and Take over control. */
export function deviceControlsLane(archetype: DeviceArchetype, tier: ChromeTier): number {
  return tier === "strip" ? 0 : CONTROLS_LANE_PX;
}

/**
 * Total drawn height of the silhouette as a multiple of its panel height.
 *
 * A monitor's neck and foot, and a laptop's base, are drawn *below* the panel
 * bounds and scale with it — a monitor panel 636px tall carries a 121px stand.
 * Reserving a fixed lane for that, as an earlier version did, worked at one
 * size and put the caption on top of the stand at every other.
 */
export function silhouetteExtent(archetype: DeviceArchetype, tier: ChromeTier): number {
  if (tier === "strip") return 1;
  const geometry = DEVICE_GEOMETRY[archetype];
  return 1 + geometry.baseHeight / geometry.height;
}

/**
 * Screen corner radius as a share of the panel height, so a caller in pane
 * pixels can reproduce the drawn curve without knowing device units.
 */
export function screenRadiusRatio(archetype: DeviceArchetype): number {
  const geometry = DEVICE_GEOMETRY[archetype];
  return geometry.screenRadius / geometry.height;
}

/** Archetypes whose silhouette extends below its panel bounds. */
export function hasStand(archetype: DeviceArchetype): boolean {
  return DEVICE_GEOMETRY[archetype].base !== "none";
}

// The terminal must fit the visible display, not the frame's outer bezel.
// Values are normalized to the outer silhouette and leave room for rounded
// corners, bezel, and the monitor/laptop base below the display.
export function screenAperture(archetype: DeviceArchetype, tier: ChromeTier): ScreenAperture {
  if (tier === "strip") return { x: 0, y: 0, width: 1, height: 1 };
  const geometry = DEVICE_GEOMETRY[archetype];
  const box = screenBox(geometry);
  return {
    x: box.x / geometry.width,
    y: box.y / geometry.height,
    width: box.width / geometry.width,
    height: box.height / geometry.height,
  };
}

/**
 * Fit the leader's grid inside a device silhouette whose proportions are fixed
 * by the archetype.
 *
 * The grid is letterboxed inside the aperture and always centred horizontally.
 * Vertically it is centred by default, and anchored to the top when the leader
 * reports a keyboard — the one case where the leftover space is not
 * incidental letterboxing but a specific thing covering the bottom of the
 * leader's screen, which the silhouette then draws.
 */
export function fitDeviceGrid(
  gridCols: number,
  gridRows: number,
  paneWidth: number,
  paneHeight: number,
  cellAspect: number,
  aperture: ScreenAperture,
  frameAspect: number,
  options: { kbOpen?: boolean; screenRadiusRatio?: number; verticalExtent?: number; captionLane?: number } = {},
): { frame: FollowerRect; screen: FollowerRect; aperture: ApertureRect; keyboardShare: number; captionOffset: number } {
  // The silhouette may draw below its panel — a stand, a laptop base — and that
  // overhang scales with the panel. Size the panel so the *whole* drawing plus
  // its caption fits the pane, rather than sizing the panel and hoping.
  const extent = options.verticalExtent ?? 1;
  const captionLane = options.captionLane ?? 0;
  const maxFrameHeight = Math.max(1, (paneHeight - captionLane) / extent);

  let width = Math.min(paneWidth, maxFrameHeight * frameAspect);
  let height = width / frameAspect;
  if (height > maxFrameHeight) { height = maxFrameHeight; width = height * frameAspect; }

  // Centre the drawn block — panel, overhang and caption — in the pane.
  const blockHeight = height * extent + captionLane;
  const frameX = (paneWidth - width) / 2;
  const frameY = Math.max(0, (paneHeight - blockHeight) / 2);

  const apertureWidth = width * aperture.width;
  const apertureHeight = height * aperture.height;
  const apertureX = frameX + width * aperture.x;
  const apertureY = frameY + height * aperture.y;

  // Reserve the keyboard first, from its own key geometry. The grid then gets
  // what is left, which is the same order of operations the leader's own
  // layout performs.
  const plateHeight = options.kbOpen
    ? keyboardPlateHeight(apertureWidth, apertureHeight * KEYBOARD_MAX_SCREEN_SHARE)
    : 0;
  const usableHeight = Math.max(1, apertureHeight - plateHeight);

  // Letterbox the grid into what remains, without distorting its cells.
  const gridAspect = (gridCols * cellAspect) / gridRows;
  let screenWidth = Math.min(apertureWidth, usableHeight * gridAspect);
  let screenHeight = screenWidth / gridAspect;
  if (screenHeight > usableHeight) { screenHeight = usableHeight; screenWidth = screenHeight * gridAspect; }

  const calculated = screenWidth / (gridCols * cellAspect);
  const fontSize = Math.max(MIN_LEGIBLE_FONT_PX, calculated);
  const scale = calculated >= MIN_LEGIBLE_FONT_PX ? 1 : calculated / MIN_LEGIBLE_FONT_PX;

  const frame = { x: frameX, y: frameY, width, height, fontSize, scale };
  const screen = {
    // Centred in the region the keyboard is not using, which puts it at the
    // middle of the screen when there is no keyboard and above the plate when
    // there is.
    x: apertureX + (apertureWidth - screenWidth) / 2,
    y: apertureY + (usableHeight - screenHeight) / 2,
    width: screenWidth,
    height: screenHeight,
    fontSize,
    scale,
  };
  return {
    frame,
    screen,
    aperture: {
      x: apertureX,
      y: apertureY,
      width: apertureWidth,
      height: apertureHeight,
      // The ratio is the drawn radius over the drawn panel height, so scaling
      // it by the frame height reproduces exactly the curve the SVG renders.
      radius: (options.screenRadiusRatio ?? 0) * height,
    },
    keyboardShare: apertureHeight > 0 ? plateHeight / apertureHeight : 0,
    // Distance from the panel's bottom edge to the caption: clears the stand.
    captionOffset: height * (extent - 1) + CAPTION_GAP_PX,
  };
}

/**
 * The full presentation decision for one follower pane: which archetype's
 * silhouette, at which chrome tier, occupying which rects.
 *
 * The tier is decided from the geometry that will actually be drawn. An
 * earlier version measured a bare grid fit, then drew different geometry, then
 * patched the mismatch with a magic minimum height afterwards.
 */
export function fitFollowerPresentation(options: {
  archetype: DeviceArchetype;
  gridCols: number;
  gridRows: number;
  paneWidth: number;
  paneHeight: number;
  cellAspect: number;
  /** The leader's virtual keyboard covers the bottom of its screen. */
  kbOpen?: boolean;
}): { frame: FollowerRect; screen: FollowerRect; aperture: ApertureRect; tier: ChromeTier; keyboardShare: number; captionOffset: number } {
  const { archetype, gridCols, gridRows, paneWidth, paneHeight, cellAspect, kbOpen } = options;
  const frameAspect = FRAME_ASPECT[archetype];

  const layoutFor = (tier: ChromeTier) => {
    const aspect = tier === "strip" ? (gridCols * cellAspect) / gridRows : frameAspect;
    return fitDeviceGrid(gridCols, gridRows, paneWidth, paneHeight, cellAspect, screenAperture(archetype, tier), aspect, {
      kbOpen,
      // The strip tier draws no enclosure, so it has no rounded screen to clip to.
      screenRadiusRatio: tier === "strip" ? 0 : screenRadiusRatio(archetype),
      verticalExtent: silhouetteExtent(archetype, tier),
      captionLane: deviceControlsLane(archetype, tier),
    });
  };

  const silhouette = layoutFor("full");
  const surplus = surplusRatio(silhouette.frame, paneWidth, paneHeight);
  const tooSmall = silhouette.frame.height < MIN_SILHOUETTE_HEIGHT_PX;

  if (tooSmall || surplus >= STRIP_MIN_SURPLUS) {
    const strip = layoutFor("strip");
    return { ...strip, tier: "strip" };
  }
  return { ...silhouette, tier: surplus < FULL_CHROME_MAX_SURPLUS ? "full" : "hairline" };
}

export function surplusRatio(rect: Pick<FollowerRect, "width" | "height">, paneWidth: number, paneHeight: number): number {
  if (paneWidth <= 0 || paneHeight <= 0) return 1;
  return 1 - (rect.width * rect.height) / (paneWidth * paneHeight);
}
