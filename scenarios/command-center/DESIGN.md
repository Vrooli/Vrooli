---
id: vrooli-command-display
version: 0.3.0
name: Vrooli Command Display
description: Fullscreen war-room, kiosk, TV, and ambient command-center display language.
colors:
  primary: "#38bdf8"
  secondary: "#22d3ee"
  neutral: "#020617"
  surface: "#0f172a"
  on-surface: "#f8fafc"
  error: "#f87171"
  success: "#22c55e"
  warning: "#f59e0b"
typography:
  display-lg:
    fontFamily: Inter
    fontSize: 84px
    fontWeight: "700"
    lineHeight: 0.9
    letterSpacing: 0em
  body-md:
    fontFamily: Inter
    fontSize: 18px
    fontWeight: "400"
    lineHeight: 1.35
    letterSpacing: 0em
  label-md:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: "600"
    lineHeight: 1.2
    letterSpacing: 0em
  telemetry-md:
    fontFamily: JetBrains Mono
    fontSize: 18px
    fontWeight: "500"
    lineHeight: 1.25
    letterSpacing: 0em
rounded:
  md: 1rem
  lg: 1.25rem
  full: 9999px
spacing:
  viewport: 2rem
  remote-target: 64px
  panel-gap: 1.25rem
  cycle-duration: 60s
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "#020617"
    typography: "{typography.label-md}"
    rounded: "{rounded.full}"
    height: "{spacing.remote-target}"
    padding: 0 1.25rem
  button-primary-loading:
    backgroundColor: "{colors.primary}"
    textColor: "#020617"
    typography: "{typography.label-md}"
    rounded: "{rounded.full}"
    height: "{spacing.remote-target}"
    padding: 0 1.25rem
  button-disabled:
    backgroundColor: "#334155"
    textColor: "#94a3b8"
    typography: "{typography.label-md}"
    rounded: "{rounded.full}"
    height: "{spacing.remote-target}"
    padding: 0 1.25rem
  input-error:
    backgroundColor: "#450a0a"
    textColor: "{colors.error}"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 1rem
  alert-error:
    backgroundColor: "#450a0a"
    textColor: "{colors.error}"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 1.25rem
  toast-success:
    backgroundColor: "#052e16"
    textColor: "{colors.success}"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 1rem
  empty-state:
    backgroundColor: "#111827"
    textColor: "#94a3b8"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 1.5rem
  skeleton:
    backgroundColor: "#1e293b"
    rounded: "{rounded.md}"
    height: 1rem
  inline-progress:
    backgroundColor: "#082f49"
    textColor: "{colors.primary}"
    typography: "{typography.telemetry-md}"
    rounded: "{rounded.full}"
    padding: 0.375rem 0.875rem
  retry-action:
    backgroundColor: "transparent"
    textColor: "{colors.primary}"
    typography: "{typography.label-md}"
    rounded: "{rounded.full}"
    height: "{spacing.remote-target}"
    padding: 0 1rem
tokens:
  color:
    background: "#020617"
    backgroundDeep: "#000000"
    surface: "#0f172a"
    surfaceMuted: "#1e293b"
    foreground: "#f8fafc"
    mutedForeground: "#94a3b8"
    border: "#334155"
    primary: "#38bdf8"
    primaryForeground: "#020617"
    accent: "#22d3ee"
    success: "#22c55e"
    warning: "#f59e0b"
    danger: "#f87171"
    gap: "#a78bfa"
    stale: "#fbbf24"
  typography:
    displayFamily: "Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif"
    numericFamily: "JetBrains Mono, Fira Code, SF Mono, Consolas, Liberation Mono, Menlo, monospace"
    baseSize: "18px"
    lineHeight: "1.35"
  motion:
    pageTransition: "fade-through-black"
    cycleDefault: "60s"
    reducedMotionFallback: "crossfade-only"
  radius:
    panel: "1rem"
    tile: "1.25rem"
    pill: "9999px"
  spacing:
    viewportPadding: "clamp(1rem, 2.2vw, 3rem)"
    remoteTarget: "64px"
constraints:
  defaultMode: "dark"
  supportedModes: ["dark"]
  primaryMedium: "fullscreen-display"
  interactionDensity: "low"
  visualImpact: "high"
  dataIntegrity: "must-not-distort"
---

# Vrooli Command Display Design

`DESIGN.md` is the source of truth for fullscreen command-center, kiosk, TV, and ambient display scenarios. Stack-specific adapters may translate these tokens into CSS, Tailwind, Three.js scene palettes, chart themes, or future native-display targets, but adapters must not redefine the design language.

## Intent

Vrooli Command Display is for screens that are meant to be seen from across a room and left running: war-room dashboards, office TVs, executive readouts, launch monitors, showcase demos, and command-center displays.

This design is not a normal CRUD application language. It should feel cinematic, technical, expensive, and alive while still being honest about the data. It should make Vrooli's work, capability growth, revenue motion, system health, and gaps feel visible in the physical world.

The emotional target is "mission control for a self-improving intelligence system." The screen should be beautiful enough to make people stop and look, but disciplined enough that operators can still understand the state of the system in seconds.

## Display Model

Design for a full-screen viewport first. The primary screen should not show ordinary app chrome, dense forms, visible settings panels, or desktop navigation. Controls are hidden by default and appear only on interaction, remote input, touch, or pointer movement.

The display may be interactive, but it should remain useful when nobody is touching it. Auto-cycle between dashboard pages is a first-class behavior. Each page should be complete enough to stand alone during a cycle interval.

Use one strong visual idea per page. Command displays may use large numbers, spatial metaphors, 3D scenes, particles, animated paths, ambient glow, charts, and full-viewport compositions. Do not place a normal app dashboard inside a decorative background.

## Layout

Prefer immersive full-bleed layouts with restrained overlay panels. Important information should fit the viewport without scrolling on TV-sized displays. If a page needs more than one viewport of content, split it into multiple cycle pages instead of creating a scroll-heavy dashboard.

Use hierarchy aggressively:

- One primary readout or scene owns the page.
- Secondary metrics orbit or support the primary readout.
- Tertiary metadata is quiet and available, never visually dominant.
- Alerts, gaps, stale data, and critical failures receive salience only when attention is needed.

Large displays can carry fewer, better-composed elements more effectively than crowded grids. Use repetition for comparable metrics, but avoid filling every pixel. Empty space and dark space are part of the design.

## Theme System

This design language supports independent dashboard themes. A theme is not just a palette variant; it can define atmosphere, typography, animation style, chart styling, panel treatment, background treatment, and 3D scene materials.

For the Command Center scenario, the intended theme family is:

- **Ground Control:** space/NASA mission control, deep black, electric blue, subtle star field, clean monospace KPIs.
- **Bioluminescent:** deep ocean, glowing greens/teals/cyans, organic nodes, neural or mycelial pathways.
- **Foundry:** charcoal, amber, molten gold, forge instrumentation, heat, throughput, rising sparks.
- **Vault:** dark green, gold, ledger paper texture, trust, finance, revenue, premium restraint.
- **Signal Tower:** black/purple/magenta, radiating waves, broadcast beams, conversion and reach.
- **Cosmos:** black/nebula, orbital systems, slow movement, panoramic system health.

Each route may have a different theme, but the application must preserve a shared contract for source freshness, gap states, focus, remote controls, and data truthfulness.

## Color

Dark mode is the default and expected mode. Use true black or near-black backgrounds for TV and OLED comfort. Use saturated accents sparingly and reserve the brightest colors for active signals, source freshness, live motion, or critical attention.

Command displays may use glow, bloom-like shadows, transparent panels, and atmospheric gradients, but decoration must support the page metaphor. If every element glows, nothing is important. Keep the base display calm enough that alerts, gaps, live state, and page identity can stand out.

Status color roles:

- Green: live, healthy, completed, improving.
- Amber: stale, delayed, pending, needs attention soon.
- Red: broken, failed, blocked, critical.
- Purple/violet: gap, missing pipeline, future capability.
- Blue/cyan: primary system signal, active route, telemetry, technical emphasis.

### Provenance is material-primary (0.3.0)

Amended 2026-09-01. For **data provenance** specifically — whether a displayed figure is measured, cached, illustrative, or absent — the load-bearing signal is **material**, not color. Color reinforces; it never carries the state alone.

| Provenance | Material | Reinforcing tone |
| --- | --- | --- |
| Measured and current | Solid fill, full contrast | The surface's own accent |
| Measured, source not answering | Same digits at reduced contrast, with an age | Amber |
| Illustrative — substrate exists, pipeline unbuilt | Hollow outline, stroked not filled | Violet where the palette allows |
| Absent — no substrate anywhere | Dotted outline | Violet where the palette allows |

Three reasons this axis is material rather than color. Hue separation is the first thing to fail at viewing distance, on a projector, and at low panel brightness. Color is already spent carrying surface identity, and a display language with independent themes cannot ask one channel to carry both. And status carried by hue alone fails accessibility, which matters more here than usual because these surfaces are almost entirely status.

**The test: a greyscale render must remain unambiguous.** If provenance states are indistinguishable without color, the surface is wrong regardless of how well it reads in color.

This amendment changes which signal is load-bearing. It does not change the palette: the `gap` token stays violet, and the status color roles above continue to govern *severity* — where red still means broken and never means "not built yet."

Layout does not change with provenance. An illustrative figure occupies exactly the space its measured counterpart will occupy, so a metric going live is a change of weight and never a change of layout.

## Typography

Use large, legible typography sized for distance. Primary KPI numbers should be readable across a room. Labels should be short and high-contrast. Avoid long paragraphs except in hidden detail panels.

Use a monospace or tabular-numeric font for metrics, counters, timestamps, scenario IDs, and telemetry. Use the display sans stack for labels, headings, and page titles. Numeric rhythm matters more than editorial personality.

Do not scale all typography linearly with viewport width. Use deliberate display, KPI, label, and metadata sizes with clamp constraints so text remains stable across TV, desktop, and tablet screens.

## Data Visualization

The display is allowed to be beautiful, but it must not lie. Do not distort quantitative values for visual drama. Chart choices must match the data. Use trends, deltas, sparklines, targets, freshness labels, and context so a number is not shown alone without meaning.

Use animation to reveal flow, change, direction, or state. Do not animate values in a way that implies movement where none exists. If source data is stale, show stale data elegantly rather than pretending it is live.

Gaps are a feature, not an embarrassment. Render gaps as future capability signals: quiet shimmer, violet glow, placeholder constellations, dotted paths, or labeled inactive nodes. Never show ugly "N/A" blocks as the final display language, and never show an empty slot where a figure belongs — render the figure in the provenance material that says what it is (see Color § Provenance is material-primary).

## Feedback & State

Command displays must make source status visible without turning an unattended screen into an error dialog. Loading, live, stale, partial, gap, degraded, offline, validation-error, request-error, failed, retrying, and recovered states should be readable at a distance and should preserve the page composition.

Never show blocking modal errors on the idle display. Use inline degraded states, source badges, freshness labels, dimmed sections, fallback visualizations, and concise recovery notes. A failed source should say whether the display is showing cached data, partial data, a known gap, or no usable data. Critical failures may interrupt the visual hierarchy, but they should still avoid stack traces, local paths, tokens, secrets, and noisy logs.

Interactive controls that trigger refresh, fullscreen, page changes, settings changes, or source reconnects need immediate visible acknowledgement. Remote and kiosk users should be able to see whether the command is pending, succeeded, failed, or unavailable.

## Request Lifecycle

For every polling request, stream, websocket, local health probe, dashboard refresh, fullscreen request, wake-lock request, or source reconnect, design the lifecycle deliberately: idle, connecting, live, stale, partial, failed, retrying, recovered, and disabled/unavailable. Freshness and retry timing are part of the visual language, especially when the display informs operational decisions.

If exact progress is unavailable, use a stable indeterminate state that does not resize the page. If an auto-refresh fails, keep the previous trustworthy value visible only with a stale label and timestamp. If the value cannot be trusted, show an explicit gap or unavailable state instead of a blank tile or fake live value.

## Motion and 3D

Motion is part of this language. Use slow, confident motion: star fields, orbiting objects, radiating waves, spark trails, flowing paths, subtle parallax, scene rotation, fade-through-black page changes, and breathing glow.

Motion rules:

- Animate transform and opacity where possible.
- Keep page transitions smooth and theme-aware.
- Pause, reduce, or simplify motion under `prefers-reduced-motion`.
- Avoid flicker, strobe, aggressive pulsing, or sudden full-screen flashes.
- Avoid motion that makes numbers hard to read.

3D and canvas scenes are appropriate when they carry the primary identity or explain system state. They must be full-bleed or integrated into the page composition, not placed inside a decorative preview card. Every 3D/canvas display needs a nonblank fallback and graceful degradation for lower-end browsers.

## Kiosk and Remote UX

Command displays must work when controlled by a TV remote, gamepad, keyboard, mouse, or touch. Targets for visible controls should be at least 64px when the UI is in TV/kiosk mode.

Core behaviors:

- Fullscreen entry and exit are explicit and recoverable.
- Wake lock is requested when appropriate and failures degrade silently.
- Auto-cycle timing is configurable.
- User interaction pauses auto-cycle and resumes after inactivity.
- Hidden controls appear on pointer movement, touch, keyboard, or D-pad input.
- Settings are available but never dominate the idle display.
- URL parameters may select dashboard, fullscreen preference, and cycle timing.

Never show blocking modal errors on an unattended TV. Use inline degraded states, stale indicators, and source badges instead.

## Accessibility and Safety

This design is visually expressive, but it still needs accessible contrast, visible focus, keyboard/remote navigation, and reduced-motion support. Avoid tiny labels, low-contrast decorative text, hover-only controls, and color-only status.

Because this display may be visible publicly, avoid exposing secrets, private user data, raw tokens, stack traces, full local paths, or detailed financial/user records unless the scenario explicitly authorizes that surface.

## Do's and Don'ts

### Do

- Make the first viewport visually striking and self-explanatory.
- Use dark-first fullscreen layouts with minimal idle chrome.
- Give each dashboard page one distinct visual concept.
- Use motion, 3D, particles, and atmospheric effects when they support meaning.
- Show live, stale, partial, and gap states beautifully and honestly.
- Verify canvas and 3D scenes render nonblank and stay performant.
- Design for TV, remote, and always-on unattended viewing.
- Show loading, stale, degraded, failed, retrying, and recovered source states without blocking the idle display.
- Pair every remote/kiosk command with visible pending, success, failure, or unavailable feedback.

### Don't

- Use the normal operational-console app shell for command displays.
- Build dense CRUD tables as the primary page experience.
- Hide data problems by making gaps look like live values.
- Let decorative effects overwhelm hierarchy or readability.
- Require pointer hover for essential controls.
- Use bright white backgrounds for the idle display.
- Create six pages that are only color variants of the same layout.
- Hide data-source failures behind blank tiles, frozen numbers, or silent console errors.
- Show stack traces, raw local paths, secrets, or blocking modal errors on public or unattended displays.
