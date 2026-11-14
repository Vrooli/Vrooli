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
| Owner-backed convergence campaign (Shape C) | The operator explicitly authorizes a continuous improvement loop, the scenario forbids Plan Manager plans, and an existing owner declaration/controller already holds the boundary, evidence, qualification and recovery state. | The scenario target and improve skill, a bounded active packet, an immutable Agent Manager workflow/declaration and a harness goal. No Plan Manager plan, worker-authored controller or second scheduler. |
| Bounded task | The work fits one session and its route is recoverable from the docs and code. Plan authoring and docs-first authoring are themselves bounded tasks. | A harness goal over the docs. No plan. |
| Investigation or review | The deliverable is a report or a verdict, not a change. | A read-only harness goal. Never a plan. |

The first two shapes are plan shapes: both are Plan Manager plans, both are
granted through a Swarm item, and either runs in sliced or goal execution mode.
Shape C is the existing owner-backed convergence exception; it is not a reason
to add a new scheduler or to let Markdown become authoritative.
[Scenario development](SCENARIO_DEVELOPMENT.md#grant-plan-shape-and-execution-mode)
owns the shape and mode definitions; `implementation-plan-authoring` authors a
phased plan and `adaptive-mandate-authoring` authors a mandate.

### Shape selection

Use this decision order for each proposed unit of work:

1. Use a discovered Action when the operation is mechanical, reversible and
   completely specified by the Action contract.
2. If the intended design, acceptance criteria or owner is missing from the
   canonical docs, make the first assignment a bounded docs-first task.
3. Use owner-backed convergence when the operator has explicitly selected that
   shape, the scenario's target and improve skill are durable, and the existing
   Agent Manager declaration/controller owns boundaries, counters, recovery,
   qualification and evidence rotation. If any of those are missing, repair the
   owner or choose a plan-backed shape; do not improvise a controller in prose.
4. Use investigation/review when the required output is a report or verdict;
   it is read-only and never needs a plan.
5. Use a bounded task when one session can complete the work and the route is
   recoverable from the docs and code.
6. Use an adaptive mandate when the target is documented, an improve skill and
   setpoint sensor exist, and the route should be selected from measured
   evidence rather than prescribed in advance.
7. Use a phased plan when route, phase order, handoffs or a stable sequential
   change boundary are part of the requirement.
8. Create a plan family only when the Plan Manager family decision applies:
   two or more independently executable plan-shaped units have explicit
   dependencies or resource overlap, and those units need independent
   identities or lifecycles. A family is not required merely because the
   effort is large.

The first two plan shapes are Plan Manager plans because Swarm uses the
accepted plan as a reviewed change boundary and grant. That does not require a
plan for every worker, every optimization experiment or every report. A large
effort may intentionally contain a phased plan, an adaptive mandate, bounded
tasks and investigations at the same time. The parent effort maps each
requirement to its actual owner reference and acceptance evidence.

When the target documentation does not describe the intended design, the first
assignment is to write it there; every later shape points at the docs instead of
restating them. A Swarm development grant binds one canonical plan only when
the selected work shape is plan-backed. The requirement is about the grant,
not about every sub-assignment inside it. The text of a harness goal is owned
by the `harness-goal-authoring` skill.

### Dispatch and fallback

Use Swarm when the work needs its typed grant, disposition, autonomous
continuation or cross-boundary ownership. Distinguish a missing human
interaction from missing authority: if Swarm's transitional implementation
requires human approval solely because it cannot yet represent an already
authorized assignment autonomously, use a qualified Agent Manager or harness
route within that assignment's fixed scope, acceptance and budget. Record that
the Swarm grant/receipt was unavailable and which guarantees were reduced; do
not fabricate a Swarm approval. If the effect genuinely requires an owner grant
(new scope, production access, paid spend, private data or an unapproved
decision), pause that effect.

Use the declared Agent Manager workflow/family route when the effort already
has the required authority and Swarm is not the execution owner. If both are
unavailable, use a qualified control-plane launcher only when the same scope,
acceptance, budget, attribution and recovery guarantees can be retained. A
direct harness is a last resort and must record the guarantees it loses. Never
replace an owner with a local ledger or an untyped launcher.

## Lifecycle

A backlog item's status is one of the values Swarm Manager defines in
`path:scenarios/swarm-manager/api/internal/backlogstatus/statuses.go`, the
single source of truth:

| Phase | Statuses |
|---|---|
| intake | `suggested` |
| planning | `backlog`, `researching`, `ready` |
| in flight | `queued`, `in_progress` |
| review | `in_review`, `review_pending` |
| terminal | `completed`, `failed`, `needs_followup`, `dropped` |

There is no `accepted` status. Query with
`swarm-manager backlog list --status <status>[,<status>] --json`; a member
command that filters on `accepted` (or on a team positional or `--context`
flag, which the command does not take) returns nothing. Replaced or no longer
useful work is `dropped`.

Team-filed items land in `backlog`. Today the consumer of stewardship-team
items is an operator decision: the operator moves an item to `ready`, queues
it, drops it, or decides its review (`swarm-manager backlog review-decide`).
No autonomous executor drains stewardship-team items. A disposition records
the actor, evidence, scope, and outcome in the Swarm Manager stream. Prompt
Manager members read the disposition through their next bounded wake; they do
not poll a local queue.

## Capability gaps and reviews

For operational oversight across ongoing work, use
[effort supervision](EFFORT_SUPERVISION.md). Its standing observer discovers
new efforts; Swarm retains its grants and disposition. In-scope supervisor
interventions use existing authority, and new targets still require amendment.

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
