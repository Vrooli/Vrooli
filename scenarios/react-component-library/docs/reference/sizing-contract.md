# Control sizing contract

`ControlBase` is the sizing authority for shared controls. A control's
`size` resolves its minimum height, minimum width, horizontal padding,
typography, direct-child icon scale, and radius as one decision. The minimum
tap target is a floor: consumer layout classes may make a control larger, but
must not reduce the floor.

The supported size scale is:

| Size | Minimum target | Intended use |
| --- | --- | --- |
| `xs` | 32px | dense chrome and floating toolbars |
| `sm` | 36px | compact toolbar actions |
| `md` | 40px | ordinary controls |
| `lg` | 44px | touch-oriented controls |
| `xl` | 48px | prominent touch controls |
| `icon` | 40px | icon-only controls using the medium icon floor |

`density` is independent of size. `comfortable` uses a 0.5rem content gap;
`compact` uses a 0.375rem gap. Density changes internal spacing only; it never
lowers the minimum target.

Toolbar guidance:

- Floating and other dense chrome use `size="xs"` and may raise the floor with
  an explicit `h-8 w-8` layout class.
- Mobile touch toolbars use `size="lg"` for the primary microphone control.
- Compact mobile rows use `size="sm"` so the mic aligns with the surrounding
  key controls.

Stories for `ControlBase`, `Button`, `IconButton`, and `VoiceInputButton` cover
every size and both density values. Each story exposes `data-control-size` and
`data-control-density` so component tests and preview tooling can verify the
resolved contract without relying on computed pixels from a particular browser
font configuration.
