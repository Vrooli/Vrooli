# Decisions — Asset Studio

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

D-001 through D-014 come from the design work that preceded scaffolding. Where a
decision inherits a position already settled elsewhere in the fleet, the source
is named — reusing a decision is cheaper than rediscovering it, and the pointer
is what lets a future agent check whether the original evidence still holds.

| ID | Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|---|
| D-001 | 2026-07-28 | Use the generated `react-vite` scenario documentation contract. | Scaffold generated from the template; it is the only general-purpose scenario template in the registry that is not landing-page-specific. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Adoption of a different template or doc contract. |
| D-002 | 2026-07-28 | **An identity block referenced by an accepted asset is immutable; a change is a new version.** | The obvious design is a mutable library you edit as a character evolves, which is how the marketing catalogue works today. That is right for canon and wrong here, because this scenario promises an artifact can be explained and re-made. A mutable identity makes both promises false the first time someone improves a description: every prior artifact then claims provenance from a definition it never saw. | No in-place update on a referenced block, unlike every other CRUD domain in the fleet. Superseded versions stay resolvable forever. **Unreferenced records stay freely editable**, so authoring a new persona carries no versioning overhead. | Never — this is the scenario. |
| D-003 | 2026-07-28 | **This scenario stores bytes; `content-desk` stores none.** | The two scenarios were designed together and this is the fork between them. Content-desk's D-018 narrowed its own media scope to attaching single images by reference; the identity-consistent and video scope landed here. | A BlobStore seam exists here and is deliberately absent there. Exactly one copy of every artifact exists, in the scenario that produced it. Consumers receive a reference carrying metadata, dimensions, alt text, and disclosure state. | Never. A change putting artifact bytes in a consuming scenario is a layering defect, not an optimisation. |
| D-004 | 2026-07-28 | **Provenance is captured at completion or the job fails.** | A partial provenance record is not a degraded success. An artifact produced without one can never acquire it, because the inputs are gone by the time anyone notices — this is the same "cannot be backfilled" property that made citation-to-span anchoring P0 in `content-desk` (its D-019). | A completed job stores spec version, every bound identity version, backend, model, seed, and resolved parameters, or it is a failure. `ASSET-P1-010` regeneration exists to *test* completeness rather than to add a feature: if regeneration needs anything the record lacks, that is a defect in `renders`. | Never expected. |
| D-005 | 2026-07-28 | **An unchecked frame is not a passing frame: release blocks on an unresolved verdict, not only a failing one.** | The gate must fail closed. A drifted identity entering the published record compounds — every later artifact anchors to it, and the drift is invisible until a campaign's worth of material is inconsistent. This is the visual analogue of content-desk's claim gate and takes the same posture for the same reason. | Release is refused with a typed cause naming the specific frame and identity. An asset depicting no identity needs no verdict and releases normally, so the gate only binds where it means something. | Never expected. Weakening it to "no failing verdict" is the most likely way this scenario decays. |
| D-006 | 2026-07-28 | **A conformance verdict comes from a human operator; automated scoring is advice.** | Scoring a frame against reference images is unvalidated — nobody has yet rendered the same character twice through this pipeline, so the comparison may be too weak to catch real drift or too strict to pass anything. An automated pass on a mis-scored frame is precisely the silent failure the gate exists to prevent. | `ASSET-P0-011` accepts verdicts only from an operator identity and records actor, actor kind, reference, frame, and time. `ASSET-P1-005` stores a score beside the verdict and never in place of it. | If scoring is ever validated against a corpus of operator judgements large enough to measure its false-pass rate. Not before. |
| D-007 | 2026-07-28 | **Cost accounting is P0, not an operational concern added later.** | Generation spend is unbounded in a way editorial work is not: a mis-specified multi-frame video spec can cost real money before any human sees a frame. Editorial mistakes cost attention; these cost money, and the feedback arrives after the spend. | Every job records an estimate before submission and an actual after, and a failed attempt still records what it consumed. Spend aggregates by spec, identity, and campaign reference. `ASSET-P1-006` adds the budget that makes it actionable. | Never expected. A pipeline that cannot answer what it spent cannot be given a budget later. |
| D-008 | 2026-07-28 | **Every inference call routes through ai-gateway; no vendor SDK, no vendor credential, no fallback.** | Fleet-wide policy that ai-gateway owns model routing, capacity, and BYOK/hosted fall-through. It also happens to be the correct hedge here: video model interfaces are moving quickly, and a payload compatible today may not be in six months. | This scenario speaks no vendor protocol and holds no vendor credential. A gateway outage queues renders rather than failing them, and no direct-to-vendor path exists to be tempted by. | Never call a model vendor directly — that would duplicate policy the gateway owns and create a second credential surface. |
| D-009 | 2026-07-28 | **Capture is a spec kind and a backend, not a second pipeline or a separate scenario.** | A generated persona frame and a recorded product demo feel like different products but differ only in source; job lifecycle, provenance, cost, library, variants, alt text, and release are identical. Modelling capture separately would duplicate all of it to avoid one discriminator field. | `ASSET-P1-003` adds a capture spec kind executed through browser-automation-studio behind a seam. **This scenario never drives a browser itself.** This is also where the orphaned `video-studio` skill's scope lands — that skill points at a scenario that was never built. | If capture ever needs a materially different provenance or gate model, which would mean the shared downstream assumption was wrong. |
| D-010 | 2026-07-28 | **Import from the marketing catalogue is one-directional and idempotent by content-addressed key.** | Inherited from `vrooli-memory` D-016 and `content-desk` D-009, on identical evidence: every source is a file a human or an agent rewrites and reorders, so positional keys break on the first reflow and watermarks desynchronise on any out-of-order edit. | `hash(source_path, normalized_content)`, unique-indexed. Re-running is a no-op for unchanged items — that is the diff, with no cursor or state file. **Accepted consequence:** an entry edited in place imports as a new *identity version* rather than mutating the block, which is correct under D-002. | If edit churn produces duplicate versions that reviewers cannot reconcile. |
| D-011 | 2026-07-28 | **The catalogue stays authoritative for strategy; this scenario is authoritative for the operational record.** | The same split `content-desk` draws between the post-type registry it owns and the strategic post-type canon it reads. Which personas exist and why is operator-curated marketing canon that moves by accepted decision; whether a record validates, what version a render bound, and which reference a verdict was judged against is operational. | Import reads; nothing writes back. After import the two can disagree — the registry wins for rendering, the catalogue wins for strategy, and a re-import sweep reconciles the first without touching the second. **`content-desk` D-013 predates this scenario and says these definitions "stay in operator-curated canon"; that remains true of the definitions and is now narrower than it reads.** | If the catalogue is ever retired in favour of authoring identities here directly, which would be a canon decision and not this scenario's to make. |
| D-012 | 2026-07-28 | **Compositing is named ordered slots, never a timeline editor.** | The pull toward frame-level editing is strong and the scenario would never finish. Templated intro, outro, caption, and lower-third slots cover the marketing shapes the post-type catalogue actually describes. | `ASSET-P1-004` assembles ordered segments with declared slots. A request needing free positioning is out of scope by design and should be answered by an external tool, not by growing this one. | If a post type is ever activated that genuinely cannot be expressed as ordered slots. Check the post-type doc first — it may be the wrong post type rather than the wrong scope. |
| D-013 | 2026-07-28 | **Look recipes and image operations are borrowed from `image-tools`, not reimplemented.** | `image-tools` already owns deterministic operations, reusable look recipes, and image analysis. Duplicating them here would create a second place visual consistency is defined — the exact failure this scenario exists to prevent, applied to itself. | Specs reference a look by identifier (`ASSET-P1-009`); variants and conformance scoring call through seams. This scenario adds identity and provenance on top of an existing toolbox rather than replacing it. | If `image-tools` ever drops look recipes, which would make the reference dangle. |
| D-014 | 2026-07-28 | **P0 is a single-identity still image, walked end to end.** | *The recurring lesson, applied a third time.* `vrooli-memory` D-022 records shipping a P0 with no writers, so compaction could not be exercised. `content-desk` sequences one manual publish before its workbench for the same reason. A render pipeline with no released artifact cannot validate its own conformance model — the central promise would ship untested. | Video, multi-frame, capture, and compositing are all deferred to P1 despite being the reason the scenario is interesting. The first slice runs the full domain chain on the cheapest possible artifact. | Never expected. Broadening P0 to include video is the most likely way this scenario ships something unvalidated. |

**Note on D-011 and content-desk D-013.** They do not conflict. Content-desk
deferred *a rendering scenario* and recorded that definitions live in canon;
this scenario is that deferred rendering scenario, and it imports those
definitions rather than relocating them. If a future change moves persona
authorship out of canon and into this scenario, that would contradict D-011 and
should be recorded as a supersession here and raised as a canon decision there.

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. Nothing here has met contact with an implementation, so no position has yet had the chance to be wrong. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
