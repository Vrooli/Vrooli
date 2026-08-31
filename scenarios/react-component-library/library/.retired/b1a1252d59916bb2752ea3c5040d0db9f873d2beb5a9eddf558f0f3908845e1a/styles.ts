export const fullPageDrawerStyles = `
[data-rcl-full-page-drawer] { position: fixed; inset: 0; z-index: var(--layer-modal, 500); padding-block-start: max(var(--space-xs), env(safe-area-inset-top)); pointer-events: none; }
.rcl-full-page-drawer__backdrop { position: absolute; inset: 0; border: 0; background: var(--color-scrim, rgb(15 23 42 / .52)); pointer-events: auto; }
.rcl-full-page-drawer__panel { position: absolute; inset: max(var(--space-xs), env(safe-area-inset-top)) 0 max(var(--space-xs), env(safe-area-inset-bottom)); display: grid; grid-template-rows: auto minmax(0,1fr) auto; overflow: hidden; border-radius: var(--radius-panel) var(--radius-panel) 0 0; background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-overlay); pointer-events: auto; }
[data-rcl-full-page-drawer][data-avoid-keyboard] .rcl-full-page-drawer__panel { inset-block-end: var(--wc-kb-height, 0px); }
.rcl-full-page-drawer__panel > header, .rcl-full-page-drawer__panel > footer { display: flex; align-items: center; justify-content: space-between; gap: var(--space-sm); padding: var(--space-md); border-block-end: 1px solid var(--color-border); }
.rcl-full-page-drawer__panel h2 { margin: 0; font: var(--text-title); } .rcl-full-page-drawer__panel button { min-inline-size: var(--tap-target-min); min-block-size: var(--tap-target-min); }
.rcl-full-page-drawer__content { min-block-size: 0; overflow: auto; padding: var(--space-md); } .rcl-full-page-drawer__panel > footer { border-block-start: 1px solid var(--color-border); border-block-end: 0; }
@media (min-width: 48rem) { .rcl-full-page-drawer__panel { inset: var(--space-md); border-radius: var(--radius-panel); } [data-rcl-full-page-drawer][data-avoid-keyboard] .rcl-full-page-drawer__panel { inset-block-end: max(var(--space-md), var(--wc-kb-height, 0px)); } }
`;
