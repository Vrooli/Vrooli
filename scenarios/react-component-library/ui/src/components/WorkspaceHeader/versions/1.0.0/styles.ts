/**
 * @vrooliComponentSource react-component-library:WorkspaceHeader
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption a4ba3a93-b98a-411e-b945-cf4ff726f2de
 * @vrooliComponentAppliedAt 2026-08-10T22:14:50Z
 * @vrooliComponentSourceSha256 c71c6c033d70ad308b4099013feb82e39f827a8499efd475da6d7cba11d7f4a4
 * @vrooliComponentDriftHash c71c6c033d70ad308b4099013feb82e39f827a8499efd475da6d7cba11d7f4a4
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
export const workspaceHeaderStyles = `
[data-rcl-workspace-header] { inline-size: 100%; min-inline-size: 0; flex-shrink: 0; overflow: hidden; border-block-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface); color: var(--color-foreground); }
[data-rcl-workspace-header] .rcl-workspace-header__row { display: flex; min-block-size: var(--tap-target-min); min-inline-size: 0; align-items: center; gap: var(--space-xs); padding: var(--space-2xs) var(--space-xs); }
[data-rcl-workspace-header] .rcl-workspace-header__leading { display: flex; min-inline-size: 0; flex-shrink: 0; align-items: center; }
[data-rcl-workspace-header] .rcl-workspace-header__copy { min-inline-size: 0; flex: 1; }
[data-rcl-workspace-header] .rcl-workspace-header__title { overflow: hidden; margin: 0; color: var(--color-foreground); font-size: var(--text-heading-size); font-weight: 700; line-height: var(--text-heading-line); letter-spacing: -0.01em; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-workspace-header] .rcl-workspace-header__description { overflow: hidden; margin: var(--space-3xs) 0 0; color: var(--color-muted-foreground); font-size: var(--text-caption-size); line-height: var(--text-caption-line); text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-workspace-header] .rcl-workspace-header__actions { display: flex; min-inline-size: 0; flex-shrink: 0; align-items: center; gap: var(--space-2xs); }
[data-rcl-workspace-header] .rcl-workspace-header__actions button { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-primary); border-radius: var(--radius-control); background: var(--color-primary); color: var(--color-primary-foreground); padding-inline: var(--space-sm); font: 600 var(--text-body-sm-size)/var(--text-body-sm-line) var(--font-sans); cursor: pointer; transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-workspace-header] .rcl-workspace-header__actions button:hover { filter: brightness(0.96); }
[data-rcl-workspace-header] .rcl-workspace-header__actions button:active { transform: translateY(1px); }
[data-rcl-workspace-header] button:focus-visible, [data-rcl-workspace-header] a:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
[data-rcl-workspace-header] .rcl-workspace-header__children { min-inline-size: 0; border-block-start: var(--border-hairline) solid var(--color-border); padding-inline: var(--space-xs); }
@media (min-width: 40rem) { [data-rcl-workspace-header] .rcl-workspace-header__row { padding-inline: var(--space-sm); } [data-rcl-workspace-header] .rcl-workspace-header__children { padding-inline: var(--space-sm); } }
@media (max-width: 30rem) { [data-rcl-workspace-header] .rcl-workspace-header__row { flex-wrap: wrap; align-items: flex-start; } [data-rcl-workspace-header] .rcl-workspace-header__copy { flex-basis: calc(100% - var(--space-xs)); } [data-rcl-workspace-header] .rcl-workspace-header__actions { inline-size: 100%; justify-content: flex-start; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-workspace-header] *, [data-rcl-workspace-header] *::before, [data-rcl-workspace-header] *::after { transition-duration: .01ms !important; animation-duration: .01ms !important; } }
@media (forced-colors: active) { [data-rcl-workspace-header] { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-workspace-header] .rcl-workspace-header__actions button { border-color: CanvasText; background: Highlight; color: HighlightText; } }
`;
