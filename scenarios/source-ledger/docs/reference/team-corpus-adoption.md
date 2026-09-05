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

## Recall

Every recall request carries the team scope explicitly. The service filters by
scope before scoring or rendering results. A member may use recall to inspect
its own durable context and the bounded wake block to orient its next run; it
must not read another team's scope implicitly.

## Surface status and gap

The Connect and API contracts for scopes, journal append, recall, and wake are
the authoritative implementation surface. As of 2026-08-08, the installed
Source Ledger CLI exposes health status only. The missing governed CLI bindings
for idempotent scope registration, prose append, bounded wake, and scoped
recall are recorded in Swarm Manager capture
`cap-ec8f3c2ee6b5f2ab`; this plan does not create a second CLI vocabulary.
