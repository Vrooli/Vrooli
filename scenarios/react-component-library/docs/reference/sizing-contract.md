# Control sizing contract

Behavioral claims are current only within the specific checks in the register. Other guidance and unverified descriptions below are **design intent**, not claims of current implementation. See the [behavior claim register](../internal/TESTING.md#behavior-claim-register).

Control geometry is implemented by `ControlBase/1.2.0`. The six documented rungs resolve their
minimum block size, padding, typography, and direct-child icon size together: `xs` 32px/12px,
`sm` 36px/14px, `md` 40px/16px, `lg` 44px/18px, `xl` 48px/20px, and `icon` 40px/16px.

Rungs below 44px are valid dense-control choices on a precise pointer. Each marks itself with
`data-control-below-tap-target="true"`, and on a coarse pointer (`@media (pointer: coarse)`) the
control raises its real box to `--tap-target-min` (44px) on both axes. A mouse keeps the rung's own
size. See [Touch floor](#touch-floor).

`ControlBase` is the sizing authority for shared controls. A control's
`size` resolves its minimum block size, horizontal padding, typography,
direct-child icon scale, and radius as one decision. On a precise pointer only
the `icon` rung also sets a minimum inline size; the other rungs grow inline
size from their content and padding.

## Touch floor

`ControlBase` owns the 44px tap-target floor, so no scenario writes `min-h-11` or its own
`[data-rcl-control]` override. Only the `@media (pointer: coarse)` query applies it; desktop density
is untouched, and a hybrid laptop whose primary pointer is a mouse keeps its dense rungs.

- The floor raises only marked rungs (`xs`, `sm`, `md`, `icon`, `default`). `lg` is already 44px and
  `xl` stays 48px; neither carries the marker.
- It raises the real box (`min-block-size` and `min-inline-size`), not a pseudo-element hit slop.
  Slop on a dense strip makes neighbouring hit areas overlap, so a tap lands on whichever sibling is
  on top; it needs `position: relative`, which fights consumers that position a control themselves;
  and a layout or accessibility audit measures the box, so slop would pass a control a thumb still
  misses. A growing box makes the row reflow around it, which is the honest answer on a phone.
- `denseTapTarget` opts one control out (`data-rcl-tap-target="dense"`). Use it only for a control
  packed into a deliberately dense strip where the caller owns hit testing. The default is
  `data-rcl-tap-target="comfortable"`. `IconButton` has the same prop and attribute, so one idea
  covers both controls.
- A consumer `min-*` utility class mounts after the library sheet and can still override the floor
  at equal specificity; prefer `denseTapTarget`, which says why.

A dense rung is still a valid choice when a layout needs one. The marker never blocks rendering.
Choose the rung for a precise pointer; the floor handles touch.

## Size scale

The supported size scale is:

| Size | Minimum target (precise pointer) | Intended use |
| --- | --- | --- |
| `xs` | 32px | dense chrome and floating toolbars |
| `sm` | 36px | compact toolbar actions |
| `md` | 40px | ordinary controls |
| `lg` | 44px | touch-oriented controls |
| `xl` | 48px | prominent touch controls |
| `icon` | 40px | icon-only controls |

`density` is independent of size. `comfortable` uses a 0.5rem content gap;
`compact` uses a 0.375rem gap. Density changes internal spacing only; it never
lowers the minimum target.

Toolbar guidance:

- Floating and other dense chrome use `size="xs"` and may provide an explicit
  `h-8 w-8` layout class when its slot needs a fixed footprint.
- Mobile touch toolbars use `size="lg"` for the primary microphone control.
- Compact mobile rows use `size="sm"` so the mic aligns with the surrounding
  key controls.

Stories for `ControlBase`, `Button`, `IconButton`, and `VoiceInputButton` cover
every size and both density values. Each story exposes `data-control-size` and
`data-control-density` so component tests and preview tooling can verify the
resolved contract without relying on computed pixels from a particular browser
font configuration.
