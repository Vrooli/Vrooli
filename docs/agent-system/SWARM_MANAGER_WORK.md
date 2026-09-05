# Swarm Manager Work

Swarm Manager is the single operator-visible work and disposition surface for
agent teams. Prompt Manager supplies team contracts, member instructions, and
bounded wake context; it does not own a second approval queue or a parallel
work ledger.

## Routing

Agents record durable observations and requests through the shared team corpus
owned by Source Ledger. When an observation needs implementation, judgment, or
operator disposition, the agent creates a Swarm Manager backlog item or capture
with evidence, scope, provenance, and the requested next action. The item is
then visible in the unified decision stream.

Mechanical, reversible work may be executed through a discovered Action when
the Action's contract completely specifies the operation. Work that changes
canon, crosses team boundaries, spends meaningful resources, or needs an
operator choice remains in Swarm Manager until it is dispositioned.

## Lifecycle

The durable lifecycle is `proposed → triaged → accepted|rejected → executed →
verified`, with `superseded` and `stale` available when the underlying work is
replaced or no longer useful. A disposition records the actor, evidence, scope,
and outcome in the Swarm Manager stream. Prompt Manager members read the
disposition through their next bounded wake; they do not poll a local queue.

## Capability gaps and reviews

A capability gap means an existing member is blocked because a required
scenario, CLI operation, Action, or source is absent. File that gap in Swarm
Manager with the blocked work and evidence. A review report is evidence about
the quality or safety of proposed work; it is not an approval queue. The
operator resolves both through the same Swarm Manager disposition surface.

## Operator contract

Every team-facing request must have one discoverable owner, one durable
provenance chain, and one visible disposition destination. New team workflows
must use the Source Ledger team scope for durable corpus and Swarm Manager for
work requests. No Prompt Manager-specific decision files, decision routes, or
member-local approval ledgers may be introduced.
