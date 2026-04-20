# Research Notes — from research/command-center-architecture

This file records guidance from `research/command-center-architecture/conclusion.md` that the theming-engine execution must incorporate.

## Per-theme staleness badge visuals (Action 6, Finding 17)

Each of the 6 themes must define its **staleness-badge variant** — how the indicator renders when a widget is showing last-cached data because an upstream source is unavailable. The indicator must:

- **Stay inside the theme's palette** so it reads as part of the theme rather than as standard chrome.
- **Be subtle and non-disruptive** — no global banner, no takeover. Discoverable by looking closely, never jarring.
- **Define color, pulse behavior, and position** relative to the widget.

Example variants (illustrative, not prescriptive — actual designs are owed by this item):

- Ground Control: a thin monospace timestamp under the value, electric-blue dimmed to 40%.
- Bioluminescent: the widget's glow dims slightly and pulses at a slower cadence.
- Foundry: an amber ember icon in the corner, slow flicker.
- Vault: a small serif "as of HH:MM" in dim gold.
- Signal Tower: the widget's magenta wave pulse slows and fades.
- Cosmos: the widget (planet) dims slightly and orbit slows.

## Cache envelope contract (Finding 7, Finding 17)

The theming-engine item is also responsible for defining the **cache envelope contract** the UI reads to drive staleness badges:

- `staleness_ts` — ISO-8601 timestamp of when the data was originally fetched from upstream.
- `from_cache` — boolean indicating the response came from cache (upstream unreachable or TTL-hit).

This contract is consumed by every dashboard page's widgets and by the kiosk-ux item's staleness UX logic (`execute/command-center-kiosk-ux`).

## Staleness is not a gap (Finding 17)

Staleness is a **runtime outage** signal. Structural gaps (metrics that have never shipped) are a separate concept handled by `/api/v1/gaps` and the `dataSource: "gap" | "partial"` registry entries. Widgets in gap mode render the existing gap treatment; widgets with a stale cache render the last value plus the theme's staleness badge. Do not conflate these two states.

## Theme isolation approach (Finding 5)

Each theme is implemented as CSS custom properties scoped to a route wrapper (`<div data-theme="ground-control">`). Theme definitions live in JSON config files applied via a ThemeProvider. Each theme defines:

- Background treatment (gradients, textures, particle effect seed parameters for R3F).
- Color palette.
- Typography (font family, weights).
- Animation style.
- Chart colors.
- Card/panel styling.
- Glow/shadow effects.
- **Staleness badge variant** (new — see above).

Per-page R3F scenes (Finding 9) live alongside the CSS theme and are code-split via `React.lazy()` (Finding 15) so only the active theme's geometry/shaders are in memory.

## Post-processing caveat

Post-processing effects (Bloom, Vignette) are currently **disabled** in prompt-manager's R3F setup due to compatibility issues. The command-center may hit the same issues — **no theme's identity should depend on post-processing**. Design each theme to read correctly without Bloom/Vignette; treat them as optional enhancements only.
