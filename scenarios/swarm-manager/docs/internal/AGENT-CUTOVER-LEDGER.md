# Agent Manager cutover ledger

Exhaustive classification of every place `scenarios/swarm-manager/` invokes Agent
Manager spawn/continuation. This ledger drives Phases 2/4/5/6 of the "Declarative
agent operations for backlog items and initiatives" migration: every call site
classified **(a)** must move behind the operating-mode operation runner; **(b)**,
**(c)**, **(d)** are allowed boundaries/chokepoints/out-of-scope with recorded
rationale.

## Method

Call sites found by grepping non-test callers of the Agent Manager client
methods discovered on its interfaces — `SpawnBacklog`, `SpawnInitiative`,
`ContinueRun` (`internal/agentmanager/{spawn,run,client}.go`) — plus the
`agentsessions` service `Continue`. The `internal/agentmanager/` package itself
(the HTTP client / adapter) is excluded: it is the transport, not a decision
site.

```
grep -rn '\.SpawnBacklog(\|\.SpawnInitiative(\|\.ContinueRun(\|\.Continue(' internal/ \
  --include=*.go | grep -v _test.go | grep -v internal/agentmanager/
```

## Classifications

- **(a) target-bound autonomous operation** — fires an agent against a backlog
  item / initiative as part of an autonomous loop. **MUST move behind operating
  modes** (become a declarative operation invoked by the generic runner).
- **(b) interactive Agent Session behavior** — user-driven session continuation.
  Allowed boundary; stays as a session concern.
- **(c) generic runtime adapter / operating-mode engine** — the activity-tracking
  chokepoint (`agentactivity`) and the operating-mode phase runner itself.
  Allowed chokepoint; this is *where* (a) operations will route through.
- **(d) explicitly out of scope** — needs a recorded scope decision; not a
  target-bound backlog/initiative operation.

## Ledger

| ✔ | file:line | function | method | class | rationale | destination operation |
|---|-----------|----------|--------|-------|-----------|-----------------------|
| ✔ | `internal/backlog/research.go:567` | `Handler.Research` | SpawnBacklog | **a** | Autonomous research pass over a backlog item; item-target loop. | **refinement/research operation** (workshop-adjacent research round) |
| ✔ | `internal/backlog/workshop_save.go:421` | `Handler.spawnWorkshopAsync` | SpawnBacklog | **a** | Autonomous workshop synthesis spawn against an item. | **workshop round operation** |
| ✔ | `internal/backlog/clarification.go:202` | `Handler.CreateClarification` | SpawnBacklog | **a** | Spawns an agent to run a workshop clarification thread on an item. | **clarification operation** (workshop clarification round) |
| ✔ | `internal/backlog/clarification.go:353` | `Handler.ContinueClarification` | ContinueRun | **a** | Continues the clarification agent run for an item thread. | **clarification continuation** (operation follow-up step) |
| ✔ | `internal/backlog/clarification_service.go:166` | `Handler.spawnWorkshopForClarification` | SpawnBacklog | **a** | Re-enters workshop synthesis after a clarification resolves. | **workshop round operation** (clarification-triggered) |
| ✔ | `internal/execution/service_queue.go:281` | `Service.QueueSpecSyncArchive` | SpawnBacklog | **a** | Queues/starts the primary execution run for an item. | **execution operation** (queue → start) |
| ✔ | `internal/execution/service_control.go:117` | `Service.startLocked` | SpawnBacklog | **a** | Starts a queued execution run for an item. | **execution operation** (start) |
| ✔ | `internal/execution/retry.go:147` | `Service.Retry` | SpawnBacklog | **a** | Fresh agent run retrying a failed execution (new run_id, parent lineage). | **execution retry operation** |
| ✔ | `internal/execution/followup.go:125` | `Service.spawnFixupRun` | SpawnBacklog | **a** | Autonomous fixup run after a review found remediable issues. | **execution fixup/recovery operation** |
| ✔ | `internal/execution/followup.go:330` | `Service.FollowUp` | SpawnBacklog | **a** | Spawns a follow-up run for an item after completion. | **follow-up operation** |
| ✔ | `internal/execution/followup.go:306` | `Service.FollowUp` | ContinueRun | **a** | Continues the parent run as the follow-up path. | **follow-up continuation** (operation follow-up step) |
| ✔ | `internal/review/service.go:234` | `Service.startReview` | SpawnBacklog | **a** | Autonomous review agent for a completed item execution. | **review round operation** |
| ✔ | `internal/review/rounds.go:197` | `Service.RequestMoreEvidence` | SpawnBacklog | **a** | Spawns an agent to gather additional evidence for a review round. | **evidence-request operation** (review sub-round) |
| ✔ | `internal/initiativereview/trigger.go:191` | `Service.startReview` | SpawnInitiative | **a** | Autonomous initiative-level review agent. | **initiative review operation** |
| ☐ | `internal/agentsessions/service.go:434` | `Service.Continue` | ContinueRun | **b** | User continues an interactive agent session with a message. | — (session boundary; stays) |
| ☐ | `internal/agentsessions/service_mutation_proposals.go:154` | `Service.RequestMutationProposalRevision` | ContinueRun | **b** | User asks a session agent to revise a proposed mutation. | — (session boundary; stays) |
| ☐ | `internal/agentsessions/handler.go:242` | `Handler.Continue` | (→ `service.Continue`) | **b** | HTTP entrypoint into the interactive session-continue boundary. | — (session boundary; stays) |
| ☐ | `internal/agentactivity/spawn.go:84` | `Service.spawnTracked` | SpawnBacklog | **c** | Activity-tracking chokepoint wrapping the real Agent Manager client. | — (chokepoint; (a) operations route through it) |
| ☐ | `internal/agentactivity/spawn.go:186` | `Service.spawnInitiativeTracked` | SpawnInitiative | **c** | Activity-tracking chokepoint for initiative spawns. | — (chokepoint) |
| ☐ | `internal/agentactivity/spawn.go:338` | `Service.continueTracked` | ContinueRun | **c** | Activity-tracking chokepoint for run continuations. | — (chokepoint) |
| ☐ | `internal/operatingmode/phase_runner.go:504` | `Service.spawnInitiative` | SpawnInitiative | **c** | The operating-mode phase runner (via activity service). | — (the engine itself) |
| ☐ | `internal/operatingmode/phase_runner.go:509` | `Service.spawnInitiative` | SpawnInitiative | **c** | The operating-mode phase runner (direct agent fallback). | — (the engine itself) |
| ☐ | `internal/captures/classify.go:146` | `Handler.spawnClassifyAgent` | SpawnBacklog | **d** | Classifies a raw capture into suggested backlog items; a pre-backlog *ingest/triage* concern, not a target-bound operation over an existing item/initiative. | — (out of scope; see scope decision) |

## Counts

| class | count |
|---|---:|
| (a) target-bound autonomous — **must cut over** | **14** |
| (b) interactive Agent Session boundary | 3 |
| (c) generic adapter / operating-mode engine chokepoint | 5 |
| (d) explicitly out of scope | 1 |
| **total** | **23** |

## The target-bound (a) set, grouped by destination operation

- **Refinement / workshop / clarification** (backlog item refinement loop):
  research (`research.go:567`), workshop synthesis (`workshop_save.go:421`,
  `clarification_service.go:166`), clarification thread + continuation
  (`clarification.go:202`, `clarification.go:353`). → Phase 5.
- **Execution** (item run lifecycle): queue/start
  (`service_queue.go:281`, `service_control.go:117`), retry (`retry.go:147`),
  fixup/follow-up (`followup.go:125`, `:330`, `:306`). → Phase 6.
- **Review / recovery / initiative review**: item review + evidence request
  (`review/service.go:234`, `review/rounds.go:197`), initiative review
  (`initiativereview/trigger.go:191`). → Phase 6.

## Recorded scope decision — (d) capture classification

`captures/classify.go:146` is **deliberately out of scope** for the operating-mode
cutover. Capture classification is an ingest/triage step that turns a free-form
capture into *suggested* backlog items; it is not an autonomous loop bound to an
existing backlog target or initiative, and it produces no target-bound run that a
mode would own. It remains a direct spawn (through the `agentactivity` chokepoint)
until/unless captures themselves become mode-driven, at which point this decision
should be revisited and this row reclassified. Re-examine in Phase 6 closeout.

## Phase 9 closeout (2026-07-15)

Verified at the final deletion sweep: re-running the ledger grep returns ONLY
class (b) session boundaries (`agentsessions`), class (c) chokepoints
(`agentactivity/spawn.go`, `operatingmode/phase_runner.go`), and the recorded
class (d) capture-classification ingest spawn. All 14 class (a) target-bound
sites are deleted or rerouted through the operation runner; the archtest
spawn-boundary allowlist (`internal/archtest/spawn_boundary_test.go`) is EMPTY
and enforced. The (d) capture-classification decision stands (captures are not
mode-driven); revisit only if captures become a declarative operation.
