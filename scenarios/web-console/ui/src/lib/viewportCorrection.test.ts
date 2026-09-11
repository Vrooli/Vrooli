import { describe, expect, it } from "vitest";
import { correctViewport, type ScreenFacts } from "./viewportCorrection";

const phone = {
  layoutWidth: 393,
  layoutHeight: 793,
  visibleWidth: 393,
  visibleHeight: 793,
  offsetLeft: 0,
  offsetTop: 0,
  scale: 1,
  keyboardInset: 0,
  keyboardVisible: false,
};

// An installed app on a 393×852 phone whose status bar is 59px tall.
const installed: ScreenFacts = { standalone: true, screenWidth: 393, screenHeight: 852, safeTop: 59 };

describe("correctViewport", () => {
  it("extends an installed app whose viewport stops short of the screen by exactly the status-bar inset to the real bottom edge, and says it did", () => {
    const { viewport, reachesBottom, extended } = correctViewport(phone, installed);
    expect(viewport.visibleHeight).toBe(852);
    expect(viewport.layoutHeight).toBe(852);
    expect(reachesBottom).toBe(true);
    expect(extended).toBe(true);
  });

  it("leaves a correct installed viewport alone and says it reaches the bottom", () => {
    const full = { ...phone, layoutHeight: 852, visibleHeight: 852 };
    expect(correctViewport(full, installed)).toEqual({ viewport: full, reachesBottom: true, extended: false });
  });

  it("with the keyboard up, keeps the measured height and clears the bottom inset — the keyboard covers the home indicator", () => {
    const typing = { ...phone, visibleHeight: 450, keyboardInset: 343, keyboardVisible: true };
    expect(correctViewport(typing, installed)).toEqual({ viewport: typing, reachesBottom: false, extended: false });
  });

  it("trusts a browser tab's viewport, whose bottom belongs to the browser", () => {
    const tab = { ...phone, visibleHeight: 700 };
    expect(correctViewport(tab, { ...installed, standalone: false })).toEqual({ viewport: tab, reachesBottom: true, extended: false });
  });

  it("does not stretch over a gap it cannot explain, and then reserves no bottom inset", () => {
    const short = { ...phone, visibleHeight: 760 };
    expect(correctViewport(short, installed)).toEqual({ viewport: short, reachesBottom: false, extended: false });
  });

  it("trusts a windowed app that is narrower than the screen", () => {
    const windowed = { ...phone, visibleWidth: 320 };
    expect(correctViewport(windowed, installed)).toEqual({ viewport: windowed, reachesBottom: true, extended: false });
  });

  it("reads the screen in the current orientation", () => {
    const landscape = { ...phone, layoutWidth: 852, visibleWidth: 852, layoutHeight: 334, visibleHeight: 334 };
    const { viewport } = correctViewport(landscape, installed);
    expect(viewport.visibleHeight).toBe(393);
  });

  it("is idempotent: a snapshot corrected once passes through unchanged and is not extended twice", () => {
    const once = correctViewport(phone, installed);
    expect(correctViewport(once.viewport, installed)).toEqual({ viewport: once.viewport, reachesBottom: true, extended: false });
  });
});
