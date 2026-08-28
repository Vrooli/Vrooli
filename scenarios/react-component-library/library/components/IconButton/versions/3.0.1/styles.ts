/**
 * IconButton owns its own surface treatment rather than inheriting the shared
 * control hover recipe.
 *
 * `ControlBase` hovers with `filter: brightness(1.06)`, a `translateY` lift, and
 * `box-shadow: var(--elev-raised)`. That recipe is written for a *filled* text
 * button. Applied to a transparent icon it produces no visible surface change
 * at all — there is nothing to brighten — while still painting a drop shadow,
 * so the hover state reads as a shadow cast by nothing. The lift also makes
 * icons jitter vertically in a dense toolbar row.
 *
 * The rules below neutralise that inheritance for this control and replace it
 * with what an icon button actually needs: a real tinted surface that appears
 * on hover and deepens on press, and no geometry change whatsoever.
 */
export const iconButtonStyles = `
[data-rcl-icon-button] {
  position: relative;
  display: inline-grid;
  place-items: center;
  padding: 0;
  border-style: solid;
  border-width: var(--border-thin, 1px);
  font-weight: 400;
  letter-spacing: normal;
  transition:
    background-color var(--dur-quick, 120ms) var(--ease-standard, ease),
    border-color var(--dur-quick, 120ms) var(--ease-standard, ease),
    color var(--dur-quick, 120ms) var(--ease-standard, ease);
}

/* Neutralise the inherited text-button hover geometry. */
[data-rcl-icon-button]:hover:not(:disabled),
[data-rcl-icon-button]:active:not(:disabled) {
  filter: none;
  transform: none;
  box-shadow: none;
}

/* The tap target is the control's business, not each call site's. A pointer
   that cannot be precise gets a comfortable target without the icon growing. */
@media (pointer: coarse) {
  [data-rcl-icon-button][data-rcl-tap-target="comfortable"] {
    min-inline-size: var(--tap-target-min, 44px);
    min-block-size: var(--tap-target-min, 44px);
  }
}

/* ---- ghost: the default. No chrome at rest; a real surface on interaction. ---- */

[data-rcl-icon-button][data-rcl-surface="ghost"] {
  background: transparent;
  border-color: transparent;
  color: var(--color-muted-foreground, currentColor);
}
[data-rcl-icon-button][data-rcl-surface="ghost"]:hover:not(:disabled) {
  background: color-mix(in srgb, currentColor 12%, transparent);
  color: var(--color-foreground, currentColor);
}
[data-rcl-icon-button][data-rcl-surface="ghost"]:active:not(:disabled) {
  background: color-mix(in srgb, currentColor 20%, transparent);
}

/* ---- soft: a standing surface, for controls that must be findable at rest ---- */

[data-rcl-icon-button][data-rcl-surface="soft"] {
  background: color-mix(in srgb, var(--color-surface-raised, var(--color-surface)) 80%, transparent);
  border-color: var(--color-border);
  color: var(--color-muted-foreground, currentColor);
  backdrop-filter: blur(4px);
}
[data-rcl-icon-button][data-rcl-surface="soft"]:hover:not(:disabled) {
  background: var(--color-surface-muted, var(--color-surface));
  color: var(--color-foreground, currentColor);
}
[data-rcl-icon-button][data-rcl-surface="soft"]:active:not(:disabled) {
  background: var(--color-surface-sunken, var(--color-surface-muted));
}

/* ---- solid / danger: filled, for the rare icon-only primary or destructive ---- */

[data-rcl-icon-button][data-rcl-surface="solid"] {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: var(--color-primary-foreground);
}
[data-rcl-icon-button][data-rcl-surface="solid"]:hover:not(:disabled) {
  background: color-mix(in srgb, var(--color-primary) 88%, var(--color-foreground));
}
[data-rcl-icon-button][data-rcl-surface="danger"] {
  background: transparent;
  border-color: transparent;
  color: var(--color-danger);
}
[data-rcl-icon-button][data-rcl-surface="danger"]:hover:not(:disabled) {
  background: color-mix(in srgb, var(--color-danger) 14%, transparent);
}
[data-rcl-icon-button][data-rcl-surface="danger"]:active:not(:disabled) {
  background: color-mix(in srgb, var(--color-danger) 22%, transparent);
}

/* ---- selected: a toggle that is on, expressed once instead of per call site ---- */

[data-rcl-icon-button][aria-pressed="true"] {
  background: var(--color-accent-subtle, color-mix(in srgb, var(--color-accent) 14%, transparent));
  border-color: color-mix(in srgb, var(--color-accent) 40%, transparent);
  color: var(--color-accent);
}
[data-rcl-icon-button][aria-pressed="true"]:hover:not(:disabled) {
  background: color-mix(in srgb, var(--color-accent) 22%, transparent);
}

/* ---- shape ---- */

[data-rcl-icon-button][data-rcl-shape="circle"] { border-radius: var(--radius-pill, 9999px); }
[data-rcl-icon-button][data-rcl-shape="rounded"] { border-radius: var(--radius-control, 0.375rem); }
[data-rcl-icon-button][data-rcl-shape="square"] { border-radius: 0; }

/* The icon slot is square and centred regardless of the control's own padding,
   so a circular button is a circle rather than a stadium. */
[data-rcl-icon-button] [data-rcl-icon-button-glyph] {
  display: inline-grid;
  place-items: center;
  pointer-events: none;
}

[data-rcl-icon-button][data-rcl-pending="true"] [data-rcl-icon-button-glyph] { visibility: hidden; }
[data-rcl-icon-button-spinner] {
  position: absolute;
  inline-size: 1em;
  block-size: 1em;
  border: var(--border-strong, 2px) solid color-mix(in srgb, currentColor 28%, transparent);
  border-block-start-color: currentColor;
  border-radius: var(--radius-pill, 9999px);
  animation: rcl-icon-button-spin var(--dur-moderate, 600ms) linear infinite;
}
@keyframes rcl-icon-button-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) {
  [data-rcl-icon-button-spinner] { animation-duration: 2.4s; }
}
`;
