import { describe, expect, it } from "vitest";

import {
  compositeOver,
  contrastRatio,
  contrastRatioFromCss,
  contrastVerdict,
  countSmallTargets,
  effectiveBackground,
  formatRatio,
  hasReadableText,
  isTouchTargetSmall,
  measureTouchTargets,
  overflowAt,
  parseCssColor,
  pickTextProbe,
  relativeLuminance,
  toLogicalSize,
  type Rgb,
} from "./instruments";

describe("colour parsing", () => {
  it("reads hex, rgb and rgba values", () => {
    expect(parseCssColor("#fff")).toEqual({ r: 255, g: 255, b: 255, a: 1 });
    expect(parseCssColor("#0c0e12")).toEqual({ r: 12, g: 14, b: 18, a: 1 });
    expect(parseCssColor("rgb(12, 14, 18)")).toEqual({ r: 12, g: 14, b: 18, a: 1 });
    expect(parseCssColor("rgba(0, 0, 0, 0.5)")).toEqual({ r: 0, g: 0, b: 0, a: 0.5 });
    expect(parseCssColor("transparent")).toEqual({ r: 0, g: 0, b: 0, a: 0 });
    expect(parseCssColor("not-a-colour")).toBeNull();
  });
});

describe("contrast", () => {
  const colour = (value: string): Rgb => {
    const parsed = parseCssColor(value);
    if (!parsed) throw new Error(`unreadable colour ${value}`);
    return parsed;
  };

  it("matches the WCAG extremes", () => {
    const black = colour("#000");
    const white = colour("#fff");
    expect(relativeLuminance(black)).toBe(0);
    expect(relativeLuminance(white)).toBeCloseTo(1, 5);
    expect(contrastRatio(black, white)).toBeCloseTo(21, 5);
    expect(contrastRatio(white, white)).toBeCloseTo(1, 5);
  });

  it("computes a known mid grey and returns null for unreadable input", () => {
    const ratio = contrastRatioFromCss("rgb(119, 119, 119)", "#ffffff");
    expect(ratio).not.toBeNull();
    expect(ratio ?? 0).toBeGreaterThan(4.4);
    expect(ratio ?? 99).toBeLessThan(4.6);
    expect(contrastRatioFromCss("var(--nope)", "#fff")).toBeNull();
    expect(contrastRatioFromCss("#fff", "transparent")).toBeNull();
  });

  it("composites a translucent foreground before measuring", () => {
    const halfBlack = colour("rgba(0, 0, 0, 0.5)");
    const white = colour("#fff");
    expect(compositeOver(halfBlack, white).r).toBe(128);
    expect(contrastRatio(halfBlack, white)).toBeGreaterThan(3);
  });

  it("bands AA verdicts and formats the ratio", () => {
    expect(contrastVerdict(7)).toBe("pass");
    expect(contrastVerdict(4.5)).toBe("pass");
    expect(contrastVerdict(3.9)).toBe("warn");
    expect(contrastVerdict(2.9)).toBe("fail");
    expect(formatRatio(8.44)).toBe("8.4:1");
  });

  it("walks up to the first opaque ancestor background", () => {
    const page = document.createElement("div");
    page.style.backgroundColor = "#0c0e12";
    const card = document.createElement("div");
    const text = document.createElement("span");
    page.append(card);
    card.append(text);

    const read = (element: Element) => getComputedStyle(element).backgroundColor;
    expect(effectiveBackground(read, text)).toBe("rgb(12, 14, 18)");

    const detached = document.createElement("span");
    expect(effectiveBackground(read, detached)).toBeNull();
    expect(effectiveBackground(read, null)).toBeNull();
  });

  it("skips transparent ancestors and blends translucent ones onto the surface", () => {
    const page = document.createElement("div");
    page.style.backgroundColor = "#000";
    const transparent = document.createElement("div");
    transparent.style.backgroundColor = "rgba(0, 0, 0, 0)";
    const tint = document.createElement("div");
    tint.style.backgroundColor = "rgba(255, 255, 255, 0.5)";
    const text = document.createElement("span");
    page.append(transparent);
    transparent.append(tint);
    tint.append(text);

    const read = (element: Element) => getComputedStyle(element).backgroundColor;
    // Half white over black is mid grey, not white and not black.
    expect(effectiveBackground(read, text)).toBe("rgb(128, 128, 128)");
    const ratio = contrastRatioFromCss("#fff", effectiveBackground(read, text) ?? "");
    expect(ratio).not.toBeNull();
    expect(ratio ?? 0).toBeGreaterThan(1);
    expect(ratio ?? 21).toBeLessThan(21);
  });

  it("blends a stack of translucent layers nearest-first", () => {
    const page = document.createElement("div");
    page.style.backgroundColor = "#000";
    const outer = document.createElement("div");
    outer.style.backgroundColor = "rgba(255, 255, 255, 0.5)";
    const inner = document.createElement("div");
    inner.style.backgroundColor = "rgba(255, 255, 255, 0.5)";
    const text = document.createElement("span");
    page.append(outer);
    outer.append(inner);
    inner.append(text);

    const read = (element: Element) => getComputedStyle(element).backgroundColor;
    // 255 * 0.5 = 128 over the first layer, then 255*0.5 + 128*0.5 = 192.
    expect(effectiveBackground(read, text)).toBe("rgb(192, 192, 192)");
  });

  it("returns null when the stack never reaches an opaque colour", () => {
    const outer = document.createElement("div");
    outer.style.backgroundColor = "rgba(255, 255, 255, 0.4)";
    const text = document.createElement("span");
    outer.append(text);
    const read = (element: Element) => getComputedStyle(element).backgroundColor;
    expect(effectiveBackground(read, text)).toBeNull();
  });
});

describe("touch targets", () => {
  it("uses the 44 px floor on either axis", () => {
    expect(isTouchTargetSmall(43, 60)).toBe(true);
    expect(isTouchTargetSmall(60, 43)).toBe(true);
    expect(isTouchTargetSmall(44, 44)).toBe(false);
    expect(isTouchTargetSmall(120, 48)).toBe(false);
  });

  it("counts offenders in a measured list", () => {
    expect(
      countSmallTargets([
        { width: 120, height: 44 },
        { width: 24, height: 24 },
        { width: 48, height: 32 },
      ]),
    ).toBe(2);
  });

  it("measures the rendered rect in the frame's logical pixels, not the zoomed screen rect", () => {
    const scaled = document.createElement("button");
    scaled.getBoundingClientRect = () =>
      ({ width: 22, height: 22, top: 0, left: 0, right: 22, bottom: 22, x: 0, y: 0 }) as DOMRect;

    // A 44 × 44 logical target at 50 % zoom is a 22 × 22 screen rect.
    expect(toLogicalSize({ width: 22, height: 22 }, 0.5)).toEqual({ width: 44, height: 44 });
    const scan = measureTouchTargets([scaled], 0.5);
    expect(scan).toMatchObject({ total: 1, small: 0 });

    // Measuring the screen rect would flag the same target.
    expect(countSmallTargets([{ width: 22, height: 22 }])).toBe(1);

    const tiny = document.createElement("button");
    tiny.getBoundingClientRect = () =>
      ({ width: 16, height: 16, top: 0, left: 0, right: 16, bottom: 16, x: 0, y: 0 }) as DOMRect;
    expect(measureTouchTargets([tiny], 0.5)).toMatchObject({ total: 1, small: 1 });
  });

  it("refuses to scan when the zoom is not a usable number", () => {
    const button = document.createElement("button");
    expect(measureTouchTargets([button], 0)).toBeNull();
    expect(measureTouchTargets([button], Number.NaN)).toBeNull();
  });
});

describe("overflow and the text probe", () => {
  it("compares scrollWidth to the logical width", () => {
    expect(overflowAt(390, 390)).toBe(false);
    expect(overflowAt(390, 391)).toBe(false);
    expect(overflowAt(390, 394)).toBe(true);
    expect(overflowAt(390, Number.NaN)).toBe(false);
    expect(overflowAt(0, 400)).toBe(false);
  });

  it("picks a text node that has text and ignores empty ones", () => {
    const root = document.createElement("div");
    const empty = document.createElement("div");
    empty.className = "tpl-text is-display";
    empty.dataset.nodeId = "empty";
    empty.textContent = "   ";
    const spacer = document.createElement("div");
    spacer.dataset.nodeId = "spacer";
    const body = document.createElement("div");
    body.className = "tpl-text is-body";
    body.dataset.nodeId = "body";
    body.textContent = "Revenue this month";
    root.append(empty, spacer, body);

    expect(hasReadableText(empty)).toBe(false);
    expect(pickTextProbe(root)?.dataset.nodeId).toBe("body");
    expect(pickTextProbe(document.createElement("div"))).toBeNull();
  });

  it("falls back to any named node with text", () => {
    const root = document.createElement("div");
    const card = document.createElement("section");
    card.dataset.nodeId = "card";
    card.textContent = "Card copy";
    root.append(card);
    expect(pickTextProbe(root)?.dataset.nodeId).toBe("card");
  });
});
