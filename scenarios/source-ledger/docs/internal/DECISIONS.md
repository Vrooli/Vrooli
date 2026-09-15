# Decisions — Source Ledger

| ID | Decision | Rationale | Revisit trigger |
|---|---|---|---|
| SL-D-001 | Keep one journal database as the corpus authority. | Multiple copies can diverge and make append-only guarantees unverifiable. | A deployment requires a separately justified shard boundary. |
| SL-D-002 | Keep compaction and summaries rebuildable. | Storage integrity must not depend on an inference provider. | A durable resumable compaction queue replaces the cache model. |
| SL-D-003 | Carry scope as an explicit request field. | One process can serve many ledgers without an engine rewrite or hidden deployment state. | A scope needs access isolation that unified read cannot honor. |
| SL-D-004 | Keep facet vocabulary and retention policy in data. | Teams and investigations need different ledgers without engine-specific labels. | A policy language requires a new governed contract. |
| SL-D-005 | Resolve the consumer endpoint once at startup. | Per-request discovery would add process-spawn latency to recall. | Discovery becomes a local in-process API with measured lower cost. |
| SL-D-006 | Expose CLI, Connect/API, UI, and search-hub surfaces. | The ledger is both a reusable programmatic capability and an operator product. | A consumer proves one surface is redundant. |
| SL-D-007 | Do not move harness adapters into this scenario. | Harness stores and projection semantics are consumer-specific. | A non-harness ledger needs the same adapter contract. |
| SL-D-008 | Use one generated service contract per moving engine domain: journal, recall, forest, facets, and scopes. | Domain-local contracts keep transport ownership aligned with the domain map and let Connect clients be generated without route mirrors. | A cross-domain transaction requires a deliberately versioned aggregate contract. |
| SL-D-009 | Treat the projection as the degraded-mode fallback and make live recall fail with a typed unavailable error when source-ledger is down. | A local read replica would create a second corpus authority and allow silent divergence. | Measured availability or recovery requirements make the single-authority tradeoff unacceptable. |
| SL-D-010 | Set the initial post-extraction ceilings at 1.20 s warm p95 for wake and 1.50 s warm p95 for recall. | The current managed `vrooli-memory` warm p95 is 0.93 s for wake and 1.16 s for recall; the allowance covers one cached discovery/network hop without hiding a material regression. | A production corpus or measured transport path invalidates the allowance. |
| SL-D-011 | Keep the original `vrooli-memory` engine copies during the pure move. | Side-by-side execution and rollback remain possible until corpus migration and consumer cutover are separately validated. | Phase 16 proves cutover and establishes the retirement policy. |
| SL-D-012 | Keep native hook commands in the memory CLI manifest but filter them from the Connect binding loader. | Hook capture and hook installation are real Go-native commands, not RPCs; declaring them preserves discovery while the loader must continue to require typed bindings for RPC commands. | The hooks become a typed service surface or are retired. |
| SL-D-013 | Make source-ledger the sole engine authority and have `vrooli-memory` use generated Connect clients for all engine domains. | A single authority prevents corpus divergence; the consumer retains only harness-specific import/projection state and can expose typed degraded behavior when the authority is unavailable. | A future deployment requires deliberately justified replication or sharding. |
| SL-D-014 | Make source-ledger the sole Search Hub federation owner, with one scope-fixed provider per policy-registry scope. | Provider descriptors are part of the ledger boundary, not the harness consumer. A source-owned descriptor and boot/create registration path removes duplicate memory providers and makes the answering scope visible in every federated result. | Revisit only if federation is extracted into a separately governed fleet registry. |
| SL-D-015 | Keep reviewable file defaults and sparse per-scope database overrides for bounded context policy. | Operators need a supported no-restart lever while fresh deployments still need a versioned default. Readback reports the winning layer for every key. | Revisit if all scopes converge permanently or policy becomes centrally governed outside Source Ledger. |
| SL-D-016 | Enforce line and character ceilings at both per-entry and whole-view levels, with pins ordered first but budgeted. | Line-only accounting does not bind on single-line JSON; exempt pins can make the ambient view unbounded. | Revisit when a shared model tokenizer becomes an approved local capability. |
| SL-D-017 | Carry wake overflow and refusal counters through prompt-manager and render an actionable truncation notice. | A producer-side signal that no consumer reads cannot protect prompt quality; the heartbeat must disclose incomplete context without raising the budget. | Revisit if heartbeat context becomes a typed retrieval tool rather than ambient prompt input. |
| SL-D-018 | Expose compaction liveness alongside policy readback instead of changing compaction in the hardening pass. | Operators first need count, oldest leaf, and last summary evidence to distinguish stale frontiers from policy sizing problems. | Revisit when a measured stale-scope policy justifies automatic compaction remediation. |
| SL-D-019 | Mutate facet retention and residency through one transactional, scope-checked Connect operation shared by the operator UI. | Vocabulary is data, but direct database edits would bypass validation, authorization boundaries, and readback. The mutation changes future policy without rewriting append-only journal history. | Revisit if a governed CLI or fleet policy service becomes the authoritative vocabulary administrator. |

## Architecture Maturity

These decisions establish the contract-first extraction boundary. They do not
claim that the ledger engine has moved or that a production corpus is present.

## Contracts And Data Flow

The decisions constrain the future wire contract: scope is explicit, authority
is singular, and harness-specific behavior stays at the consumer boundary.

## Purpose Of This Document

This is the durable decision log for the Source Ledger extraction boundary.

## Decision Log

`SL-D-001` through `SL-D-007` are active decisions for the contract phase.

## Superseded Decisions

No Source Ledger decisions are superseded yet.

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
- [`PROGRESS.md`](PROGRESS.md)
- [`PROBLEMS.md`](PROBLEMS.md)
