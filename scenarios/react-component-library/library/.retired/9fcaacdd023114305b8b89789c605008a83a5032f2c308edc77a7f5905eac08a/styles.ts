export const contextMenuStyles = `
[data-rcl-context-menu] { position: fixed; inset: 0; z-index: var(--layer-menu, 610); pointer-events: none; }
.rcl-context-menu__backdrop { position: absolute; inset: 0; border: 0; background: var(--color-scrim, rgb(15 23 42 / .42)); pointer-events: auto; }
.rcl-context-menu__surface { position: absolute; inset-inline: var(--space-xs); inset-block-end: max(var(--space-xs), env(safe-area-inset-bottom)); display: grid; gap: var(--space-3xs); max-block-size: 70dvh; overflow: auto; padding: var(--space-2xs); border: 1px solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-floating); pointer-events: auto; }
.rcl-context-menu__handle { min-block-size: var(--tap-target-min); border: 0; background: transparent; touch-action: none; } .rcl-context-menu__handle span { display: block; inline-size: 3rem; block-size: .25rem; margin: auto; border-radius: var(--radius-pill); background: var(--color-border-strong); }
.rcl-context-menu__surface h2 { margin: var(--space-xs); font: var(--text-title); }
[data-rcl-context-menu-item-wrap][data-separator] { border-block-start: 1px solid var(--color-border); padding-block-start: var(--space-3xs); margin-block-start: var(--space-3xs); }
[data-rcl-context-menu-item-wrap] > button[role="menuitem"] { display: flex; align-items: center; inline-size: 100%; gap: var(--space-sm); min-block-size: var(--tap-target-min); padding: var(--space-xs) var(--space-sm); border: 0; border-radius: var(--radius-control); background: transparent; color: inherit; text-align: start; }
[data-rcl-context-menu-item-wrap] > button[role="menuitem"]:hover, [data-rcl-context-menu-item-wrap] > button[data-active] { background: var(--color-surface-muted); }
[data-rcl-context-menu-item-wrap] > button[data-destructive] { color: var(--color-danger-foreground, #b42318); }
[data-rcl-context-menu-item-icon] { display: inline-flex; flex: 0 0 auto; } [data-rcl-context-menu-item-wrap] kbd { margin-inline-start: auto; }
@media (min-width: 48rem) { .rcl-context-menu__surface { inset: auto; min-inline-size: 14rem; max-inline-size: 24rem; transform: translateX(var(--context-menu-align, 0)); } .rcl-context-menu__surface[data-placement="bottom-end"] { transform: translateX(-100%); } }
`;
