# Routines App — Orchestration Summary

**Origin:** Morning Vision Walk, 2026-04-23 (first calibration walk).

**Why this exists:** The Vrooli project has a clear pipeline for automating *digital* chores — new scenarios get built, friction at the agent level drops. Physical chores (cleaning, inspection, maintenance, repair) cannot be automated the same way. The best achievable form of automation for physical tasks is **removing the planning and executive-function load from the user** — telling them what to do, in what order, with what products, at what cadence. This is the purpose of the routines app: close the loop on physical chores the way new scenarios close the loop on digital ones, so the Morning Vision Walk's chore-audit phase has a mechanism for converting friction into future absence-of-friction.

The scenario targets users — the operator first and foremost — who prefer being told what to do over figuring out *how* to do it. The Bar Keeper's Friend-for-sink-discoloration insight is the archetype: non-obvious product/technique knowledge that most people would spend years not discovering. Routines are the vehicle for that knowledge to compound.

## Vision framing (must be preserved in initiative docs and agent handoffs)

### The flexibility ladder

Routines must support multiple representation levels so the authoring cost matches the task's structural complexity:

1. **Markdown prose** — default. Steps as paragraphs or bullets. No structural overhead. Appropriate for most routines.
2. **Structured list** — ordered steps, each potentially with duration / required products / optional branches.
3. **Light DAG** — for routines with parallel branches, fallbacks, or dependencies between steps.
4. **Rich graph (BPMN-level concepts, not the spec)** — events, parallel gateways, compensation, interrupt boundaries. Needed for long-horizon use cases (humanoid-robot orchestration of a multi-room deep clean with interrupts and sensor integration). Not supported in v0.

**Explicitly not adopting BPMN 2.0 as a spec.** The original Vrooli project died partly because it committed to BPMN 2.0 proper. A Vrooli-native rich-graph representation with the concepts needed (events, gateways, compensation) is lighter, stays domain-appropriate, and doesn't lock us into a 2010-era business-process standard. If BPMN 2.0 XML export is ever wanted for interop, it's an export path, not a foundation.

### Routines are a resource/primitive — not a user-facing "build your own workflow" app

The original Vrooli sold "build your own workflows" to end users. Almost nobody wants to build workflows; they want outcomes. This scenario must keep authoring infrastructure **internal**. The authoring surface exists for the operator (and agents) to curate routines; the user-facing surfaces are the library (browse/search) and the executor (follow a routine, step through, upload evidence). The routines-as-a-product trap is explicitly rejected.

### One app, not many branded wrappers

The worst design outcome is "generic routines app + branded scenarios on top" — branded wrappers degenerate into thin UIs, the user sees generic routine-builder UX bleed through, specialization collapses. Two acceptable shapes:

- (a) **Routines are a resource** other scenarios consume — invisible to the end user as a distinct app.
- (b) **One opinionated app** with a curated library across home/car/electronics — no user-visible "build your own" surface.

v0 goes with (b). (a) may emerge later as the routine representation stabilizes.

### Library vs executor are different surfaces

The library (browse / search / author / manage) and the executor (actively follow a routine, tick steps, handle branches, upload photos for evidence) share data but have distinct UX. Do not collapse them into one confused page.

### Steps as calls to scenario CLIs

Routine steps can invoke any scenario's CLI — browser-automation, agent-manager, inventory, image-classifier, etc. This replaces the original project's seven-step-type schema with composition over all of Vrooli's existing capabilities. It's also the source of **hidden coupling** — a routine authored against scenario X's CLI today breaks when X renames a flag next quarter. A versioning / contract story is required.

### Validation is separate from authoring

A routine exists ≠ a routine works. As the library scales, some form of validation (user feedback on completion, agent review before publication, photo/video evidence on critical steps, failure-mode replay) is required so the library doesn't silently rot as scenario CLIs evolve. Not in v0; required in a later iteration.

## v0 scope (narrow, deliberate)

The v0 acceptance criterion is **the operator using it for his own personal routines** on home + car + basic electronics maintenance. This is the validation gate — if v0 doesn't improve the operator's physical-chore experience, the scenario failed regardless of technical correctness.

v0 includes:

- React-vite scenario scaffold (UI + Go API + CLI, per Vrooli template standards)
- Routine = folder on disk containing a `routine.md` (markdown representation only in v0) plus assets (images, videos, reference docs, manifests of required products)
- Semantic search over the library (consistent with swarm-manager / prompt-manager patterns)
- Library UI (browse / search / filter by category, tags, required inventory)
- Executor UI (follow a routine, step through, mark complete, optionally attach photos)
- Integration with the **inventory scenario** (separate initiative) — surface which routines are available now vs blocked on missing products
- Starter curated routine set — minimum 10, spanning home cleaning (including the Bar Keeper's Friend sink-routine), car maintenance (basic interior + exterior), and small-appliance / electronics care. Selected around the operator's personal needs.

v0 explicitly **does not** include: routine authoring UI for end users, light-DAG or rich-graph representations, affiliate links, consumer-product offers, validation feedback loops, calendar integration, phone-agent voice surface, robot-orchestration concepts.

## v1+ directions (not committed; documented for context)

- Light-DAG representation for routines with parallel/branched steps
- Calendar integration (timed reminders when routines are due)
- Phone-agent voice surface ("what should I clean today?")
- Validation / feedback loop
- Agent-assisted authoring
- Inventory-driven product recommendations (gated on the affiliate / consumer-product revenue-line architecture and its recommendation-blindness post-processor — see `docs/monetization/revenue-lines/`)
- Long-horizon: humanoid-robot orchestration (rich-graph representation, physical-digital task blending)

## Compound value ties

- **Inventory scenario (separate initiative)** — substrate. Determines what routines are runnable. Gates consumer-product / affiliate monetization downstream.
- **Contact-book-plus scenario (separate initiative)** — enables gift-purchase surface and relationship-aware routine customization (e.g., different cleaning cadence when hosting).
- **Calendar scenario (future)** — timing layer.
- **Phone-agent (future)** — voice input.
- **Lifestyle-bundle hero candidate** — if v0 validates, this is a natural headliner for the lifestyle bundle once it activates.

## Monetization implications (held separate)

This scenario is *the substrate* for the lifestyle-bundle's `consumer-products` and `affiliate-commerce` revenue lines but does **not** activate them in v0. The revenue-line architecture (recommendation-blindness post-processor, inventory-gated offer insertion, FTC disclosure UI, truthful opt-out) must exist before any routine surfaces a purchase prompt. See `docs/monetization/revenue-lines/consumer-products.md` and `docs/monetization/revenue-lines/affiliate-commerce.md`.

## Relationship to the archived "chore-tracking" idea

Not applicable. The archived chore-tracking attempt was a gamified family-first system (points, achievements, leaderboards). This scenario has a different center of gravity (authority-layer content for solo adults) and should not be framed as a revival.

## Risks

- **Scope creep.** Shipping anything beyond markdown-only v0 delays the validation gate. Resist.
- **Hidden coupling of routine steps to scenario CLIs.** Versioning story required before the library crosses ~50 routines that invoke scenario commands.
- **Content acquisition.** Operator curates v0. The long-term acquisition strategy is a separate research item (see `lifestyle-demand-validation` initiative).
- **Validation debt.** A growing library without a validation loop rots. Track this as a dependency for v1.
