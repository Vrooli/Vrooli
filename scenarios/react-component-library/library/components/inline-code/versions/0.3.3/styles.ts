export const inlineCodeStyles = `
[data-rcl-inline] { display: inline-flex; position: relative; align-items: center; gap: var(--space-3xs); color: var(--color-foreground); }
[data-rcl-inline].rcl-inline__token { border-radius: var(--radius-control); background: var(--color-surface-muted); padding: var(--space-3xs) var(--space-2xs); color: var(--color-foreground); font-family: var(--font-mono, ui-monospace, monospace); }
[data-rcl-inline] .rcl-inline__token { border-radius: var(--radius-control); background: var(--color-surface-muted); padding: var(--space-3xs) var(--space-2xs); color: var(--color-foreground); font-family: var(--font-mono, ui-monospace, monospace); }
[data-rcl-inline].rcl-inline__link { color: var(--color-accent); text-decoration: underline; text-underline-offset: var(--space-3xs); }
[data-rcl-inline] .rcl-inline__copy { visibility: hidden; border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); padding: var(--space-3xs) var(--space-2xs); font: var(--text-label); cursor: pointer; }
[data-rcl-inline]:hover .rcl-inline__copy, [data-rcl-inline]:focus-within .rcl-inline__copy { visibility: visible; }
[data-rcl-inline] .rcl-inline__copy:hover { background: color-mix(in srgb, var(--color-accent) 10%, transparent); }
[data-rcl-inline]:focus-visible, [data-rcl-inline] :focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
@media (prefers-reduced-motion: reduce) { [data-rcl-inline], [data-rcl-inline] * { transition-duration: .01ms; } }
@media (forced-colors: active) { [data-rcl-inline].rcl-inline__token, [data-rcl-inline] .rcl-inline__token { border: var(--border-hairline) solid CanvasText; background: Canvas; color: CanvasText; } }
`;
