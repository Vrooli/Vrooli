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

An overlay resolves the usable viewport through the `BaseStyles` host viewport contract —
`--rcl-viewport-height`, `--rcl-safe-top`/`-right`/`-bottom`/`-left`, and `--rcl-keyboard-inset` —
and never through `env(safe-area-inset-*)` or `100dvh` directly. The defaults published by
`BaseStyles` *are* the raw environment, so a host that says nothing behaves as before; a host that
manages its own scrolling, keyboard handling, or chrome assigns the six properties on the root
element and every surface follows it.

This matters most at the bottom edge. `env(safe-area-inset-bottom)` describes the layout viewport,
which an application has often already narrowed, so a surface that insets itself by that value can
end up floating above the edge it slid in from. A sheet reaches the edge; the safe area and the
keyboard are cleared as padding inside the scroll region and the footer.

## Content and bands

`contentPadding` selects the scroll region's gutter: `comfortable` (the default) or `none` for a
caller that owns its own. Scrolling stays with the overlay either way. A caller that needs a fixed
band above the scroll region — tabs, a filter row, a search field — uses the full-bleed `subheader`
slot rather than nesting a second scroller inside the content.

An overlay applies no blanket tap-target floor to its descendants. The floor belongs to the
overlay's own affordances; a dense control rung placed inside an overlay keeps the size its caller
selected, as [the sizing contract](sizing-contract.md) requires.

