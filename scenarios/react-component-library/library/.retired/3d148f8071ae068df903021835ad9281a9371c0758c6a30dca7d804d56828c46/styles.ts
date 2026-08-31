export const swipeActionsStyles = `
/* Clipped with overflow:clip rather than overflow:hidden, and the pair is
   deliberate. Hidden establishes a block formatting context, which contains a
   child's bottom margin instead of letting it collapse out; a consumer whose
   row carries its own margin then makes this box taller than the face on top
   of it, and the action track -- which spans the full box -- shows through the
   difference as a stripe under every row. Clip does not establish that context,
   so the margin collapses away as it normally would and the track can never
   exceed the face it sits behind. Hidden stays first as the fallback for
   engines that do not know clip. */
[data-rcl-swipe-actions] { position: relative; overflow: hidden; overflow: clip; isolation: isolate; }

/* The row is the one descendant of a swipe-enabled SidebarShell that must keep
   horizontal pointer movement. The shell locks every child to pan-y so a drag
   down its nav list still scrolls; data-rcl-pan-x is the opt-out it publishes
   for children that genuinely claim the inline axis. */
[data-rcl-swipe-actions][data-swipe="true"] { touch-action: pan-y; }
/* The action buttons too, not just the row box. The drag can begin anywhere on
   the row -- once the actions are revealed they occupy most of it, and a
   dismiss that only works on the shrinking sliver of face left over is a
   dismiss the user cannot find. A button left at the default touch-action lets
   the browser claim horizontal panning and cancel the gesture mid-drag. */
[data-rcl-swipe-actions][data-swipe="true"] * { touch-action: pan-y; }

[data-rcl-swipe-actions-track] {
  position: absolute; inset-block: 0; display: flex; align-items: stretch;
  z-index: 0;
}
[data-rcl-swipe-actions-track][data-side="left"] { inset-inline-start: 0; flex-direction: row-reverse; }
[data-rcl-swipe-actions-track][data-side="right"] { inset-inline-end: 0; }

[data-rcl-swipe-action] {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: var(--space-3xs);
  border: 0; cursor: pointer;
  padding-inline: var(--space-2xs);
  font: inherit; font-size: var(--text-xs); font-weight: var(--weight-medium);
  background: var(--color-surface-muted); color: var(--color-foreground);
}
[data-rcl-swipe-action][data-tone="primary"] { background: var(--color-primary); color: var(--color-primary-foreground); }
[data-rcl-swipe-action][data-tone="destructive"] { background: var(--color-danger); color: var(--color-danger-foreground); }
[data-rcl-swipe-action]:focus-visible { outline: var(--focus-ring-width) solid var(--color-focus-ring); outline-offset: calc(var(--focus-ring-width) * -1); }
[data-rcl-swipe-action][data-armed="true"] { filter: brightness(1.12); }

/* Closed actions are inert rather than merely covered. A button hidden behind
   the face is still in the tab order and still announced, which would offer a
   screen-reader user a control they cannot see and cannot meaningfully undo. */
[data-rcl-swipe-actions][data-open="false"] [data-rcl-swipe-action] { pointer-events: none; }

[data-rcl-swipe-actions-face] {
  position: relative; z-index: 1;
  background: var(--color-surface);
  transform: translateX(0);
  will-change: auto;
}
[data-rcl-swipe-actions-face][data-settling="true"] {
  transition: transform var(--dur-quick) var(--ease-standard);
}
[data-rcl-swipe-actions-face][data-dragging="true"] { transition: none; will-change: transform; }

/* The global reduced-motion reset collapses transition-duration, but the
   library ships no stylesheet of its own, so a consuming app that never added
   that reset would still animate. Declaring it here keeps the settle honest
   without depending on an app-level contract the package cannot enforce. */
@media (prefers-reduced-motion: reduce) {
  [data-rcl-swipe-actions-face][data-settling="true"] { transition-duration: 0.01ms; }
}
`;
