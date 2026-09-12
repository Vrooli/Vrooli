# Swarm Manager Architecture

## Plan-backed execution

Swarm owns the backlog item, the exact Plan Manager revision, acceptance, the
queue, execution identity and the final disposition. Agent Manager owns the
declared workflow or run, its child runs, budget accounting, cancellation and
the durable journal. Product evidence stays with the scenario and its evidence
providers.

A plan-backed item is described by two independent choices:

| Axis | Values | Where it is set |
| --- | --- | --- |
| **Plan shape** — what the work is | `phased` or `mandate` | A field on the Plan Manager plan. |
| **Execution mode** — how the item runs | `sliced` or `goal` | The Run dialog. The item records the default. |

Any shape runs under either mode. A **phased plan** is an ordered list of
phases, each with steps, acceptance and validation. An **adaptive mandate**
points at the scenario docs, names sensors (setpoint programs), defines done as
bands, and states scope and authority, stop rules, a suggested arc and a journal
location. It has no step lists; its progress is the setpoint board, not a phase
frontier.

| Mode | What Agent Manager runs | Bound | Review |
| --- | --- | --- | --- |
| `sliced` | The workflow `swarm-manager/phased-plan-drain`. | `max_slices` per execution. | An independent review per slice, then a decision node. |
| `goal` | One Agent Manager run in harness goal mode. No workflow. | The item allowance only. | Swarm finalization after the run ends. |

**Sliced mode.** Each slice is a fresh worker run with bounded turns and time.
The worker reads prior-slice handoffs from the append-only journal. A review
child verifies the cited evidence against the plan's completion policy; a
worker's summary is not its own acceptance receipt. A rejected review continues
the named worker in a correction, then reviews the replacement result again. A
routine phase boundary uses the item's approval policy. The slice result
identifies `approvalReason=operator-decision` for a target or grant change, an
explicit operator pause, or another genuine authority boundary. That reason
waits even when routine approval is automatic. Missing reasons stay
conservative. The workflow counts actual slice attempts, preserves waits and
blocked outcomes, and returns `budget_exhausted` when it cannot start the next
permitted slice. No mode may reinterpret a protected target or expand authority.

**Goal mode.** Swarm composes the goal message from `harness-goal-authoring`
(Shape A for a phased plan, Shape B for a mandate) and dispatches one run. Agent
Manager installs `/goal <finish line>` in a warm interactive session when the
runner declares native goal support for the sandbox mode. It carries the finish
line in the prompt when native support is absent. The finish line is a
condition, never a feeling of doneness. For a phased plan it is every phase
recorded finished in Plan Manager with evidence. For a mandate it is every
setpoint row in band and a passing evidence audit. There is no slice cap and no
per-session reviewer. The run ends with an agent-decided verdict or an
involuntary interruption (see Continuation). Swarm finalization is the
completion authority. Goal mode is the recommended default for a mandate.

**Finalization** is Swarm's item-level completion path after any execution.
Swarm restarts the scenario, runs the health check, gathers evidence, runs the
review agent, then lands the item at done, `needs_review`, or follow-up. Sliced
mode reaches finalization after its per-slice reviews. Goal mode reaches it
directly.

### Item contract

Queue admission binds the canonical plan hash and the item contract. Backlog
acceptance and execution share one subject-version contract. Editing the
contract clears acceptance. The UI preserves these values through its API
mapping, shows the aggregate limits before acceptance and launch, and lets one
run narrow its slice allowance. Bulk launch keeps each item's own settings.

| Field | Values | Notes |
| --- | --- | --- |
| `plan_ref`, `plan_acceptance` | A Plan Manager plan; acceptance pins the content hash. | The approval digest. |
| `execution_mode` | `sliced` \| `goal` | Default from the plan shape. A mandate recommends `goal`. |
| `execution_limits` | `max_slices` (sliced only), `max_tokens`, `max_wall_seconds`, `max_turns`, `max_charge_micro_usd`, `max_children`, `max_node_attempts`, `max_retries` | One aggregate allowance across resumes. |
| `continuation` | `manual` \| `until-allowance` | Resume policy for involuntary interruptions only. |
| `scope_policy` | `fixed` \| `extend-with-record` | See Scope policy. |
| `acceptance_allow`, `acceptance_deny` | Path globs | Narrow to the target scenario, its packages, protos and docs. Use `extend-with-record` for the rest. |
| `operator_note` | Free text | Reaches the agent verbatim in the goal message and in the slice prompt: intent, reminders, authority to fix shared packages with a record. A run may add its own note. |
| `execution_preferences` | `preferred_runner`, `model`, `effort` | Carried on the queue request and the execution record, not stored on the item. The runner is a preference, not a pin. It reorders role candidates; the fallthrough reason is visible. |

An item without explicit limits keeps the bounded ordinary defaults. A larger
allowance requires explicit reviewed limits. Workflow catalog capacity is an
admission ceiling, not an allowance granted to every item. Agent Manager pins
the supplied grant to an execution. A resume or retry accounts for earlier
measured usage; missing authoritative terminal usage never becomes a fresh
budget. Cost is measured or reported unknown, never zero by default. Money for
coding agents and money for product inference are separate authorities.

### Scope policy

The item's authored `acceptance_allow` and `acceptance_deny` remain the
acceptance and plan-acceptance contract. With `scope_policy: fixed`, an
out-of-scope edit is an operator decision. With `scope_policy:
extend-with-record`, the agent records `plan-manager exec boundary-extend` with
a reason before editing. Swarm re-reads Plan Manager's append-only
`boundary_extensions` at every start, rebuild and apply: the goal message at
run start, each slice projection, each resume, and finalization. It unions the
`added_allow` paths into the effective `writeScope` and carries the policy in
`constraints.scopePolicy`. The same effective scope drives finalization
scenario selection and review expectations. A recorded extension that overlaps
`acceptance_deny` is refused. Neither the authored allow list nor
`plan_acceptance` is rewritten. Opening review or a Run dialog does not start
work.

### Continuation

Every stop belongs to one of two classes:

| Stop class | Outcomes | What Swarm does |
| --- | --- | --- |
| Agent-decided verdict | `complete`, `blocked`, `abstained` | Finalization. The verdict is final. Nothing resumes it. |
| Involuntary interruption | Usage window, timeout, crash, or session lost, with no terminal result. | Resume only under `continuation: until-allowance`. |

Under `continuation: manual` every stop is a stop. The operator starts the next
run.

Under `continuation: until-allowance` Swarm resumes an involuntary interruption:

1. Confirm that allowance remains on every aggregate dimension, that the
   acceptance digest and the effective scope are still valid, and that no halt
   flag is set.
2. Derive the remaining allowance from settled usage under the same approval
   digest. Unknown usage blocks the resume.
3. Prefer resuming the same session. Start a fresh run with the last handoff
   only when the session is dead.
4. Copy the parent's mode and execution preferences. Create one pending resume
   at a time.
5. Halt the item for an operator after three resumes with no observed Plan
   Manager progress. Record `no_progress` as `continuation_stopped_reason`.

An exhausted dimension (`tokens`, `charge`, `wall`, `slices`, or another
aggregate limit) is recorded as `continuation_stopped_reason`. Operators can
halt or resume an item without cancelling a running execution. Parent and child
IDs and the halt/stop state are projected in execution and backlog responses.
Operational continuation fields never change the accepted item digest.

This route is for trusted coding agents with ordinary workspace and owner scope
controls. It does not claim a hard in-flight token ceiling or qualified
containment of every external effect. Native-goal capability, tracking,
protected containment, workflow admission, and accounting completeness are
distinct facts. Use current owner capability evidence and executed
qualification, rather than inferring one from another or from a declaration's
existence.

Swarm accepts a terminal workflow or run result only when workflow or run
identity, definition digest, consumer identity, entity version and frontier
digest match. A local claim applies the typed transition once. A successful
worker result moves work toward review; it does not prove that every promised
product outcome passed or authorize publication. Final review must inspect the
approved outcome denominator and actual receipts, including unavailable, stale
and failed evidence.

**Implementation status (2026-09-11).** The code still stores
`execution_strategy` with three values: `phased-plan-drain`,
`adaptive-improvement`, and `goal-session`. The first two run the
`swarm-manager/phased-plan-drain` workflow. The third runs an interim
`swarm-manager/goal-session-drain` workflow that goal mode supersedes and that
will be removed. The generated [transition catalog](../reference/transition-catalog.md)
shows `plan.execute` bound to that one workflow key, while the registry behind it
still declares the three strategies. Plans have no `shape` field and items have
no `operator_note` field. Continuation keys on a `budget_exhausted` workflow
status that the typed apply never accepts, so involuntary interruptions are not
resumed today. The strategy-to-mode mapping lives in
[SCENARIO_DEVELOPMENT.md](../../../../docs/agent-system/SCENARIO_DEVELOPMENT.md#implementation-status).

## Retired route: contract-development

The earlier development-contract route is retired in the target design. Its
transition, workflow, Development contract panel, and the
`swarm-manager development` CLI are removed. A plan-backed item carries its own
development grant: the operator authorizes one item within its acceptance
globs, effect policy and aggregate limits through ordinary plan acceptance. No
second approval exists, and no item binds two lifecycles.

Its preview step (`TransitionService.PreviewDevelopment`) resolved selected
skill, program and target files and reported field and source completeness.
Its approval, amendment, revocation and acceptance endpoints required the
configured verified-human identity and write capability. Those two properties
survive in the plan-backed route: plan acceptance requires a verified human, and
a complete preview is never launch qualification.

**Implementation status (2026-09-11).** The `contract-development` transition is
still registered, and the preview, development CLI and panel still exist. The
route is never launch-ready: it has no registered product evidence resolver, and
its coordinator is not composed into a launch transition. Do not create new
items on it.

## Qualification and target adoption

[Scenario development](../../../../docs/agent-system/SCENARIO_DEVELOPMENT.md)
owns the shared method. Qualify each mode with one bounded disposable item
before using it as evidence of approval-ready product execution:

1. Accept and launch one exact plan revision. Duplicate admission returns the
   same work rather than another allowance.
2. Repair two distinct defects with independently inspected evidence and no
   intermediate operator approval. Recover the next repair from durable context.
3. Preserve original authority and aggregate usage across correction, fresh
   runs, interruption and resume. Stop new dispatch on revocation or exhausted
   allowance.
4. Refuse a weaker target, stale acceptance, unknown required evidence and an
   unsupported hard-ceiling request. Preserve uncertainty through owner outages.
5. Verify actual workspace changes and apply provenance. A complete harness with
   failed finalization remains a failed execution boundary until owner recovery.

Source tests, catalog reconciliation, live workflow and run receipts, and real
provider observations establish different parts of this proof. Record the exact
handles and limits in the active infrastructure plan. Product plans remain
unaccepted and unstarted while this infrastructure is qualified.

Until Tech Tree Designer's generic revisioned bundles are implemented, a reviewed
plan can retain proposed target text directly in its canonical content. Its hash
then binds the target bytes. A mutable external path alone cannot carry
acceptance; keep any file copy's digest and source role explicit. Future draft
bundles replace this verbose fallback only after owner-mediated
review/application is qualified.

### Transition dispatch

The transition registry is the runtime catalog for every declared agent
capability. `internal/transitionrunner` is the only Swarm component permitted
to select a declared workflow, dispatch a goal-mode run, invoke Agent Manager,
collect a terminal result, or persist the shared `claimed`/`complete`
correlation journal. Sliced mode dispatches the `plan.execute` transition to
the workflow `swarm-manager/phased-plan-drain`. Goal mode dispatches one Agent
Manager run with the composed goal message; there is no workflow key. Subject
domains contribute only immutable input builders and typed apply functions.
Startup verifies that every workflow and deterministic `applyAction` resolves
to a registered function; an incomplete dispatch table fails closed. The
Connect `TransitionService` and the CLI/UI catalog clients are the generic
discovery and execution surfaces. Session transitions remain catalog-visible
but stay on the Agent Session path.

Swarm Manager is the **operator command center for autonomous change work**. A backlog item moves through one arc — intake → Plan Workshop authoring → explicit operator acceptance → mode-selected execution (sliced or goal) → evidence-backed review → operator decision → follow-up proposals — and Goals sit above the items as intent statements with milestones and acceptance criteria. Captures are ephemeral intake events: the ground-and-shape workflow may propose items, goals, and milestones, lands work at `suggested`, or emits one `research` item when evidence is incomplete; the capture is then deleted. The narrative version of both arcs lives in [OPERATOR-JOURNEYS.md](./OPERATOR-JOURNEYS.md); the authority model lives in [TARGET-OPERATING-MODEL.md](./TARGET-OPERATING-MODEL.md).

The primary operator surface is the **Plan board** at `/plan`; the **Graph workspace** at `/graph` is the secondary, topology-first navigation surface (see "Operator Surfaces" below).

```
   intake                    swarm-manager                     substrate
┌───────────────┐   ┌────────────────────────────────┐   ┌─────────────────┐
│ operator      │──▶│ BACKLOG ITEM                   │   │ plan-manager    │
│ captures      │──▶│ Plan Workshop → accept → run  │◀─▶│ (canonical plan)│
│ session/goal  │──▶│  → review → decide → follow-up │   ├─────────────────┤
│ proposals     │   ├────────────────────────────────┤   │ agent-manager   │
└───────────────┘   │ GOALS + MILESTONES             │◀─▶│ (declared       │
                    │  intent → proposals → DoD      │   │  workflows)     │
                    │  review → progress truth       │   ├─────────────────┤
                    └────────────────────────────────┘   │ test-genie /    │
                                                         │ git-control-    │
                                                         │ tower (evidence)│
                                                         └─────────────────┘
```

**Why this matters:** agents (sessions, Plan Workshop, reviews, goal workflows, and capture grounding) analyze and propose, but they never mutate project work directly. Swarm Manager is the single place where proposals are decided, plan references are accepted, execution is authorized, evidence is retained, and terminal statuses are written — the project-work equivalent of a pull-request review boundary.

## Domain Concepts

| Concept | Description | Lifecycle States | Implementation |
|---------|-------------|------------------|----------------|
| **Backlog Item** | Unit of work stored as git-tracked folders (`idea`, `research`, `fix`, `execute`, `chore`) | `suggested` -> `backlog`/`ready`; normal flow: `backlog` -> `researching` -> `ready` -> `queued` -> `in_progress` -> `completed`/`failed`/`archived` | [CODE: ui/src/types/domain.ts#BacklogItem] |
| **Milestone** | Lightweight grouping of related backlog items by a shared label plus explicit milestone metadata | Derived from member items with explicit operator-managed metadata (`name`, `title`, `description`, `status`) | [CODE: api/internal/backlog/store.go] |
| **Dependency** | Directed edge between backlog items (`depends_on` field in spec.json) | N/A (structural, validated on write) | [CODE: api/internal/depgraph/graph.go] |
| **Execution Run** | Governed execution-control record linked to backlog work | `pending` -> `scheduled` -> `running` -> `completed`/`failed`/`canceled` | [CODE: ui/src/types/domain.ts#ExecutionRecord] |
| **Agent Activity** | Durable record for one tracked AgentManager interaction (`spawn` or `continue`) across backlog, scenario, capture, and session flows | `pending` -> `starting`/`running`/`needs_review` -> `complete`/`failed`/`cancelled` | [CODE: ui/src/types/domain.ts#AgentActivity] |
| **Agent Session** | Durable human-led conversation for meta-orchestration and Swarm operations, with proposals, artifacts, and verified attribution | `starting` -> `running` -> `waiting_for_user`/`proposal_ready` -> `complete`/`failed`/`canceled` | [DOC: docs/internal/AGENT-SESSIONS.md] |
| **Scenario** | Runtime scenario in the Vrooli ecosystem | `running`, `stopped`, `error`, `unknown` | [CODE: ui/src/types/domain.ts#Scenario] |
| **Capture** | Ephemeral raw operator/agent observation (text + optional images) grounded into proposals, one research item, or discard | `pending` -> `classifying` -> deleted after proposal/discard recording | [CODE: api/internal/captures/io.go] |
| **Record** | Immutable narrative artifact of completed work (`trigger`, `approach`, `ruled_out`, `commit`, `files_changed`, `outcome`); mirrors `BacklogKind`; supports `supersedes` chains for amendments | Stub (auto-created on backlog completion) -> filled (once, via `records edit`) -> immutable (further changes require supersedes) | [CODE: api/internal/records/types.go] |
| **Event** | Append-only audit entry for entity state deltas (backlog status, record created/superseded, etc.); queried by named measures | N/A (immutable) | [CODE: api/internal/eventlog/types.go] |

### Four-Entity Model

Swarm-manager's domain forms a four-entity pipeline that closes the recursive-learning loop:

1. **Captures** — raw operator/agent input; the front door for observations and ideas before they have shape.
2. **Backlog** — current-state work tracking; what is being done now, its plan reference, and its queue/execution lifecycle.

### Backlog next-action policy

`backlog.ResolveNextAction` is the single read-only action policy for first-party
backlog surfaces. It calls the same execution `ProcessPreflight` boundary used
by queueing and combines its evidence with dependency and lifecycle facts. The
policy never treats `plan_ref` presence as execution readiness. Detail views use
the single-item projection; list surfaces use the bounded batch endpoint, so a
visible card list does not issue one Plan Manager validation request per card.
3. **Records** — narrative artifacts of completed work; what was learned, including hypotheses ruled out, files touched, commit, and outcome. Records are the **write side of the recursive-learning loop**: future agents query them through `search query --type record`. Stub records are auto-created on backlog terminal transitions (`review decide --accept|--fail`) and filled by the executing agent; records can also be created for work that never touched the backlog.
4. **Events** — audit log of state deltas (backlog status changes, record creations and supersedes, etc.) queried by named measures with explicit provenance.

### Milestones

Milestones are a lightweight grouping mechanism. Each backlog item may carry an `milestone` string field in its `spec.json`. Items sharing the same milestone value are considered members of that milestone. Milestone metadata is managed through the milestones API or supplied inline during backlog batch-create preview/create. Milestone updates are partial: callers only send the fields they intend to change.

### Dependencies

Backlog items can declare dependencies on other items via the `depends_on` field in `spec.json`. Each entry is a `"kind/name"` reference (e.g., `"fix/auth-bug"`). The `depgraph` package builds a directed acyclic graph from these references and provides:

- **Cycle detection** -- rejects writes that would introduce circular dependencies
- **Topological sort** -- determines safe execution order for batch operations
- **Validation** -- ensures all referenced items actually exist

## Key Flows

1. **Backlog creation and refinement**
   ```
   Team finding -> Backlog item (idea/research/fix/execute/chore) -> declared refinement workflow or Session -> plan-manager plan_ref -> queue
   ```
   Every backlog kind uses the same plan-backed readiness lifecycle. Research artifacts remain ordinary item files, while Plan Manager is the readiness authority for the canonical plan bound through `spec.json.plan_ref`. See [DOC: docs/reference/transition-catalog.md] for the active catalog.

2. **Archive scenario into backlog context**
   ```
   Scenario delete with archive=true -> scenario removed -> archived backlog idea created with preserved files
   ```

3. **Batch operations**
   ```
   POST /api/v1/backlog/batch (preview=true) -> validate items + milestone plan + dependency refs -> no writes
   POST /api/v1/backlog/batch -> apply milestone changes -> create items atomically -> assign milestone membership
   POST /api/v1/backlog/batch/queue -> topological sort via depgraph -> queue items in dependency order
   ```

4. **Execution lifecycle**
   ```
   Queue backlog item (manual/scheduled/yolo) -> execution record -> declared Agent Manager workflow (sliced) or one run (goal) -> typed terminal result -> Swarm authorized apply -> finalization
   ```

5. **Backlog auto-filer**
   ```
   ticker / feature-queue wake / operator run-now
     -> targeting strategy (feature_pending or importance)
     -> GCT readiness finding source
     -> policy gates (enabled, cap, velocity brake, dismissal memory)
     -> filer/reconciler
     -> backlog item + explicit automated-maintenance goal target
   ```
   The auto-filer is the governed intake loop for programmatic maintenance
   findings. In `suggest` mode it creates `suggested` backlog items that an
   operator can accept into the normal flow or dismiss. In `auto_add` mode it
   creates normal backlog items while still applying the open-item cap,
   velocity brake, and dismissal memory. Reconciliation runs through the same
   loop: findings that no longer hold archive untouched suggestions and add a
   note to already-accepted work instead of deleting operator history.

   Implementation references: [CODE: api/internal/autofiler/sweeper.go],
   [CODE: api/internal/autofiler/policy.go], [CODE: api/internal/autofiler/filer.go].

6. **Graph workspace projection**
   ```
   GET /api/v1/plan -> proto PlanBoardResponse -> plan store -> Now/Next/Later/Done board
   GET /api/v1/graph?lens=topology -> proto GraphResponse -> typed graph store -> Graph surface (full topology by default; client-side focus mode)
   WS /ws/graph invalidate (lenses incl. "plan") -> silent refresh + runtime node pulse
   ```

7. **Native agent sessions**
   ```
   Graph launcher -> draft agent session -> composer message + context/images -> Agent Manager run -> proposal -> API-owned apply -> artifact attribution
   ```
   Agent Sessions support longer human-led planning and operations conversations inside Swarm Manager. Session details uses the shared composer also used by Quick Capture, with session-only context chips for existing backlog items, milestones, captures, executions, agent activity, scenarios, prior sessions, and the current operations briefing. Message context is resolved by the API before it reaches Agent Manager, and uploaded images are stored as session-owned attachments. Meta-orchestration sessions can create multiple milestones and backlog items through the batch apply seam. Swarm operations sessions receive a bounded `operations_briefing/latest` context by default, answer broad current-status questions from that packet first, then drill down through the operations, overview, and named-measures commands only when needed. See [DOC: docs/internal/AGENT-SESSIONS.md].

8. **UI route navigation**
   ```
   /plan -> Plan board (first-class route, default landing; ?drawer=decisions opens the decision drawer)
   /graph -> Graph surface (full topology projection by default)
   /graph?mode=focus -> Graph surface in attention-filtered focus mode
   /backlog/:kind/:name -> backlog detail
   /scenarios/:name -> scenario detail
   /executions/:executionId -> execution detail
   /milestones/:name -> milestone detail
   /captures/:captureId -> capture detail
   /graph/plan -> redirect to /plan (legacy graph path; query state preserved)
   /graph/focus -> redirect to /graph?mode=focus (legacy graph path; query state preserved)
   /graph/topology -> redirect to /graph (legacy graph path; query state preserved)
   /operations, /command-post, /command-post/decisions -> redirect to /plan (Command Post and the Operations Center were absorbed by the Plan board)
   /executions, /scenarios (bare list paths) -> redirect to /plan (the retired ExecutionPage/ScenariosPage list surfaces; detail routes above are unaffected)
   ```

   Fullscreen operator surfaces are first-class routes inside a shared app shell. The shell owns the global sidebar. Page close/back controls use route-aware history with a direct-load fallback to `/plan`.

9. **Global sidebar shell**
   ```
   AppShell -> persisted, resizable desktop sidebar + floating mobile sheet -> routed page outlet
   ```

   Sidebar open/collapsed state, desktop width, active tab, search mode/query, filters, and sort options are stored in localStorage. The sidebar no longer writes ambient UI preferences into the current route query string.

## Operator Surfaces

Swarm Manager exposes two operator navigation surfaces: **Plan** and **Graph**. Plan is the primary control surface at `/plan`. Graph is the single graph surface at `/graph`; it renders the full topology by default and can enter focus mode through query state (`mode=focus`). Topology remains the server projection name (`GET /api/v1/graph?lens=topology`), not a user-facing tab.

### Plan
**Purpose:** One forward-looking board answering "what is running, what is actionable, in what order will the rest happen, and where am I needed."

Four columns computed by the server plan projection (`GET /api/v1/plan`, `internal/planview`):

- **Now** — in-flight agent runs (cards from `GET /api/v1/operations` via the proven polling path) with lane utilization bars, queue chip, group-by milestone/phase, select-mode bulk stop, spawn and refresh actions.
- **Next** — actionable immediately: server-owned next-action cards (decide / review / plan / run) at dependency wave 0, plus capture proposal gates in the decision stream. Suggested work is visibly distinct and filterable. Header bulk actions: Run all ready (threshold-confirmed) and Answer all (decision drawer).
- **Later** — not yet actionable, grouped by nearest blocker (gate-blocked groups sort above item-blocked), with honest ordinal wave badges from `depgraph.Waves` frontier peeling. Waves deeper than 5 collapse into a "beyond horizon" rollup; dependency cycles surface as diagnostics.
- **Done** — window-capped recent outcomes (1h–24h picker on the column header).

Filters (search / status / owner-type / lane / group-by / show-snoozed) live in a shared drawer and persist in URL query params. Snooze remains client-side (localStorage). The decision drawer hosts the full decision stream (`?drawer=decisions` deep link) and per-item scoped answering from server-owned decide cards. No drag: columns are derived, so cards act through explicit menus mapped to real levers (run / workshop / finalize / archive / status / snooze / focus).

**Navigation:** First-class `/plan` route; the default landing for `/`, `/graph/plan`, and all retired-surface redirects. Keyboard shortcut: `1`.

### Graph
**Purpose:** Full structural exploration, selection, inspector actions, and attention-filtered focus mode on one graph surface.

The default graph mode renders the topology projection (`GET /api/v1/graph?lens=topology`) on the node/edge canvas. Focus mode is a client-side filter over the same topology payload: nodes pass `computeNodeAttention` (pending decisions, review-ready, failures) and their milestone/scenario context is re-attached via `member_of`/`targets` edges. Node click applies BFS visual focus; the inspector panel offers per-entity actions.

**Navigation:** Graph tab at `/graph`; the board's per-card "Focus on graph" action and detail-page focus links navigate to `/graph?mode=focus&select=<node>`. Keyboard shortcut: `2`.

**Edges:** `depends_on`, `member_of`, `targets`

10. **Scenario lifecycle control**
   ```
   List scenarios -> inspect details -> start/stop/restart/delete/archive
   ```

## Logical Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ PRESENTATION LAYER (UI)                                     │
│ App shell + graph workspace + sidebar/search + routes        │
├─────────────────────────────────────────────────────────────┤
│ API GATEWAY LAYER (Go API)                                  │
│ HTTP/proto endpoints, validation, response contracts         │
├─────────────────────────────────────────────────────────────┤
│ DOMAIN LOGIC LAYER                                           │
│ Backlog + milestones + depgraph + overview + scenarios +    │
│ execution + agentactivity + agentsessions + promptcatalog +  │
│ settings orchestration                                       │
├─────────────────────────────────────────────────────────────┤
│ INTEGRATION LAYER                                            │
│ agent-manager + prompt-manager + CLI                         │
├─────────────────────────────────────────────────────────────┤
│ PERSISTENCE LAYER                                            │
│ Filesystem: backlog folders + .vrooli/*.json state           │
└─────────────────────────────────────────────────────────────┘
```

### Current Implementation State

| Layer | Status | Notes |
|-------|--------|-------|
| Presentation | Functional | Shared app shell owns global navigation; the Plan board is primary (`/plan`), with one Graph surface (`/graph`) backed by the topology projection plus focus mode, canonical detail routes for backlog, milestones, scenarios, executions, and captures, and a sidebar Sessions tab |
| API Gateway | Implemented | Health, graph, backlog (incl. batch), agent sessions, scenarios, settings, queue, execution, prompts, milestones, overview, captures, agent-manager status |
| Domain Logic | Implemented | CRUD, archive, queue, research, batch ops, dependency graph, milestones, agent sessions, overview aggregation, execution scheduling and run control |
| Integration | Implemented | Discovery-based clients (agent-manager, prompt-manager) and CLI-backed scenario operations |
| Persistence | Filesystem-first | Backlog items and execution/agent-activity/settings/queue JSON persisted on disk |

## Historical workshop readiness model

The previous workshop system used a 5-dimension readiness model to measure how prepared a backlog item was for execution. This is retained only to explain historical round records and migration data; it is not an active orchestration contract or a replacement for Plan Manager validation.

| Dimension | Measures |
|-----------|----------|
| `problem_clarity` | Is the problem well-understood? |
| `scope_defined` | Are boundaries and deliverables defined? |
| `approach_solid` | Is the technical approach viable? |
| `testable` | Can success be verified? |
| `risk_awareness` | Are risks identified and mitigated? |

The score and boost formula are historical-data semantics only. The active
contract is a Plan Workshop session with a typed packet, one idempotent
operator response, Plan Manager candidate validation, and explicit plan
acceptance. A backlog item queues only while its accepted canonical plan hash
and work-contract version remain current. Research evidence is stored as normal
item artifacts and never replaces the implementation plan.

See [DOC: docs/guides/workshop-workflow.md] for the active Plan Workshop operator contract.

## Physical Structure

Key implementation files:
- Domain types: [CODE: ui/src/types/domain.ts]
- API routes/composition: [CODE: api/main.go]
- Graph workspace: [CODE: ui/src/surfaces/graph/components/GraphWorkspace.tsx]
- Graph projection API: [CODE: api/internal/graph/projection.go]
- Backlog service: [CODE: ui/src/services/backlog-service.ts]
- Graph service: [CODE: ui/src/services/graph-service.ts]
- Scenarios service: [CODE: ui/src/services/scenarios-service.ts]
- Execution service: [CODE: ui/src/services/execution-service.ts]
- CLI commands: [CODE: cli/app.go]

### API Package Structure

The backlog handler has been decomposed from a single large file into focused modules:

```
api/internal/
├── backlog/           # Backlog domain (refactored)
│   ├── types.go       # Domain types and interfaces
│   ├── store.go       # Filesystem CRUD abstraction
│   ├── handler.go     # HTTP route registration and core handlers
│   ├── files.go       # File upload/download handlers
│   ├── research.go    # Research spawn handlers
│   ├── queue_ops.go   # Queue/dequeue handlers
│   ├── archive_handlers.go  # Archive operations
│   ├── kind_config.go # Per-kind metadata (deliverable filename, directory)
│   ├── batch_handler.go     # Batch create (all-or-nothing)
│   └── batch_queue_handler.go # Batch queue (topological order)
├── depgraph/          # Dependency graph (pure computation)
│   └── graph.go       # Cycle detection, topological sort
├── milestones/       # Milestone CRUD + rollup status
├── overview/          # Aggregation endpoint (backlog + milestones + graph)
├── captures/          # Capture intake, grounding, and proposal recording
├── promptcatalog/     # Canonical runtime prompt inventory and resolvers
├── planworkshop/      # Active Plan Workshop session and response model
├── execution/         # Execution run lifecycle
├── graph/             # Graph projection + websocket invalidation
├── scenarios/         # Scenario CRUD and lifecycle
├── queue/             # Queue state operations
├── settings/          # Settings persistence
├── prompts/           # Prompt skill CRUD
└── integrations/      # agent-manager and prompt-manager clients
```

## API Boundaries

- `/health`, `/api/v1/health` - health and readiness
- `/api/v1/backlog/*` - backlog CRUD, queue, and research work
- `/api/v1/backlog/batch` - batch create (all-or-nothing with dependency validation)
- `/api/v1/backlog/batch/queue` - batch queue (topologically sorted, dependency-aware)
- `/api/v1/milestones/*` - milestone CRUD with rollup status from member items
- `/api/v1/goals/*` - goal CRUD, targets, priority; each response carries the transitive-closure scope (progress %) and a p50/p80 ETA band
- `/api/v1/plan-import` - POST an existing `{plan_id}` or adopted `{source_path|markdown}` plan, choose `container: "items"` or `container: "milestone"`, and land idempotent plan-bound work with `plan_ref` populated and created/linked/updated counts
- `/api/v1/plan-import/plans` - list canonical plan-manager plans for the Create-Work-From-Plan picker
- `/api/v1/execution/auto-drain` - GET/PUT the continuous goal-directed auto-enqueue toggle (default OFF; a scenario-local flag, not a proto setting)
- `/api/v1/overview` - aggregated view (backlog, milestones, dependency graph)
- `/api/v1/operations/brief` - bounded current operations briefing for CLI, UI, and Swarm operations session prompts
- `/api/v1/graph?lens=topology` - the topology projection (Graph focus mode filters it client-side)
- `/api/v1/plan?window_seconds=...` - the Plan board projection (waves + next-action markers)
- `/ws/graph` - graph invalidation and node pulse websocket
- `/api/v1/captures/*` - capture creation, attachment storage, workflow launch, proposal apply, and deletion
- `/api/v1/scenarios/*` - scenario list/detail/lifecycle/delete/archive
- `/api/v1/settings/*` - settings persistence
- `/api/v1/queue/*` - queue state operations
- `/api/v1/execution/*` - execution runs and policy operations
- `/api/v1/agent-activities/*` - tracked agent activity history and active runtime telemetry
- `/api/v1/prompts/*` - prompt catalog, skill CRUD, versions, revert, preview, simulate
- `/api/v1/agent-manager/status` - agent-manager availability
- `BacklogService` - **Connect-RPC**; owns typed backlog create, read, update,
  delete, and `DecideAttempt` mutations. `DecideAttempt` addresses a durable
  attempt by subject and round, so a review decision has one audited typed
  mutation path. Legacy REST domains remain tracked in
  `docs/internal/PROBLEMS.md`.
- `/vrooli.swarm_manager.v1.discovery.DiscoveryService/GetAudioToolsEndpoint` -
  **Connect-RPC**; resolves audio-tools' base URL for the browser.

### Audio capability (via audio-tools)

Voice input in `MessageComposer` (Session Details + Quick Capture) and
agent-message TTS in `ChatThread` flow through the `audio-tools`
scenario via the discovery endpoint above. The browser builds an
`AudioToolsClient` against the resolved base URL at boot, then the
copy-paste `ui/src/audio-integration/` module owns all STT / TTS /
summarize calls. swarm-manager contains zero audio synthesis or
transcription code. See `docs/internal/SEAMS.md` for the seam map.

## Meta-Orchestrator Skill

The `swarm-manager-meta-orchestrator` skill is the primary entry point for turning large goals into structured backlog imports. Its actual flow is:

1. Parse high-level input into clusters and candidate items
2. Discuss and refine the plan with the user, potentially across many turns before creation
3. Inspect existing scenarios/codepaths when the target systems already exist
4. Shape items with canonical backlog fields (`milestone`, `depends_on`, `acceptance_allow`, `acceptance_deny`)
5. Preview the multi-milestone import through `backlog batch-create --preview`
6. Create the items only after user approval

The skill intentionally supports long pre-creation planning so workshop auto-spawn happens only after the backlog descriptions are front-loaded with useful context.

## Priority Ranking

Backlog items are sorted using a three-tier system applied consistently across
the sidebar and command post:

1. **Dependency depth** (primary) — computed via `computeDepthMap()` in
   `dependency-sort.ts`. Items whose dependencies are incomplete sort below
   those dependencies. Depth 0 = no incomplete deps, depth N = depends on
   something at depth N-1. This axis is absolute and never overridden.

2. **Effective priority** (tiebreaker within same depth) — combines the item's
   manual priority (1-10) with an **unblocking value boost** based on how many
   incomplete items transitively depend on it:

   ```
   effectivePriority = manualPriority - min(transitiveDependentCount * 0.5, 3)
   ```

   Computed by `computeUnblockingMap()` + `computeEffectivePriority()` in
   `dependency-sort.ts`. Items that unblock more downstream work naturally
   surface higher. The boost is capped at 3 priority points so it influences
   but doesn't completely override manual priority.

3. **Recency** (final tiebreaker) — most recently updated items sort first
   within the same effective priority.

### Attention signals

`attention.ts` computes why an item needs user attention (pending decisions,
ready plans, completed research). These reasons power the sidebar tab badges
and backlog card badges. (The Activity-tab unified feed that once consumed
them was retired 2026-07-13.)

### Key invariants

- Dependency depth is never violated by priority boosts
- Completed and archived items are excluded from transitive dependent counts
- The unblocking map is computed once per sort call (O(V+E)), not per comparison

## Design Principles

1. **Backlog-first governance**: all planned scenario changes are represented as backlog artifacts.
2. **Execution visibility**: governed work is visible through execution records, and all agent usage is visible through agent activity records.
3. **Delegated implementation**: Swarm Manager governs work; agent-manager performs work.
4. **File-based context**: backlog artifacts remain human-readable and git-trackable.
5. **Prompt-manager team ownership**: research and recommendations are generated by teams and written into backlog items.
6. **Dependency-aware ordering**: batch operations respect the dependency graph to ensure items are processed in safe topological order.
7. **Canonical backlog contract**: change boundaries are expressed with `acceptance_allow` / `acceptance_deny`; `scope` is not part of the backlog model.
