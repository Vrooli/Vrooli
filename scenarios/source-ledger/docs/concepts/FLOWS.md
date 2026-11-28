# Flows — Source Ledger

## Append

1. A client sends an entry with an explicit scope and provenance.
2. The journal validates identity, writes the entry, and advances its
   high-water mark in one transaction.
3. Enrichment may classify, derive facet text, embed, and queue failures.
4. The response reports durable append success independently of enrichment.

The file-backed Journal Connect recovery test submits the same scoped Walk
briefing while embedding is unavailable, confirms its retry is deferred, then
closes the handler and SQLite pools. A new handler over the same database
returns the identical entry ID and body; after embedding recovers, the retry
completes and replaying the same request key still returns that one entry.
This proves the SQLite file-backed owner path across a test-simulated restart.
It does not prove production process orchestration, another database driver,
or an ambiguous append response after commit.

The opt-in cross-product test runs Command Center's current Walk API source
against this production Journal handler over Connect and the same SQLite owner.
Its no-base-URL row injects a deterministic api-core resolver failure, selects
the local owner through `SOURCE_LEDGER_API_PORT`, injects metric-provider loss
through the user-facing route, reconstructs the Journal handler over the same
file, then verifies recovery and exact request-key replay. This verifies the
port fallback without consulting shared peer/registry/CLI state or restarting
shared services; the supervisor-managed process and served Command Center build
remain outside this test.

## Embedding and missing-vector recovery

1. Each `GatewayClient.Embed` invocation makes at most one `ExecuteRoute` RPC.
2. An unusable response or transport error ends that invocation without
   resubmission; it does not prove that provider work did not complete.
3. Journal append persists an `embed` retry record for missing derived vectors.
4. A later `ProcessEmbeddingRetries` pass serializes queue processing, invokes
   `Embed` for missing vectors, and stores successful vectors before
   acknowledging the retry record. Journal prose and facet-text rows remain
   immutable.

The Journal queue has no durable provider receipt or unknown-outcome
reconciliation. Later invocations can repeat completed provider work after a
lost response. One RPC per invocation does not provide exactly-once execution
across queue attempts or processes; durable reconciliation remains unresolved.

Sources: `path:scenarios/source-ledger/api/internal/inference/client.go` and
`path:scenarios/source-ledger/api/internal/journal/service.go`.

## Recall and Wake

1. The request boundary normalizes the scope once.
2. Policy resolves the scope budgets and vocabulary.
3. Recall filters journal and derived nodes by scope before scoring.
4. Wake selects pinned and resident items within the scope budget.
5. The response labels the scope and distinguishes source entries from
   rebuildable summaries.

## Compaction

1. Forest reads the eligible frontier for one scope.
2. Policy excludes pinned, protected, and non-compactable nodes.
3. The scorer selects the highest-value candidate cluster.
4. Inference summarizes the cluster within a bounded timeout.
5. The forest atomically writes the summary and edges.
6. The journal remains unchanged and the next pass can resume.

## Federation

1. Source Ledger registers the `source-ledger.scopes` descriptor at boot.
2. Search Hub forwards the original query and requested scope to the recall
   engine that also backs Vrooli Memory. Category phrases influence memory
   ranking within that query.
3. Empty scope defaults to agent-memory; a named scope selects that ledger.
4. Creating a scope does not add another provider or duplicate default recall.
5. Search hits carry the node ID and scope for inspecting evidence with zoom.

## Architecture Maturity

These flows define the intended service boundary and failure semantics. They
are executable only after the service contract and moved engine are present.

## Contracts And Data Flow

The flows preserve append-only authority while derived enrichment, compaction,
and federation consume scope-filtered records.


This document describes the intended source-ledger flows and their failure
boundaries.

## Flow Inventory

Append, recall/wake, compaction, and federation are the initial flow families.

## Flow Details

Each flow is scope-aware and preserves the separation between durable source
records and rebuildable derived records.

## State Machines

Append and compaction report explicit durable, queued, failed, and completed
states rather than hiding provider failures.

## Maturity Ladder

The scaffold documents flow intent; executable flow evidence begins after the
service contract and engine move.

## Production Shape

Production wiring will use lifecycle-managed API, CLI, UI, backup, and
federation boundaries.

## Deferred / Unmodeled Flows

Team migration and multi-corpus rollout are deferred to later plan phases.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md)
- [`DATA.md`](DATA.md)
- [`ARCHITECTURE.md`](ARCHITECTURE.md)
