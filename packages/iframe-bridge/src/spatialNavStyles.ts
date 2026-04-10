/**
 * Default focus-ring CSS for spatial navigation.
 *
 * Injected at runtime when `injectDefaultFocusStyle` is `true` (the default).
 * Scenarios can override these styles in their own CSS via the
 * `[data-spatial-focus]` attribute selector, or opt out entirely by passing
 * `{ injectDefaultFocusStyle: false }` to `initSpatialNav`.
 */

// ---------------------------------------------------------------------------
// Style content
// ---------------------------------------------------------------------------

/**
 * Blue ring + glow that works on both light and dark backgrounds.
 *
 * - `[data-spatial-active]` on `<html>` hides the browser cursor.
 * - `[data-spatial-focus="true"]` highlights the focused element.
 * - The `*:focus` rule suppresses the browser's default focus outline while
 *   spatial mode is active so only our ring is visible.
 */
export const DEFAULT_SPATIAL_FOCUS_CSS = `
[data-spatial-active] {
  cursor: none !important;
}

[data-spatial-focus="true"] {
  outline: 2px solid #60a5fa !important;
  outline-offset: 2px;
  box-shadow: 0 0 0 4px rgba(96, 165, 250, 0.3);
}

[data-spatial-active] *:focus {
  outline: none;
}
`.trim();

// ---------------------------------------------------------------------------
// Injection helpers
// ---------------------------------------------------------------------------

const STYLE_ATTR = 'data-spatial-styles';

/**
 * Inject the default spatial-nav focus CSS into `<head>`.
 * Returns the `<style>` element for later removal.
 */
export function injectSpatialStyles(): HTMLStyleElement {
  // Avoid duplicates.
  const existing = document.querySelector(`style[${STYLE_ATTR}]`);
  if (existing instanceof HTMLStyleElement) return existing;

  const style = document.createElement('style');
  style.setAttribute(STYLE_ATTR, '');
  style.textContent = DEFAULT_SPATIAL_FOCUS_CSS;
  document.head.appendChild(style);
  return style;
}

/**
 * Remove a previously-injected spatial-nav style element.
 */
export function removeSpatialStyles(style: HTMLStyleElement): void {
  style.remove();
}
