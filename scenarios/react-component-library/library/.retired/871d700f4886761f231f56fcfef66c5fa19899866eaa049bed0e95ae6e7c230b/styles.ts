export const codeBlockStyles = `
[data-rcl-code-block] { min-inline-size: 0; overflow: hidden; margin-block: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-code-block] .rcl-code-block__header { display: flex; align-items: center; justify-content: space-between; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-xs) var(--space-sm); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-code-block] button { border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-accent); padding: var(--space-3xs) var(--space-2xs); font: var(--text-label); cursor: pointer; }
[data-rcl-code-block] button:hover { background: color-mix(in srgb, var(--color-accent) 10%, transparent); }
[data-rcl-code-block] .rcl-code-block__body { overflow-x: auto; padding: var(--space-sm); color: var(--color-foreground); font: var(--text-body); }
[data-rcl-code-block] .rcl-code-block__body > pre, [data-rcl-code-block] .rcl-code-block__body pre { margin: 0; background: transparent; padding: 0; }
`;
