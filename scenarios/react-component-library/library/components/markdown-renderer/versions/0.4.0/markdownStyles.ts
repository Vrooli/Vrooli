export const markdownStyles = `
[data-rcl-markdown] { min-inline-size: 0; color: var(--color-foreground); font: var(--text-body); line-height: 1.55; }
[data-rcl-markdown] a, [data-rcl-markdown] .rcl-md__link, [data-rcl-markdown].rcl-md__link { color: var(--color-accent); text-decoration: underline; text-underline-offset: var(--space-3xs); }
[data-rcl-markdown] .rcl-md__blockquote { margin-block: var(--space-sm); border-inline-start: var(--border-strong) solid var(--color-accent); padding-inline-start: var(--space-sm); color: var(--color-muted-foreground); font-style: italic; }
[data-rcl-markdown] .rcl-md__table-scroll { margin-block: var(--space-sm); overflow-x: auto; }
[data-rcl-markdown] table { border-collapse: collapse; font: var(--text-body); }
[data-rcl-markdown] th, [data-rcl-markdown] td { border: var(--border-hairline) solid var(--color-border); padding: var(--space-2xs) var(--space-xs); text-align: start; }
[data-rcl-markdown].rcl-md__code, [data-rcl-markdown] .rcl-md__code, [data-rcl-markdown].rcl-md__diagram, [data-rcl-markdown] .rcl-md__diagram { margin-block: var(--space-sm); overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-muted); }
[data-rcl-markdown] .rcl-md__code-header, [data-rcl-markdown] .rcl-md__diagram-header { display: flex; align-items: center; justify-content: space-between; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-xs) var(--space-sm); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-markdown] .rcl-md__code-header button, [data-rcl-markdown] .rcl-md__diagram-header button, [data-rcl-markdown] .rcl-md__inline-copy { border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-accent); padding: var(--space-3xs) var(--space-2xs); font: var(--text-label); cursor: pointer; }
[data-rcl-markdown] .rcl-md__code-header button:hover, [data-rcl-markdown] .rcl-md__diagram-header button:hover, [data-rcl-markdown] .rcl-md__inline-copy:hover { background: color-mix(in srgb, var(--color-accent) 10%, transparent); }
[data-rcl-markdown] .rcl-md__code-body, [data-rcl-markdown] .rcl-md__diagram-body { overflow-x: auto; padding: var(--space-sm); color: var(--color-foreground); font: var(--text-body); }
[data-rcl-markdown] .rcl-md__code-body > pre, [data-rcl-markdown] .rcl-md__code-body pre { margin: 0; background: transparent; padding: 0; }
[data-rcl-markdown].rcl-md__inline, [data-rcl-markdown] .rcl-md__inline { display: inline-flex; position: relative; align-items: center; gap: var(--space-3xs); }
[data-rcl-markdown].rcl-md__inline-token, [data-rcl-markdown] .rcl-md__inline-token { border-radius: var(--radius-control); background: var(--color-surface-muted); padding: var(--space-3xs) var(--space-2xs); color: var(--color-foreground); font-family: var(--font-mono, ui-monospace, monospace); }
[data-rcl-markdown] .rcl-md__inline-copy { visibility: hidden; color: var(--color-muted-foreground); }
[data-rcl-markdown].rcl-md__inline:hover .rcl-md__inline-copy, [data-rcl-markdown] .rcl-md__inline:hover .rcl-md__inline-copy, [data-rcl-markdown].rcl-md__inline:focus-within .rcl-md__inline-copy, [data-rcl-markdown] .rcl-md__inline:focus-within .rcl-md__inline-copy { visibility: visible; }
[data-rcl-markdown] .rcl-md__diagram-actions, [data-rcl-markdown] .rcl-md__diagram-tabs { display: flex; align-items: center; gap: var(--space-xs); }
[data-rcl-markdown] .rcl-md__diagram-header button[aria-pressed="true"] { color: var(--color-foreground); background: color-mix(in srgb, var(--color-accent) 12%, transparent); }
[data-rcl-markdown] .rcl-md__error { padding: var(--space-sm); color: var(--color-danger); font: var(--text-caption); }
`;
