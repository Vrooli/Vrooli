# Overlay contract

All interactive overlays compose `useOverlaySurface`. The shared substrate owns portal placement,
layer ordering, top-layer Escape dismissal, modal background inertness, nested scroll locking,
focus containment, focus return, presence state, and — from 1.3.0 — swipe dismissal. Presentation
components own geometry and the dismissal choices in this table.

A presentation asks for a gesture with `dismiss: { swipe: "down" }` and renders the returned
`grabberProps` on its own affordance. The substrate writes the drag offset to
`--rcl-overlay-progress` on the surface and flags `data-dragging` while a gesture is in flight; a
presentation follows the finger from its own stylesheet and suspends its transform transition on
that flag. Routing the offset through a custom property rather than React state keeps a drag from
re-rendering the overlay subtree on every pointer move. The grabber is a real button: Enter and
Space dismiss, so the gesture is never the only way out.

| Presentation | Component | Mobile | Desktop | Modal | Escape | Backdrop | Swipe | Role |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| page | `FullPageDrawer` | sheet flush with the bottom edge | inset card | yes | close | close | down | `dialog` |
| dialog | `ResponsiveDialog` | bottom sheet, bounded by the host viewport | centered token-sized card | yes | close | close | down | `dialog` |
| alert | `AlertDialog` | centered card | centered card | yes | cancel | none | none | `alertdialog` |
| rail | `SidebarShell` | drawer over scrim | persistent resizable column | mobile | mobile | mobile | edgeward on mobile | `dialog` / `complementary` |
| menu | `ContextMenu` | bottom action sheet | anchored popover | no | close | close | down on mobile | `menu` |

`ResponsiveDialog` and `ContextMenu` keep one mounted child subtree while their presentation
changes at `--breakpoint-md`. Resizing must preserve uncontrolled input values, focus, scroll
position, and the open state. `AlertDialog` never dismisses through its backdrop or a gesture; its
safe action receives initial focus.

Every interactive element has a pointer and keyboard path. Test identifiers use a unique rooted
shape: `<catalog-id>` for the surface, `<catalog-id>.backdrop`, `<catalog-id>.close`,
`<catalog-id>.grabber`, and `<catalog-id>.subheader`.

## Viewport

`useViewportEnvironment` owns generic browser measurement. One singleton external store coalesces
window and VisualViewport resize/scroll signals with `requestAnimationFrame` and exposes layout
dimensions, visible dimensions, offsets, scale, and qualified keyboard state. Ordinary resizes and
pinch zoom never become a keyboard: entry requires editable focus, scale 1, a meaningful occlusion,
and two settled samples. An established keyboard follows its animation and exits immediately when
focus or occlusion clears. Browsers without VisualViewport use the layout viewport; SSR receives a
safe zero snapshot until hydration.

`useOverlaySurface` consumes that snapshot and applies `--rcl-viewport-width`,
`--rcl-viewport-height`, `--rcl-viewport-offset-left`, `--rcl-viewport-offset-top`, and
`--rcl-keyboard-inset` on the overlay presentation root. A keyboard-aware root uses the visual
viewport rectangle directly; it does not also subtract the keyboard inset. Its panel sizes against
that root, so the geometry has one coordinate system and one keyboard adjustment. No generic
viewport code mutates `html` or scrolls the document. `BaseStyles` keeps dynamic-viewport and safe-area defaults for first paint.
Embedded and remote hosts can replace browser measurement with `ViewportEnvironmentProvider`
without changing overlay consumers.

This matters most at the bottom edge. `env(safe-area-inset-bottom)` describes the layout viewport,
which an application has often already narrowed, so a surface that insets itself by that value can
end up floating above the edge it slid in from. A sheet reaches the edge; the safe area and the
keyboard are cleared as padding inside the scroll region and the footer.

## Responsive and gesture transitions

An open responsive overlay keeps one mounted content subtree across breakpoint changes. Every
viewport transition cancels an in-flight gesture, releases pointer capture, and clears the inline
progress property, dragging attributes, origin, measured extent, and grabber reference. The same
reset occurs on commit, pointer cancellation, close, direction change, and swipe-policy change.
An idle resize must therefore leave the surface at its resting transform; a drag cannot leak into
the next presentation. A committed drag suppresses its synthetic follow-up click, while an
unmoved grabber click—including assistive activation—invokes the normal close action.

An open surface keeps its layer registration across ordinary consumer renders. The registry order
records when overlays opened; changing an inline close callback must not promote a lower parent
above a nested child. Escape, backdrop, and swipe dismissal therefore resolve the same visually
topmost surface when two drawers are open.

## Content and bands

`contentPadding` selects the scroll region's gutter: `comfortable` (the default) or `none` for a
caller that owns its own. Scrolling stays with the overlay either way. A caller that needs a fixed
band above the scroll region — tabs, a filter row, a search field — uses the full-bleed `subheader`
slot rather than nesting a second scroller inside the content.

An overlay applies no blanket tap-target floor to its descendants. The floor belongs to the
overlay's own affordances; a dense control rung placed inside an overlay keeps the size its caller
selected, as [the sizing contract](sizing-contract.md) requires.
