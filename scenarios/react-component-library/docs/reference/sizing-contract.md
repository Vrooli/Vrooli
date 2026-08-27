# Control sizing contract

Control geometry is implemented by `ControlBase/1.1.0`. The six documented rungs resolve their
minimum block size, padding, typography, and direct-child icon size together: `xs` 32px/12px,
`sm` 36px/14px, `md` 40px/16px, `lg` 44px/18px, `xl` 48px/20px, and `icon` 40px/16px.

Rungs below 44px are valid dense-control choices. Development builds mark them with
`data-control-below-tap-target="true"` and warn once per component/rung; the warning never blocks
rendering and no blanket tap-target minimum overrides the selected rung.

`ControlBase` is the sizing authority for shared controls. A control's
`size` resolves its minimum block size, horizontal padding, typography,
direct-child icon scale, and radius as one decision. Only the `icon` rung also
sets a minimum inline size; the other rungs grow inline size from their
content and padding.

The 44px tap-target guidance is not a blanket CSS floor. Dense rungs are valid
when the layout requires them. Controls below 44px expose
`data-control-below-tap-target="true"` and emit one development-mode warning
per component and rung. The warning never blocks rendering. A consumer that
needs a larger control may add ordinary layout styles; a consumer that needs a
documented dense size selects the corresponding rung.

The supported size scale is:

| Size | Minimum target | Intended use |
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
