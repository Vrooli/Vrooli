export const bottomSheetStyles = `
[data-rcl-bottom-sheet] { position: fixed; inset: 0; z-index: var(--layer-modal, 500); display: grid; align-items: end; padding: var(--space-xs); padding-block-end: max(var(--space-xs), env(safe-area-inset-bottom)); pointer-events: none; }
[data-rcl-bottom-sheet][data-avoid-keyboard] { inset-block-end: var(--wc-kb-height, 0px); }
.rcl-bottom-sheet__backdrop { position: absolute; inset: 0; border: 0; background: var(--color-scrim, rgb(15 23 42 / .52)); pointer-events: auto; }
.rcl-bottom-sheet__panel { position: relative; display: grid; grid-template-rows: auto auto minmax(0,1fr) auto; inline-size: min(100%, 42rem); max-block-size: calc(100dvh - (var(--space-lg) * 2)); margin-inline: auto; overflow: hidden; border: 1px solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-overlay); pointer-events: auto; transform: translateY(calc(var(--rcl-sheet-progress, 0) * 100%)); transition: transform var(--dur-quick) var(--ease-standard); }
.rcl-bottom-sheet__handle { min-block-size: var(--tap-target-min); border: 0; background: transparent; display: grid; place-items: center; touch-action: none; }
.rcl-bottom-sheet__handle span { inline-size: 3rem; block-size: .25rem; border-radius: var(--radius-pill); background: var(--color-border-strong); }
.rcl-bottom-sheet__header, .rcl-bottom-sheet__footer { display: flex; align-items: center; justify-content: space-between; gap: var(--space-sm); padding: var(--space-sm) var(--space-md); border-block-end: 1px solid var(--color-border); }
.rcl-bottom-sheet__header h2 { margin: 0; font: var(--text-title); } .rcl-bottom-sheet__panel :is(button, input, select, textarea, [role="button"]) { min-inline-size: var(--tap-target-min); min-block-size: var(--tap-target-min); }
.rcl-bottom-sheet__content { min-block-size: 0; overflow: auto; padding: var(--space-md); } .rcl-bottom-sheet__footer { border-block-start: 1px solid var(--color-border); border-block-end: 0; }
@media (prefers-reduced-motion: reduce) { .rcl-bottom-sheet__panel { transition: none; } }
`;
