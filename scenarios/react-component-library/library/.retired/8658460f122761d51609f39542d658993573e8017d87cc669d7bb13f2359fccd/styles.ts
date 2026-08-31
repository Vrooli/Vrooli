export const mermaidStyles = `
[data-rcl-mermaid] { min-inline-size: 0; overflow: hidden; margin-block: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-mermaid] .rcl-mermaid__header { display: flex; align-items: center; justify-content: space-between; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-xs) var(--space-sm); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-mermaid] .rcl-mermaid__tabs, [data-rcl-mermaid] .rcl-mermaid__actions { display: flex; align-items: center; gap: var(--space-xs); }
[data-rcl-mermaid] button { border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-accent); padding: var(--space-3xs) var(--space-2xs); font: var(--text-label); cursor: pointer; }
[data-rcl-mermaid] button:hover, [data-rcl-mermaid] button[aria-pressed="true"] { background: color-mix(in srgb, var(--color-accent) 10%, transparent); color: var(--color-foreground); }
[data-rcl-mermaid] .rcl-mermaid__body { overflow-x: auto; padding: var(--space-sm); color: var(--color-foreground); font: var(--text-body); }
[data-rcl-mermaid] .rcl-mermaid__error { padding: var(--space-sm); color: var(--color-danger); font: var(--text-caption); }
[data-rcl-mermaid] .rcl-mermaid__body > svg { display: block; max-inline-size: 100%; block-size: auto; margin-inline: auto; }
`;
