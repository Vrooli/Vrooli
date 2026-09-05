# Decisions

Durable decisions for this scenario, newest first. A decision recorded here is binding on implementation; changing one is a new entry, not an edit.

---

## 2026-09-01 — Command Center becomes the Director Swarm instrument

**Decision.** Command Center takes the full instrument shape defined in `path:docs/agent-system/TARGET_MODEL.md`: authored space joined live, a setpoint it reads but does not own, trust-qualified readings, one ranked surface, an open-loop self-report, and a describe endpoint. It becomes the team's single address, closing the `two addresses` deviation recorded in the team's own record on 2026-08-13.

**Why.** The team declares its instrument `partial` and names Command Center as the only viable candidate, because it owns none of what it would observe. The portfolio control loop runs open-loop today: the prediction ledger records falsifiable predictions against named Command Center metrics with horizon dates, and none can be scored because the payload carries labels and no values.

**Alternatives rejected.** Values-and-display-only, which would leave the loop open and compute nothing about which sensor to build next. Phasing the instrument after the display, which would design the registry schema twice.

---

## 2026-09-01 — The objective join is read as a transmitter, not absorbed

**Decision.** The objective set stays in `prompt-manager graph objectives` and is read over the standardised verb. It does not move into this scenario.

**Why.** This is the fork the team's gap marker is blocked on. Invariant 2 forbids the alternative: the instrument never authors the denominators it measures against, and an observer that writes its own reference model is confirming itself. The target model's role table names the read verb a *bus contract* whose value scales with the number of conforming devices.

---

## 2026-09-01 — Provenance is material-primary; DESIGN.md is amended

**Decision.** A reading's honesty is carried by material — solid, dimmed, hollow, dotted — with colour as reinforcement. `DESIGN.md`'s status-colour section is amended to say so. Violet remains the sample tone where a room's palette allows.

**Why.** Colour alone fails at distance, fails on a projector, fails for colourblind viewers, and is already spent on room identity. A greyscale render of any room must remain unambiguous.

**Consequence.** The design contract changes rather than quietly going stale. Recorded because a future reader comparing `DESIGN.md` against its git history needs to know this was deliberate.

---

## 2026-09-01 — Coverage and trust are independent axes

**Decision.** A reading carries `coverage` (`NOW` / `IN-REACH` / `MISSING` / `UNREGISTERED`) and `trust` (`VALID` / `CACHED` / `UNAVAILABLE` / `UNTRUSTED`) as separate fields. The API never emits a composed status or a pre-resolved ink.

**Why.** Invariant 3: three axes, never merged. The 2026-04 model had one field, `dataSource: live | partial | gap`, which could not express "the sensor exists and this fetch failed" — the single most common real state.

**Consequence.** `UNREGISTERED` is added beyond the standard three, because an outcome nobody named renders nowhere and is the only kind of hole that never ages.

---

## 2026-09-01 — Sample values are authored, never generated

**Decision.** Illustrative values are checked-in registry data with a required `basis` string. No code path may construct a sample from an upstream response, a previous reading, or a runtime computation.

**Why.** The requested feature is "always show a number, and mark it when it is not real." The dangerous implementation invents plausible numbers at runtime, which is indistinguishable from a bug. Authored samples are reviewable in diff, stable across reloads, and stamped so nothing downstream can mistake them for measurements.

**Rejected.** A `reviewBy` date on each sample that fails CI. That was a primitive version of something the platform already does properly — the gap-closure loop in the outcomes charter routes non-live metrics to `outcome-gap` work with operator approval. This scenario's obligation is to emit a ranked, dated signal, not to nag on a timer.

---

## 2026-09-01 — Surface and rank only; no actuation

**Decision.** No write path of any kind. No work-item filing, no setpoint writes, no upstream mutation, and no UI action that performs any of these.

**Why.** Invariant 5, and the target model's tag-letter argument: `FT` is a transmitter, `FIC` is a controller, and the `C` is what confers the right to act. The charter's gap-closure loop already assigns proposal to `outcome-strategist` and approval to the operator at the vision walk.

---

## 2026-09-01 — No composite scores; the archetype is production-ledger

**Decision.** The board reports counts with stated denominators, queue state, and staleness against a window. It does not report synthesised percentages or composite health scores.

**Why.** Director Swarm's declared archetype is `production-ledger`, and the target model is explicit: such a board returns "queue state, staleness against a window, outcome evidence. No percentages. Do not force a coverage ratio onto a production team: a denominator nobody can defend is worse than an honest ledger."

**Consequence.** The Panorama composition drafted during design used a composite health score of 82 and five ring gauges. Both are disallowed and must be re-expressed as ledger state. The archetype belongs to the team; if `team.json` later declares `coverage-board`, this relaxes — as a team decision, not a design change made here.

---

## 2026-09-01 — The taxonomy is data, not code

**Decision.** Room list, metric set and source bindings are derived at read time. The registry is versioned with a required migration path, metric ids are stable and never reused, rooms are a grouping query, and retirement leaves a dated tombstone.

**Why.** Nothing about what Vrooli tracks is settled. Adding a seventh outcome category, splitting a room, or re-pointing a source must be a data change with a migration, never a code change with a release.

**Consequence.** The UI generates routes from `/api/v1/board`. No room id is hardcoded in the router.

---

## 2026-09-01 — Runs everywhere; the capability ladder is the architecture

**Decision.** One build serving a desktop browser, a phone, a gamepad-controlled TV and a wall panel, reached through `tunnel-manager`, with a runtime capability probe selecting the scene tier. Each room ships designed landscape *and* portrait compositions.

**Why.** The scenario is expected to work like every other scenario on every surface. Xbox support specifically means gamepad control, which the shared `@vrooli/iframe-bridge/spatial` package already provides — `GamepadInputManager` emits `page-next` / `page-prev` / `menu` / `select`, mapping directly onto the intent vocabulary.

**Consequence.** The figure layer is identical at every tier; the reading is never degraded to protect the decoration. A mounted scene that draws nothing is a failure, not a pass.

---

## 2026-09-01 — Both display topologies, from the first commit

**Decision.** Auto-cycle ships first, but the pinned-wall path is designed alongside it: the room is a pure URL parameter, ambient motion seeds per display so adjacent screens never run in sync, and nothing assumes exactly one room is live.

---

## 2026-09-01 — Audience modes gate illustrative readings

**Decision.** `samples=hide|mark|full`. Outward-facing displays default to `hide` and compose from real readings only; internal displays default to `mark` with a persistent legend.

**Why.** The board is sometimes seen by people outside the team. "Hollow means illustrative" is a convention defined in this scenario's documentation and known to nobody outside it.

---

## 2026-09-01 — The blank-scene defect is root-caused before scene work

**Decision.** Investigate the existing five-of-six blank canvases on hardware with a GPU before building any new scene.

**Why.** The cause was narrowed to the lighting path but never proven. If it lives in the canvas wrapper or the renderer version rather than in the placeholder scene, it follows into all six new rooms.

---

## Superseded

- **2026-04-18 — Command Center is a read-only kiosk aggregator.** Superseded by the entries above. The aggregation, TTL cache and staleness behaviour it produced are retained and restated against the honesty contract; the framing, the payload shape and the UI are not.
