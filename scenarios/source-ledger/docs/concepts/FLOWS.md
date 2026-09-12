# Flows — Source Ledger

## Append

1. A client sends an entry with an explicit scope and provenance.
2. The journal validates identity, writes the entry, and advances its
   high-water mark in one transaction.
3. Enrichment may classify, derive facet text, embed, and queue failures.
4. The response reports durable append success independently of enrichment.

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

1. Source Ledger reads the policy registry during boot and materializes one
   descriptor per existing scope.
2. Federation derives a stable provider ID from the scope ID
   (`source-ledger.agent-memory` for the fleet memory scope and
   `source-ledger.scope.<scope-id>` otherwise).
3. Search Hub self-registration upserts every descriptor; creating a scope
   repeats the descriptor and registration steps without restarting the API.
4. A federated query returns scope-labelled results with no legacy
   `vrooli-memory` provider or duplicate ledger hit.

## Architecture Maturity

These flows define the intended service boundary and failure semantics. They
are executable only after the service contract and moved engine are present.

## Contracts And Data Flow

The flows preserve append-only authority while derived enrichment, compaction,
and federation consume scope-filtered records.

## Purpose Of This Document

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
