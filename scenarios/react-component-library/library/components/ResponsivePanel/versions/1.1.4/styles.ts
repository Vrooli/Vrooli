export const responsivePanelStyles = `
[data-rcl-responsive-panel-root] { position: relative; min-block-size: 0; min-inline-size: 0; }
[data-rcl-responsive-panel] { position: relative; display: flex; min-block-size: 0; min-inline-size: 0; flex-direction: column; overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); box-shadow: var(--elev-raised); }
[data-rcl-responsive-panel][data-mobile="false"] { block-size: 100%; inline-size: var(--rcl-responsive-panel-width, var(--sidebar-width)); }
[data-rcl-responsive-panel][data-mobile="true"] { position: fixed; inset-block-start: env(safe-area-inset-top); inset-block-end: 0; inset-inline-start: 0; z-index: var(--layer-modal); inline-size: min(var(--rcl-responsive-panel-width, var(--sidebar-width)), calc(100vw - (var(--space-md) * 2))); max-inline-size: 100%; border-block-start: 0; border-inline-start: 0; border-block-end: 0; border-radius: 0 var(--radius-sheet) var(--radius-sheet) 0; box-shadow: var(--elev-modal); animation: rcl-responsive-panel-enter var(--dur-moderate) var(--ease-enter) both; }
[data-rcl-responsive-panel-backdrop] { position: fixed; inset: 0; z-index: calc(var(--layer-modal) - 1); border: 0; background: color-mix(in srgb, var(--color-shell) var(--opacity-scrim), transparent); cursor: default; }
[data-rcl-responsive-panel-header] { display: flex; min-block-size: var(--tap-target-min); min-inline-size: 0; flex-shrink: 0; align-items: flex-start; justify-content: space-between; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-sm); padding-block-start: max(var(--space-sm), env(safe-area-inset-top)); }
[data-rcl-responsive-panel-heading] { min-inline-size: 0; flex: 1; }
[data-rcl-responsive-panel-heading] h2 { overflow-wrap: anywhere; margin: 0; color: var(--color-foreground); font-size: var(--text-heading-size); font-weight: 700; line-height: var(--text-heading-line); letter-spacing: -0.01em; }
[data-rcl-responsive-panel-heading] p { overflow-wrap: anywhere; margin: var(--space-3xs) 0 0; color: var(--color-muted-foreground); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); }
[data-rcl-responsive-panel-close] { display: inline-grid; flex: 0 0 auto; place-items: center; inline-size: var(--tap-target-min); block-size: var(--tap-target-min); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); cursor: pointer; font: 700 var(--text-heading-size)/1 var(--font-sans); }
[data-rcl-responsive-panel-close]:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-responsive-panel-content] { min-block-size: 0; min-inline-size: 0; flex: 1; overflow: auto; overscroll-behavior: contain; padding: var(--space-md); }
[data-rcl-responsive-panel-resize] { position: absolute; inset-block: 0; inset-inline-end: calc(var(--space-3xs) * -1); z-index: var(--layer-sticky); inline-size: var(--space-xs); block-size: 100%; appearance: none; border: 0; background: transparent; padding: 0; cursor: col-resize; touch-action: none; writing-mode: vertical-lr; }
[data-rcl-responsive-panel-resize]::-webkit-slider-runnable-track { inline-size: var(--space-3xs); block-size: 100%; border-radius: var(--radius-pill); background: transparent; }
[data-rcl-responsive-panel-resize]::-webkit-slider-thumb { inline-size: var(--space-3xs); block-size: var(--space-lg); appearance: none; border: 0; border-radius: var(--radius-pill); background: transparent; }
[data-rcl-responsive-panel-resize]::-moz-range-track { inline-size: var(--space-3xs); block-size: 100%; border-radius: var(--radius-pill); background: transparent; }
[data-rcl-responsive-panel-resize]::-moz-range-thumb { inline-size: var(--space-3xs); block-size: var(--space-lg); border: 0; border-radius: var(--radius-pill); background: transparent; }
[data-rcl-responsive-panel-resize]:hover, [data-rcl-responsive-panel-resize]:focus-visible { background: color-mix(in srgb, var(--color-primary) 20%, transparent); }
[data-rcl-responsive-panel] :focus-visible, [data-rcl-responsive-panel-backdrop]:focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: var(--space-3xs); }
@keyframes rcl-responsive-panel-enter { from { opacity: 0; transform: translateX(-100%); } to { opacity: 1; transform: translateX(0); } }
@media (prefers-reduced-motion: reduce) { [data-rcl-responsive-panel][data-mobile="true"] { animation: none; } }
@media (forced-colors: active) { [data-rcl-responsive-panel] { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-responsive-panel-backdrop] { background: Canvas; opacity: .8; } }
`;
