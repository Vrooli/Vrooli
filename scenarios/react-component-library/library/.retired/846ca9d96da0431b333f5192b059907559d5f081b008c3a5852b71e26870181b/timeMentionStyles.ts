/**
 * Mention and detail-card rules, keyed on data attributes so they apply in any
 * renderer that maps `time` to `MarkdownTimeMention`, not only this library's.
 *
 * The underline is the only mark on the original text. Everything the reader
 * did not write — their local time, a countdown — sits beside it in a quieter,
 * monospaced tag that is never part of a text selection.
 */
export const timeMentionStyles = `
[data-rcl-time-mention] { cursor: pointer; color: inherit; border-radius: var(--radius-sm); text-decoration-line: underline; text-decoration-style: dotted; text-decoration-thickness: 1.5px; text-decoration-color: color-mix(in srgb, var(--color-accent) 70%, transparent); text-underline-offset: 0.24em; -webkit-tap-highlight-color: transparent; transition: background-color var(--dur-quick) var(--ease-standard); }
[data-rcl-time-mention]:hover, [data-rcl-time-mention][aria-expanded="true"] { background: color-mix(in srgb, var(--color-accent) 12%, transparent); }
[data-rcl-time-mention]:focus-visible { outline: var(--border-strong, 2px) solid var(--color-focus); outline-offset: 1px; }
[data-rcl-time-mention][data-tone="unzoned"] { text-decoration-color: var(--color-border-strong); }
[data-rcl-time-mention][data-tone="lint"] { text-decoration-style: wavy; text-decoration-thickness: 1px; text-decoration-color: var(--color-warning); }
[data-rcl-time-mention-tag] { display: inline-flex; align-items: center; gap: 0.3em; margin-inline-start: 0.35em; padding: 0 0.42em; border: 1px solid var(--color-border-subtle); border-radius: var(--radius-sm); background: var(--color-surface-muted); color: var(--color-muted-foreground); font-family: var(--font-mono); font-size: 0.78em; font-weight: 500; line-height: 1.5; font-variant-numeric: tabular-nums; white-space: nowrap; vertical-align: 0.06em; user-select: none; -webkit-user-select: none; }
[data-rcl-time-mention-tag] svg { inline-size: 1em; block-size: 1em; flex: none; }
[data-rcl-time-mention-tag] [data-night] { color: var(--color-info); }
[data-rcl-time-mention-tag][data-tone="lint"], [data-rcl-time-mention-tag][data-tone="stale"] { border-color: transparent; background: var(--color-warning-subtle); color: var(--color-warning-foreground); }
[data-rcl-time-mention-tag][data-tone="unzoned"] { border-style: dashed; background: transparent; }
[data-rcl-time-mention-badge] { display: inline-flex; align-items: center; gap: 0.4em; margin-inline-start: 0.35em; padding: 0 0.55em; border-radius: var(--radius-pill); background: color-mix(in srgb, var(--color-accent) 14%, transparent); color: var(--color-foreground); font-family: var(--font-mono); font-size: 0.74em; font-weight: 600; line-height: 1.55; font-variant-numeric: tabular-nums; white-space: nowrap; vertical-align: 0.08em; user-select: none; -webkit-user-select: none; }
[data-rcl-time-mention-badge]::before { content: ""; inline-size: 0.5em; block-size: 0.5em; border-radius: 50%; background: var(--color-accent); }
[data-rcl-time-mention-badge][data-state="soon"] { background: var(--color-warning-subtle); color: var(--color-warning-foreground); }
[data-rcl-time-mention-badge][data-state="soon"]::before { background: var(--color-warning); }
[data-rcl-time-mention-badge][data-state="past"] { background: var(--color-surface-muted); color: var(--color-muted-foreground); }
[data-rcl-time-mention-badge][data-state="past"]::before { background: transparent; box-shadow: inset 0 0 0 1.5px currentColor; }

[data-rcl-time-mention-card] { display: grid; min-inline-size: 0; color: var(--color-foreground); font: var(--text-body-sm); }
[data-rcl-time-mention-card] > * + * { border-block-start: 1px solid var(--color-border-subtle); }
[data-rcl-time-mention-card-head] { display: grid; gap: var(--space-3xs); padding: var(--space-md); }
[data-rcl-time-mention-card-primary] { display: flex; align-items: center; gap: var(--space-xs); margin: 0; font-family: var(--font-mono); font-size: 1.375rem; font-weight: 600; line-height: 1.2; letter-spacing: -0.01em; font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
[data-rcl-time-mention-card-primary] svg { inline-size: 1.1rem; block-size: 1.1rem; flex: none; color: var(--color-info); }
[data-rcl-time-mention-card-secondary] { margin: 0; color: var(--color-muted-foreground); }
[data-rcl-time-mention-card-source] { margin: var(--space-3xs) 0 0; color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-time-mention-card-source] code { padding: 0 var(--space-3xs); border-radius: var(--radius-sm); background: var(--color-surface-muted); color: var(--color-foreground); font-family: var(--font-mono); font-size: 0.95em; }
[data-rcl-time-mention-card-rows] { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: center; gap: var(--space-2xs) var(--space-md); margin: 0; padding: var(--space-sm) var(--space-md); }
[data-rcl-time-mention-card-rows] dt { color: var(--color-muted-foreground); }
[data-rcl-time-mention-card-rows] dd { display: flex; align-items: center; justify-content: flex-end; gap: var(--space-3xs); margin: 0; min-inline-size: 0; font-family: var(--font-mono); font-size: 0.92em; font-variant-numeric: tabular-nums; text-align: end; overflow-wrap: anywhere; }
[data-rcl-time-mention-card] [data-rcl-time-mention-card-rows] dd :is(button) { min-block-size: 0; min-inline-size: 0; }
[data-rcl-time-mention-card-note] { display: flex; align-items: flex-start; gap: var(--space-xs); margin: 0; padding: var(--space-sm) var(--space-md); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-time-mention-card-note] svg { inline-size: 1rem; block-size: 1rem; flex: none; margin-block-start: 0.1em; }
[data-rcl-time-mention-card-note][data-tone="warning"] { background: var(--color-warning-subtle); color: var(--color-warning-foreground); }
[data-rcl-time-mention-card-display] { display: grid; gap: var(--space-xs); padding: var(--space-sm) var(--space-md) var(--space-md); background: var(--color-surface-muted); }
[data-rcl-time-mention-card-display-label] { display: flex; justify-content: space-between; gap: var(--space-sm); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-time-mention-card-display] [data-rcl-button-group] { display: flex; inline-size: 100%; flex-wrap: nowrap; }
[data-rcl-time-mention-card-display] [data-rcl-button-group] > [data-rcl-button] { flex: 1 1 0; }
@media (prefers-reduced-motion: reduce) { [data-rcl-time-mention] { transition: none; } }
`;