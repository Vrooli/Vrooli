/**
 * @vrooliComponentSource react-component-library:AssetDetailShell
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption a1876380-85df-4ed9-a8b1-ce6008e168a0
 * @vrooliComponentAppliedAt 2026-08-11T00:47:57Z
 * @vrooliComponentSourceSha256 8ae7dfd06f5739cb5d943ed24e5d864346aef272b3e8a873287109ad51fa6e56
 * @vrooliComponentDriftHash 8ae7dfd06f5739cb5d943ed24e5d864346aef272b3e8a873287109ad51fa6e56
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
export const assetDetailShellStyles = `
[data-rcl-asset-detail] { display: grid; inline-size: min(100%, calc(var(--space-xl) * 32)); min-inline-size: 0; margin-inline: auto; grid-template-columns: minmax(0, 1fr); gap: var(--space-md); padding: var(--space-md); color: var(--color-foreground); }
[data-rcl-asset-detail-primary] { display: grid; min-inline-size: 0; gap: var(--space-md); }
[data-rcl-asset-detail-header] { display: flex; min-inline-size: 0; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: var(--space-xs); }
[data-rcl-asset-detail-title] { min-inline-size: 0; margin: 0; color: var(--color-foreground); font: var(--text-title); letter-spacing: -0.02em; overflow-wrap: anywhere; }
[data-rcl-asset-detail-actions] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-2xs); }
[data-rcl-asset-detail-actions] > button:not([data-rcl-control]), [data-rcl-asset-detail-actions] > a:not([data-rcl-control]) { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); padding-inline: var(--space-sm); font: var(--text-label); text-decoration: none; cursor: pointer; }
[data-rcl-asset-detail-preview], [data-rcl-asset-detail-metadata], [data-rcl-asset-detail-activity], .rcl-asset-detail-activity { min-inline-size: 0; align-self: start; overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-flat); }
[data-rcl-asset-detail-preview] { min-block-size: calc(var(--space-xl) * 6); display: grid; place-items: center; overflow: hidden; background: radial-gradient(circle at 20% 10%, color-mix(in srgb, var(--color-primary) 13%, transparent), transparent 45%), var(--color-surface); }
[data-rcl-asset-detail-preview] img { display: block; max-inline-size: 100%; block-size: auto; }
[data-rcl-asset-detail-preview] > * { max-inline-size: 100%; }
[data-rcl-asset-detail-metadata] { padding: var(--space-md); }
[data-rcl-asset-detail-activity], .rcl-asset-detail-activity { padding: var(--space-md); }
[data-rcl-asset-detail] :focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: var(--space-3xs); }
@media (min-width: 60rem) { [data-rcl-asset-detail] { grid-template-columns: minmax(0, 1fr) minmax(0, min(var(--sidebar-width), 36%)); } }
@media (max-width: 36rem) { [data-rcl-asset-detail] { padding: var(--space-sm); } [data-rcl-asset-detail-header] { align-items: flex-start; } [data-rcl-asset-detail-actions] { inline-size: 100%; } [data-rcl-asset-detail-actions] > button:not([data-rcl-control]), [data-rcl-asset-detail-actions] > a:not([data-rcl-control]) { flex: 1 1 auto; } }
@media (forced-colors: active) { [data-rcl-asset-detail-preview], [data-rcl-asset-detail-metadata], [data-rcl-asset-detail-activity] { border-color: CanvasText; background: Canvas; box-shadow: none; } }
`;
