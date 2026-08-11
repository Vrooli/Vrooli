/**
 * @vrooliComponentSource react-component-library:CommandCenterShell
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 84c3cc79-aa23-4837-9062-63b1430c03da
 * @vrooliComponentAppliedAt 2026-08-11T00:47:54Z
 * @vrooliComponentSourceSha256 3645b9e97e5ac510886045b948f21e566fc50071cb85012bb1ac778883443b6f
 * @vrooliComponentDriftHash 3645b9e97e5ac510886045b948f21e566fc50071cb85012bb1ac778883443b6f
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
export const commandCenterShellStyles = `
[data-rcl-command-center] { min-block-size: 100%; display: grid; grid-template-columns: minmax(0, 1fr); gap: var(--space-md); padding: var(--space-md); background: var(--color-background); color: var(--color-foreground); }
[data-rcl-command-center] .rcl-command-center__navigation { min-inline-size: 0; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-flat); padding: var(--space-sm); }
[data-rcl-command-center] .rcl-command-center__navigation > * { min-inline-size: 0; }
[data-rcl-command-center] .rcl-command-center__navigation a { display: flex; min-block-size: var(--tap-target-min); align-items: center; border-radius: var(--radius-control); color: var(--color-muted-foreground); padding-inline: var(--space-xs); text-decoration: none; font: 600 var(--text-body-sm-size)/var(--text-body-sm-line) var(--font-sans); transition: background-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard); }
[data-rcl-command-center] .rcl-command-center__navigation a:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-command-center] .rcl-command-center__navigation a:focus-visible, [data-rcl-command-center] button:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
[data-rcl-command-center] .rcl-command-center__body { min-inline-size: 0; display: grid; align-content: start; gap: var(--space-md); }
[data-rcl-command-center] .rcl-command-center__header { display: flex; min-inline-size: 0; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: var(--space-sm); }
[data-rcl-command-center] .rcl-command-center__title { min-inline-size: 0; margin: 0; color: var(--color-foreground); font-size: var(--text-title-size); font-weight: 700; line-height: var(--text-title-line); letter-spacing: -0.015em; }
[data-rcl-command-center] .rcl-command-center__controls { display: flex; min-inline-size: 0; flex-wrap: wrap; gap: var(--space-2xs); }
[data-rcl-command-center] .rcl-command-center__controls button { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); cursor: pointer; padding-inline: var(--space-sm); font: 600 var(--text-body-sm-size)/var(--text-body-sm-line) var(--font-sans); transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-command-center] .rcl-command-center__controls button:hover { border-color: var(--color-primary); background: var(--color-surface-muted); }
[data-rcl-command-center] .rcl-command-center__controls button:active { transform: translateY(1px); }
[data-rcl-command-center] .rcl-command-center__metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 10rem), 1fr)); gap: var(--space-sm); margin: 0; }
[data-rcl-command-center] .rcl-command-center__metric { min-inline-size: 0; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-flat); padding: var(--space-sm); }
[data-rcl-command-center] .rcl-command-center__metric-label { color: var(--color-muted-foreground); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); }
[data-rcl-command-center] .rcl-command-center__metric-value { margin: var(--space-3xs) 0 0; color: var(--color-foreground); font-size: var(--text-title-size); font-weight: 700; line-height: var(--text-title-line); letter-spacing: -0.015em; }
[data-rcl-command-center] .rcl-command-center__metric-detail { margin: var(--space-3xs) 0 0; color: var(--color-muted-foreground); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); }
[data-rcl-command-center] .rcl-command-center__primary { min-inline-size: 0; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-flat); padding: var(--space-md); }
[data-rcl-command-center] .rcl-command-center__primary > :not([role="status"]) { max-inline-size: 100%; }
@media (min-width: 48rem) { [data-rcl-command-center] { grid-template-columns: minmax(12rem, 15rem) minmax(0, 1fr); } }
@media (prefers-reduced-motion: reduce) { [data-rcl-command-center] *, [data-rcl-command-center] *::before, [data-rcl-command-center] *::after { transition-duration: .01ms !important; animation-duration: .01ms !important; } }
@media (forced-colors: active) { [data-rcl-command-center] .rcl-command-center__navigation, [data-rcl-command-center] .rcl-command-center__metric, [data-rcl-command-center] .rcl-command-center__primary, [data-rcl-command-center] .rcl-command-center__controls button { border-color: CanvasText; background: Canvas; color: CanvasText; } }
`;
