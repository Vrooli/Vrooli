export const drawerShellStyles = `
[data-rcl-drawer-shell-root] { position: fixed; inset: 0; z-index: var(--layer-modal); }
[data-rcl-drawer-shell-backdrop] { position: absolute; inset: 0; border: 0; background: color-mix(in srgb, var(--color-shell) var(--opacity-scrim), transparent); cursor: default; }
[data-rcl-drawer-shell] { position: absolute; inset-block-start: max(var(--space-sm), env(safe-area-inset-top)); inset-inline: 0; inset-block-end: 0; display: flex; min-block-size: 0; flex-direction: column; overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-block-end: 0; border-radius: var(--radius-sheet) var(--radius-sheet) 0 0; background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-modal); animation: rcl-drawer-shell-enter var(--dur-moderate) var(--ease-enter) both; }
[data-rcl-drawer-shell][data-size="compact"] { inset-block-start: auto; inset-inline: var(--space-xs); inset-block-end: max(var(--space-xs), env(safe-area-inset-bottom)); max-block-size: calc(100dvh - (var(--space-lg) * 2)); border-block-end: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); }
[data-rcl-drawer-shell][data-avoid-keyboard="true"] { inset-block-end: var(--wc-kb-height, env(safe-area-inset-bottom)); }
[data-rcl-drawer-shell-header] { flex-shrink: 0; border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-sm) var(--space-md); }
[data-rcl-drawer-shell-title-row] { display: flex; min-inline-size: 0; align-items: center; gap: var(--space-xs); }
[data-rcl-drawer-shell-title] { min-inline-size: 0; flex: 1; overflow: hidden; margin: 0; color: var(--color-foreground); font-family: var(--font-sans); font-size: var(--text-heading-size); font-weight: 700; line-height: var(--text-heading-line); text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-drawer-shell-actions] { display: flex; flex-shrink: 0; align-items: center; gap: var(--space-3xs); }
[data-rcl-drawer-shell-close] { display: inline-grid; flex-shrink: 0; min-block-size: var(--tap-target-min); min-inline-size: var(--tap-target-min); place-items: center; border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); cursor: pointer; font: 700 var(--text-heading-size)/1 var(--font-sans); transition: background-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-drawer-shell-close]:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-drawer-shell-close]:active { transform: scale(.96); }
[data-rcl-drawer-shell-extra] { margin-block-start: var(--space-3xs); color: var(--color-muted-foreground); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); }
[data-rcl-drawer-shell-body] { min-block-size: 0; flex: 1; overflow: auto; overscroll-behavior: contain; }
[data-rcl-drawer-shell] :focus-visible, [data-rcl-drawer-shell-backdrop]:focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: var(--space-3xs); }
@keyframes rcl-drawer-shell-enter { from { opacity: 0; transform: translateY(var(--space-md)); } to { opacity: 1; transform: translateY(0); } }
@media (min-width: 48rem) {
  [data-rcl-drawer-shell] { inset-block-start: var(--space-md); inset-inline: var(--space-md); inset-block-end: var(--space-md); border-block-end: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); }
  [data-rcl-drawer-shell][data-size="compact"] { inset-block-start: 50%; inset-inline: 50%; inset-block-end: auto; inline-size: min(100% - (var(--space-lg) * 2), 32rem); max-block-size: 80dvh; transform: translate(-50%, -50%); }
}
@media (prefers-reduced-motion: reduce) { [data-rcl-drawer-shell], [data-rcl-drawer-shell] * { animation: none; transition-duration: .01ms; } }
@media (forced-colors: active) { [data-rcl-drawer-shell] { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-drawer-shell-backdrop] { background: Canvas; opacity: .8; } }
`;
