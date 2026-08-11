/**
 * @vrooliComponentSource react-component-library:StatusBadge
 * @vrooliComponentVersion 1.2.0
 * @vrooliComponentAdoption bf6f0ee0-c331-4c85-85eb-3107221d2f45
 * @vrooliComponentAppliedAt 2026-08-10T19:46:56Z
 * @vrooliComponentSourceSha256 a760758680c90def11bec2f3c1380309c9139459610fcfff68b4f15c5eb032c8
 * @vrooliComponentDriftHash a760758680c90def11bec2f3c1380309c9139459610fcfff68b4f15c5eb032c8
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
export const statusBadgeStyles = `
[data-rcl-status-badge] { display: inline-flex; min-block-size: calc(var(--text-label-line) + (var(--space-3xs) * 2)); max-inline-size: 100%; align-items: center; gap: var(--space-3xs); box-sizing: border-box; border: var(--border-hairline) solid var(--rcl-status-border, var(--color-border)); border-radius: var(--radius-pill); background: var(--rcl-status-surface, var(--color-surface-muted)); color: var(--rcl-status-accent, var(--color-muted-foreground)); padding: var(--space-3xs) var(--space-xs); font: var(--text-label); white-space: nowrap; }
[data-rcl-status-badge][data-tone="neutral"] { --rcl-status-accent: var(--color-muted-foreground); --rcl-status-surface: var(--color-surface-muted); --rcl-status-border: var(--color-border); }
[data-rcl-status-badge][data-tone="success"] { --rcl-status-accent: var(--color-success); --rcl-status-surface: color-mix(in srgb, var(--color-success) 10%, var(--color-surface)); --rcl-status-border: color-mix(in srgb, var(--color-success) 32%, var(--color-border)); }
[data-rcl-status-badge][data-tone="warning"] { --rcl-status-accent: var(--color-warning); --rcl-status-surface: color-mix(in srgb, var(--color-warning) 11%, var(--color-surface)); --rcl-status-border: color-mix(in srgb, var(--color-warning) 38%, var(--color-border)); }
[data-rcl-status-badge][data-tone="danger"] { --rcl-status-accent: var(--color-danger); --rcl-status-surface: color-mix(in srgb, var(--color-danger) 10%, var(--color-surface)); --rcl-status-border: color-mix(in srgb, var(--color-danger) 34%, var(--color-border)); }
[data-rcl-status-badge][data-tone="info"] { --rcl-status-accent: var(--color-info); --rcl-status-surface: color-mix(in srgb, var(--color-info) 10%, var(--color-surface)); --rcl-status-border: color-mix(in srgb, var(--color-info) 32%, var(--color-border)); }
[data-rcl-status-badge-indicator] { inline-size: var(--space-2xs); block-size: var(--space-2xs); flex: 0 0 auto; border-radius: var(--radius-pill); background: currentColor; box-shadow: 0 0 0 var(--space-3xs) color-mix(in srgb, currentColor 12%, transparent); }
[data-rcl-status-badge-label] { min-inline-size: 0; overflow: hidden; text-overflow: ellipsis; }
@media (forced-colors: active) { [data-rcl-status-badge] { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-status-badge-indicator] { background: CanvasText; box-shadow: none; } }
`;
