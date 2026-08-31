export const dialogStyles = `
[data-rcl-dialog] { position: fixed; inset: 0; z-index: var(--layer-modal); display: flex; align-items: end; justify-content: center; padding: var(--space-2xs) var(--space-xs) var(--space-xs); background: color-mix(in srgb, var(--color-shell) 60%, transparent); }
[data-rcl-dialog] .rcl-dialog__backdrop { position: absolute; inset: 0; border: 0; background: transparent; cursor: default; }
[data-rcl-dialog] .rcl-dialog__surface { position: relative; z-index: 1; display: flex; min-block-size: 0; max-block-size: calc(100dvh - var(--space-sm)); inline-size: min(100%, 32rem); flex-direction: column; overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); box-shadow: var(--elev-modal); }
[data-rcl-dialog] .rcl-dialog__header { display: flex; align-items: start; justify-content: space-between; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-sm) var(--space-md); }
[data-rcl-dialog] .rcl-dialog__heading { min-inline-size: 0; }
[data-rcl-dialog] .rcl-dialog__title { margin: 0; color: var(--color-foreground); font-family: var(--font-sans); font-size: var(--text-heading-size); font-weight: 650; line-height: var(--text-heading-line); }
[data-rcl-dialog] .rcl-dialog__description { margin-block-start: var(--space-3xs); color: var(--color-muted-foreground); font-family: var(--font-sans); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); }
[data-rcl-dialog] button { min-block-size: var(--tap-target-min); min-inline-size: var(--tap-target-min); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); cursor: pointer; }
[data-rcl-dialog] .rcl-dialog__surface :is(button, a[href], input, select, textarea, [role="button"]) { min-block-size: var(--tap-target-min); min-inline-size: var(--tap-target-min); }
[data-rcl-dialog] button:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-dialog] .rcl-dialog__body { min-block-size: 0; flex: 1; overflow: auto; padding: var(--space-md); }
[data-rcl-dialog] .rcl-dialog__footer { border-block-start: var(--border-hairline) solid var(--color-border); padding: var(--space-sm) var(--space-md); }
[data-rcl-dialog] :focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
@media (min-width: 768px) { [data-rcl-dialog] { align-items: center; padding: var(--space-md); } }
@media (prefers-reduced-motion: reduce) { [data-rcl-dialog] *, [data-rcl-dialog] *::before, [data-rcl-dialog] *::after { transition-duration: .01ms; } }
@media (forced-colors: active) { [data-rcl-dialog] .rcl-dialog__surface { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-dialog] .rcl-dialog__backdrop { background: Canvas; opacity: .8; } }
`;
