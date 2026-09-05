export const cardStyles = `
[data-rcl-card] { min-inline-size: 0; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); }
[data-rcl-card] .rcl-card__header { display: flex; min-inline-size: 0; flex-direction: column; gap: var(--space-2xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-sm) var(--space-md); }
[data-rcl-card] .rcl-card__title { color: var(--color-foreground); font-family: var(--font-sans); font-size: var(--text-heading-size); font-weight: 650; line-height: var(--text-heading-line); }
[data-rcl-card] .rcl-card__description { color: var(--color-muted-foreground); font-family: var(--font-sans); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); }
[data-rcl-card] .rcl-card__content { min-inline-size: 0; padding: var(--space-md); }
[data-rcl-card] :focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
@media (prefers-reduced-motion: reduce) { [data-rcl-card] *, [data-rcl-card] *::before, [data-rcl-card] *::after { transition-duration: .01ms; } }
@media (forced-colors: active) { [data-rcl-card] { border-color: CanvasText; background: Canvas; color: CanvasText; } }
`;
