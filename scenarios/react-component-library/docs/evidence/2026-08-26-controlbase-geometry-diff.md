# ControlBase geometry evidence

Measured 2026-08-26 against `docs/reference/sizing-contract.md`.

| Size | Documented minimum | Implemented source | Status |
| --- | ---: | --- | --- |
| `xs` | 32px | `var(--control-size-xs)` | pass |
| `sm` | 36px | `var(--control-size-sm)` | pass |
| `md` | 40px | `var(--control-size-md)` | pass |
| `lg` | 44px | `var(--control-size-lg)` | pass |
| `xl` | 48px | `var(--control-size-xl)` | pass |
| `icon` | 40px | `var(--control-size-icon)` | pass |

The blanket `[data-rcl-control]` tap-target minimum is absent from the 1.1.0
ControlBase foundation. Below-floor sizes carry `data-control-below-tap-target`
and the development warning; the warning does not block rendering.

Evidence: `ui/src/components/control-base-geometry.test.ts` passed 2/2 and
`library/components/ControlBase/versions/1.1.0/ControlBase.tsx` contains the
six-rung mapping.
