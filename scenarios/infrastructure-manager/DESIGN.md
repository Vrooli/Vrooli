---
id: vrooli-annunciator
version: 1.0.0
name: Vrooli Annunciator Panel
description: Control-room annunciator language for instrument scenarios — lit lamps on a dark panel, where the unlit regions are the content.
colors:
  primary: "#F2A93B"
  secondary: "#4FC7D9"
  neutral: "#0E1214"
  surface: "#151B1E"
  on-surface: "#DDE5E4"
  error: "#E3543C"
  success: "#F2A93B"
  warning: "#4FC7D9"
typography:
  display-lg:
    fontFamily: Barlow Condensed
    fontSize: 44px
    fontWeight: "600"
    lineHeight: 0.95
    letterSpacing: -0.012em
  display-md:
    fontFamily: Barlow Condensed
    fontSize: 26px
    fontWeight: "600"
    lineHeight: 1.04
    letterSpacing: 0.002em
  legend-md:
    fontFamily: Barlow Condensed
    fontSize: 19px
    fontWeight: "600"
    lineHeight: 1.2
    letterSpacing: 0.13em
  body-md:
    fontFamily: IBM Plex Sans
    fontSize: 15.5px
    fontWeight: "400"
    lineHeight: 1.62
    letterSpacing: 0em
  body-sm:
    fontFamily: IBM Plex Sans
    fontSize: 13px
    fontWeight: "400"
    lineHeight: 1.5
    letterSpacing: 0em
  label-md:
    fontFamily: IBM Plex Sans
    fontSize: 13px
    fontWeight: "600"
    lineHeight: 1.25
    letterSpacing: 0.02em
  tag-sm:
    fontFamily: IBM Plex Mono
    fontSize: 12.5px
    fontWeight: "500"
    lineHeight: 1.45
    letterSpacing: 0.1em
  telemetry-md:
    fontFamily: IBM Plex Mono
    fontSize: 14px
    fontWeight: "500"
    lineHeight: 1.4
    letterSpacing: 0.02em
rounded:
  sm: 0.125rem
  md: 0.1875rem
  lg: 0.25rem
  full: 9999px
spacing:
  unit: 0.25rem
  touch: 44px
  sidebar: 20rem
  panel-gap: 1px
  hairline: 1px
components:
  button-primary:
    backgroundColor: "rgba(242,169,59,0.14)"
    textColor: "{colors.primary}"
    typography: "{typography.label-md}"
    rounded: "{rounded.md}"
    height: "{spacing.touch}"
    padding: 0 1rem
  button-primary-loading:
    backgroundColor: "rgba(242,169,59,0.08)"
    textColor: "#93A3A3"
    typography: "{typography.label-md}"
    rounded: "{rounded.md}"
    height: "{spacing.touch}"
    padding: 0 1rem
  button-disabled:
    backgroundColor: "#1B2327"
    textColor: "#637374"
    typography: "{typography.label-md}"
    rounded: "{rounded.md}"
    height: "{spacing.touch}"
    padding: 0 1rem
  input-error:
    backgroundColor: "#151B1E"
    textColor: "{colors.error}"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 0.75rem
  alert-error:
    backgroundColor: "rgba(227,84,60,0.13)"
    textColor: "{colors.error}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.md}"
    padding: 1rem
  toast-success:
    backgroundColor: "rgba(242,169,59,0.14)"
    textColor: "{colors.primary}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.md}"
    padding: 0.75rem
  empty-state:
    backgroundColor: "#151B1E"
    textColor: "#637374"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 1.5rem
  skeleton:
    backgroundColor: "#1B2327"
    rounded: "{rounded.sm}"
    height: 1rem
  inline-progress:
    backgroundColor: "rgba(79,199,217,0.13)"
    textColor: "{colors.secondary}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.sm}"
    padding: 0.25rem 0.625rem
  retry-action:
    backgroundColor: "transparent"
    textColor: "{colors.secondary}"
    typography: "{typography.label-md}"
    rounded: "{rounded.md}"
    height: "{spacing.touch}"
    padding: 0 0.75rem
  legend-plate:
    backgroundColor: "transparent"
    textColor: "{colors.on-surface}"
    typography: "{typography.legend-md}"
    rounded: "0"
    padding: 0.55rem 0
  lamp-cell:
    backgroundColor: "#202A2E"
    textColor: "#637374"
    typography: "{typography.tag-sm}"
    rounded: "{rounded.sm}"
    height: 26px
    padding: 0 0.35rem
  stat-plate:
    backgroundColor: "#151B1E"
    textColor: "{colors.on-surface}"
    typography: "{typography.display-md}"
    rounded: "0"
    padding: 1.4rem 1.5rem 1.5rem
tokens:
  color:
    background: "#0E1214"
    shell: "#0B0F10"
    surface: "#151B1E"
    surfaceMuted: "#131A1D"
    surfaceRaised: "#1B2327"
    foreground: "#DDE5E4"
    mutedForeground: "#93A3A3"
    subtleForeground: "#637374"
    border: "#263033"
    borderLit: "#3A484D"
    primary: "#F2A93B"
    primaryForeground: "#0E1214"
    accent: "#4FC7D9"
    success: "#F2A93B"
    danger: "#E3543C"
    warning: "#4FC7D9"
    info: "#4FC7D9"
    blind: "#202A2E"
    darkBackground: "#0E1214"
    darkSurface: "#151B1E"
    darkSurfaceMuted: "#131A1D"
    darkForeground: "#DDE5E4"
    darkMutedForeground: "#93A3A3"
    darkBorder: "#263033"
  signal:
    covered: "#F2A93B"
    coveredSoft: "rgba(242,169,59,0.14)"
    partial: "#4FC7D9"
    partialSoft: "rgba(79,199,217,0.13)"
    excursion: "#E3543C"
    excursionSoft: "rgba(227,84,60,0.13)"
    blind: "#202A2E"
    blindEdge: "#3A484D"
  radius:
    control: "0.1875rem"
    panel: "0.1875rem"
    sheet: "0.25rem"
    pill: "9999px"
  typography:
    fontFamily: "IBM Plex Sans, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif"
    displayFamily: "Barlow Condensed, Bahnschrift, Avenir Next Condensed, Liberation Sans Narrow, DejaVu Sans Condensed, Arial Narrow, ui-sans-serif, system-ui, sans-serif"
    monoFamily: "IBM Plex Mono, JetBrains Mono, ui-monospace, SF Mono, DejaVu Sans Mono, Consolas, Liberation Mono, Menlo, monospace"
    baseSize: "15.5px"
    lineHeight: "1.62"
  spacing:
    unit: "0.25rem"
    touchTarget: "44px"
    desktopSidebar: "20rem"
    desktopSidebarMin: "16.25rem"
    desktopSidebarMax: "30rem"
constraints:
  letterSpacing: "0"
  cardRadiusMaximum: "0.25rem"
  defaultMode: "dark"
  supportedModes: ["dark"]
  responsiveBaseline: "mobile-first"
  dominantPalette: "instrument-dark-with-amber-lamp-cyan-partial-and-rare-vermilion-excursion"
---

# Vrooli Annunciator Panel

`DESIGN.md` is the source of truth for this scenario's UI decisions. Stack-specific
adapters may translate these tokens into CSS, Tailwind, or future targets, but
adapters must not redefine the design language.

## Why this scenario has its own language

Infrastructure Manager previously adopted the `vrooli-default` Operational Console
kit whole and unmodified. Three of the four arguments for doing so were correct and
are carried forward here unchanged:

- **The users are operators and agents, not consumers.** They arrive to answer one
  question — what should I do next? — and leave.
- **Density is the product.** This scenario replaces hand-walked triage across
  several sensor sources with one ranked list. Compact information surfaces are the
  mechanism by which that saves time.
- **Status semantics carry real meaning, and more of it than usual.** Nearly every
  element on every page is a status: in band, out of band, untrusted, unavailable,
  unmeasurable, open-loop. The rule that **status is never encoded by colour alone**
  is load-bearing rather than stylistic, and it survives into this language intact.
  See the `status-not-colour-alone` and `instrument-vs-plant-distinct` claims in
  `experience/pages/`.

The fourth argument — that a generic operational console was therefore the right
*visual* language — did not follow. Density and calm are properties of information
design, not of a palette. The stock kit is a light, blue, rounded admin surface: a
reasonable default for a CRUD scenario and a poor fit for an instrument whose entire
subject is what a machine cannot see about itself.

## The thesis: blindness is the figure, not the ground

This scenario renders one idea that almost no monitoring product renders: **declared
blindness**. A region that is dark is not empty and is not "no data" — it is a
first-class, dated, aged measurement meaning *nobody is watching this, and here is
how long that has been true*.

A conventional dashboard cannot express that. It has no visual vocabulary for
absence, so absence renders as a blank card, a zero, or a green tick — which is the
precise dishonesty this instrument exists to remove.

A control-room annunciator does have that vocabulary, natively. On a real annunciator
panel the unlit lamps are the most important thing in the room. That is why this
language is drawn from one, and it is why the visual treatment is not decoration
here: it is the only part of the surface that makes the model legible to somebody who
will never read the architecture.

The reference vernacular is already the codebase's own. `infrastructure-manager`
cites ISA-18.2 and EEMUA 191 and uses instrumentation tag letters. Lit lamps on a
dark panel, engraved legend plates, precise hairlines, and monospace tag numbers are
this domain's native visual language, not a borrowed aesthetic.

## The committed dark world

`constraints.supportedModes` is `["dark"]`. This is a deliberate single-world
commitment, not an unfinished light theme.

The lit-lamp metaphor inverts under a light theme: on a light ground, an unlit lamp
is the *brightest* region on the page, which reverses the one relationship the whole
surface is built to communicate. A light variant would therefore need a completely
different encoding for blindness — texture rather than darkness — which is a second
design language, not a mode of this one.

Because the commitment is deliberate, two rules follow and are enforced:

1. **Every colour is painted explicitly.** No element inherits a ground, a text
   colour, or a border from the host shell or from browser chrome. A surface that
   renders correctly only because something behind it happened to be dark is a bug.
2. **No colour is defined only inside a media query or a `[data-theme]` block.**
   Tokens are defined once on `:root` and consumed by name.

## Type roles

Three roles, each with a full fallback stack, because **no web font is loaded**.
Vrooli scenarios run locally and must render correctly offline, and no font package
is currently approved through `scenario-dependency-analyzer`. The stacks below are
ordered so that if the operator later approves `@fontsource` packages for the first
name in each stack, the language upgrades with no other change.

| Role | Stack | Used for |
|---|---|---|
| **Display** | `Barlow Condensed` → `Bahnschrift` → `Avenir Next Condensed` → `Liberation Sans Narrow` → `DejaVu Sans Condensed` → `Arial Narrow` | Legend plates, headings, stat numerals |
| **Body** | `IBM Plex Sans` → system sans | Prose, descriptions, labels |
| **Mono** | `IBM Plex Mono` → `JetBrains Mono` → `ui-monospace` → `DejaVu Sans Mono` | All data, all tags, all cell references, every numeral in a column |

Every fallback in the display stack is a genuine condensed face present by default on
at least one of Linux, macOS, or Windows, so the condensed voice survives on a host
with no webfont. `Liberation Sans Narrow` and `DejaVu Sans Condensed` cover Linux;
`Bahnschrift` covers Windows 10+; `Avenir Next Condensed` covers macOS.

All tabular data sets `font-variant-numeric: tabular-nums`.

## The semantic signal system

Four states, and they are a *system*, not an accent palette. Colour is supplementary
in every one: each state also carries a distinct mark and a text label, per the
existing closed vocabulary in `src/theme/instrument.ts`.

| State | Token | Colour | Meaning |
|---|---|---|---|
| **Covered** | `--signal-covered` | `#F2A93B` amber | The rung is instrumented and read |
| **Partial** | `--signal-partial` | `#4FC7D9` cyan | Instrumented with a stated limit |
| **Blind** | `--signal-blind` | `#202A2E` unlit | Declared blindness, dated |
| **Excursion** | `--signal-excursion` | `#E3543C` vermilion | Out of band, or a claim contradicted by evidence |

**The excursion colour stays rare.** If more than a few vermilion elements appear on
one screen the alarm has lost its meaning — which is exactly the failure mode
EEMUA 191 exists to prevent. It is reserved for a genuine out-of-band condition or a
declaration proven false; it is never used for "missing" or "unknown".

Two further facts are distinct from all four above and must never be rendered as any
of them:

- **`unmeasurable`** — the device exists and the platform cannot read this signal
  from it, with a stated reason (for example, `smartctl` present but permission
  denied). Rendered as a lamp with a struck-through field and its reason adjacent.
  It is *not* blind, because it is declared and understood; it is not zero; it is not
  healthy.
- **`UNAVAILABLE`** — the sensor source could not be reached at read time. This is a
  fact about the *instrument*, not about the *plant*. Rendered on the instrument
  chrome rather than in the plant data, so an owner outage can never read as a
  coverage collapse.

## Structural devices encode real data

- Section headers are **engraved legend plates**: a mono tag, an uppercase condensed
  legend, a hairline rule running to the panel edge, and an optional right-aligned
  aside.
- **A plate's tag is a real reference** — an actual substrate cell id such as `SB1`,
  a rung number, or a device address. It is never a decorative counter. If a section
  has no real reference to carry, it gets no tag.
- Hairlines are `1px` at `--color-border`. Panels are separated by a 1px gap over a
  border-coloured ground, so the seams read as machined joints rather than as
  shadowed cards. Radii stay at or below `0.25rem`; this language has no soft cards.

## Motion

Motion is used once, for one reason: lamps come up in sequence on first paint, the
way an annunciator panel self-tests at power-on. It runs for well under a second, it
never loops, and it never repeats on re-render.

Everything else is still. There are no hover animations, no sliding panels, and no
ambient effects. All motion is wrapped in
`@media (prefers-reduced-motion: no-preference)` and the panel is fully legible with
motion disabled.

## Accessibility

- Every interactive element has a visible focus state: a `2px` `--signal-covered`
  outline at `3px` offset. Focus is never removed without replacement.
- Status is never colour alone. Every lamp carries a mark and an accessible label.
- The device constellation is an `<svg role="img">` with a `<title>` and a `<desc>`
  that states what the shape shows, plus an adjacent text alternative listing the
  same devices and rung states. A screen-reader user gets the finding, not the
  picture.
- Contrast: `--color-foreground` on `--color-background` is well above 4.5:1, as is
  every signal colour on both `--color-background` and `--color-surface`. The
  subtlest token, `--color-subtle-foreground`, is reserved for non-essential
  supporting text and never carries a status.

## What this language is not

It is not a war-room kiosk. `vrooli-command-display` exists for ambient TV and
fullscreen display surfaces and optimises for legibility at three metres with 84px
type. This is a desk instrument read at arm's length, and it optimises for density
and precision instead. The two share a dark-first world and nothing else.
