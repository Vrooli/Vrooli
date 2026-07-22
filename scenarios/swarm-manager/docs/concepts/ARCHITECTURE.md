# Swarm Manager Architecture

## Typed workflow pilots

The backlog workshop-round pilot uses Agent Manager as a consumer-neutral workflow runtime. A small scenario-owned definition captures one immutable backlog snapshot and returns one discriminated result. Agent Manager executes the graph and returns that result with run/profile provenance; it has no backlog vocabulary and performs no Swarm mutation.

Swarm Manager remains the domain owner. It validates the typed result and applies it exactly once through its backlog API, including stale-snapshot and replay protection. This boundary is intentionally narrow: it introduces no classifier, target registry, domain-action framework, compatibility wrapper, or duplicated result schema.

The plan-execution pilot hard-cuts only `execution.Service.startPlanOperationLocked`
to `swarm-manager/phased-plan-drain`. The workflow owns slice composition:
each visit to `slice` creates an independent Run using
`swarm-manager/deep-work`; ordered compact handoffs come from the append-only
journal; `correction` explicitly continues the named slice; and the reusable
`phased-plan-slice-review` child uses `swarm-manager/analysis`. A rejected
review is bound into a same-conversation correction and the corrected handoff
is reviewed again before it can become terminal. The consumer-supplied
`maxSlices` value is enforced from durable `slice` attempt entries; reaching it
while more work is requested yields `budget_exhausted` without starting another
Run. Cycle edges and global budgets are finite, and approval is a durable
external signal. Blocked and abstained results retain their distinct Agent
Manager workflow statuses rather than masquerading as success.

Swarm still owns the plan frontier and execution lifecycle. It hashes the live
Plan Manager rendering and backlog snapshot, starts idempotently, and accepts a
terminal result only when workflow id, definition digest, consumer id, entity
version, and frontier digest all match. A two-stage local claim applies the
typed terminal transition exactly once. Retry, fixup, follow-up, research
conclusion, legacy records, and UI paths are outside this pilot.

### Pilot decision

Decision: **go for the generic Agent Manager workflow primitive and retain both
narrow pilots**. The evidence supports the execution-identity model: fresh Run
per loop visit, node-local profiles, named continuation, child workflow,
bounded journal context, finite cycles, durable waits, and consumer-owned
exactly-once mutation all pass focused and race gates without domain vocabulary
in Agent Manager. This is not authorization for a broad Swarm replacement.
The legacy operating-mode and agent-operation runtime has been removed. Legacy
records and event projections remain read-only for audit and migration history.

## Mental Model

Swarm Manager is the **staging and review layer** for agent-generated plans. Its primary role is to receive work proposals from prompt-manager agent teams, let operators review and refine them, and then control when and how they execute.

The primary operator surface is now the **graph workspace**. Operators navigate backlog items, milestones, scenarios, executions, agent activities, runs, and captures primarily through a graph-first view, with sidebar lists and detail routes serving as search, drill-down, and non-graph navigation paths.

```
prompt-manager                         swarm-manager
┌────────────────────────┐             ┌──────────────────────────────────┐
│  Agent Teams           │             │  STAGING / REVIEW / EXECUTION    │
│  ┌──────────────────┐  │             │                                  │
│  │ Feature Team     │──┼── idea ────▶│  ┌──────────┐                    │
│  │ QA Team          │──┼── fix ─────▶│  │ BACKLOG  │  Operator reviews  │
│  │                  │──┼── execute ─▶│  │ items    │  plans, uses       │
│  │                  │  │            ▶│  └────┬─────┘  transitions       │
│  └──────────────────┘  │             │       │        and plans          │
│                        │             │       ▼                           │
│  Skills define reviewed│             │  ┌──────────┐                    │
│  workflow or session   │             │  │ DECLARED │  typed result      │
│  instructions          │             │  │ WORKFLOW │  → plan_ref        │
│                        │             │  └────┬─────┘                    │
└────────────────────────┘             │       │                           │
                                       │       ▼                           │
                                       │  ┌──────────────┐                │
                                       │  │  EXECUTION   │  manual /      │
                                       │  │  CONTROL     │  scheduled /   │
                                       │  └──────┬───────┘  yolo          │
                                       │         │                         │
                                       │         ▼                         │
                                       │  Generator / Improver agents      │
                                       │  build or iterate the scenario    │
                                       └──────────────────────────────────┘
```

**Why this matters:** Agent teams in Prompt Manager can analyze and produce proposals, but they do not mutate project work directly. Swarm Manager is the single place to review proposals, retain a Plan Manager plan reference, authorize execution, and retain evidence. Programmatic refinement is selected from the registered workflow catalog; an operator can instead use a natural Session when the next step is conversation rather than a typed result. This is the project-work equivalent of a pull-request review boundary.

Recommendation generation lives in Prompt Manager teams, not in Swarm Manager.

## Domain Concepts

| Concept | Description | Lifecycle States | Implementation |
|---------|-------------|------------------|----------------|
| **Backlog Item** | Unit of work stored as git-tracked folders (`idea`, `research`, `fix`, `execute`, `chore`) | `suggested` -> `backlog`/`ready`; normal flow: `backlog` -> `researching` -> `ready` -> `queued` -> `in_progress` -> `completed`/`failed`/`archived` | [CODE: ui/src/types/domain.ts#BacklogItem] |
| **Milestone** | Lightweight grouping of related backlog items by a shared label plus explicit milestone metadata | Derived from member items with explicit operator-managed metadata (`name`, `title`, `description`, `status`) | [CODE: api/internal/milestones/service.go] |
| **Dependency** | Directed edge between backlog items (`depends_on` field in spec.json) | N/A (structural, validated on write) | [CODE: api/internal/depgraph/graph.go] |
| **Execution Run** | Governed execution-control record linked to backlog work | `pending` -> `scheduled` -> `running` -> `completed`/`failed`/`canceled` | [CODE: ui/src/types/domain.ts#ExecutionRecord] |
| **Agent Activity** | Durable record for one tracked AgentManager interaction (`spawn` or `continue`) across backlog, scenario, capture, and session flows | `pending` -> `starting`/`running`/`needs_review` -> `complete`/`failed`/`cancelled` | [CODE: ui/src/types/domain.ts#AgentActivity] |
| **Agent Session** | Durable human-led conversation for meta-orchestration and Swarm operations, with proposals, artifacts, and verified attribution | `starting` -> `running` -> `waiting_for_user`/`proposal_ready` -> `complete`/`failed`/`canceled` | [DOC: docs/internal/AGENT-SESSIONS.md] |
| **Scenario** | Runtime scenario in the Vrooli ecosystem | `running`, `stopped`, `error`, `unknown` | [CODE: ui/src/types/domain.ts#Scenario] |
| **Capture** | Raw operator/agent observation (text + optional images) classified into a candidate kind | `pending` -> `classified` -> consumed (converted to backlog or discarded) | [CODE: api/internal/captures/io.go] |
| **Record** | Immutable narrative artifact of completed work (`trigger`, `approach`, `ruled_out`, `commit`, `files_changed`, `outcome`); mirrors `BacklogKind`; supports `supersedes` chains for amendments | Stub (auto-created on backlog completion) -> filled (one-shot via `records edit`) -> immutable (further changes require supersedes) | [CODE: api/internal/records/types.go] |
| **Event** | Append-only audit entry for entity state deltas (backlog status, record created/superseded, etc.); folded by stats engine | N/A (immutable) | [CODE: api/internal/eventlog/types.go] |

### Four-Entity Model

Swarm-manager's domain forms a four-entity pipeline that closes the recursive-learning loop:

1. **Captures** — raw operator/agent input; the front door for observations and ideas before they have shape.
2. **Backlog** — current-state work tracking; what is being done now, its plan reference, and its queue/execution lifecycle.
3. **Records** — narrative artifacts of completed work; what was learned, including hypotheses ruled out, files touched, commit, and outcome. Records are the **write side of the recursive-learning loop**: future agents query them via `ai-search query --kind fix` and `records search`. Stub records are auto-created on backlog terminal transitions (`review-decide --accept|--fail`) and filled by the executing agent; records can also be created for work that never touched the backlog.
4. **Events** — audit log of state deltas (backlog status changes, record creations and supersedes, etc.) consumed by the stats engine to surface throughput, regression rate, and visibility split.

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
   Queue backlog item (manual/scheduled/yolo) -> execution record -> declared Agent Manager workflow -> typed terminal result -> Swarm authorized apply
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
   Agent Sessions support longer human-led planning and operations conversations inside Swarm Manager. Session details uses the shared composer also used by Quick Capture, with session-only context chips for existing backlog items, milestones, captures, executions, agent activity, scenarios, prior sessions, and the current operations briefing. Message context is resolved by the API before it reaches Agent Manager, and uploaded images are stored as session-owned attachments. Meta-orchestration sessions can create multiple milestones and backlog items through the batch apply seam. Swarm operations sessions receive a bounded `operations_briefing/latest` context by default, answer broad current-status questions from that packet first, then drill down through the operations/overview/stats commands only when needed. See [DOC: docs/internal/AGENT-SESSIONS.md].

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
- **Next** — actionable immediately: human gate cards (decide / review / classify, from the `internal/gates` read-model) plus runnable and needs-workshop item cards at dependency wave 0. Header bulk actions: Run all ready (threshold-confirmed) and Answer all (decision drawer).
- **Later** — not yet actionable, grouped by nearest blocker (gate-blocked groups sort above item-blocked), with honest ordinal wave badges from `depgraph.Waves` frontier peeling. Waves deeper than 5 collapse into a "beyond horizon" rollup; dependency cycles surface as diagnostics.
- **Done** — window-capped recent outcomes (1h–24h picker on the column header).

Filters (search / status / owner-type / lane / group-by / show-snoozed) live in a shared drawer and persist in URL query params. Snooze remains client-side (localStorage). The decision drawer hosts the full decision stream (`?drawer=decisions` deep link) and per-item scoped answering from decide gate cards. No drag: columns are derived, so cards act through explicit menus mapped to real levers (run / workshop / finalize / archive / status / snooze / focus).

**Navigation:** First-class `/plan` route; the default landing for `/`, `/graph/plan`, and all retired-surface redirects. Keyboard shortcut: `1`.

### Graph
**Purpose:** Full structural exploration, selection, inspector actions, and attention-filtered focus mode on one graph surface.

The default graph mode renders the topology projection (`GET /api/v1/graph?lens=topology`) on the node/edge canvas. Focus mode is a client-side filter over the same topology payload: nodes pass `computeNodeAttention` (pending decisions, review-ready, failures) and their milestone/scenario context is re-attached via `member_of`/`targets` edges. Node click applies BFS visual focus; the inspector panel offers per-entity actions.

**Navigation:** Graph tab at `/graph`; the board's per-card "Focus on graph" action and detail-page focus links navigate to `/graph?mode=focus&select=<node>`. Keyboard shortcut: `2`.

**Edges:** `depends_on`, `member_of`, `classified_as`, `targets`

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

## Historical Workshop Readiness Model

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

See [DOC: docs/guides/workshop-workflow.md] for the active operator contract.

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
├── overview/          # Aggregation endpoint (backlog + milestones + graph + stats)
├── captures/          # Capture CRUD and classification
├── promptcatalog/     # Canonical runtime prompt inventory and resolvers
├── workshop/          # Readiness scoring, round I/O
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
- `/api/v1/backlog/*` - backlog CRUD, queue, research (workshop)
- `/api/v1/backlog/batch` - batch create (all-or-nothing with dependency validation)
- `/api/v1/backlog/batch/queue` - batch queue (topologically sorted, dependency-aware)
- `/api/v1/milestones/*` - milestone CRUD with rollup status from member items
- `/api/v1/goals/*` - goal CRUD, targets, priority; each response carries the transitive-closure scope (progress %) and a p50/p80 ETA band
- `/api/v1/plan-import` - POST an existing `{plan_id}` or adopted `{source_path|markdown}` plan, choose `container: "items"` or `container: "milestone"`, and land idempotent plan-bound work with `plan_ref` populated and created/linked/updated counts
- `/api/v1/plan-import/plans` - list canonical plan-manager plans for the Create-Work-From-Plan picker
- `/api/v1/execution/auto-drain` - GET/PUT the continuous goal-directed auto-enqueue toggle (default OFF; a scenario-local flag, not a proto setting)
- `/api/v1/overview` - aggregated view (backlog, milestones, dependency graph, summary stats)
- `/api/v1/operations/brief` - bounded current operations briefing for CLI, UI, and Swarm operations session prompts
- `/api/v1/graph?lens=topology` - the topology projection (Graph focus mode filters it client-side)
- `/api/v1/plan?window_seconds=...` - the Plan board projection (waves + gates read-model)
- `/ws/graph` - graph invalidation and node pulse websocket
- `/api/v1/captures/*` - capture CRUD and AI classification
- `/api/v1/scenarios/*` - scenario list/detail/lifecycle/delete/archive
- `/api/v1/settings/*` - settings persistence
- `/api/v1/queue/*` - queue state operations
- `/api/v1/execution/*` - execution runs and policy operations
- `/api/v1/agent-activities/*` - tracked agent activity history and active runtime telemetry
- `/api/v1/prompts/*` - prompt catalog, skill CRUD, versions, revert, preview, simulate
- `/api/v1/agent-manager/status` - agent-manager availability
- `/vrooli.swarm_manager.v1.discovery.DiscoveryService/GetAudioToolsEndpoint` - **Connect-RPC**; resolves audio-tools' base URL for the browser. The only Connect-RPC surface in swarm-manager today; remaining REST domains tracked in `docs/internal/PROBLEMS.md`.

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
