// Pure, deterministic transform math for the Mermaid zoom/pan surface. Keeping
// these free of DOM/React makes the zoom behavior unit-testable in isolation.
export const MIN_SCALE = 0.1;
export const MAX_SCALE = 8;
/** Cap the initial fit so a tiny diagram is not blown up past this scale. */
export const MAX_FIT_SCALE = 1.5;
/** Multiplicative step used by the zoom in/out buttons and keyboard shortcuts. */
export const ZOOM_STEP = 1.2;
export const IDENTITY_TRANSFORM = { scale: 1, x: 0, y: 0 };
/** Clamp a scale into the allowed range. */
export function clampScale(scale, min = MIN_SCALE, max = MAX_SCALE) {
    if (!Number.isFinite(scale))
        return min;
    return Math.min(Math.max(scale, min), max);
}
/**
 * Zoom by `factor` while keeping the content point under (px, py) — surface
 * coordinates relative to the zoom surface's top-left — visually fixed.
 */
export function zoomAroundPoint(t, factor, px, py, min = MIN_SCALE, max = MAX_SCALE) {
    const newScale = clampScale(t.scale * factor, min, max);
    // Translate the focal point into content space using the current transform,
    // then re-anchor it under the same surface point at the new scale.
    const worldX = (px - t.x) / t.scale;
    const worldY = (py - t.y) / t.scale;
    return {
        scale: newScale,
        x: px - worldX * newScale,
        y: py - worldY * newScale,
    };
}
/** Translate the content by a pixel delta without changing scale. */
export function panBy(t, dx, dy) {
    return { scale: t.scale, x: t.x + dx, y: t.y + dy };
}
/** Canonical 100%, centered-at-origin transform. */
export function resetTransform() {
    return { ...IDENTITY_TRANSFORM };
}
/**
 * Fit `content` inside `viewport` with padding and center it. Falls back to the
 * identity transform when either size is missing/zero/non-finite so callers
 * never produce NaN translations.
 */
export function fitTransform(viewport, content, opts = {}) {
    const padding = opts.padding ?? 24;
    const maxScale = opts.maxScale ?? MAX_FIT_SCALE;
    const valid = Number.isFinite(viewport.width) &&
        Number.isFinite(viewport.height) &&
        Number.isFinite(content.width) &&
        Number.isFinite(content.height) &&
        viewport.width > 0 &&
        viewport.height > 0 &&
        content.width > 0 &&
        content.height > 0;
    if (!valid)
        return resetTransform();
    const availW = Math.max(viewport.width - padding * 2, 1);
    const availH = Math.max(viewport.height - padding * 2, 1);
    const rawScale = Math.min(availW / content.width, availH / content.height);
    const scale = clampScale(rawScale, MIN_SCALE, maxScale);
    return {
        scale,
        x: (viewport.width - content.width * scale) / 2,
        y: (viewport.height - content.height * scale) / 2,
    };
}
/** Format a scale as an integer percentage for display. */
export function formatScalePercent(scale) {
    return `${Math.round(scale * 100)}%`;
}
/** Euclidean distance between two points (for pinch tracking). */
export function distance(ax, ay, bx, by) {
    return Math.hypot(ax - bx, ay - by);
}
/** Midpoint between two points (for pinch tracking). */
export function midpoint(ax, ay, bx, by) {
    return { x: (ax + bx) / 2, y: (ay + by) / 2 };
}
/**
 * Parse a viewBox string ("minX minY width height") into a Size, or null when
 * the value is absent or malformed.
 */
export function parseViewBox(viewBox) {
    if (!viewBox)
        return null;
    const parts = viewBox.trim().split(/[\s,]+/).map(Number);
    if (parts.length !== 4)
        return null;
    const width = parts[2];
    const height = parts[3];
    if (width === undefined || height === undefined)
        return null;
    if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0)
        return null;
    return { width, height };
}
