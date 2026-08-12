import { contrastRatioRgb, hexToRgb, isLightColor } from "./paneColor";
const CYAN_HUE = 195;
const MIN_L = 0.02;
const MAX_L = 0.98;
export const CHROME_PALETTE_TOKEN_NAMES = [
    "--wc-surface-base",
    "--wc-surface-raised",
    "--wc-surface-input",
    "--wc-surface-header",
    "--wc-text-primary",
    "--wc-text-secondary",
    "--wc-text-muted",
    "--wc-text-faint",
    "--wc-border-default",
    "--wc-border-hover",
    "--wc-accent",
    "--wc-accent-fg",
    "--wc-accent-border",
    "--wc-accent-active",
];
function clamp(value, min, max) {
    return Math.min(max, Math.max(min, value));
}
function srgbToLinear(channel) {
    const c = channel / 255;
    return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
}
function linearToSrgb(channel) {
    const c = clamp(channel, 0, 1);
    const s = c <= 0.0031308 ? 12.92 * c : 1.055 * Math.pow(c, 1 / 2.4) - 0.055;
    return Math.round(clamp(s, 0, 1) * 255);
}
export function rgbToOklch(rgb) {
    const r = srgbToLinear(rgb.r);
    const g = srgbToLinear(rgb.g);
    const b = srgbToLinear(rgb.b);
    const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b);
    const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b);
    const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);
    const okL = 0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s;
    const okA = 1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s;
    const okB = 0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s;
    const c = Math.sqrt(okA * okA + okB * okB);
    const h = ((Math.atan2(okB, okA) * 180) / Math.PI + 360) % 360;
    return { l: okL, c, h };
}
export function oklchToRgb(oklch) {
    const a = Math.cos((oklch.h * Math.PI) / 180) * oklch.c;
    const b = Math.sin((oklch.h * Math.PI) / 180) * oklch.c;
    const lPrime = oklch.l + 0.3963377774 * a + 0.2158037573 * b;
    const mPrime = oklch.l - 0.1055613458 * a - 0.0638541728 * b;
    const sPrime = oklch.l - 0.0894841775 * a - 1.291485548 * b;
    const l = lPrime * lPrime * lPrime;
    const m = mPrime * mPrime * mPrime;
    const s = sPrime * sPrime * sPrime;
    return {
        r: linearToSrgb(+4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s),
        g: linearToSrgb(-1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s),
        b: linearToSrgb(-0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s),
    };
}
function rgbToTriple(rgb) {
    return `${rgb.r} ${rgb.g} ${rgb.b}`;
}
function rgbToHex(rgb) {
    const channel = (value) => value.toString(16).padStart(2, "0");
    return `#${channel(rgb.r)}${channel(rgb.g)}${channel(rgb.b)}`;
}
function tone(seed, lightness, chromaScale = 0.55) {
    return oklchToRgb({
        l: clamp(lightness, MIN_L, MAX_L),
        c: Math.min(seed.c * chromaScale, 0.08),
        h: seed.h,
    });
}
function textTone(surface, seed, target, polarity, baseL) {
    let best = tone(seed, baseL, 0.12);
    for (let i = 0; i <= 90; i += 1) {
        const l = polarity === "light" ? clamp(baseL + i * 0.006, MIN_L, MAX_L) : clamp(baseL - i * 0.006, MIN_L, MAX_L);
        const candidate = tone(seed, l, 0.12);
        best = candidate;
        if (contrastRatioRgb(candidate, surface) >= target)
            return candidate;
    }
    return best;
}
function accentTone(surface, polarity) {
    let best = oklchToRgb({ l: polarity === "light" ? 0.78 : 0.42, c: 0.12, h: CYAN_HUE });
    for (let i = 0; i <= 80; i += 1) {
        const l = polarity === "light" ? clamp(0.72 + i * 0.004, MIN_L, MAX_L) : clamp(0.48 - i * 0.004, MIN_L, MAX_L);
        for (const c of [0.12, 0.1, 0.08, 0.06]) {
            const candidate = oklchToRgb({ l, c, h: CYAN_HUE });
            best = candidate;
            if (contrastRatioRgb(candidate, surface) >= 3)
                return candidate;
        }
    }
    return best;
}
function readableTextPair(background) {
    const dark = { r: 15, g: 23, b: 42 };
    const light = { r: 248, g: 250, b: 252 };
    return contrastRatioRgb(dark, background) >= contrastRatioRgb(light, background) ? dark : light;
}
export function deriveChromePalette(seedHex) {
    const seedRgb = hexToRgb(seedHex) ?? { r: 15, g: 23, b: 42 };
    const seed = rgbToOklch(seedRgb);
    const lightSeed = isLightColor(seedHex);
    const raised = tone(seed, seed.l, 0.7);
    const base = lightSeed ? tone(seed, seed.l - 0.12, 0.5) : tone(seed, seed.l - 0.08, 0.5);
    const input = lightSeed ? tone(seed, seed.l - 0.2, 0.45) : tone(seed, seed.l + 0.1, 0.45);
    const polarity = lightSeed ? "dark" : "light";
    const primary = textTone(base, seed, 4.5, polarity, lightSeed ? 0.16 : 0.9);
    const secondary = textTone(base, seed, 4.5, polarity, lightSeed ? 0.22 : 0.82);
    const muted = textTone(raised, seed, 3, polarity, lightSeed ? 0.34 : 0.68);
    const faint = textTone(input, seed, 3, polarity, lightSeed ? 0.42 : 0.58);
    const accent = accentTone(raised, polarity);
    const accentActive = lightSeed
        ? oklchToRgb({ l: 0.76, c: 0.09, h: CYAN_HUE })
        : oklchToRgb({ l: 0.42, c: 0.1, h: CYAN_HUE });
    return {
        "--wc-surface-base": rgbToTriple(base),
        "--wc-surface-raised": rgbToTriple(raised),
        "--wc-surface-input": rgbToTriple(input),
        "--wc-surface-header": rgbToTriple(raised),
        "--wc-text-primary": rgbToTriple(primary),
        "--wc-text-secondary": rgbToTriple(secondary),
        "--wc-text-muted": rgbToTriple(muted),
        "--wc-text-faint": rgbToTriple(faint),
        "--wc-border-default": `${rgbToTriple(muted)} / 0.26`,
        "--wc-border-hover": `${rgbToTriple(primary)} / 0.42`,
        "--wc-accent": rgbToTriple(accent),
        "--wc-accent-fg": rgbToTriple(readableTextPair(accent)),
        "--wc-accent-border": `${rgbToTriple(accent)} / 0.46`,
        "--wc-accent-active": `${rgbToTriple(accentActive)} / 0.34`,
    };
}
export function paletteTripleToHex(value) {
    const [r = 0, g = 0, b = 0] = value.split(/\s+/).map((part) => Number.parseInt(part, 10));
    return rgbToHex({ r, g, b });
}
