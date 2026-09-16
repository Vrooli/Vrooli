# Decisions

Durable decisions for this scenario, newest first. A decision recorded here is binding on implementation; changing one is a new entry, not an edit.

---

## 2026-09-16 — Ledger is a measured monetization instrument

**Decision.** Ledger owns the revenue and credit-economy story: LPBS remains
the producer for provider-reported AI cost, revenue-by-line totals, and credit
usage; Command Center reads those contracts and computes credit margin only at
read time. Offer Desk contributes the live Money Ledger posture, including the
declared default-alive buffer, without storing or correcting financial values.

**Why.** Authored samples cannot answer whether the business is working. The
room must distinguish measured producer data from in-reach projections while
remaining useful when production or an optional upstream is unavailable.

**Consequence.** Credit readings stay out of Broadcast, production promotion is
evidence-gated, and missing operator inputs remain absent rather than zero.

---

## 2026-09-15 — Panorama is a star atlas of the whole board

**Decision.** Panorama answers its charter question, "what is the state of the whole, and what is unmeasurable?", instead of repeating one room's headline. The hero counts signals measured across every other room, with the total and a reason for each unmeasured signal (sensor failing, in reach, no substrate). The scene is a star atlas: each room is a constellation, each of its signals a star in the provenance material of where it stands, and a constellation line is solid only between two measured stars. The room's own readings, the five room headlines, all become supporting tiles. `GET /api/v1/rooms/{id}` carries `constellations` for a room of category `panorama`, built from the registry's room list.

**Why.** The previous composition was five identical rings around a glow that stood for the retired composite score, with its room list hard-coded in the scene, and a hero ("healthy apps") already shown in The Hive. It encoded provenance and nothing else. Counting the whole board is the one thing no other room can show, and it is the board's own measurement, so it is solid ink.

**Constraints kept.** Counts, never a ratio (production-ledger archetype, 2026-09-01). One classifier, `ui/src/lib/sky.ts`, over the shared ink resolver, used by both hero and scene; an `IN-REACH` reading that returns a number still counts as in reach, as the resolver draws it. Star positions are seeded by room and metric id, so a signal going live changes weight, never place. A new room appears in the atlas with no code change.

**Alternatives rejected.** Polishing the constellation, which would still draw five circles that encode nothing. A wall of the five rooms' scenes, which repeats the cycle and runs several canvases on a kiosk. A WebGL scene, which reopens the three.js removal for no gain in meaning.

---

## 2026-09-15 — The release ladder is a structured reading, not a flattened panel

**Decision.** Offer Desk's release ladder is read as its own reading kind, `ladder`, carrying every ranked rung with what it opens, the enabling work due by its rank, linked goals and readiness. It renders as Next Rung (standard beat), the Reach Map (a new `wide` beat layout) and a strip summary. Offer Desk stamps `generated_at` on the ladder so the reading is trusted on producer time.

**Why.** The ladder was projected into panel rows: fourteen rungs broke the six-row panel cap, the response carried no producer time, and the rows dropped everything that makes the ladder worth showing. The room was blank, and the scene drew five identical unlabelled lines even when rows passed.

**Alternatives rejected.** Raising the panel cap, which would still flatten the schedule. Stamping observation time in Command Center, which would break the producer-time contract every reading honours. A side rail for the summary, which the room layout does not have; the supporting strip already is that rail.

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
