# UI Architecture

**Status:** verified implementation architecture. The governing design language is `DESIGN.md` (`vrooli-command-display`); integration condition is projected into the existing source strips and Focus surface without a new settings surface.

## The board is not an app shell

`DESIGN.md` lists "use the normal operational-console app shell for command displays" under **Don't**, and lists "create six pages that are only color variants of the same layout" under **Don't** as well. The 2026-04 build did both. The target is the opposite of both: full-bleed, zero idle chrome, one composed visual idea per room.

This also disqualifies the component library's `CommandCenterShell` despite its name — it is a sidebar-nav-plus-metric-cards operational console. See [Component library](#component-library) below.

## Layer model

Every room is the same four layers, in this order. The order is load-bearing.

```
4  Control layer      hidden until input; 64px targets; fades after 4s idle (CC-P1-007)
3  Figure layer       hero readout, supporting readings, provenance ink
                      ── composites ABOVE postprocessing ──
2  Scene layer        ambient composition, capability-tiered, quiet zones respected
1  Ground             theme background, painted explicitly
```

**The figure layer composites above any scene postprocessing pass** (`CC-P1-006`). The sample and absent inks are hairline strokes; a bloom pass smears them into unreadable haze and takes the honesty system with it. The scene gets bloom; the numbers never do.

## Quiet zones and the contrast floor

Each room declares regions the scene may not render into, plus a scrim under every figure region. This is structural, not cosmetic: in a static mockup the designer controls every pixel behind a numeral, but in a live scene a bright element drifts behind a digit and the contrast ratio changes frame to frame.

The acceptance measure is the **worst case across sampled frames of a running scene**, not the average and not a single still (`CC-P1-005`).

## Capability ladder

The board runs on a phone, a laptop, a gamepad-controlled TV and a wall panel, reached through `tunnel-manager`. One build, tiered at runtime (`CC-P1-012`).

| Tier | Scene | Selected when |
|---|---|---|
| Full | Composition with postprocessing | Probe reports capable GPU and sufficient budget |
| Reduced | Composition without postprocessing, lower element counts | Probe reports WebGL but limited budget |
| Still | Composed static frame | No WebGL, reduced-motion, or probe failure |

Three rules:

- Tier selection **never blocks first paint**. The still frame renders immediately and upgrades.
- **The figure layer is identical at every tier.** The reading is never degraded to protect the decoration.
- A mounted scene that draws nothing is a **failure, not a pass**. Every scene surface performs a first-frame render check and falls back to the still (`CC-P1-003`). This closes the 2026-04 defect where five of six scenes mounted correctly and drew nothing, and where the test suite asserted only that a `<canvas>` existed.

## Orientation

Each room ships two designed compositions, not one scaled layout (`CC-P1-004`):

- **Landscape** — figure in the void, scene owning the opposite side, supporting readings in a row.
- **Portrait** — figure at the top, scene reduced to a band, supporting readings stacked.

Room identity survives at 390px. No surface scrolls horizontally at any width. The portrait view is the one people check from their desk, so it is designed, not tolerated.

## Input and intent

Four input classes resolve to one intent vocabulary **before anything reacts** (`CC-P1-009`), built on the shared `GamepadAction` vocabulary from `@vrooli/iframe-bridge/spatial` rather than a scenario-local set.

| Intent | Gamepad | Keyboard | Touch | Pointer |
|---|---|---|---|---|
| `nextRoom` / `prevRoom` | `page-next` / `page-prev`, D-pad ◀ ▶ | ← → , 1–n | Horizontal swipe, rubber-banded | Control-bar arrows |
| `pauseCycle` | `select` | Space | Long-press 400ms | Movement pauses 20s |
| `revealControls` | `menu` | Any key | Tap | Movement |
| `toggleFullscreen` | `menu` (long) | F | Control bar | Control bar |
| `inspectReading` | `select` on a focused figure | Enter | Tap a figure | Click a figure |
| `showHelp` | `back` | ? | Control bar | Control bar |

**Every command acknowledges** (`CC-P1-010`). Fullscreen and wake-lock fail silently on some TV browsers; that surfaces as a stated line in the control bar, never a console warning and never a blocking dialog — `DESIGN.md` forbids blocking modal errors on an unattended display.

## Cycle

Auto-cycle is a first-class behaviour, not a setting. Interaction pauses it; twenty seconds of inactivity resumes it. The cycle rail at the top edge visibly stops and restarts, so a paused board never reads as a frozen one (`CC-P1-008`).

URL parameters seed the whole state so a kiosk boots configured with no interaction: `?room=forge&cycle=45&samples=mark&fullscreen=1`.

Both topologies are supported from the first commit: one display cycling, and several displays each pinned to a room. That means the room is a pure URL parameter, ambient motion seeds per display so adjacent screens never run in sync, and nothing anywhere assumes exactly one room is live.

## Motion

| Register | Rate | Purpose |
|---|---|---|
| Ambient | 0.02–0.2 Hz | Atmosphere. If a viewer perceives it as motion rather than atmosphere, it is too fast. |
| Freshness | Per source TTL | The board showing its own pulse. Replaces a `STALE` badge. |
| Value change | 380ms, changed digits only | Legibility. A whole-number crossfade reads as a flicker at distance. |
| Room change | 900ms fade-through-black | Lets six colour worlds sit next to each other without a hue slam. |

Two deliberate absences: **no pulsing alerts** — a threshold crossing gets one 1.2s bloom and holds, because a loop trains people to stop seeing it; and **no entrance animation on cycle** beyond the fade.

Under `prefers-reduced-motion` (`CC-P1-013`): scenes hold a composed still rather than blanking, the freshness indicator steps in quarters, digits swap without rolling, room changes crossfade.

## Theming

Six themes as **React Component Library semantic token overrides** — `--color-*`, `--space-*`, `--text-*` — so library components inherit the active room's palette (`CC-P1-014`).

The scenario-private `--cc-*` vocabulary is retired. It and the library's token names do not intersect, so every library component dropped into a room today renders in library defaults and ignores the theme entirely. Rewriting the themes onto library tokens is a prerequisite for using any library component, not a follow-up.

## Component library

Assets to adopt, and the work each needs. Sizes observed 2026-09-01.

**Use as-is:** `Presence`, `MotionPrimitive`, `useSwipeGesture`, `ShortcutRegistry`, `GlobalCommandSystem`, `GestureDirection`, `ChromeTheme` (per-room OS status-bar colour), `RelativeTime`, `useReducedMotion`, `useMediaQuery`, `StyleSheet`, `ClassMerge`, `Tokens`.

**Extend before use:**

| Asset | Needed |
|---|---|
| `Stat` | A `provenance` prop and a `scale` register above card size; guaranteed `tabular-nums` so rolling digits do not shift layout. |
| `Chart` | Per-point provenance so a sample series draws dashed; a height not pinned at 240px. |
| `MotionPrimitive` | A `fade-through-black` variant — the contract's mandated page transition, which none of the existing variants expresses — and a stagger API. |
| `Tokens` | Provenance tokens, a glow/bloom token, and display sizes above `--text-display` for wall distances. |

**Known defects to fix or route:** `CartesianCharts` ignores its `kind` prop entirely and renders a line chart for every value; `NetworkGraph` sets a CSS variable as a canvas 2D `strokeStyle` (which cannot resolve) and renders empty node buttons; `Meter` is a vacuous stub superseded by `BoundedMeter`; `CommandCenterShell` is an operational console and should be renamed so it stops being reached for here.

**New assets to build in the library, not in this scenario** — every ambient-display scenario after this one will want them: `ProvenanceInk` (foundation), `HeroReadout`, `AmbientDisplayShell`, `CycleController` (service), `AmbientCanvas`, `FreshnessArc`, `RollingNumber`, `SampleSeries`.

## One resolver for provenance

The UI receives `coverage` and `trust` and maps the pair to an ink through **one shared resolver**. No component decides independently how a sample looks. A second place that makes that decision is a second source of truth about honesty, and they will diverge.

## Cross-references

- [PROVENANCE-MODEL.md](PROVENANCE-MODEL.md) — what the inks mean
- [ARCHITECTURE.md](ARCHITECTURE.md) — the API the board reads
- `DESIGN.md` — the display design contract
- `../../experience/index.json` — the per-surface experience specs
