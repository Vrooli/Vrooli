---
name: large-effort-orchestration
description: Deliver a long product goal through one orchestrator and substantial independent worker outcomes. Assess the product against governing requirements and selected designs, maintain an evolving allocation, coordinate parallel implementation, and independently accept results. Use for finite orchestrated delivery; ordinary tasks and phased plans keep their own work models.
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [orchestration, multi-agent, effort, epochs, continuity, recovery]
  icon: network
  status: active
  revision: 59
  createdAt: "2026-09-10T00:00:00Z"
  updatedAt: "2026-10-24T00:00:00Z"
  requires:
    scenarios: [prompt-manager, agent-manager]
    commands: [prompt-manager skill read, agent-manager]
  origin:
    kind: authored
---

## Practice focus: Large effort orchestration

Deliver the whole product destination in the charter. Own the technical strategy,
work allocation and independent acceptance. Workers implement substantial outcomes;
the orchestrator keeps the overall product moving toward completion.

Use this practice for long delivery goals. Phased plans, bounded tasks and direct
requests keep their own work models. This is the only home of the epoch standard:
an **epoch** is one coherent worker outcome with its brief and execution record.

Resolve supporting-file links below relative to the repository directory
`scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/`.
Workers receive the full repository-relative worker-card path in their assignment.

### 1. Start from the product

Read the team's `CHARTER.md`, current `STATUS.md`, open `FEEDBACK.md` and the native
`agent-manager effort board --effort-ref <ref>`. Read the relevant accepted results
in `LOG.md` and any retained active brief. Run `agent-manager effort lint <home>`.
Resolve new steering before relying on an old handoff. Acknowledge open feedback
by ID without claiming it resolved. Reconcile retained owners; parked is not terminal.
An uncertain dispatch does not authorize a replacement orchestrator.

Inspect the current product and its governing sources. For visual work, open the
selected original images for the actual screen/state. Compare the served product
at matching theme and viewport. An atlas is an implementation target, not a mood
board. Tests and screenshots alone do not establish visual fidelity.

Build a gap assessment that covers the whole destination:

| Decide | Required result |
| --- | --- |
| What must users be able to do? | Required journeys, states and preserved behavior, including nonvisual requirements |
| What should it look like? | Exact selected originals and justified adaptations |
| What already works? | Current source/served evidence; historical or unavailable evidence stays labelled |
| What makes delivery simpler? | Shared composition, reusable components, needed custom components, API/domain changes and removal of superseded code |
| What can run together? | Independent outcomes with owned files/interfaces and explicit dependencies |
| What remains uncertain? | A bounded investigation, named external return or genuine human decision |

A missing mockup does not erase required behavior. Resolve source conflicts and
human-only decisions explicitly. Keep roadmap features separate from approved scope.
Approved shared-contract work excluded from one child returns to the orchestrator
for another assignment; that exclusion is not a new approval gate for the whole goal.

### 2. Maintain one evolving allocation

Keep the allocation in `STATUS.md`; do not create another plan or ledger for normal
delivery planning. Each outcome states its user benefit, references, dependencies,
owner, acceptance and remaining work. Plan all required surfaces, not only the next
small fix. Reassess this allocation after material inspection, acceptance or feedback.
Prepare independent remaining work while a worker is active; do not repeat unchanged
planning or waits. Keep estimates provisional until current evidence supports them.

Resolve shared shell, theme or API ownership before dependent screen polish. Reuse
suitable shared components through their owner routes. Assign necessary refactors
and API work inside the approved destination. Include focused self-validation and
handoff in the implementation outcome. Remove superseded paths and preserve unique
test protection. Do not split source, tests, captures and receipts into separate
workers solely to create more dispatches. Do not invent deletion quotas for features.

Maintain a `## Forecast`: destination target, current evidenced gap, proposed outcome
order/dependencies, estimates and uncertainty. After acceptance, compare actual result
with the estimate and adjust the remaining forecast. An empty queue with an open gap
means plan more work. Repeated low yield or a queue that cannot reach the destination
needs a strategy change, not more of the same. Detailed sensors belong to the
[worker card](references/worker.md#progress-and-step-back).

### 3. Delegate outcomes with context

Keep one orchestrator. Use the existing selected profiles: the standard orchestrator
is Luna-high and each implementation worker is Luna-medium. Record each run's model;
record a reason before escalating within native profile permissions. The orchestrator
does not implement product code. It can inspect, plan, coordinate and review.

An assignment may span many files and several implementation steps. Size it around
one user-visible outcome or a whole replacement/refactor. A defect-only assignment
is justified when it closes that outcome. Split an oversized outcome at a meaningful
behavior or ownership boundary; retain the full remainder in STATUS.

Give every worker the following in its task description:

```text
Mission: <the team's whole product destination and why this outcome matters>
Outcome: <one substantial user-visible result or replacement>
References: <exact governing sections, selected original image paths and current evidence>
Ownership: <allowed modules/files, shared interfaces, peer ownership and non-goals>
Implementation: <necessary UI/API/refactor work; preserved behavior and known decisions>
Verification: <focused commands, operated journeys and served-original comparisons>
Return: changed, verified, remaining, unverified; exact source/evidence identities
Execution: <brief path, parent ID, worker card, applicable directives and native until>
```

Keep the brief self-contained. Include requirement/source IDs and exact selected
states; do not substitute a broad atlas link for the assigned originals. Include the
[worker card](references/worker.md), not this entire orchestrator skill. The complete
brief is the execution ledger. The worker reads earlier history only when its assigned
requirements depend on it. Use the [brief format and dispatch route](references/execution.md).

Run independent epochs concurrently within the native commission limit. Descendants
count toward that limit. Team heartbeat concurrency is a separate control. Serialize
actual file, lease, host/device or interface conflicts. Do not launch competing workers
on one outcome or a second leader. When a required enabling change crosses a child's
scope, amend through the owner or assign the enabling outcome and its dependency.

### 4. Review, learn and continue

On a worker return, inspect that exact run's result, current source and preserved
baseline. Independently inspect behavior, code ownership, required checks and visual
comparisons. A worker's summary alone is not acceptance. Use existing evidence for
unchanged inputs; rerun only affected, missing, stale or materially uncertain checks
under `path:docs/TESTING.md`. A separate reviewer is useful when independence or risk
requires it; do not create a reviewer dispatch for every small return.

| Result | Orchestrator action |
| --- | --- |
| Full assigned outcome and required proof pass | Record acceptance and evidence in LOG; update STATUS and forecast |
| Useful partial result | Retain delivered evidence, concrete gaps and the next correction; keep full acceptance open |
| Shared dependency blocks one check | Record its owner/return condition; continue independent approved outcomes |
| Source or attribution is unknown | Preserve uncertainty; obtain the missing evidence before the affected claim |
| Repeated weak progress | Repartition, refactor, switch to enabling work or raise a real decision |

Retain acceptance gates set at admission. Record a justified correction when a gate
is wrong or needs a different proof route; never lower it merely to pass. Archive
resolved history verbatim and keep unresolved feedback active. Update one STATUS
`## Handoff` with changed, verified, remaining and next action. Follow the acceptance,
steering and native park commands in the execution reference. Never poll children.
Once due independent work is complete, park on the real pending condition. A neutral
check-in consumes the old wait; inspect current children and park again when still due.

### 5. Keep state and time with their owners

| Owner | Authoritative state |
| --- | --- |
| Prompt Manager team registry | Identity, selected profile, heartbeat, homePath |
| CHARTER | Operator-owned product destination, acceptance intent and selected sources |
| STATUS | Surface scoreboard, active units, allocation, dependencies, forecast and one handoff |
| LOG | Stable accepted-unit IDs, exact worker runs, evidence and reopen records |
| FEEDBACK | Verbatim operator/supervisor directions and dispositions |
| epochs/E<n>.md | One assignment's brief, directives and worker slice log |
| Agent Manager enrollment | Commission, worker limit, persisted clock and frozen terminal handback |

Keep the resume set (CHARTER + STATUS + open FEEDBACK) below 30,000 bytes. Use
`unassessed → queued → rough → close → accepted` for surfaces. Accepted assignments
do not automatically accept whole surfaces. Record each active unit with its stable
ID and worker runs. Archive resolved history; never discard unresolved obligations.

Read native deliveryPressure. ON TRACK pursues full intent; APPROACHING prioritizes
remaining value; OVER explicitly defers optional work; CONVERGE secures coherent partial
delivery; HARD STOP checkpoints and hands back. Required behavior and acceptance truth
remain intact. The owner enforces admission/stop and freezes an outcome after terminal
readback. A timed stop is a partial handback, not successful product acceptance.
A stopped commission cannot restart itself. Disabling heartbeat does not pause the
clock or retained execution. Native clock commands and thresholds are in
`path:scenarios/agent-manager/docs/reference/effort-supervision.md`.

### Troubleshooting & Edge Cases

| Situation | Read/action |
| --- | --- |
| Preparing or recommissioning a team | [Commissioning](references/commissioning.md); keep it disabled until the existing owner route is qualified |
| Dispatch, gates, directives, park or operator decision | [Execution reference](references/execution.md); load the relevant section once |
| Lost worker, quota, continuation or authority refusal | [Recovery](references/recovery.md); preserve the existing owner and real denial |
| Shared infrastructure fails | Keep the affected proof open; use an authorized equivalent or return condition and continue independent work; supervisor owns shared repair |
| Historical brief conflicts with current charter | Explicitly supersede it before dispatch; retain unique product requirements and evidence |

Blocked means a decision, credential or approval you lack. Name it. Diagnosable friction
is not blocked. Repairs inside the approved destination stay with this effort; work
outside it follows `path:docs/agent-system/SWARM_MANAGER_WORK.md`. Preserve native identity,
permissions, human-only choices and governed dependency/generation/lifecycle operations.
Do not invent an extra grant, isolation or approval requirement from assignment wording.

### Output expectations and anti-patterns

Deliver current source and proportionate outcome evidence, an honest STATUS and exact
LOG attribution. Finish successfully when the charter destination is met. If all
remaining work requires a real operator/external return, keep it open and park. At a
native timed stop, publish the honest handback. Record a work-record through the Memory
contract and clean up authorized goal-only installs/processes on closure.

| Avoid | Do instead |
| --- | --- |
| Optimize process receipts or line counts instead of the product | Assess required behavior and matched visual results |
| Repeated tiny assignments | Assign coherent outcomes with implementation and self-validation |
| Read all history on every wake | Read current state and the exact changed evidence |
| Stop at green tests or a worker's completion claim | Independently assess the full assigned outcome |
| Freeze the first decomposition | Update the allocation when evidence changes |
