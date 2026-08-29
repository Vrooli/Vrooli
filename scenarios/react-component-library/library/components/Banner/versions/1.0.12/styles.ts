/**
 * Tone palettes are built with `color-mix` over the semantic colour tokens
 * rather than written as literals.
 *
 * The literal form is tempting because of a real trap: several design tokens
 * already carry an alpha component, and composing a second opacity modifier on
 * top of one produces `rgb(R G B / a / b)`, which no browser parses — the
 * declaration is dropped and the element renders unpainted. `color-mix` avoids
 * that entirely while keeping the palette themeable, which literals do not.
 */
export const bannerStyles = `
[data-rcl-banner-region] { display: flex; flex-direction: column; flex-shrink: 0; max-block-size: var(--rcl-banner-region-max, 40vh); overflow-y: auto; overscroll-behavior: contain; }

[data-rcl-banner] {
  --rcl-banner-accent: var(--color-info, #0284c7);
  --rcl-banner-surface: color-mix(in srgb, var(--rcl-banner-accent) 18%, var(--color-surface, #ffffff));
  --rcl-banner-border: color-mix(in srgb, var(--rcl-banner-accent) 35%, transparent);
  --rcl-banner-title: var(--color-foreground, #0f172a);
  --rcl-banner-detail: var(--color-muted-foreground, #64748b);
  --rcl-banner-action-bg: color-mix(in srgb, var(--rcl-banner-accent) 18%, transparent);
  --rcl-banner-action-border: color-mix(in srgb, var(--rcl-banner-accent) 45%, transparent);

  box-sizing: border-box;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  align-items: start;
  gap: var(--space-2xs, 8px);
  padding-block: var(--space-2xs, 8px);
  padding-inline-start: max(var(--space-sm, 16px), var(--rcl-safe-left, 0px));
  padding-inline-end: max(var(--space-sm, 16px), var(--rcl-safe-right, 0px));
  border-block-end: var(--border-hairline, 1px) solid var(--rcl-banner-border);
  background: var(--rcl-banner-surface);
  color: var(--rcl-banner-title);
  font: var(--text-body-sm, 400 var(--text-body-sm-size) / var(--text-body-sm-line) var(--font-sans));
  line-height: 1.4;
}

[data-rcl-banner][data-tone="danger"] { --rcl-banner-accent: var(--color-danger, #dc2626); }
[data-rcl-banner][data-tone="warning"] { --rcl-banner-accent: var(--color-warning, #d97706); }
[data-rcl-banner][data-tone="success"] { --rcl-banner-accent: var(--color-success, #16a34a); }

[data-rcl-banner][data-compact="true"] { padding-block: var(--space-3xs, 4px); background: color-mix(in srgb, var(--rcl-banner-accent) 10%, transparent); }

/* Held on screen after its condition cleared, purely so it stays readable.
 * Muted rather than removed — a banner that vanishes mid-sentence is the
 * flicker this whole layer exists to prevent. */
[data-rcl-banner][data-settling="true"] { opacity: 0.72; }

[data-rcl-banner-icon] { display: grid; place-items: center; margin-block-start: 1px; color: var(--rcl-banner-accent); }
[data-rcl-banner-icon] svg { inline-size: var(--icon-size-sm, 16px); block-size: var(--icon-size-sm, 16px); }
[data-rcl-banner-icon][data-spin="true"] svg { animation: rcl-banner-spin 900ms linear infinite; }
@keyframes rcl-banner-spin { to { transform: rotate(360deg); } }

[data-rcl-banner-content] { display: flex; flex-direction: column; gap: 2px; min-inline-size: 0; }
[data-rcl-banner-title] { font-weight: 600; overflow-wrap: anywhere; }
[data-rcl-banner-description] { color: var(--rcl-banner-detail); overflow-wrap: anywhere; }
[data-rcl-banner-detail] { color: color-mix(in srgb, var(--rcl-banner-accent) 60%, var(--rcl-banner-detail)); overflow-wrap: anywhere; }

[data-rcl-banner-actions] { display: flex; flex-wrap: wrap; gap: var(--space-3xs, 4px); }

[data-rcl-banner-action] {
  display: inline-flex; align-items: center; gap: var(--space-3xs, 4px); flex-shrink: 0;
  padding-block: var(--space-3xs, 4px); padding-inline: var(--space-2xs, 8px);
  border: var(--border-hairline, 1px) solid var(--rcl-banner-action-border);
  border-radius: var(--radius-control, 0.375rem);
  background: transparent; color: inherit; font: inherit; font-weight: 600; cursor: pointer;
  transition: background-color var(--dur-quick, 180ms) ease;
}
[data-rcl-banner-action][data-primary="true"] { background: var(--rcl-banner-action-bg); }
[data-rcl-banner-action]:hover:not(:disabled) { background: var(--rcl-banner-action-bg); }
[data-rcl-banner-action]:disabled { opacity: 0.6; cursor: not-allowed; }
[data-rcl-banner-action] svg { inline-size: var(--icon-size-sm, 16px); block-size: var(--icon-size-sm, 16px); }

/* An icon button, not a bordered control. Close is the least important thing
 * in the banner and reads as clutter when boxed like an action — the actions
 * are the controls that need to look pressable.
 *
 * The touch target is an ::after overlay rather than real size, because real
 * size would set the height of the whole row: this is compact chrome, and a
 * 44px control would nearly double it. The overlay gives a conforming hit area
 * at zero layout cost. */
[data-rcl-banner-dismiss] {
  position: relative;
  display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0;
  padding: var(--space-3xs, 4px);
  border: 0; border-radius: var(--radius-control, 0.375rem);
  background: transparent; color: var(--rcl-banner-detail); cursor: pointer;
  transition: background-color var(--dur-quick, 180ms) ease, color var(--dur-quick, 180ms) ease;
}
[data-rcl-banner-dismiss]::after {
  content: ""; position: absolute; inset-block-start: 50%; inset-inline-start: 50%;
  inline-size: var(--tap-target-min, 44px); block-size: var(--tap-target-min, 44px);
  transform: translate(-50%, -50%);
}
[data-rcl-banner-dismiss]:hover:not(:disabled) { background: var(--rcl-banner-action-bg); color: var(--rcl-banner-title); }
[data-rcl-banner-dismiss]:disabled { opacity: 0.5; cursor: not-allowed; }
[data-rcl-banner-dismiss] svg { inline-size: var(--icon-size-sm, 16px); block-size: var(--icon-size-sm, 16px); }

[data-rcl-banner-overflow-toggle] {
  display: flex; align-items: center; justify-content: center; gap: var(--space-3xs, 4px);
  inline-size: 100%; padding-block: var(--space-3xs, 4px);
  padding-inline: max(var(--space-sm, 16px), var(--rcl-safe-left, 0px)) max(var(--space-sm, 16px), var(--rcl-safe-right, 0px));
  border: 0; border-block-end: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1);
  background: color-mix(in srgb, var(--color-foreground, #0f172a) 8%, transparent);
  color: var(--color-muted-foreground, #64748b);
  font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); cursor: pointer;
}
[data-rcl-banner-overflow-toggle]:hover { color: var(--color-foreground, #0f172a); }
[data-rcl-banner-overflow-toggle] svg { inline-size: var(--icon-size-xs, 12px); block-size: var(--icon-size-xs, 12px); }

[data-rcl-banner-action]:focus-visible,
[data-rcl-banner-dismiss]:focus-visible,
[data-rcl-banner-overflow-toggle]:focus-visible { outline: var(--focus-ring-width, 2px) solid var(--color-focus-ring, var(--color-focus)); outline-offset: 1px; }

/* Tone must survive a mode that discards author colour, so the glyph carries
 * the same distinction the palette does. */
@media (forced-colors: active) {
  [data-rcl-banner] { border-block-end: 1px solid CanvasText; background: Canvas; color: CanvasText; }
  [data-rcl-banner-icon] { color: CanvasText; }
  [data-rcl-banner-action] { border-color: CanvasText; }
}

@media (prefers-reduced-motion: reduce) {
  [data-rcl-banner-icon][data-spin="true"] svg { animation: none; }
  [data-rcl-banner-dismiss], [data-rcl-banner-action] { transition: none; }
}
`;
