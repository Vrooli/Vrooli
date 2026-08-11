/**
 * @vrooliComponentSource react-component-library:BottomNav
 * @vrooliComponentVersion 1.3.0
 * @vrooliComponentAdoption f8ff9216-a8af-44bb-a4b1-c443315e2ad6
 * @vrooliComponentAppliedAt 2026-08-11T00:11:48Z
 * @vrooliComponentSourceSha256 c47b0abd7c03c3f56213d73e1154ad7615741105590f775315719b91571a3951
 * @vrooliComponentDriftHash c47b0abd7c03c3f56213d73e1154ad7615741105590f775315719b91571a3951
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
export const bottomNavStyles = `
[data-rcl-bottom-nav] { position: fixed; inset-inline: 0; inset-block-end: 0; z-index: var(--layer-sticky); display: grid; grid-template-columns: repeat(auto-fit, minmax(0, 1fr)); box-sizing: border-box; border-block-start: var(--border-hairline) solid var(--color-border); background: color-mix(in srgb, var(--color-surface-raised) 94%, transparent); box-shadow: var(--elev-raised); padding-block: var(--space-2xs) env(safe-area-inset-bottom); padding-inline: env(safe-area-inset-left) env(safe-area-inset-right); backdrop-filter: blur(var(--space-xs)); }
[data-rcl-bottom-nav-item] { position: relative; display: flex; min-inline-size: 0; min-block-size: var(--tap-target-min); flex-direction: column; align-items: center; justify-content: center; gap: var(--space-3xs); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); padding: var(--space-2xs) var(--space-3xs); font: var(--text-caption); text-decoration: none; cursor: pointer; transition: background-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-bottom-nav-item]::before { position: absolute; inset-block-start: calc(var(--space-2xs) * -1); inline-size: var(--space-lg); block-size: var(--space-3xs); border-radius: var(--radius-pill); background: var(--color-primary); content: ""; opacity: 0; transform: scaleX(.5); transition: opacity var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-bottom-nav-item][data-active="true"] { color: var(--color-primary); font-weight: 700; }
[data-rcl-bottom-nav-item][data-active="true"]::before { opacity: 1; transform: scaleX(1); }
[data-rcl-bottom-nav-item]:hover:not(:disabled) { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-bottom-nav-item]:active:not(:disabled) { transform: translateY(var(--space-3xs)); }
[data-rcl-bottom-nav-item][data-disabled="true"], [data-rcl-bottom-nav-item]:disabled { cursor: not-allowed; opacity: .58; }
[data-rcl-bottom-nav-icon] { display: grid; inline-size: var(--space-md); block-size: var(--space-md); flex: 0 0 auto; place-items: center; }
[data-rcl-bottom-nav-label] { max-inline-size: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-bottom-nav] :focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: calc(var(--space-3xs) * -1); }
@media (prefers-reduced-motion: reduce) { [data-rcl-bottom-nav] *, [data-rcl-bottom-nav] *::before { transition: none; } }
@media (forced-colors: active) { [data-rcl-bottom-nav] { border-color: CanvasText; background: Canvas; } [data-rcl-bottom-nav-item] { color: CanvasText; } [data-rcl-bottom-nav-item][data-active="true"]::before { background: CanvasText; } }
`;
