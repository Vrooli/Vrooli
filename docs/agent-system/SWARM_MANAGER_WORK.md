# Swarm Manager Work

Swarm Manager is the single operator-visible work and disposition surface for
agent teams. Prompt Manager supplies team contracts, member instructions, and
bounded wake context; it does not own a second approval queue or a parallel
work ledger.

## Routing

Agents record durable observations and requests through the shared team corpus
owned by Source Ledger. When an observation needs implementation, judgment, or
operator disposition outside already-authorized work, the agent creates a Swarm Manager backlog item or capture
with evidence, scope, provenance, and the requested next action. The item is
then visible in the unified decision stream.

Mechanical, reversible work may be executed through a discovered Action when
the Action's contract completely specifies the operation. Work that changes
canon, crosses team boundaries, spends meaningful resources, or needs an
operator choice remains in Swarm Manager until it is dispositioned. A disposition
may authorize a development mandate, not just one predicted code change. Within
that mandate, record successive repairs under the existing work item instead of
creating another item for each decision. New targets or grants require amendment;
implementation choices inside the grant do not. The authority and artifact rules
live in [Scenario Development Under a Grant](SCENARIO_DEVELOPMENT.md).

## Work shapes

Work reaches an agent in one of four shapes. Choose the cheapest shape that
survives the work's real durability and authority needs. A plan is one shape,
not the default.

| Shape | Use when | Carries |
|---|---|---|
| Phased plan (plan shape `phased`) | The route or phase order is the requirement, several sessions or agents will touch the work, or Swarm must review and grant it. | A canonical Plan Manager plan with ordered phases and a harness goal that points at it. |
| Adaptive mandate (plan shape `mandate`) | The work is large or its architecture is not yet clear, and the scenario has a documented target with an improve skill and sensors. | A canonical Plan Manager plan holding the target pointer, sensors, bands, scope, stop rules and arc; a development grant; a goal whose finish line is the setpoint board. |
| Bounded task | The work fits one session and its route is recoverable from the docs and code. Plan authoring and docs-first authoring are themselves bounded tasks. | A harness goal over the docs. No plan. |
| Investigation or review | The deliverable is a report or a verdict, not a change. | A read-only harness goal. Never a plan. |

The first two shapes are plan shapes: both are Plan Manager plans, both are
granted through a Swarm item, and either runs in sliced or goal execution mode.
[Scenario development](SCENARIO_DEVELOPMENT.md#grant-plan-shape-and-execution-mode)
owns the shape and mode definitions; `implementation-plan-authoring` authors a
phased plan and `adaptive-mandate-authoring` authors a mandate.

Two rules follow. When the target documentation does not describe the intended
design, the first assignment is to write it there; every later shape points at
the docs instead of restating them. A Swarm development grant binds one
canonical plan because the plan is the reviewed change boundary and acceptance
contract; that requirement is about the grant, not about every sub-assignment
inside it. The text of a harness goal is owned by the `harness-goal-authoring`
skill.

## Lifecycle

The durable lifecycle is `proposed → triaged → accepted|rejected → executed →
verified`, with `superseded` and `stale` available when the underlying work is
replaced or no longer useful. A disposition records the actor, evidence, scope,
and outcome in the Swarm Manager stream. Prompt Manager members read the
disposition through their next bounded wake; they do not poll a local queue.

## Capability gaps and reviews

A capability gap means a required scenario, CLI operation, Action, or source is
absent. When its repair is within accepted work, record and repair it under that
work's authority. Otherwise request disposition through Swarm Manager with the
affected work and evidence. Absence alone does not prove an authority blocker.
A review report is evidence about
the quality or safety of proposed work; it is not an approval queue. The
operator resolves both through the same Swarm Manager disposition surface.

## Operator contract

Every team-facing request must have one discoverable owner, one durable
provenance chain, and one visible disposition destination. New team workflows
must use the Source Ledger team scope for durable corpus and Swarm Manager for
work requests. No Prompt Manager-specific decision files, decision routes, or
member-local approval ledgers may be introduced.
