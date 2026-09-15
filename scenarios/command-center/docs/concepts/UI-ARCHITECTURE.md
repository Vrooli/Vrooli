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

- **Landscape** — figure in the void, scene owning the opposite side, supporting readings in a row. **A landscape room never scrolls**: it is the wall display, and content below the fold is content nobody sees.
- **Portrait** — figure at the top, scene reduced to a band, supporting readings stacked. Portrait is the desk view and may scroll vertically.

Room identity survives at 390px. No surface scrolls horizontally at any width. The portrait view is the one people check from their desk, so it is designed, not tolerated.

## Fitting a landscape room

The landscape figure layer is a fixed budget, not a flow (`CC-P1-017`):

- **The strip is capped, the hero takes the rest.** The supporting strip gets at most a third of the figure layer's height. The hero region is `minmax(0, 1fr)` and sizes its figures to its own box through container units, not to the viewport.
- **A strip that does not fit tightens, then pages.** It first drops to a compact density. If that still does not fit, it splits into pages that cross-fade in place, with the column count held across pages and a page counter in the corner.
- **A list that is tall by nature auto-scrolls.** Panel rows, Next Rung blockers and Reach Map lanes sit in an auto-scroll viewport. It holds the first rows for four seconds, steps up one row at a time, holds at the end, then fades back to the top; it never scrolls back up. Edges fade where more rows wait, and a counter names the rows in view. Under reduced motion it steps a viewport at a time without animation. A list that fits does not move.
- **Paging and scrolling hold the beat.** A beat does not end until its strip has shown every page and its list has made one full pass, bounded at 90 seconds past the authored dwell. The rail waits at the segment's end so a held beat reads as reading, not frozen. Manual navigation is never held.
- **The room reports its own fit.** The room element carries `data-fit`: `ok`, `overflow` when a figure leaves the viewport or the hero runs into the strip, or `scroll` in portrait. Workflow cases assert `ok` on the densest beats at 1280×720.

## Supporting tiles speak on exception

A supporting tile draws its label, its figure and the freshness hairline. Its text qualifier appears only when the reading needs explaining (cached, not answering, untrusted, illustrative or absent), on one line with the full text on hover. A live reading's source is already named in the room's source strip and its freshness is the hairline, so writing both under every figure spent a third of the strip saying nothing new. The live text stays in the accessibility tree (`PROVENANCE-MODEL.md` §"Every figure carries its qualifier").

Origin is shown only where it differs. A production reading in a room of local readings carries a `PROD` mark on its qualifier line. A room whose readings all share a non-local origin names it once in the source strip.

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

URL parameters seed the whole state so a kiosk boots configured with no interaction: `?room=forge&cycle=45&samples=mark&fullscreen=1`. `?beat=N` opens a room on one beat.

## Beats, layouts and list readings

A beat features one reading as the hero. Its `layout` decides how much of the figure layer the hero takes:

- **`standard`** (the default) — the hero holds the left column and the beat's composition owns the band beside it.
- **`wide`** — the hero spans both figure columns and the scene steps back to 30% opacity. Use it only for a reading that is an axis, not a figure.

The hero component follows the reading's `kind`: `scalar` renders the wall figure, `panel` renders ranked rows, and `ladder` renders **Next Rung** on a standard beat and the **Reach Map** on a wide one. The Forge shows Next Rung beside the funnel-cascade scene, which draws the actual next rungs with their names; the Hive shows the Reach Map.

A supporting tile follows the same kind. A list reading has no single figure, so a `panel` tile shows its leading rows and a `ladder` tile shows the next release and one segment per rung, instead of a dash (`CC-P1-016`).

**A ladder beat has no supporting strip.** When the hero is the release ladder (Next Rung with the funnel-cascade scene, or the Reach Map), the schedule is the whole page: the strip is omitted and the hero takes its height. On every other beat the ladder still appears in the strip as its tile.

Both topologies are supported from the first commit: one display cycling, and several displays each pinned to a room. That means the room is a pure URL parameter, ambient motion seeds per display so adjacent screens never run in sync, and nothing anywhere assumes exactly one room is live.

## Motion

| Register | Rate | Purpose |
|---|---|---|
| Ambient | 0.02–0.2 Hz | Atmosphere. If a viewer perceives it as motion rather than atmosphere, it is too fast. |
| Freshness | Per source TTL | The board showing its own pulse. Replaces a `STALE` badge. |
| Value change | 380ms, changed digits only | Legibility. A whole-number crossfade reads as a flicker at distance. |
| Beat change | Authored dwell, segmented rail | Rotates a room through complete readings without adding persistent chrome. |
| Room change | 900ms fade-through-black | Lets six colour worlds sit next to each other without a hue slam. |

Two deliberate absences: **no pulsing alerts** — a threshold crossing gets one 1.2s bloom and holds, because a loop trains people to stop seeing it; and **no entrance animation on cycle** beyond the fade. Beat changes carry information through the segmented rail and complete-page swap, so they do not add an entrance animation.

Under `prefers-reduced-motion` (`CC-P1-013`): scenes hold a composed still rather than blanking, the freshness indicator steps in quarters, digits swap without rolling, room changes crossfade.

## Theming

Six themes as **React Component Library semantic token overrides** — `--color-*`, `--space-*`, `--text-*` — so library components inherit the active room's palette (`CC-P1-014`).

The scenario-private `--cc-*` vocabulary is retired. The room themes now override the library semantic tokens directly. The former collision between the four shorthand typography tokens (`--text-body`, `--text-label`, `--text-display`, and `--text-title`) and size-only room values was corrected: each has a complete `font` shorthand, with separate `*-size` and `*-line` tokens for size-only declarations. Library components therefore inherit the active room theme without invalidating their typography.

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
