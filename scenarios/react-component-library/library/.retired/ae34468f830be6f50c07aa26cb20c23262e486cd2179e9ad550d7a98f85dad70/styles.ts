export const bottomNavStyles = `
[data-rcl-bottom-nav] { position: fixed; inset-inline: 0; inset-block-end: 0; z-index: var(--layer-sticky); box-sizing: border-box; border-block-start: var(--border-hairline) solid var(--color-border); background: color-mix(in srgb, var(--color-surface-raised) 96%, transparent); box-shadow: var(--elev-raised); padding-block-start: var(--space-2xs); padding-inline: env(safe-area-inset-left, 0px) env(safe-area-inset-right, 0px); backdrop-filter: blur(var(--space-xs)); }
[data-rcl-bottom-nav][data-rcl-bottom-nav-presentation="flow"] { position: relative; inset: auto; z-index: auto; }
[data-rcl-bottom-nav][data-rcl-bottom-nav-safe-area="inset"] { padding-block-end: env(safe-area-inset-bottom, 0px); }
[data-rcl-bottom-nav][data-rcl-bottom-nav-safe-area="floor"] { padding-block-end: max(var(--space-lg), env(safe-area-inset-bottom, 0px)); }
[data-rcl-bottom-nav][data-rcl-bottom-nav-safe-area="none"] { padding-block-end: 0; }
[data-rcl-bottom-nav-track] { position: relative; display: grid; grid-template-columns: repeat(auto-fit, minmax(0, 1fr)); min-inline-size: 0; }
[data-rcl-bottom-nav-item] { position: relative; display: flex; min-inline-size: 0; min-block-size: var(--tap-target-min); flex-direction: column; align-items: center; justify-content: center; gap: var(--space-3xs); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); padding: var(--space-2xs) var(--space-3xs); font: var(--text-caption); text-decoration: none; cursor: pointer; transition: background-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-bottom-nav-item][data-active="true"] { color: var(--color-primary); font-weight: 700; }
[data-rcl-bottom-nav-item]:hover:not(:disabled) { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-bottom-nav-item]:active:not(:disabled) { transform: translateY(var(--space-3xs)); }
[data-rcl-bottom-nav-item][data-disabled="true"], [data-rcl-bottom-nav-item]:disabled { cursor: not-allowed; opacity: .58; }
[data-rcl-bottom-nav-active-indicator] { position: absolute; inset-block-end: 0; inset-inline-start: 0; z-index: 1; block-size: var(--space-3xs); border-radius: var(--radius-pill); background: var(--color-primary); pointer-events: none; transition: transform var(--dur-moderate) var(--ease-standard), inline-size var(--dur-moderate) var(--ease-standard); }
[data-rcl-bottom-nav][data-rcl-bottom-nav-indicator="static"] [data-rcl-bottom-nav-active-indicator] { transition: none; }
[data-rcl-bottom-nav-icon] { display: grid; inline-size: var(--space-md); block-size: var(--space-md); flex: 0 0 auto; place-items: center; }
[data-rcl-bottom-nav-icon] svg { inline-size: 100%; block-size: 100%; }
[data-rcl-bottom-nav-label] { max-inline-size: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
@media (prefers-reduced-motion: reduce) { [data-rcl-bottom-nav-active-indicator] { transition: none; } }
`;
