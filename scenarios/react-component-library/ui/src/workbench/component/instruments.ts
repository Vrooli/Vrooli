/**
 * Instrument maths for the in-context view.
 *
 * Every number the checks panel shows comes from here: contrast ratios are
 * computed from computed styles, touch targets from measured boxes, overflow
 * from measured scroll widths. Nothing in this module knows React, so the
 * arithmetic can be tested against the WCAG definitions directly.
 */

export interface Rgb {
  r: number;
  g: number;
  b: number;
  a: number;
}

const HEX = /^#([0-9a-f]{3,8})$/i;
const RGB = /^rgba?\(([^)]+)\)$/i;

function clampChannel(value: number): number {
  return Math.max(0, Math.min(255, Math.round(value)));
}

function parseAlpha(raw: string | undefined): number {
  if (raw === undefined) return 1;
  const trimmed = raw.trim();
  if (trimmed.endsWith("%")) {
    const percent = Number.parseFloat(trimmed);
    return Number.isFinite(percent) ? Math.max(0, Math.min(1, percent / 100)) : 1;
  }
  const parsed = Number.parseFloat(trimmed);
  return Number.isFinite(parsed) ? Math.max(0, Math.min(1, parsed)) : 1;
}

function parseChannel(raw: string): number {
  const trimmed = raw.trim();
  if (trimmed.endsWith("%")) {
    const percent = Number.parseFloat(trimmed);
    return Number.isFinite(percent) ? clampChannel((percent / 100) * 255) : 0;
  }
  const parsed = Number.parseFloat(trimmed);
  return Number.isFinite(parsed) ? clampChannel(parsed) : 0;
}

/** Parses the colour formats a computed style can return. Null means "unknown". */
export function parseCssColor(value: string | undefined | null): Rgb | null {
  if (!value) return null;
  const input = value.trim();
  if (!input || input === "transparent") return { r: 0, g: 0, b: 0, a: 0 };

  const hex = HEX.exec(input);
  if (hex?.[1]) {
    const digits = hex[1];
    if (digits.length === 3 || digits.length === 4) {
      const [r, g, b, a] = digits.split("");
      return {
        r: Number.parseInt(`${r}${r}`, 16),
        g: Number.parseInt(`${g}${g}`, 16),
        b: Number.parseInt(`${b}${b}`, 16),
        a: a ? Number.parseInt(`${a}${a}`, 16) / 255 : 1,
      };
    }
    if (digits.length === 6 || digits.length === 8) {
      return {
        r: Number.parseInt(digits.slice(0, 2), 16),
        g: Number.parseInt(digits.slice(2, 4), 16),
        b: Number.parseInt(digits.slice(4, 6), 16),
        a: digits.length === 8 ? Number.parseInt(digits.slice(6, 8), 16) / 255 : 1,
      };
    }
    return null;
  }

  const rgb = RGB.exec(input);
  if (!rgb?.[1]) return null;
  const parts = rgb[1].split(/[/,]/).map((part) => part.trim()).filter(Boolean);
  if (parts.length < 3) return null;
  const [red = "0", green = "0", blue = "0", alpha] = parts;
  return {
    r: parseChannel(red),
    g: parseChannel(green),
    b: parseChannel(blue),
    a: parseAlpha(alpha),
  };
}

/** Flattens a translucent foreground onto an opaque background. */
export function compositeOver(foreground: Rgb, background: Rgb): Rgb {
  const a = foreground.a;
  return {
    r: clampChannel(foreground.r * a + background.r * (1 - a)),
    g: clampChannel(foreground.g * a + background.g * (1 - a)),
    b: clampChannel(foreground.b * a + background.b * (1 - a)),
    a: 1,
  };
}

/** WCAG 2.1 relative luminance. */
export function relativeLuminance(color: Rgb): number {
  const channel = (value: number) => {
    const srgb = value / 255;
    return srgb <= 0.03928 ? srgb / 12.92 : ((srgb + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * channel(color.r) + 0.7152 * channel(color.g) + 0.0722 * channel(color.b);
}

/** WCAG 2.1 contrast ratio, 1–21. */
export function contrastRatio(foreground: Rgb, background: Rgb): number {
  const back = background.a < 1 ? { ...background, a: 1 } : background;
  const front = compositeOver(foreground, back);
  const light = Math.max(relativeLuminance(front), relativeLuminance(back));
  const dark = Math.min(relativeLuminance(front), relativeLuminance(back));
  return (light + 0.05) / (dark + 0.05);
}

/** Null when either colour cannot be parsed; a wrong ratio is worse than none. */
export function contrastRatioFromCss(foreground: string, background: string): number | null {
  const front = parseCssColor(foreground);
  const back = parseCssColor(background);
  if (!front || !back || back.a === 0) return null;
  return contrastRatio(front, back);
}

export function isTouchTargetSmall(width: number, height: number, minimum = 44): boolean {
  if (!Number.isFinite(width) || !Number.isFinite(height)) return false;
  return width < minimum || height < minimum;
}

export function countSmallTargets(
  sizes: Array<{ width: number; height: number }>,
  minimum = 44,
): number {
  return sizes.filter((size) => isTouchTargetSmall(size.width, size.height, minimum)).length;
}

/** WCAG AA for body text. Large text is allowed 3:1; we score the stricter gate. */
export function contrastVerdict(ratio: number): "pass" | "warn" | "fail" {
  if (ratio >= 4.5) return "pass";
  if (ratio >= 3) return "warn";
  return "fail";
}

export function formatRatio(ratio: number): string {
  return `${ratio.toFixed(1)}:1`;
}

/** Serialises a composited (opaque) colour so it can be logged and shown. */
export function formatCssColor(color: Rgb): string {
  return `rgb(${clampChannel(color.r)}, ${clampChannel(color.g)}, ${clampChannel(color.b)})`;
}

/**
 * The background a text node actually sits on.
 *
 * A text node's own background is usually `rgba(0,0,0,0)`, and cards are often
 * painted with a translucent tint. Stopping at the first ancestor with
 * non-zero alpha would measure the text against a surface the eye never sees,
 * so every translucent layer is composited onto the first opaque ancestor,
 * nearest layer first. Null means the stack never reaches an opaque colour:
 * the ratio is unknown, and the caller must report it unmeasured.
 */
export function effectiveBackground(
  read: (element: Element) => string,
  start: Element | null,
): string | null {
  const layers: Rgb[] = [];
  let current: Element | null = start;
  while (current) {
    const parsed = parseCssColor(read(current));
    if (parsed) {
      if (parsed.a >= 1) {
        // `layers` is nearest-first, so fold from the farthest layer down.
        const composite = layers.reduceRight<Rgb>(
          (base, layer) => compositeOver(layer, base),
          parsed,
        );
        return formatCssColor(composite);
      }
      if (parsed.a > 0) layers.push(parsed);
    }
    current = current.parentElement;
  }
  return null;
}

/**
 * Converts a rect measured on screen back into the frame's logical CSS pixels.
 *
 * The frame renders its page at the viewport width and then applies one
 * `transform: scale()`. Reading `getBoundingClientRect()` therefore returns the
 * zoomed rect; a 44 px target looks 22 px at 50 % zoom and would be reported as
 * failing if the screen rect were measured directly.
 */
export function toLogicalSize(
  rect: Pick<DOMRect, "width" | "height">,
  scale: number,
): { width: number; height: number } {
  const factor = Number.isFinite(scale) && scale > 0 ? scale : 1;
  return { width: rect.width / factor, height: rect.height / factor };
}

export interface MeasuredTarget {
  element: Element;
  /** Logical size in the frame's CSS pixels, before the frame's zoom. */
  width: number;
  height: number;
  /** The on-screen rect, used to draw the outline over the zoomed page. */
  rect: DOMRect;
}

export interface TouchScan {
  total: number;
  small: number;
  targets: MeasuredTarget[];
}

/**
 * Measures interactive elements at their logical size. Null means the zoom is
 * not a usable number, so nothing was measured and the caller must not report
 * a verdict.
 */
export function measureTouchTargets(
  elements: ArrayLike<Element>,
  scale: number,
  minimum = 44,
): TouchScan | null {
  if (!Number.isFinite(scale) || scale <= 0) return null;
  const targets = Array.from(elements).map((element) => {
    const rect = element.getBoundingClientRect();
    const size = toLogicalSize(rect, scale);
    return { element, width: size.width, height: size.height, rect };
  });
  return {
    total: targets.length,
    small: targets.filter((target) => isTouchTargetSmall(target.width, target.height, minimum)).length,
    targets,
  };
}

/**
 * Overflow is read from `scrollWidth` against the page's logical width, never
 * from a scaled rect. Invalid input returns false; the check layer treats such
 * a row as unmeasured rather than trusting the default.
 */
export function overflowAt(width: number, scrollWidth: number, tolerance = 1): boolean {
  if (!Number.isFinite(width) || !Number.isFinite(scrollWidth) || width <= 0) return false;
  return scrollWidth > width + tolerance;
}

/** True when a node has text a person could actually read. */
export function hasReadableText(element: Element): boolean {
  const text = element.textContent;
  return typeof text === "string" && text.replace(/\s+/g, " ").trim().length > 0;
}

/**
 * Picks a text node for the contrast probe. The previous fallback selected the
 * first `.tpl-text` even when it was empty, which produced a ratio for a box
 * nobody can read. Text nodes are preferred; any named node with text is the
 * fallback.
 */
export function pickTextProbe(root: ParentNode): HTMLElement | null {
  const textNodes = Array.from(root.querySelectorAll<HTMLElement>(".tpl-text"));
  for (const candidate of textNodes) {
    if (hasReadableText(candidate)) return candidate;
  }
  for (const candidate of Array.from(root.querySelectorAll<HTMLElement>("[data-node-id]"))) {
    if (hasReadableText(candidate)) return candidate;
  }
  return null;
}
