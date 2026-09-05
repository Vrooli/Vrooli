# System Monitor design adoption contract

Status: Phase 15 baseline and design decision, 2026-08-24.

## Decision

System Monitor keeps its `system-monitor-instrument` identity: graphite/phosphor
dark mode, calibrated paper-light mode, telemetry-specific chart colors, compact
tabular readouts, and explicit focus/density rules. The product adopts the RCL
catalog as a structural and tokenized substrate through a hybrid strategy, not
as a wholesale visual replacement.

The measured RCL set is the reference boundary:

- Card 1.2.0: approved for direct use after mapping its semantic tokens to the
  System Monitor token contract.
- Dialog 1.2.0: adapted; preserve instrument surfaces, focus, and density.
- SidebarShell 1.3.0: structural-only; navigation remains System Monitor's
  own responsive instrument header.
- EvidenceCarousel 1.0.9: approved for evidence-oriented screens, not a
  general page layout primitive.

Generic chart adoption is deferred until range selection, tooltip behavior,
keyboard access, empty/error states, dark mode, and dense telemetry readability
are proven in the same desktop/mobile matrix.

## Token contract

All new page work uses the existing `tokens.css` roles rather than literal
colors or arbitrary spacing. Required semantic roles are background,
surface, raised surface, border, text, muted text, primary trace, info,
success, warning, error, selected, disabled, focus ring, chart grid, chart
line, chart fill, and chart endpoint.

The structural scale is `xs/sm/md/lg/xl/xxl` spacing, `sm/md/lg` radius,
`sm/md/lg/header` elevation, the display/readout/body font roles, and tabular
numeric figures for measurements. Light mode re-picks signal colors for paper
contrast; it is not a mechanical inversion of dark mode.

The responsive contract is mobile 390px, tablet 768–834px, desktop 1440px,
and wide desktop 2560px. At mobile width, controls may stack, timestamps may
move to a second row, and notices must remain inside the viewport without
covering primary content. Native controls remain keyboard reachable with a
visible focus ring and a minimum 44px hit area.

Every interactive surface must define hover, pressed, selected, disabled,
loading, empty, and error states. Error and unavailable states are content,
not decoration: the capture baseline's API-unavailable toast and 502 console
entries remain findings until the runtime path is repaired or explicitly
classified as an external environment limitation.

## Baseline findings to carry into implementation

The durable capture set is in `docs/evidence/phase15-design-baseline-2026-08-24/`
with its machine-readable manifest beside it. It records an API-unavailable
toast on all routes, log geometry collisions, mobile toast clipping, and a
capacity text-wrap defect. These are not design decisions to conceal; they are
acceptance criteria for the later route repairs and screenshot re-capture.

## Visual acceptance checklist

- Light and dark captures are explicitly reproducible with `?theme=`.
- Desktop and mobile captures contain no horizontal clipping, overlapping
  labels, or notices covering primary content.
- API errors expose a concise recovery action and do not masquerade as healthy
  data.
- Charts retain a distinguishable trace/grid/fill hierarchy in both themes.
- Focus, selected, disabled, loading, empty, and error states are visible and
  keyboard reachable.
- BAS screenshot, accessibility, console, and bounded performance evidence
  remain available through the same-origin embedded proxy.
