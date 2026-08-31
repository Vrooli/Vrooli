export const sidebarShellStyles = `
[data-rcl-sidebar-shell] { min-block-size: 0; min-inline-size: 0; display: flex; flex-direction: column; background: var(--color-surface); color: var(--color-foreground); }

/* 2.5.0 — the seam between the panel and the content it sits beside follows the
   physical edge the panel is anchored to. Physical properties are paired with
   data-edge on purpose: data-edge is already a resolved side, so restating it
   logically would re-introduce the very indirection that made 2.4.0 disagree
   with itself in RTL. */
[data-rcl-sidebar-shell][data-edge="left"] { border-right: var(--border-hairline) solid var(--color-border); }
[data-rcl-sidebar-shell][data-edge="right"] { border-left: var(--border-hairline) solid var(--color-border); }

[data-rcl-sidebar-shell][data-mode="persistent"] { position: relative; z-index: auto; block-size: 100%; flex-shrink: 0; box-shadow: var(--elev-flat); }
[data-rcl-sidebar-shell][data-mode="overlay"], [data-rcl-sidebar-shell][data-mode="responsive"] { position: fixed; inset-block: 0; z-index: var(--layer-modal); block-size: 100dvh; inline-size: 100%; max-inline-size: none; padding-block: var(--rcl-safe-top, env(safe-area-inset-top)) var(--rcl-safe-bottom, env(safe-area-inset-bottom)); box-shadow: var(--elev-modal); transform: translateX(0); transition: transform var(--dur-quick) var(--ease-standard), visibility var(--dur-quick) var(--ease-standard); }

/* 2.5.0 — where the drawer rests. 2.4.0 pinned this with inset-inline-start,
   which mirrors with the locale, while the closed transform below stayed
   physical. In RTL the panel therefore moved to the right edge and its closed
   transform pushed it further into the viewport instead of out of it. */
[data-rcl-sidebar-shell][data-mode="overlay"][data-edge="left"],
[data-rcl-sidebar-shell][data-mode="responsive"][data-edge="left"] { left: 0; right: auto; }
[data-rcl-sidebar-shell][data-mode="overlay"][data-edge="right"],
[data-rcl-sidebar-shell][data-mode="responsive"][data-edge="right"] { right: 0; left: auto; }

/* The closed state travels out through the edge the panel is anchored to.
   Four attributes; the opening rule below matches that count deliberately. */
[data-rcl-sidebar-shell][data-mode="overlay"][data-open="false"][data-edge="left"],
[data-rcl-sidebar-shell][data-mode="responsive"][data-open="false"][data-edge="left"] { visibility: hidden; transform: translateX(-100%); }
[data-rcl-sidebar-shell][data-mode="overlay"][data-open="false"][data-edge="right"],
[data-rcl-sidebar-shell][data-mode="responsive"][data-open="false"][data-edge="right"] { visibility: hidden; transform: translateX(100%); }

[data-rcl-sidebar-shell][data-mode="responsive"] { display: flex; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__header { display: flex; align-items: center; justify-content: space-between; gap: var(--space-xs); min-block-size: var(--tap-target-min); border-block-end: var(--border-hairline) solid var(--color-border); padding-inline: var(--space-xs); }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__header-content { min-inline-size: 0; overflow-wrap: anywhere; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__close { min-block-size: var(--tap-target-min); min-inline-size: var(--tap-target-min); flex-shrink: 0; border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); cursor: pointer; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__close:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__icon { inline-size: var(--space-sm); block-size: var(--space-sm); }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__content { min-block-size: 0; min-inline-size: 0; flex: 1; overflow: auto; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__resize { position: absolute; inset-block: 0; inset-inline-end: calc(var(--space-3xs) * -1); z-index: var(--layer-sticky); inline-size: var(--space-xs); border: 0; background: transparent; cursor: col-resize; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__resize:hover { background: color-mix(in srgb, var(--color-primary) 25%, transparent); }
[data-rcl-sidebar-backdrop] { position: fixed; inset: 0; z-index: calc(var(--layer-modal) - 1); border: 0; background: color-mix(in srgb, var(--color-shell) 60%, transparent); cursor: default; }

/* 2.1.0 — drag-to-dismiss.
   pan-y is what lets the gesture coexist with the nav list: the browser
   keeps vertical scrolling and hands horizontal movement to the pointer
   handlers, so a drag down the list still scrolls.
   While a drag is live the transform is written every frame, so the easing
   that animates open/close must be off or it fights the finger, and the
   panel is promoted once at pointerdown rather than mid-gesture. */
[data-rcl-sidebar-shell][data-swipe="true"] { touch-action: pan-y; }
/* Every descendant, not just the shell's own content box. A consumer's own
   scrolling list is a scroll container, and a scroll container resolves
   touch-action against itself before the shell above it — so with only the
   shell restricted, a drag starting inside the list was claimed as a scroll
   and cancelled the pointer stream. That is why the gesture used to work
   only in the strip above the list, where nothing scrolls.
   A child that genuinely claims the inline axis opts out with data-rcl-pan-x;
   SwipeActions is the component that does, and the pairing is covered by that
   component's "inside a drawer" story. */
[data-rcl-sidebar-shell][data-swipe="true"] *:not([data-rcl-pan-x]):not([data-rcl-pan-x] *) { touch-action: pan-y; }
[data-rcl-sidebar-shell][data-dragging="true"] { transition: none; will-change: transform; }

/* 2.2.0 — the opening drag.
   The strip sits under the backdrop layer and only exists while the drawer is
   closed, so it never competes with the panel it opens. pan-y keeps a
   vertical flick on the page behind it scrolling the page. */
[data-rcl-sidebar-edge] { position: fixed; inset-block: 0; z-index: calc(var(--layer-modal) - 2); touch-action: pan-y; background: transparent; }
[data-rcl-sidebar-edge][data-edge="left"] { left: 0; right: auto; }
[data-rcl-sidebar-edge][data-edge="right"] { right: 0; left: auto; }

/* While an opening drag is live the panel is on screen but not yet "open", so
   it needs the visibility the closed rule takes away, and none of the easing
   that would fight the finger.
   Matched to the closed rule's specificity on purpose, and placed after it.
   The closed rule is four attributes; a three-attribute opening rule loses the
   cascade, and the panel then tracked the finger correctly and did all of it
   invisibly — the drag appeared to do nothing until release, when data-open
   flipped and the ordinary transition slid the panel in. [data-edge] is present
   to match the count, not to filter. */
[data-rcl-sidebar-shell][data-mode="overlay"][data-opening="true"][data-edge],
[data-rcl-sidebar-shell][data-mode="responsive"][data-opening="true"][data-edge] { visibility: visible; transition: none; will-change: transform; }

/* Desktop last, so its resets win every tie against the drawer rules above at
   equal specificity. A responsive shell stops being a drawer here. */
@media (min-width: 768px) {
  [data-rcl-sidebar-shell][data-mode="responsive"] { position: relative; inset: auto; left: auto; right: auto; z-index: auto; block-size: 100%; inline-size: auto; padding-block: 0; box-shadow: var(--elev-flat); transform: none; visibility: visible; }
  [data-rcl-sidebar-shell][data-mode="responsive"][data-open="false"][data-edge] { transform: none; visibility: visible; }
  [data-rcl-sidebar-shell][data-mode="responsive"] .rcl-sidebar-shell__header { display: none; }
  [data-rcl-sidebar-shell][data-mode="overlay"] { inline-size: min(22rem, 100vw); max-inline-size: 22rem; }
  [data-rcl-sidebar-backdrop][data-mode="responsive"] { display: none; }
}
`;

// Appended in 2.0.0: the shell now owns the resize affordance, so it also owns
// the rule that a drawer has no seam to drag. Below the desktop breakpoint a
// responsive shell is a full-width dialog; an overlay shell always is.
export const sidebarShellResizeStyles = `
[data-rcl-sidebar-shell][data-mode="overlay"] [data-rcl-resize-handle] { display: none; }
[data-rcl-sidebar-shell][data-mode="responsive"] [data-rcl-resize-handle] { display: none; }
@media (min-width: 768px) {
  [data-rcl-sidebar-shell][data-mode="responsive"] [data-rcl-resize-handle] { display: flex; }
}
`;
