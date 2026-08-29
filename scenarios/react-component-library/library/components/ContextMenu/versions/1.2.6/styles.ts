export const contextMenuStyles = `
[data-rcl-context-menu] { position: fixed; inset: 0; z-index: var(--layer-menu, 610); pointer-events: none; }
[data-rcl-context-menu][data-presentation="sheet"] { display: grid; align-items: end; }

.rcl-context-menu__backdrop { position: absolute; inset: 0; margin: 0; padding: 0; border: 0; background: var(--color-scrim, rgb(15 23 42 / .42)); pointer-events: auto; opacity: 1; transition: opacity var(--dur-quick) var(--ease-standard); }
[data-rcl-context-menu][data-state="closed"] .rcl-context-menu__backdrop { opacity: 0; }

.rcl-context-menu__surface { position: relative; display: grid; gap: var(--space-3xs); align-content: start; inline-size: 100%; max-block-size: calc(var(--rcl-viewport-height, 100dvh) - var(--rcl-safe-top, 0px) - var(--overlay-drawer-top-gap, 32px)); overflow: auto; overscroll-behavior: contain; padding: var(--space-2xs); padding-block-end: calc(var(--space-2xs) + var(--rcl-safe-bottom, 0px)); border: var(--border-hairline) solid var(--color-border); border-block-end: 0; border-radius: var(--radius-sheet) var(--radius-sheet) 0 0; background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-modal); pointer-events: auto; transition: transform var(--dur-quick) var(--ease-standard); animation: rcl-context-menu-enter-sheet var(--dur-moderate) var(--ease-enter); }
.rcl-context-menu__surface[data-dragging="true"] { transition: none; will-change: transform; }
[data-rcl-context-menu][data-state="closed"] .rcl-context-menu__surface { transform: translateY(100%); animation: none; }
@keyframes rcl-context-menu-enter-sheet { from { transform: translateY(100%); } }

.rcl-context-menu__handle { position: absolute; z-index: 1; inset-block-start: 0; inset-inline-start: 50%; translate: -50% 0; inline-size: min(60%, 12rem); min-block-size: var(--tap-target-min); display: grid; justify-items: center; align-content: start; padding: var(--space-2xs) 0 0; margin: 0; border: 0; background: transparent; color: inherit; touch-action: none; cursor: grab; }
.rcl-context-menu__handle[data-rcl-overlay-dragging="true"] { cursor: grabbing; }
.rcl-context-menu__handle > span { display: block; inline-size: var(--overlay-grabber-inline, 2.25rem); block-size: var(--overlay-grabber-block, .25rem); border-radius: var(--radius-pill); background: var(--color-border-strong, var(--color-border)); }

.rcl-context-menu__surface h2 { margin: var(--space-sm) var(--space-xs) var(--space-3xs); font: var(--text-heading); }
[data-rcl-context-menu-item-wrap][data-separator] { border-block-start: var(--border-hairline) solid var(--color-border); padding-block-start: var(--space-3xs); margin-block-start: var(--space-3xs); }
[data-rcl-context-menu-item-wrap] > button[role="menuitem"], [data-rcl-context-menu-item-wrap] > button[role="menuitemcheckbox"] { display: flex; align-items: center; inline-size: 100%; gap: var(--space-sm); min-block-size: var(--tap-target-min); padding: var(--space-xs) var(--space-sm); border: 0; border-radius: var(--radius-control); background: transparent; color: inherit; text-align: start; }
[data-rcl-context-menu-item-wrap] > button:hover, [data-rcl-context-menu-item-wrap] > button[data-active] { background: var(--color-surface-muted); }
[data-rcl-context-menu-item-wrap] > button[data-destructive] { color: var(--color-danger); }
[data-rcl-context-menu-item-icon] { display: inline-flex; flex: 0 0 auto; } [data-rcl-context-menu-item-wrap] kbd { margin-inline-start: auto; }
.rcl-context-menu__surface [data-icon] { flex: 0 0 auto; inline-size: var(--icon-size-md); block-size: var(--icon-size-md); }

@media (min-width: 48rem) {
  .rcl-context-menu__surface { position: absolute; inset: auto; inline-size: auto; min-inline-size: 14rem; max-inline-size: 24rem; max-block-size: calc(var(--rcl-viewport-height, 100dvh) - (var(--space-lg) * 2)); padding-block-end: var(--space-2xs); border-block-end: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); box-shadow: var(--elev-overlay); transform: translateX(var(--overlay-menu-align, 0)); animation-name: rcl-context-menu-enter-menu; }
  [data-rcl-context-menu][data-state="closed"] .rcl-context-menu__surface { transform: translateX(var(--overlay-menu-align, 0)) translateY(var(--space-3xs)); opacity: 0; }
  .rcl-context-menu__surface[data-placement="bottom-end"] { transform: translateX(-100%); }
  [data-rcl-context-menu][data-state="closed"] .rcl-context-menu__surface[data-placement="bottom-end"] { transform: translateX(-100%) translateY(var(--space-3xs)); }
  .rcl-context-menu__surface h2 { margin: var(--space-xs); font: var(--text-label); }
}
@keyframes rcl-context-menu-enter-menu { from { opacity: 0; } }
`;
