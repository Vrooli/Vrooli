/**
 * @vrooliComponentSource react-component-library:EmptyState
 * @vrooliComponentVersion 1.2.0
 * @vrooliComponentAdoption 08245e9a-4333-4e0a-ae3c-f6178d1d06e1
 * @vrooliComponentAppliedAt 2026-08-18T01:12:47Z
 * @vrooliComponentSourceSha256 3675e77e7bba30d5ee41b13e71592a38a3d563fb9d5bfaf07391c1a7c9e6ef1d
 * @vrooliComponentDriftHash 3675e77e7bba30d5ee41b13e71592a38a3d563fb9d5bfaf07391c1a7c9e6ef1d
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
export const emptyStateStyles = `
[data-rcl-empty-state] { display: flex; min-inline-size: 0; max-inline-size: 48rem; flex-direction: column; align-items: flex-start; gap: var(--space-sm); box-sizing: border-box; border: var(--border-hairline) solid var(--color-border); border-block-start: var(--border-strong) solid var(--color-primary); border-radius: var(--radius-panel); background: linear-gradient(135deg, var(--color-surface-muted), color-mix(in srgb, var(--color-primary) 4%, var(--color-surface))); color: var(--color-foreground); padding: var(--space-lg); box-shadow: var(--elev-flat); }
[data-rcl-empty-state-icon] { display: grid; inline-size: var(--space-xl); block-size: var(--space-xl); flex: 0 0 auto; place-items: center; border: var(--border-hairline) solid color-mix(in srgb, var(--color-primary) 22%, var(--color-border)); border-radius: var(--radius-control); background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface)); color: var(--color-primary); }
[data-rcl-empty-state-copy] { min-inline-size: 0; }
[data-rcl-empty-state-title] { margin: 0; color: var(--color-foreground); font: var(--text-title); letter-spacing: -0.02em; }
[data-rcl-empty-state-description] { max-inline-size: 55ch; margin: var(--space-3xs) 0 0; color: var(--color-muted-foreground); font: var(--text-body-sm); overflow-wrap: anywhere; }
[data-rcl-empty-state-action] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-2xs); padding-block-start: var(--space-3xs); }
[data-rcl-empty-state-action] > button:not([data-rcl-control]), [data-rcl-empty-state-action] > a:not([data-rcl-control]) { min-block-size: var(--tap-target-min); box-sizing: border-box; border: var(--border-hairline) solid var(--color-primary); border-radius: var(--radius-control); background: var(--color-primary); color: var(--color-on-primary); padding-inline: var(--space-md); font: var(--text-label); text-decoration: none; cursor: pointer; transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-empty-state-action] > button:not([data-rcl-control]):hover, [data-rcl-empty-state-action] > a:not([data-rcl-control]):hover { border-color: var(--color-primary-hover); background: var(--color-primary-hover); }
[data-rcl-empty-state-action] > button:not([data-rcl-control]):active, [data-rcl-empty-state-action] > a:not([data-rcl-control]):active { transform: translateY(var(--space-3xs)); }
[data-rcl-empty-state] :focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: var(--space-3xs); }
@media (max-width: 36rem) { [data-rcl-empty-state] { padding: var(--space-md); } [data-rcl-empty-state-action] { inline-size: 100%; } [data-rcl-empty-state-action] > button:not([data-rcl-control]), [data-rcl-empty-state-action] > a:not([data-rcl-control]) { inline-size: 100%; justify-content: center; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-empty-state-action] > button, [data-rcl-empty-state-action] > a { transition: none; } }
@media (forced-colors: active) { [data-rcl-empty-state] { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-empty-state-icon] { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-empty-state-action] > button, [data-rcl-empty-state-action] > a { border-color: CanvasText; background: CanvasText; color: Canvas; } }
`;
