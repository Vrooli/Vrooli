export const responsiveDialogStyles = `
[data-rcl-responsive-dialog] { position: fixed; inset: 0; z-index: var(--layer-modal, 400); display: grid; align-items: end; pointer-events: none; }
[data-rcl-responsive-dialog][data-avoid-keyboard] { inset: auto; inset-inline-start: var(--rcl-viewport-offset-left, 0px); inset-block-start: var(--rcl-viewport-offset-top, 0px); inline-size: var(--rcl-viewport-width, 100vw); block-size: var(--rcl-viewport-height, 100dvh); }

.rcl-responsive-dialog__backdrop { position: absolute; inset: 0; margin: 0; padding: 0; border: 0; background: var(--color-scrim, rgb(15 23 42 / .52)); pointer-events: auto; opacity: 1; transition: opacity var(--dur-quick) var(--ease-standard); }
[data-rcl-responsive-dialog][data-state="closed"] .rcl-responsive-dialog__backdrop { opacity: 0; }

.rcl-responsive-dialog__panel { position: relative; display: flex; flex-direction: column; inline-size: 100%; max-block-size: calc(100% - var(--rcl-safe-top, 0px) - var(--overlay-drawer-top-gap, 32px)); overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-block-end: 0; border-radius: var(--radius-sheet) var(--radius-sheet) 0 0; background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-modal); pointer-events: auto; transition: transform var(--dur-quick) var(--ease-standard); animation: rcl-responsive-dialog-enter-sheet var(--dur-moderate) var(--ease-enter); }
.rcl-responsive-dialog__panel[data-dragging="true"] { transition: none; will-change: transform; }
[data-rcl-responsive-dialog][data-state="closed"] .rcl-responsive-dialog__panel { transform: translateY(100%); animation: none; }
@keyframes rcl-responsive-dialog-enter-sheet { from { transform: translateY(100%); } }

.rcl-responsive-dialog__grabber { position: absolute; z-index: 1; inset-block-start: 0; inset-inline-start: 50%; translate: -50% 0; inline-size: min(60%, 12rem); min-block-size: var(--tap-target-min); display: grid; justify-items: center; align-content: start; padding: var(--space-2xs) 0 0; margin: 0; border: 0; background: transparent; color: inherit; touch-action: none; cursor: grab; }
.rcl-responsive-dialog__grabber[data-rcl-overlay-dragging="true"] { cursor: grabbing; }
.rcl-responsive-dialog__grabber > span { inline-size: var(--overlay-grabber-inline, 2.25rem); block-size: var(--overlay-grabber-block, .25rem); border-radius: var(--radius-pill); background: var(--color-border-strong, var(--color-border)); }

.rcl-responsive-dialog__panel > header, .rcl-responsive-dialog__panel > footer { flex: 0 0 auto; display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm); padding: var(--space-sm) var(--space-md); }
.rcl-responsive-dialog__panel > header { border-block-end: var(--border-hairline) solid var(--color-border); }
.rcl-responsive-dialog__panel > footer { border-block-start: var(--border-hairline) solid var(--color-border); padding-block-end: calc(var(--space-sm) + var(--rcl-safe-bottom, 0px)); }
.rcl-responsive-dialog__panel > header > *:first-child { min-inline-size: 0; }
.rcl-responsive-dialog__panel > header > *:last-child { display: flex; flex: 0 0 auto; align-items: center; gap: var(--space-3xs); }
.rcl-responsive-dialog__panel h2 { margin: 0; font: var(--text-heading); }

.rcl-responsive-dialog__subheader { flex: 0 0 auto; min-block-size: 0; border-block-end: var(--border-hairline) solid var(--color-border); }

.rcl-responsive-dialog__content { flex: 1 1 auto; min-block-size: 0; overflow: auto; overscroll-behavior: contain; padding-block-end: var(--rcl-safe-bottom, 0px); }
[data-rcl-responsive-dialog][data-content-padding="comfortable"] .rcl-responsive-dialog__content { padding: var(--space-md); padding-block-end: calc(var(--space-md) + var(--rcl-safe-bottom, 0px)); }
[data-rcl-responsive-dialog][data-has-footer] .rcl-responsive-dialog__content { padding-block-end: 0; }
[data-rcl-responsive-dialog][data-has-footer][data-content-padding="comfortable"] .rcl-responsive-dialog__content { padding-block-end: var(--space-md); }

.rcl-responsive-dialog__panel [data-icon] { flex: 0 0 auto; inline-size: var(--icon-size-md); block-size: var(--icon-size-md); }

@media (min-width: 48rem) {
  [data-rcl-responsive-dialog] { place-items: center; padding: var(--space-md); }
  .rcl-responsive-dialog__panel { max-block-size: calc(100% - (var(--space-lg) * 2)); border-block-end: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); animation-name: rcl-responsive-dialog-enter-dialog; }
  [data-rcl-responsive-dialog][data-state="closed"] .rcl-responsive-dialog__panel { transform: translateY(var(--space-2xs)); opacity: 0; }
  [data-rcl-responsive-dialog][data-size="sm"] .rcl-responsive-dialog__panel { inline-size: var(--overlay-dialog-sm); }
  [data-rcl-responsive-dialog][data-size="md"] .rcl-responsive-dialog__panel { inline-size: var(--overlay-dialog-md); }
  [data-rcl-responsive-dialog][data-size="lg"] .rcl-responsive-dialog__panel { inline-size: var(--overlay-dialog-lg); }
  .rcl-responsive-dialog__panel > header, .rcl-responsive-dialog__panel > footer { padding: var(--space-md); }
  .rcl-responsive-dialog__panel > footer { padding-block-end: var(--space-md); }
  .rcl-responsive-dialog__content { padding-block-end: 0; }
  [data-rcl-responsive-dialog][data-content-padding="comfortable"] .rcl-responsive-dialog__content { padding: var(--space-md); }
}
@keyframes rcl-responsive-dialog-enter-dialog { from { transform: translateY(var(--space-2xs)); opacity: 0; } }
`;
