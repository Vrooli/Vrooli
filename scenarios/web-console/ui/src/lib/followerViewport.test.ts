import { describe, expect, it } from "vitest";
import { DEVICE_ARCHETYPES, type DeviceArchetype } from "./deviceArchetype";
import { DEVICE_GEOMETRY, KEYBOARD_MAX_SCREEN_SHARE, screenBox } from "./deviceGeometry";
import {
	deviceControlsLane,
	fitFollowerPresentation,
  FRAME_ASPECT,
  MIN_LEGIBLE_FONT_PX,
  MIN_SILHOUETTE_HEIGHT_PX,
  CAPTION_GAP_PX,
  CAPTION_HEIGHT_PX,
  screenAperture,
  screenRadiusRatio,
  silhouetteExtent,
  surplusRatio,
} from "./followerViewport";

const CELL_ASPECT = 0.5;

function present(archetype: DeviceArchetype, cols: number, rows: number, paneWidth = 1200, paneHeight = 800, kbOpen = false) {
  return fitFollowerPresentation({ archetype, gridCols: cols, gridRows: rows, paneWidth, paneHeight, cellAspect: CELL_ASPECT, kbOpen });
}

describe("geometry is one source of truth", () => {
  it("derives the frame aspect from the drawn enclosure", () => {
    for (const archetype of DEVICE_ARCHETYPES) {
      const geometry = DEVICE_GEOMETRY[archetype];
      expect(FRAME_ASPECT[archetype]).toBeCloseTo(geometry.width / geometry.height, 6);
    }
  });

  it("derives the aperture from the same bezel the silhouette draws", () => {
    for (const archetype of DEVICE_ARCHETYPES) {
      const geometry = DEVICE_GEOMETRY[archetype];
      const box = screenBox(geometry);
      const aperture = screenAperture(archetype, "full");
      expect(aperture.x).toBeCloseTo(box.x / geometry.width, 6);
      expect(aperture.y).toBeCloseTo(box.y / geometry.height, 6);
      expect(aperture.width).toBeCloseTo(box.width / geometry.width, 6);
      expect(aperture.height).toBeCloseTo(box.height / geometry.height, 6);
    }
  });

  it("gives every archetype a portrait or landscape frame that matches its kind", () => {
    expect(FRAME_ASPECT.phone).toBeLessThan(1);
    expect(FRAME_ASPECT.tablet).toBeLessThan(1);
    expect(FRAME_ASPECT.laptop).toBeGreaterThan(1);
    expect(FRAME_ASPECT.monitor).toBeGreaterThan(FRAME_ASPECT.laptop);
    expect(FRAME_ASPECT.ultrawide).toBeGreaterThan(FRAME_ASPECT.monitor);
  });
});

describe("fitFollowerPresentation", () => {
  // The core invariant. The enclosure is a property of the device; the grid is
  // letterboxed inside it. Previously the frame aspect *was* the grid aspect,
  // so a keyboard opening reshaped the silhouette.
  it("keeps the frame aspect fixed when the grid aspect changes", () => {
    const closed = present("phone", 46, 26);
    const open = present("phone", 46, 13);
    const aspectOf = (rect: { width: number; height: number }) => rect.width / rect.height;
    expect(aspectOf(open.frame)).toBeCloseTo(aspectOf(closed.frame), 5);
    expect(aspectOf(closed.frame)).toBeCloseTo(FRAME_ASPECT.phone, 5);
    expect(open.frame.width).toBeCloseTo(closed.frame.width, 5);
    expect(open.frame.height).toBeCloseTo(closed.frame.height, 5);
  });

  it("reserves no screen space when the leader reports no keyboard", () => {
    const layout = present("phone", 46, 26);
    expect(layout.keyboardShare).toBe(0);
    // With nothing covering the screen, the grid is centred in the aperture.
    const gapAbove = layout.screen.y - layout.aperture.y;
    const gapBelow = layout.aperture.y + layout.aperture.height - (layout.screen.y + layout.screen.height);
    expect(gapAbove).toBeCloseTo(gapBelow, 5);
  });

  // The plate used to be sized by whatever the grid did not use, so a
  // width-limited grid — the normal case — left it covering three quarters of
  // the screen. It is now sized from its own key geometry.
  it("sizes the keyboard from its keys, not from what the grid left over", () => {
    const wideGrid = present("phone", 46, 13, 1200, 800, true);
    const tallGrid = present("phone", 46, 40, 1200, 800, true);
    expect(wideGrid.keyboardShare).toBeCloseTo(tallGrid.keyboardShare, 5);
    expect(wideGrid.keyboardShare).toBeGreaterThan(0.15);
    expect(wideGrid.keyboardShare).toBeLessThan(0.35);
  });

  it("never lets the keyboard claim more than its share of any screen", () => {
    for (const archetype of DEVICE_ARCHETYPES) {
      const layout = present(archetype, 100, 30, 1200, 800, true);
      expect(layout.keyboardShare).toBeGreaterThan(0);
      expect(layout.keyboardShare).toBeLessThanOrEqual(KEYBOARD_MAX_SCREEN_SHARE + 1e-9);
    }
  });

  it("keeps the grid clear of the keyboard it reserved", () => {
    const layout = present("phone", 46, 13, 1200, 800, true);
    const plateTop = layout.aperture.y + layout.aperture.height * (1 - layout.keyboardShare);
    expect(layout.screen.y + layout.screen.height).toBeLessThanOrEqual(plateTop + 1e-6);
  });

  // The terminal is clipped to the aperture at render time, but it must fit
  // inside it geometrically too, or the clip would be silently cropping cells.
  it.each([[false], [true]])("keeps the grid inside the screen opening (keyboard=%s)", (kbOpen) => {
    for (const archetype of DEVICE_ARCHETYPES) {
      const { aperture, screen, frame } = present(archetype, 100, 30, 1200, 800, kbOpen);
      expect(screen.x).toBeGreaterThanOrEqual(aperture.x - 1e-6);
      expect(screen.y).toBeGreaterThanOrEqual(aperture.y - 1e-6);
      expect(screen.x + screen.width).toBeLessThanOrEqual(aperture.x + aperture.width + 1e-6);
      expect(screen.y + screen.height).toBeLessThanOrEqual(aperture.y + aperture.height + 1e-6);
      // And the opening itself is inside the enclosure.
      expect(aperture.x).toBeGreaterThanOrEqual(frame.x - 1e-6);
      expect(aperture.x + aperture.width).toBeLessThanOrEqual(frame.x + frame.width + 1e-6);
      expect(aperture.y + aperture.height).toBeLessThanOrEqual(frame.y + frame.height + 1e-6);
    }
  });

  it("reports a clip radius matching the curve the silhouette draws", () => {
    for (const archetype of DEVICE_ARCHETYPES) {
      const layout = present(archetype, 100, 30);
      expect(layout.aperture.radius).toBeCloseTo(screenRadiusRatio(archetype) * layout.frame.height, 5);
    }
  });

  it("never distorts the character cell", () => {
    for (const archetype of DEVICE_ARCHETYPES) {
      const { screen } = present(archetype, 80, 24);
      expect(screen.width / screen.height).toBeCloseTo((80 * CELL_ASPECT) / 24, 4);
    }
  });

  // A stand is drawn below the panel and scales with it, so the whole drawing
  // — panel, stand and caption — has to fit. Measuring only the panel and a
  // fixed lane passed happily while a monitor's foot hung out of the pane.
  it.each([[1200, 700], [900, 500], [1600, 900], [640, 380]] as const)(
    "keeps the whole silhouette and its caption inside a %ix%i pane",
    (paneWidth, paneHeight) => {
      for (const archetype of DEVICE_ARCHETYPES) {
        const layout = present(archetype, 100, 30, paneWidth, paneHeight);
        expect(layout.frame.x).toBeGreaterThanOrEqual(0);
        expect(layout.frame.y).toBeGreaterThanOrEqual(0);
        expect(layout.frame.x + layout.frame.width).toBeLessThanOrEqual(paneWidth + 0.001);

        const drawnBottom = layout.frame.y + layout.frame.height * silhouetteExtent(archetype, layout.tier);
        const captionTop = layout.frame.y + layout.frame.height + layout.captionOffset;
        const captionBottom = captionTop + CAPTION_HEIGHT_PX;

        // Nothing drawn may leave the pane, caption included.
        expect(drawnBottom).toBeLessThanOrEqual(paneHeight + 0.001);
        expect(captionBottom).toBeLessThanOrEqual(paneHeight + 0.001);
        // And the caption must sit below the stand, not on top of it.
        expect(captionTop).toBeGreaterThanOrEqual(drawnBottom - 0.001);
      }
    },
  );

  it("scales the stand allowance with the panel instead of assuming one", () => {
    // The same archetype in two pane sizes: the offset must track the panel.
    const small = present("monitor", 100, 30, 900, 500);
    const large = present("monitor", 100, 30, 1600, 900);
    expect(large.frame.height).toBeGreaterThan(small.frame.height);
    const ratio = large.frame.height / small.frame.height;
    expect((large.captionOffset - CAPTION_GAP_PX) / (small.captionOffset - CAPTION_GAP_PX)).toBeCloseTo(ratio, 4);
  });

  it("asks no clearance of an archetype that draws no stand", () => {
    for (const archetype of DEVICE_ARCHETYPES) {
      const layout = present(archetype, 100, 30);
      const expected = DEVICE_GEOMETRY[archetype].base === "none";
      expect(silhouetteExtent(archetype, layout.tier) === 1).toBe(expected);
      if (expected) expect(layout.captionOffset).toBeCloseTo(CAPTION_GAP_PX, 6);
    }
  });

  it("reserves a control lane for a tall phone in a short pane", () => {
    const paneHeight = 360;
    const layout = present("phone", 45, 90, 1200, paneHeight);
    expect(layout.frame.y + layout.frame.height + deviceControlsLane("phone", layout.tier)).toBeLessThanOrEqual(paneHeight + 0.001);
  });

  it("degrades to the caption strip rather than drawing an unreadable silhouette", () => {
    const layout = present("phone", 80, 24, 160, 110);
    expect(layout.tier).toBe("strip");
    expect(deviceControlsLane("phone", "strip")).toBe(0);
  });

  it("draws a silhouette whenever there is room for one", () => {
    const layout = present("monitor", 100, 30, 1200, 800);
    expect(layout.tier).not.toBe("strip");
    expect(layout.frame.height).toBeGreaterThanOrEqual(MIN_SILHOUETTE_HEIGHT_PX);
  });

  it("keeps the font legible or scales the surface instead", () => {
    for (const archetype of DEVICE_ARCHETYPES) {
      const layout = present(archetype, 240, 80, 400, 300);
      expect(layout.screen.fontSize).toBeGreaterThanOrEqual(MIN_LEGIBLE_FONT_PX);
      expect(layout.screen.scale).toBeGreaterThan(0);
      expect(layout.screen.scale).toBeLessThanOrEqual(1);
    }
  });
});
