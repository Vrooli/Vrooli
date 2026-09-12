# Provenance Model

**Status:** contract canon for this scenario. This is the rendering half of [COVERAGE-MODEL.md](COVERAGE-MODEL.md) — how a reading's honesty becomes something visible from ten feet away.

## The problem this solves

A board that runs unattended on a wall has one failure mode that matters more than any other: a number that looks real and is not. The previous design handled this by putting a red `GAP` chip next to the metric's *label* and showing no number at all. That failed twice — it showed nothing where a figure should be, so nobody could see what the finished display would look like; and it coloured a future capability the same as a critical failure, so a room of unbuilt pipelines read as a room on fire.

The requirement is the opposite of both: **always render a figure, and make its truthfulness a property of the figure itself.**

## Ink

Every figure on every surface is drawn in one of four **inks**. An ink is a *material* — solid, dimmed, hollow, dotted — not a colour.

| Ink | Composition | Rendering | Reads as |
|---|---|---|---|
| **Live** | `NOW` + `VALID` | Solid fill, full contrast. A hairline beneath drains over the source TTL. | This is measured and current. |
| **Cached** | `NOW` + `CACHED` | Same digits at reduced contrast, with an age readout and a dashed freshness rule. | This was measured; the sensor is not answering now. |
| **Sample** | `IN-REACH` | Hollow outline, stroked not filled. | This is a drawing of a number. The shape is known; the pipeline is not built. |
| **Absent** | `MISSING` | Dotted outline. | The shape of a future capability. Nothing collects this anywhere. |

`UNREGISTERED` has no ink because it has no figure — an outcome nobody named cannot appear on a room. It surfaces only in the self-report, which is precisely why that surface has to exist.

### Why material and not colour

Three reasons, all load-bearing:

1. **Colour fails at distance and on hardware.** A wall panel at its dimmest configured brightness, a projector, a phone in sunlight — hue separation is the first thing to go. Stroke weight and fill/no-fill survive all of them.
2. **Colour is already spent.** Each room has its own accent, and the accent is what makes the six rooms distinguishable at a glance. If provenance also needed a hue, either provenance or room identity would have to give way.
3. **Colour alone is not accessible.** WCAG forbids status carried by hue alone, and this board is almost entirely status.

Colour still reinforces — the sample ink leans toward the violet the design contract assigns to future capability where a room's palette allows it. But **a greyscale render of any room must remain unambiguous** (`CC-P1-001`). That is the test.

### Layout does not change with ink

A metric going live is a change of *weight*, not a change of layout. The hollow figure occupies exactly the space the solid figure will occupy, at the same size, in the same position, with the same supporting furniture. Two consequences:

- The room is complete today. A passer-by sees the finished composition, not a placeholder.
- The day the pipeline lands, dotted strokes become solid fills and **not one pixel moves**. No reflow, no re-layout, no visual event that reads as a bug.

This is also why the sample state is safe to leave running for months, and why it must never be a smaller or greyer version of the real thing.

## Every figure carries its qualifier

No value appears alone (`CC-P1-002`). Each ink has a required qualifier, sized to be read at distance:

| Ink | Required qualifier |
|---|---|
| Live | Source name and freshness — the draining hairline plus "refresh in Ns" |
| Cached | Age of the last good reading, and that a retry is in flight |
| Sample | What is needed to make it real, naming the missing surface |
| Absent | The owning team, and the days the gap has stood |

An `UNTRUSTED` reading is a special case: the figure is drawn but the integrity finding is shown *with* it. A number that arrived and cannot be believed is more dangerous than no number, so it never appears unqualified.

## Sample values are authored, never generated

This is the rule that makes the whole system trustworthy, and it is easy to get wrong.

**Sample values are checked-in registry data.** They are authored by a person, reviewed in diff like any other content, and stable across reloads. The API passes them through with their provenance attached; it never computes them, never derives them from a real reading, and never randomises them.

```json
{
  "id": "revenue_mrr",
  "label": "Monthly recurring revenue",
  "unit": "usd",
  "format": "currency.compact",
  "coverage": "IN-REACH",
  "owner": "team:monetization",
  "whatIsNeeded": "A revenue surface on the monetization instrument",
  "firstObservedMissing": "2026-04-18",
  "sample": {
    "value": 12400,
    "series": [8100, 8800, 9600, 10200, 11300, 12400],
    "basis": "hand-authored, mid-scale, reviewed 2026-09-01"
  }
}
```

This row is `IN-REACH` rather than `MISSING` because the monetization instrument is `live` and the substrate exists — what is absent is a pipeline, not a control loop. The distinction is not cosmetic: it decides which ink is drawn, which owner the finding names, and whether the ranked surface calls for plumbing or for an instrument. See [SOURCE-MAP.md](SOURCE-MAP.md).

Two invariants, each with a test (`CC-P0-003`):

1. **A sample may never originate from an upstream.** No code path constructs a sample from a response, a previous reading, or a runtime computation.
2. **A sample is stamped.** Every emitted sample carries the `basis` string that authored it, so nothing downstream — a CLI consumer, an export, a screenshot pipeline — can mistake it for a measurement.

The dangerous version of this feature invents plausible numbers at runtime. That version is indistinguishable from a bug, and it is forbidden here.

## Audience modes

"Hollow means illustrative" is a convention this document defines. Nobody outside it knows that, which matters because the board is sometimes seen by people outside the team (`CC-P1-011`).

| Mode | Behaviour |
|---|---|
| `samples=full` | Sample and absent inks render normally. Internal default when the legend would waste space. |
| `samples=mark` | Same, plus a persistent on-screen legend explaining the inks. The default. |
| `samples=hide` | Sample and absent readings are withheld entirely; rooms compose from real readings only, and a room with nothing real says so. Outward-facing default. |

The active mode is part of the board's own visible state, so a screenshot is self-describing. A photograph of the board taken in `mark` mode carries its own legend; one taken in `hide` mode contains no illustrative figure to misread.

## Motion is part of the contract

Two motion behaviours carry provenance meaning and are therefore not decoration:

- **The freshness hairline** drains over the source TTL and refills on fetch. It replaces a `STALE` badge with staleness as a continuously visible property, and it makes the board show its own pulse.
- **Digit rolls** animate only the digits that changed, at 380ms. A whole-number crossfade reads as a flicker at distance. Under `prefers-reduced-motion` digits swap instantly and the hairline steps in quarters (`CC-P1-013`).

Bloom and other scene postprocessing must never touch the figure layer. The sample and absent inks are hairline strokes; a bloom pass smears them into unreadable haze and takes the honesty system with it. The figure layer composites *above* postprocessing (`CC-P1-006`).

## Relationship to the design contract

`DESIGN.md` assigns violet to gaps and treats colour as the primary status signal. That was amended on 2026-09-01 by operator decision to make **material primary and colour reinforcing**, for the three reasons above. The amendment is recorded in [../internal/DECISIONS.md](../internal/DECISIONS.md); the design contract itself carries the amended language. Where a room's palette permits, sample and absent inks still lean violet — the amendment changed which signal is load-bearing, not the palette.

## Cross-references

- [COVERAGE-MODEL.md](COVERAGE-MODEL.md) — the two axes these inks compose
- [DATA.md](DATA.md) — the registry and reading shapes
- [UI-ARCHITECTURE.md](UI-ARCHITECTURE.md) — where the rendering primitives live
- `DESIGN.md` — the display design contract this operates inside
