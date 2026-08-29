export const fullPageDrawerStyles = `
[data-rcl-full-page-drawer] { position: fixed; inset: 0; z-index: var(--layer-modal, 400); pointer-events: none; }

.rcl-full-page-drawer__backdrop { position: absolute; inset: 0; margin: 0; padding: 0; border: 0; background: var(--color-scrim, rgb(15 23 42 / .52)); pointer-events: auto; opacity: 1; transition: opacity var(--dur-quick) var(--ease-standard); }
[data-rcl-full-page-drawer][data-state="closed"] .rcl-full-page-drawer__backdrop { opacity: 0; }

.rcl-full-page-drawer__panel { position: absolute; inset-inline: 0; inset-block-start: calc(var(--rcl-safe-top, 0px) + var(--overlay-drawer-top-gap, 32px)); inset-block-end: 0; display: flex; flex-direction: column; min-block-size: 0; overflow: hidden; border-radius: var(--radius-sheet) var(--radius-sheet) 0 0; background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-modal); pointer-events: auto; transition: transform var(--dur-moderate) var(--ease-standard); animation: rcl-full-page-drawer-enter var(--dur-moderate) var(--ease-enter); }
.rcl-full-page-drawer__panel[data-dragging="true"] { transition: none; will-change: transform; }
[data-rcl-full-page-drawer][data-state="closed"] .rcl-full-page-drawer__panel { transform: translateY(100%); animation: none; }
[data-rcl-full-page-drawer][data-avoid-keyboard] .rcl-full-page-drawer__panel { inset-block-end: var(--rcl-keyboard-inset, 0px); }
@keyframes rcl-full-page-drawer-enter { from { transform: translateY(100%); } }

.rcl-full-page-drawer__grabber { position: absolute; z-index: 1; inset-block-start: 0; inset-inline-start: 50%; translate: -50% 0; inline-size: min(60%, 12rem); min-block-size: var(--tap-target-min); display: grid; justify-items: center; align-content: start; padding: var(--space-2xs) 0 0; margin: 0; border: 0; background: transparent; color: inherit; touch-action: none; cursor: grab; }
.rcl-full-page-drawer__grabber[data-rcl-overlay-dragging="true"] { cursor: grabbing; }
.rcl-full-page-drawer__grabber > span { inline-size: var(--overlay-grabber-inline, 2.25rem); block-size: var(--overlay-grabber-block, .25rem); border-radius: var(--radius-pill); background: var(--color-border-strong, var(--color-border)); }

.rcl-full-page-drawer__panel > header, .rcl-full-page-drawer__panel > footer { flex: 0 0 auto; display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm); padding: var(--space-sm) var(--space-md); }
.rcl-full-page-drawer__panel > header { border-block-end: var(--border-hairline) solid var(--color-border); }
.rcl-full-page-drawer__panel > footer { border-block-start: var(--border-hairline) solid var(--color-border); padding-block-end: calc(var(--space-sm) + var(--rcl-safe-bottom, 0px)); }
.rcl-full-page-drawer__panel > header > *:first-child { min-inline-size: 0; }
.rcl-full-page-drawer__panel > header > *:last-child { display: flex; flex: 0 0 auto; align-items: center; gap: var(--space-3xs); }
.rcl-full-page-drawer__panel h2 { margin: 0; font: var(--text-heading); }

.rcl-full-page-drawer__subheader { flex: 0 0 auto; min-block-size: 0; border-block-end: var(--border-hairline) solid var(--color-border); }

.rcl-full-page-drawer__content { flex: 1 1 auto; min-block-size: 0; overflow: auto; overscroll-behavior: contain; padding-block-end: var(--rcl-safe-bottom, 0px); }
[data-rcl-full-page-drawer][data-content-padding="comfortable"] .rcl-full-page-drawer__content { padding: var(--space-md); padding-block-end: calc(var(--space-md) + var(--rcl-safe-bottom, 0px)); }
[data-rcl-full-page-drawer][data-has-footer] .rcl-full-page-drawer__content { padding-block-end: 0; }
[data-rcl-full-page-drawer][data-has-footer][data-content-padding="comfortable"] .rcl-full-page-drawer__content { padding-block-end: var(--space-md); }

@media (min-width: 48rem) {
  .rcl-full-page-drawer__panel { inset: var(--space-md); border-radius: var(--radius-panel); animation-name: rcl-full-page-drawer-enter-inset; }
  [data-rcl-full-page-drawer][data-state="closed"] .rcl-full-page-drawer__panel { transform: translateY(var(--space-2xs)); opacity: 0; }
  [data-rcl-full-page-drawer][data-avoid-keyboard] .rcl-full-page-drawer__panel { inset-block-end: max(var(--space-md), var(--rcl-keyboard-inset, 0px)); }
  .rcl-full-page-drawer__panel > header, .rcl-full-page-drawer__panel > footer { padding: var(--space-md); }
  .rcl-full-page-drawer__panel > footer { padding-block-end: var(--space-md); }
  .rcl-full-page-drawer__content { padding-block-end: 0; }
  [data-rcl-full-page-drawer][data-content-padding="comfortable"] .rcl-full-page-drawer__content { padding: var(--space-md); }
}
@keyframes rcl-full-page-drawer-enter-inset { from { transform: translateY(var(--space-2xs)); opacity: 0; } }
.rcl-full-page-drawer__panel [data-icon] { flex: 0 0 auto; inline-size: var(--icon-size-md); block-size: var(--icon-size-md); }
`;
