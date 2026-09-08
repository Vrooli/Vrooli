# Team corpus adoption contract

Source Ledger is the sole durable corpus authority for prompt-manager team
memory. This document defines the consumer contract for the six prompt-manager
teams. It does not implement or duplicate the ledger engine.

## Vocabulary

Consumers use the existing Source Ledger vocabulary:

- `scope`: one named corpus partition with its own facets and budgets;
- `journal`: append-only source entries and provenance;
- `wake`: a bounded ambient block selected for the next heartbeat;
- `recall`: scoped semantic retrieval over journal and derived nodes;
- `frontier`: the rebuildable compaction boundary used to keep wake selection
  bounded.

The consumer must not introduce `knowledge`, `handoff`, `decision`, or another
parallel append-only vocabulary as a ledger API concept. Those words may occur
inside an entry body when they describe historical source material.

### Team facets

Every `team:*` scope carries the same six semantic facets. Facet IDs are
scope-prefixed in storage (`prompt-manager-<team>-<facet>`) because the current
schema uses a global facet-definition key.

| Facet | Holds | Retention | Compaction | Residency |
|---|---|---|---|---:|
| `standing-lesson` | Durable lessons and rules that should survive recency | `pinned-or-review` | no | 6 |
| `decision` | Accepted decisions with evidence and tradeoffs | `retain` | no | 6 |
| `episode` | Completed runs, scans, snapshots, and work outcomes | `compact` | yes | 8 |
| `handoff` | Member handoff snapshots | `expire-on-resolution` | yes | 4 |
| `thread` | Open or unresolved lines of work | `expire-on-resolution` | no | 4 |
| `rehearsal` | Test-channel artifacts whose kind ends in `-test` | `retain` | no | 0 |

Residency is an ambient-wake entry ceiling. The line and character ceilings
remain scope-wide. A zero-residency rehearsal remains in the append-only
journal for audit and recall but never enters ambient wake.

### Deterministic classification

Team classification is owned by the `classification_rules` table. A rule
selects either an exact `kind` or a `kind_glob`; a rule may not provide both.
Exact kinds and glob patterns are evaluated in ascending priority and then by
rule ID. `kind_glob` uses Go `path.Match` semantics, so slash-containing kinds
need explicit one-level or deep-path rules. The classifier remains a reported
fallback for kinds with no rule; `source-ledger rules measure-distribution`
shows that tail.

Operators create rules disabled, run `dry-run`, inspect the match count, then
`enable` and `refacet` the scope. This makes classification changes reversible
until a forest summary is written.

### The write vocabulary decides the facet spread

Rules can only separate what the writer distinguishes. A rule keys on `kind`, so
a team that writes every entry under one generic kind gets one facet no matter
how many rules exist, and a team that writes semantic kinds gets a spread. This
is observable across the fleet: `marketing-crew` writes `marketing-handoff`,
`marketing-decision` and `marketing-knowledge` and lands across four facets,
while teams writing only `team-knowledge` land 82% of their corpus in `episode`.

Team members therefore choose a kind by what the entry *is*. The generated
Knowledge Log block in every heartbeat prompt offers four:

| Kind | Facet | Use for |
|---|---|---|
| `team-standing-lesson` | `standing-lesson` | a durable rule or invariant that should outlive this run |
| `team-decision` | `decision` | an accepted decision with its evidence and stated tradeoff |
| `team-thread` | `thread` | an open, unresolved line of work or follow-up |
| `team-knowledge` | `episode` | a completed run, scan, snapshot, or work outcome — the default |

`team-decision` is matched by the `*decision*` glob; the other three carry
explicit exact-kind rules. Concentration in `episode` is not by itself a defect:
scans, snapshots and completed runs *are* episodes under this vocabulary. It is
a defect only when durable lessons and accepted decisions are being written and
are indistinguishable from them.

Rules are created per scope through the CLI, not seeded from scenario code. A
newly registered team scope receives the facet vocabulary from
`prompt-manager`'s `TeamScopeFacets` but **no classification rules**, so its
entries reach the LLM classifier until an operator adds them. Check a new scope
with `source-ledger rules list --scope team:<id>`.

### Rehearsal contract

Kinds ending in `-test` route to the scope's `rehearsal` facet. Rehearsal data is
retained in the journal and its current assignment is visible to curation, but
the facet's residency is zero, so it is never emitted by Wake. This rule is a
ledger policy, not a consumer-side filter.

### Compaction schedule

`vrooli-memory` runs the canopy every six hours over **every registered scope**,
not a named subset. The default bounded pass limit is two merges, applied
independently to each scope. The per-scope result is persisted in
`maintenance_compactions`; a failed or stalled scope is therefore visible
without averaging it into the fleet result.

The pass covers every scope because the registry is the list, and restricting it
to a prefix silently strands whatever falls outside — an earlier `team:`-only
filter dropped `agent-memory`, the largest corpus in the system. Breadth is
affordable because compaction is a no-op at target: a scope whose eligible
frontier is already at or below its `frontier_target` performs zero merges and
calls no provider, and most non-team scopes declare no compaction-eligible facet
at all. Cost tracks the backlog, not the scope count.

Operators can inspect a scope with:

```bash
source-ledger policy show --scope team:director-swarm --json
source-ledger forest frontier --scope team:director-swarm --json
```

Read `liveness.last_summary_at` and the eligible frontier together. For a
configured scope, `liveness.unsummarized_leaf_count` uses the same
compaction-eligible, unpinned, aged, vector-backed root gates as the forest, so
it can be compared directly with `frontier_target`. Recent entries and
non-eligible durable facets remain in the journal but are intentionally absent
from this compaction pressure signal. Compaction never deletes journal rows.

`last_summary_at` is a receipt for a summary-producing compaction. It can be
absent when a scope is already at or below its target and the bounded pass
correctly creates no summary. In that case, use the per-scope maintenance
receipt (`maintenance_compactions`, surfaced by the maintenance-status
workflow) together with the frontier; do not manufacture a summary or a
timestamp merely to fill the field.

## Team scope contract

Each team maps to one stable scope ID:

| Team | Scope ID |
|---|---|
| director-swarm | `team:director-swarm` |
| infra-health | `team:infra-health` |
| marketing-crew | `team:marketing-crew` |
| meta-optimization | `team:meta-optimization` |
| monetization | `team:monetization` |
| scenario-qa | `team:scenario-qa` |

Scope creation is idempotent. A member start must create the missing scope
with the team's facet vocabulary and budgets, then reuse that scope on later
starts. A start must fail with a typed `source-ledger unavailable` error when
the service cannot register or verify the scope. It must not create a local
memory file as a fallback.

## Append and provenance

The member that observes a fact appends the prose entry to its team scope. The
entry carries the verified actor/profile identity, source runtime, timestamp,
and any run or workflow correlation available at the request boundary. A
heartbeat attempt and a task board item are operational records, not team
corpus content; they remain in their owning event/task systems and are never
appended to a team scope.

## Wake and frontier rule

The scope's wake budget is the hard upper bound for one heartbeat's ambient
block. The frontier target must be no larger than the number of entries that
can be compacted and scored within that budget. Compaction may replace derived
summaries and edges, but it must never delete journal rows. A wake response must
identify its scope and remain bounded even when the journal grows.

Wake can emit only facets with resident, currently eligible entries. A
resident-budget-zero facet, or an occupied facet whose entries are not in the
bounded wake result, is not represented in every individual wake. Fleet-level
facet coverage is therefore measured as the union of the team wakes; no dummy
entries are written to make an empty facet appear in a wake.

## Recall

Every recall request carries the team scope explicitly. The service filters by
scope before scoring or rendering results. A member may use recall to inspect
its own durable context and the bounded wake block to orient its next run; it
must not read another team's scope implicitly.

## Surface status

The Connect and API contracts for scopes, journal append, recall, wake, forest,
facets, and classification rules are the authoritative implementation surface.
The governed Source Ledger CLI exposes those operations with an explicit
`--scope` and does not create a local fallback or a second team-memory store.
