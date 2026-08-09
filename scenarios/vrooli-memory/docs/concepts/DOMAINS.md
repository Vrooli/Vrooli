# Domains — Vrooli Memory

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The scaffold also ships one clearly fenced worked example domain (never
product scope) as a copyable reference; `template-manager detemplate
<scenario>` removes every fenced example once the real domains are green.

**Status: designed, not implemented.** The domain map below is the output of
a completed design workshop (see [`../internal/DECISIONS.md`](../internal/DECISIONS.md)).
Only `health` exists in code today. Each row states the boundary the
implementation is expected to honour.

The load-bearing split is between the **journal** (immutable truth) and
everything derived from it. `forest` and `recall` may be rebuilt from
`journal` at any time without data loss; `journal` may never be rebuilt
from them. If a change would make a derived domain authoritative for
something, it belongs in `journal` instead.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/vrooli-memory/v1/shared/health.proto` |
| journal | Own the append-only entry log and the single write path. | The one authority for what the fleet remembers; everything else is derived and rebuildable. | Entries, facet texts, embeddings, attribution, correlation. | crud | service | Entry, FacetText, Attribution | `api/internal/journal/` |
| facets | Own the facet taxonomy, classification, retention policy, and pin curation. | Different memory has different decay laws; the facet decides which policy applies. Pins have no automatic relief valve, so their bounding lives here too. | Facet assignments, pin state, pin review and merge proposals, supersession and expiry marks. | policy | service | Facet, Policy, Pin, PinProposal | `api/internal/facets/` |
| forest | Own the frontier, clustering, and compaction passes. | Keep ambient recall inside a fixed budget as the corpus grows without bound. | Summary nodes, parent/child edges, frontier membership. | service | scheduler | Frontier, Cluster, Summary, Span | `api/internal/forest/` |
| recall | Own query, cross-depth ranking, and the wake budget. | Retrieval is the product; compaction only changes what is *ambient*, never what is *findable*. | No data; reads journal and forest. | query | reporting | Hit, Cover, Budget | `api/internal/recall/` |
| federation | Own the search-hub provider descriptor and control surface. | One registry row makes memory reachable from federated query with no router change. | Descriptor config; no memory content. | integration | service | Descriptor, ResultMapping | `api/internal/federation/` |
| harness | Own the memory-file projection, prompt-block install, native-write capture, and store import. | The integration point that makes every agent runtime share one memory. | Projection and import-key state only. | integration | service | Projection, PromptBlock, Capture, Import | `api/internal/harness/` |

## Domain Details

### journal

- Purpose: hold the append-only log of everything an agent deliberately remembers.
- Primary archetype: CRUD (append + read only — no update, no delete).
- Secondary traits: write-path orchestration across facets and ai-gateway.
- Owns: entry records (prose body, facet tag, derived facet texts, embeddings,
  author attribution, receipt correlation, timestamps); the append operation;
  the classification/embedding call sequence at write time.
- Does not own: what a facet *means* (`facets`), summaries (`forest`), ranking
  (`recall`). It stores the facet tag; it does not decide policy from it.
- Invariant: no code path rewrites or deletes an entry. A correction is a new
  entry plus a supersession mark, never an edit.
- Degradation rule: classification or embedding failure must not lose a write.
  The entry is appended unclassified and queued for retry.
- Requirements: `VMEM-P0-001`, `VMEM-P0-002`, `VMEM-P1-002`.

### facets

- Purpose: assign exactly one facet per entry and apply that facet's retention policy.
- Primary archetype: policy / rules.
- Owns: the closed facet set (standing-rule, environment-fact, gotcha, episode,
  thread, entity-record); the policy table mapping facet → retention behaviour;
  pin state; supersession and expiry marks; operator re-facet corrections; the
  pin budget, review dates, and the merge/trade-off proposal queue.
- Does not own: compaction mechanics (`forest`) — it only decides *eligibility*.
  Pin consolidation *borrows* the forest's cohesion scoring but terminates in an
  operator proposal, never a collapse (D-018).
- Invariant: the facet set is closed. An unrecognised facet is a hard error at
  write, never a silent default; a default would route an entry to the wrong
  decay law without anyone noticing.
- Where the set lives: seeded **data**, not Go constants (D-019). Closed means
  every write is validated against a known set, not that the set is fixed at
  compile time. This costs nothing now and is the cheap insurance behind the
  deferred source-ledger idea in
  [`ARCHITECTURE.md`](ARCHITECTURE.md) § Deliberately Not Built.
- Highest-consequence error in the scenario: a standing rule misclassified as an
  episode becomes compaction-eligible and can vanish from ambient context. This
  is why operator re-facet is a first-class action rather than an admin tool.
- Requirements: `VMEM-P0-005`, `VMEM-P0-006`.

### forest

- Purpose: keep the compaction-eligible portion of the frontier under its
  target size by collapsing the cheapest available cluster.
- Primary archetype: service with a scheduled pass.
- Owns: summary nodes and their spans, parent/child edges, frontier membership,
  cluster scoring, and the compaction pass itself.
- Does not own: the summarization prompt's product wording (shared with
  `facets` policy) or leaf storage (`journal`).
- Invariant: compaction is a context-budget device, never a storage device.
  A pass adds summary nodes; it never removes a leaf.
- Frontier = the antichain of roots: every journal leaf not yet inside a summary
  plus every summary not yet inside a summary. It mixes depths freely, so a
  fresh leaf can cluster with a depth-2 summary.
- Compaction pressure counts only roots whose current facet policy is eligible
  (currently `episode`). Non-episode roots remain in the mixed frontier for
  full-fidelity recall; counting them against a target they are forbidden to
  satisfy would make the pressure loop impossible.
- Requirements: `VMEM-P0-007`, `VMEM-P1-004`, `VMEM-P2-003`.

### recall

- Purpose: answer queries across every depth and render the budgeted ambient view.
- Primary archetype: query / reporting.
- Owns: cross-depth ranking, descendant collapse, the `cover` budget selection
  for `wake`, and `zoom` descent.
- Does not own: any data. It reads `journal` and `forest` and stores nothing.
- Invariant: compaction status never excludes a leaf from the candidate set.
- Requirements: `VMEM-P0-003`, `VMEM-P0-004`, `VMEM-P0-008`,
  `VMEM-P1-003`.

### federation

- Purpose: document the consumer boundary; source-ledger owns federated
  provider registration and serves the control surface.
- Primary archetype: integration.
- Owns: no provider descriptor. Search Hub metadata, per-scope descriptors,
  boot registration, and scope-creation registration belong to source-ledger.
- Does not own: routing or Search Control operations. The router holds no
  memory content and no vectors; it routes on source-ledger metadata.
- Requirements: `VMEM-P0-009`, `VMEM-P2-004`.

### harness

- Purpose: make every agent runtime read and write the same memory.
- Primary archetype: integration.
- Owns: the generated memory-file projection, the idempotent prompt-block
  installer, native-write capture (hook and store-diff channels), the
  declarative import adapters and their content-addressed keys, and the
  run-correlation lookup that lists sibling events.
- Does not own: agent identity or run data. Correlation ids are stored; run
  payloads are never copied — `vrooli-events` stays the one truth about a run.
- Does not own: coding-agent install, update, or permissions. Those belong to
  `resources/<agent>/`. This domain **extends** that resource with projection,
  prompt-block, and hook install; it never edits an agent binary.
- Invariant: the projection is one-directional. The projected file is never read
  back as a memory source, so there is no bidirectional sync to conflict.
  Captured native writes are a *separate* input path — they read the harness's
  own store, never the projection this scenario wrote.
- Invariant: the prompt block marks the generated wake block read-only,
  prefers the harness-native memory tool, and names `vrooli-memory journal note` only
  as the fallback when the runtime has no native write surface (D-041).
- Invariant: import is idempotent by content hash, so a sweep may run at any
  frequency without duplicating (D-016).
- Requirements: `VMEM-P0-010`, `VMEM-P1-002`, `VMEM-P1-007`,
  `VMEM-P1-008`, `VMEM-P0-011`, `VMEM-P2-002`.

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `distillation` — propose memories from run receipts | The deliberate write verb is the primary path because it reaches every harness; distillation only reaches agent-manager-spawned agents. Signal-to-noise over the receipt stream is an open empirical question. | `VMEM-P2-001`: once the P0 loop has real data, run the experiment on the same substrate. |
| `eval` — golden retrieval suites for memory | Requires a real corpus. A generated-only corpus cannot certify under the search-hub provider contract. | `VMEM-P2-004`: once there is enough reviewed memory content to author positives and junk negatives. |

## Explicitly Not Domains (Decided)

| Considered | Verdict | Reason |
|---|---|---|
| A separate `standing` store for rules and facts | **Rejected.** | An earlier design had two derived structures — a resolved key/value state store beside the episodic tree. It collapsed to a *pin flag plus policy* on one journal once frontier clustering was shown to co-locate contradictions. One storage, one tree, one flag. |
| A `records` domain mirroring swarm-manager | **Rejected.** | Work records are a *kind* of memory, not a separate context. They enter through the same write path with a required field set (`VMEM-P1-001`). |
| A `scope` or `tenant` domain for access control | **Rejected.** | Unified read across all scenarios is the product. Partitioning memory for privacy would undercut the reason to build it. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
