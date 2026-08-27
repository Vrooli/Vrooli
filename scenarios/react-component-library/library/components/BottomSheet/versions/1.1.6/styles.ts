export const bottomSheetStyles = `
[data-rcl-bottom-sheet] { position: fixed; inset: 0; z-index: var(--layer-modal, 400); display: grid; align-items: end; pointer-events: none; }
[data-rcl-bottom-sheet][data-avoid-keyboard] { inset-block-end: var(--rcl-keyboard-inset, 0px); }

.rcl-bottom-sheet__backdrop { position: absolute; inset: 0; margin: 0; padding: 0; border: 0; background: var(--color-scrim, rgb(15 23 42 / .52)); pointer-events: auto; opacity: 1; transition: opacity var(--dur-quick) var(--ease-standard); }
[data-rcl-bottom-sheet][data-state="closed"] .rcl-bottom-sheet__backdrop { opacity: 0; }

.rcl-bottom-sheet__panel { --rcl-overlay-progress: 0; position: relative; display: flex; flex-direction: column; inline-size: 100%; max-block-size: calc(var(--rcl-viewport-height, 100dvh) - var(--rcl-safe-top, 0px) - var(--overlay-drawer-top-gap, 32px)); overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-block-end: 0; border-radius: var(--radius-sheet) var(--radius-sheet) 0 0; background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-modal); pointer-events: auto; transform: translateY(calc(var(--rcl-overlay-progress) * 100%)); transition: transform var(--dur-quick) var(--ease-standard); animation: rcl-bottom-sheet-enter var(--dur-moderate) var(--ease-enter); }
.rcl-bottom-sheet__panel[data-dragging="true"] { transition: none; will-change: transform; }
[data-rcl-bottom-sheet][data-state="closed"] .rcl-bottom-sheet__panel { transform: translateY(100%); animation: none; }
@keyframes rcl-bottom-sheet-enter { from { transform: translateY(100%); } }

.rcl-bottom-sheet__grabber { position: absolute; z-index: 1; inset-block-start: 0; inset-inline-start: 50%; translate: -50% 0; inline-size: min(60%, 12rem); min-block-size: var(--tap-target-min); display: grid; justify-items: center; align-content: start; padding: var(--space-2xs) 0 0; margin: 0; border: 0; background: transparent; color: inherit; touch-action: none; cursor: grab; }
.rcl-bottom-sheet__grabber[data-rcl-overlay-dragging="true"] { cursor: grabbing; }
.rcl-bottom-sheet__grabber > span { inline-size: var(--overlay-grabber-inline, 2.25rem); block-size: var(--overlay-grabber-block, .25rem); border-radius: var(--radius-pill); background: var(--color-border-strong, var(--color-border)); }

.rcl-bottom-sheet__header, .rcl-bottom-sheet__footer { flex: 0 0 auto; display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm); padding: var(--space-sm) var(--space-md); }
.rcl-bottom-sheet__header { border-block-end: var(--border-hairline) solid var(--color-border); }
.rcl-bottom-sheet__footer { border-block-start: var(--border-hairline) solid var(--color-border); padding-block-end: calc(var(--space-sm) + var(--rcl-safe-bottom, 0px)); }
.rcl-bottom-sheet__header > *:first-child { min-inline-size: 0; }
.rcl-bottom-sheet__header > *:last-child { display: flex; flex: 0 0 auto; align-items: center; gap: var(--space-3xs); }
.rcl-bottom-sheet__header h2 { margin: 0; font: var(--text-heading); }

.rcl-bottom-sheet__subheader { flex: 0 0 auto; min-block-size: 0; border-block-end: var(--border-hairline) solid var(--color-border); }

.rcl-bottom-sheet__content { flex: 1 1 auto; min-block-size: 0; overflow: auto; overscroll-behavior: contain; padding-block-end: var(--rcl-safe-bottom, 0px); }
[data-rcl-bottom-sheet][data-content-padding="comfortable"] .rcl-bottom-sheet__content { padding: var(--space-md); padding-block-end: calc(var(--space-md) + var(--rcl-safe-bottom, 0px)); }
[data-rcl-bottom-sheet][data-has-footer] .rcl-bottom-sheet__content { padding-block-end: 0; }
[data-rcl-bottom-sheet][data-has-footer][data-content-padding="comfortable"] .rcl-bottom-sheet__content { padding-block-end: var(--space-md); }

.rcl-bottom-sheet__panel [data-icon] { flex: 0 0 auto; inline-size: var(--icon-size-md); block-size: var(--icon-size-md); }

@media (min-width: 48rem) {
  [data-rcl-bottom-sheet] { padding: var(--space-xs); padding-block-end: max(var(--space-xs), var(--rcl-safe-bottom, 0px)); }
  [data-rcl-bottom-sheet][data-avoid-keyboard] { padding-block-end: var(--space-xs); }
  .rcl-bottom-sheet__panel { inline-size: min(100%, 42rem); margin-inline: auto; border-block-end: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-sheet); }
  [data-rcl-bottom-sheet][data-state="closed"] .rcl-bottom-sheet__panel { transform: translateY(calc(100% + var(--space-xs))); }
  .rcl-bottom-sheet__footer { padding-block-end: var(--space-sm); }
  .rcl-bottom-sheet__content { padding-block-end: 0; }
  [data-rcl-bottom-sheet][data-content-padding="comfortable"] .rcl-bottom-sheet__content { padding: var(--space-md); }
}
`;
