export const emptyStateStyles = `
[data-rcl-empty-state] { min-inline-size: 0; max-inline-size: 48rem; overflow: hidden; }
[data-rcl-empty-state] .rcl-empty-state__content { display: grid; justify-items: center; gap: var(--space-md); padding: var(--space-2xl) var(--space-lg); text-align: center; }
[data-rcl-empty-state-icon] { display: grid; inline-size: var(--space-2xl); block-size: var(--space-2xl); flex: 0 0 auto; place-items: center; border: var(--border-hairline) solid color-mix(in srgb, var(--color-primary) 22%, var(--color-border)); border-radius: var(--radius-pill); background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface)); color: var(--color-primary); }
[data-rcl-empty-state-copy] { display: grid; min-inline-size: 0; gap: var(--space-3xs); max-inline-size: 55ch; }
[data-rcl-empty-state-title] { margin: 0; }
[data-rcl-empty-state-description] { margin: 0; overflow-wrap: anywhere; }
[data-rcl-empty-state-action] { display: flex; flex-wrap: wrap; align-items: center; justify-content: center; gap: var(--space-2xs); }
[data-rcl-empty-state] :focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: var(--space-3xs); }
@media (max-width: 36rem) { [data-rcl-empty-state] .rcl-empty-state__content { padding: var(--space-xl) var(--space-md); } [data-rcl-empty-state-action] { inline-size: 100%; } [data-rcl-empty-state-action] > [data-rcl-control] { inline-size: 100%; justify-content: center; } }
@media (forced-colors: active) { [data-rcl-empty-state] { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-empty-state-icon] { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-empty-state-action] > button, [data-rcl-empty-state-action] > a { border-color: CanvasText; background: CanvasText; color: Canvas; } }
`;
