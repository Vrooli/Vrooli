/**
 * @vrooliComponentSource react-component-library:InspectorLayout
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption 6323c8e4-8bee-4081-9279-97931a8c27a3
 * @vrooliComponentAppliedAt 2026-08-11T00:47:54Z
 * @vrooliComponentSourceSha256 b893efaadcaaa50288a45c6296a5a1ae940219cfc2615b8a2ad3129e0de401ec
 * @vrooliComponentDriftHash b893efaadcaaa50288a45c6296a5a1ae940219cfc2615b8a2ad3129e0de401ec
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
export const inspectorLayoutStyles = `
[data-rcl-inspector-layout] { display: grid; min-inline-size: 0; min-block-size: 100%; grid-template-columns: minmax(0, 1fr); gap: var(--space-md); padding: var(--space-md); color: var(--color-foreground); }
[data-rcl-inspector-canvas] { display: grid; min-inline-size: 0; min-block-size: calc(var(--space-xl) * 6); grid-template-rows: auto minmax(0, 1fr); overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-flat); }
[data-rcl-inspector-toolbar] { display: flex; min-inline-size: 0; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface-raised); padding: var(--space-sm) var(--space-md); }
[data-rcl-inspector-title] { min-inline-size: 0; margin: 0; color: var(--color-foreground); font: var(--text-title); overflow-wrap: anywhere; }
[data-rcl-visually-hidden] { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap; }
[data-rcl-inspector-toolbar] > button:not([data-rcl-control]), [data-rcl-inspector-toolbar] > a:not([data-rcl-control]) { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); padding-inline: var(--space-sm); font: var(--text-label); text-decoration: none; cursor: pointer; }
[data-rcl-inspector-canvas-body] { min-inline-size: 0; min-block-size: 0; padding: var(--space-md); background: radial-gradient(circle at 1px 1px, color-mix(in srgb, var(--color-border) 55%, transparent) 1px, transparent 0) 0 0 / var(--space-sm) var(--space-sm), var(--color-surface-muted); }
[data-rcl-inspector-panel], .rcl-inspector-panel { min-inline-size: 0; align-self: start; overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); padding: var(--space-md); box-shadow: var(--elev-flat); }
[data-rcl-inspector-layout] :focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: var(--space-3xs); }
@media (min-width: 60rem) { [data-rcl-inspector-layout] { grid-template-columns: minmax(0, 1fr) minmax(0, min(var(--sidebar-width), 36%)); } }
@media (max-width: 36rem) { [data-rcl-inspector-layout] { padding: var(--space-sm); } [data-rcl-inspector-toolbar] { align-items: flex-start; } [data-rcl-inspector-toolbar] > * { max-inline-size: 100%; } }
@media (forced-colors: active) { [data-rcl-inspector-canvas], [data-rcl-inspector-panel] { border-color: CanvasText; background: Canvas; box-shadow: none; } [data-rcl-inspector-canvas-body] { background: Canvas; } }
`;
