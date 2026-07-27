# Decisions — Signal Inbox

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

## Provenance

Decisions D-001 through D-025 come from an operator design workshop on
2026-07-27, held after an audit of the predecessor scenario
`bookmark-intelligence-hub` found it stalled at 25/100 completeness with 34 of
34 requirements unimplemented and a test suite that could not execute. The
workshop reshaped the capability rather than repairing the implementation.

The session that produced them began with an external signal — a GitHub project
for local X-bookmark storage, surfaced from the operator's own bookmark stream —
which is itself the class of material this scenario exists to capture. That
signal reached the project by being retold in conversation, because no capture
substrate existed. **That is the problem statement, observed rather than
hypothesized.**

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-07-27 | D-001 — The unit is a **signal**, not a bookmark. | The material the operator saves includes videos with a timestamp, AI-chat share links, screenshots, and pasted text. None of those are bookmarks; a bookmark is one arrival mechanism among several. | The core model is source-agnostic. Adding a source kind is an adapter, not a schema change. Several questions that were hard under a bookmark model — is a video in scope, is a pasted screenshot a bookmark — stop being questions. | Revisit if a source kind emerges that cannot be reduced to *artifact plus operator note*. |
| 2026-07-27 | D-002 — This is a general capture substrate; alpha extraction is one category and one consumer. | The near-term driver is feeding the morning vision walk, but the operator also wants categories with no relation to Vrooli strategy — meals, fitness — consumed by future scenarios. | No alpha-specific structure may enter the core model. Categories are domain-agnostic, and the system privileges none of them. The intake-pipeline vocabulary lives inside one category's taxonomy (D-011), not in the schema. | Revisit if a second consumer needs first-class structure the category/taxonomy pair cannot express. |
| 2026-07-27 | D-003 — Renamed from `bookmark-intelligence-hub` to `signal-inbox`. | The old name mis-describes the scope under D-001, and was going to keep mis-describing it. | 15 live external references must be repointed, five of them structural under `prompt-manager/store/` where `topics.json` `external_producers` entries are validated by `prompt-manager graph topics`. Rejected `signal-hub` (echoes search-hub, which is a router rather than a corpus) and `signal-intake` (names only the front door). | Revisit only if the scope narrows back to bookmarks, which would be a different scenario. |
| 2026-07-27 | D-004 — Regenerate from the template; do not migrate the predecessor. | The predecessor's API was mux-and-REST against Postgres, its UI was static JavaScript, its Twitter adapter's `GetBookmarks` was a `TODO` stub, and its categorization was a hand-written keyword matcher. | The entire implementation is discarded. Exactly one artifact is carried forward: the conversation-extraction recipe (D-016). Since D-003 changes the directory name anyway, regeneration is also the cleanest possible diff. | None. This is a one-time transition. |
| 2026-07-27 | D-005 — One operator; no multi-tenancy. | The predecessor's `OT-P0-001` was multi-tenant authentication with profile-scoped processing, aimed at "knowledge workers and teams". There is one operator. | No profile table, no per-user partitioning, no auth scenario dependency. Every query is global. Removing this assumption removes a whole subsystem from P0. | Revisit only if this scenario is productized for external users, which would be a different PRD. |
| 2026-07-27 | D-006 — The journal is append-only and nothing is ever deleted. | Capture is worthless if the corpus can silently lose entries, and a source that has gone offline cannot be re-fetched. | Signals are written once. Category, disposition, and annotations live in separate structures that reference the journal and never rewrite it. A `dropped` signal is still a stored, indexed, searchable signal. | None. This is a load-bearing invariant. |
| 2026-07-27 | D-007 — Categories are operator-defined at runtime, with an open set. | The predecessor hard-coded seven categories in code and stored keyword rules in a `category_rules` table. The operator's real categories are unpredictable and change over time. | Adding a category is data, never a code change. A reserved `uncategorized` always exists. Retiring a category reassigns its signals to `uncategorized` rather than deleting them. | Revisit if the category set grows large enough that flat naming becomes unmanageable, which would argue for hierarchy. |
| 2026-07-27 | D-008 — Classification proposes; only operator confirmation is authoritative. | Automatic classification is useful and unreliable, and the operator explicitly requires override for ambiguous content and for disagreement. | A proposal carries a confidence score and does not satisfy confirmed-category queries. Reassignment is available forever, including after a signal is `done`. A wrong category is the error that hides a signal from the consumer who needed it, so the classifier is never trusted alone. | Revisit the confirmation requirement if measured accuracy (D-009) is high enough that auto-confirmation above a threshold becomes defensible. |
| 2026-07-27 | D-009 — Every override is recorded as an annotation. | Classification quality was asserted by the predecessor PRD (">85%, >95% with learning") with no mechanism that could measure it. | Overrides accumulate a labeled corpus as a side effect of normal use, which `SIG-P1-008` measures against. Accuracy becomes observed rather than claimed. Cost is one annotation row per override. | None. |
| 2026-07-27 | D-010 — One primary category per signal, plus free tags. | A signal can genuinely relate to two categories, but disposition needs exactly one owner or "handled" becomes ambiguous when two consumers disagree. | Multi-category is deferred to `SIG-P2-003` and blocked on defining the multi-consumer disposition rule first. Tags cover the practical need meanwhile. | Revisit when evidence shows tags are insufficient — not before, since the relaxation costs the disposition model its clarity. |
| 2026-07-27 | D-011 — A category may optionally declare a typed taxonomy. | Alpha categories need subtypes (`workflow`, `competitor`, `hook`…) that are meaningless for `meals`. | Alpha structure lives inside one category's declaration instead of the core schema, satisfying D-002. Categories without a taxonomy are unaffected. | Revisit if most categories end up declaring taxonomies, which would argue for making it mandatory. |
| 2026-07-27 | D-012 — The alpha taxonomy reuses the vision-walk signal-type vocabulary verbatim. | `morning-vision-walk/SKILL.md` already enumerates the signal types, and `INTAKE_PIPELINE.md` already routes on them. | Signals emit tokens the intake pipeline already understands, so routing needs no translation layer and this scenario is a Mode-2 deterministic-prefix producer requiring no classifier on the draining side. A test pins the vocabulary against drift. | Revisit if the walk skill's vocabulary changes; the test will catch it. |
| 2026-07-27 | D-013 — Disposition governs ambient surfacing only, never storage or search. | The operator's stated problem: a signal that already produced a scenario keeps reappearing and burning context every heartbeat. | `done` and `dropped` signals never appear in the ambient view and always appear in search. This is the mechanism that makes a growing corpus cost bounded context rather than growing context. | None. This is the second load-bearing invariant, paired with D-006. |
| 2026-07-27 | D-014 — Annotations carry typed outcome links. | The workshop that created `vrooli-memory` could not answer "where did this idea come from" without re-reading a conversation transcript. | A signal records what it became — a scenario, a backlog item, an idea-pipeline entry, a knowledge topic. Provenance becomes a query. One signal may carry several outcome links, since one signal can produce several results. | None. |
| 2026-07-27 | D-015 — Every source adapter declares a risk tier, enforced by the runner. | The operator's hard constraint is not losing a platform account to automated capture. | Tier is declared in the descriptor and enforced centrally, not left to each adapter's good behaviour. An adapter with no declared tier fails to load. The contract ships at P0 even though only tier-0 adapters do, so the first networked adapter has a tier to declare rather than a precedent to set. | None. |
| 2026-07-27 | D-016 — Any anomalous response disables the adapter; it is never retried. | Retrying through a soft block is what escalates a throttle into a ban. | A 429, 403, or challenge response disables the adapter, raises an alert, and persists disabled across restart until an operator re-enables it. This is deliberately more conservative than a backoff policy, and it accepts stalled capture as the cost of not risking the account. | Revisit only with evidence that a specific platform tolerates retry, per-adapter and never globally. |
| 2026-07-27 | D-017 — Only tier-0 sources ship at P0. | Manual capture and operator-supplied exports carry no account risk and require no credential. | The scenario is fully usable at P0 with zero platform exposure, and import supplies the real corpus that makes classification and search measurable. Networked adapters follow once the substrate is proven. | None. |
| 2026-07-27 | D-018 — Enrichment delegates to the scenario that owns each capability. | `image-tools` owns image-to-text, `video-downloader` owns video, `browser-automation-studio` owns browser sessions. | No OCR library, media downloader, or browser driver is linked into this scenario. `SIG-P1-005` is explicitly blocked on `video-downloader` exposing a transcript-only request, which it does not today — declared as a dependency rather than assumed. | Revisit per capability only if the owning scenario cannot meet the contract. |
| 2026-07-27 | D-019 — An empty extraction is a failure, not an empty document. | Carried from the predecessor's reverse-engineering notes: naive extraction of a client-rendered share page returns page chrome and reads as an empty conversation. | Extraction asserts it found content; a zero-content result marks the signal `needs-attention` for manual entry rather than storing a blank body. Without this, the system is silently lossy in exactly the cases where the content mattered most. | None. |
| 2026-07-27 | D-020 — SQLite in-process, not Postgres. | The predecessor declared Postgres with a seven-table schema and shipped `schema.sql` plus `seed.sql`. Current scenario convention is embedded SQLite. | No resource dependency for storage, no migration code carried in a field scenario. FTS5 covers keyword and structured filtering. | Revisit only if the corpus outgrows single-file storage, which is implausible for one operator's saved material. |
| 2026-07-27 | D-021 — Proto and Connect-RPC for all contracts. | The predecessor used gorilla/mux with hand-rolled REST handlers. | Contracts are proto-owned; the CLI is a thin wrapper over the API rather than a parallel implementation, so CLI and API cannot drift. | None; this is fleet doctrine. |
| 2026-07-27 | D-022 — Search indexes every signal regardless of category, confidence, or disposition. | The tempting optimization is to index only signals in categories the project cares about. | A category assignment that later proves wrong does not cost the signal its retrievability. Since a source may be gone by the time the error is noticed, re-indexing may be impossible — so the cheap decision is unrecoverable and the expensive one is not. Classification is a display and routing device, never a storage device. | None. |
| 2026-07-27 | D-023 — Federation is a declarative descriptor, not router code. | `search-hub` registers providers through `.vrooli/search.json`; the router holds no corpus content and no vectors. | Registration adds a registry row and changes nothing in search-hub. The descriptor also makes the scenario search-applicable to fleet scan and the test-genie `search` phase, which requires a minimum eval corpus to certify. | None. |
| 2026-07-27 | D-024 — The ambient view is budgeted and disposition-filtered. | Its cost is paid by every consumer on every heartbeat. | An unbounded ambient view converts a growing corpus into a growing per-run context tax. The budget is independent of any search limit, since the two answer different questions. | Revisit the default budget once real consumption exists; the initial value is unvalidated. |
| 2026-07-27 | D-025 — Domain events carry identifiers only. | Consumers need to react without polling, but a copied signal body would become a second, drifting truth. | Events carry signal id, category, and disposition; consumers resolve content through the query contract. | None. |
| 2026-07-27 | D-026 — Inference routes only through ai-gateway. | `ai-go/search` defaults to a direct provider subprocess, while the scenario integration contract requires ai-gateway to own routing, capacity, and failure policy. The live gateway role inventory exposes `embedding.default` and `classify.routing`. | `internal/inference.Client` is the sole scenario boundary; its `Embedder` adapter preserves separate `search_document:` and `search_query:` instructions. Domains receive this client through composition and may not call provider resources directly. | Only if ai-gateway retires either role; replace the gateway role policy, never with a direct provider call. |
| 2026-07-27 | D-027 — Post-capture extraction is an append-only enrichment sidecar. | A URL or image must be journaled before a slow or unavailable extractor runs, but updating its `signal` row afterwards would violate D-006. | `signal_enrichment` records each extraction attempt and its readable content or attention outcome. Signal reads compose the immutable capture row with the latest enrichment record; no extraction path updates or deletes `signal`. | Revisit only if a measured query cost requires a materialized read model; that model must be rebuildable from these append-only records. |

## Withdrawn Positions

Recorded because each was argued and lost, and re-arguing them wastes a future
session.

| Position | Why it was withdrawn |
|---|---|
| Rewrite the predecessor in place rather than regenerating. | The objection was that relocating a tracked folder makes the diff unreviewable. That objection only holds when the destination path is unchanged — D-003 renames the scenario, so a fresh directory is created either way and regeneration is strictly cleaner. |
| Model relevance as its own axis (`alpha` / `personal` / `noise`) beside category. | Alpha-centric framing. Under D-002 relevance collapses into the category set, where `uncategorized` is the noise bucket. A separate axis would have added a dimension that only the alpha consumer used. |
| Keep the predecessor's PRD and edit it. | Its operational targets describe a multi-tenant consumer product with an action-approval engine and recipe/workout integrations. The goals were wrong, not the details, and editing would have preserved the framing that made them wrong. |
| Carry the predecessor's action-suggestion engine forward. | It existed to push bookmarks into other scenarios. Under D-013 and D-014 the "action" is routing plus an outcome link, which the intake pipeline already performs — the reframe made a whole subsystem redundant. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| 2026-07-27 | Use the generated `react-vite` documentation contract as shipped. | Retained, with scenario-specific content authored over the stubs. | The template contract is kept; only its placeholder content is replaced. Recorded so the generated row is not mistaken for an unmade decision. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — domain boundaries
- [`../concepts/DATA.md`](../concepts/DATA.md) — storage model these invariants constrain
- [`../reference/conversation-extraction.md`](../reference/conversation-extraction.md) — the one artifact carried from the predecessor
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
