# Workshop Refinement Workflow

The workshop system is a universal iterative refinement loop that transforms any backlog item (idea, research, fix, execute, chore) into a fully-specified, ready-for-execution plan. It replaces the earlier idea-only 3-phase pipeline (clarify/suggest/enhance) with a single loop that works across all 5 backlog kinds.

## End-to-End Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│  BacklogDetailsPage                                                 │
│  User clicks "Workshop" button on any backlog item                  │
│  ┌───────────────────────────────────────────────────────────┐      │
│  │  WorkshopPanel                                             │      │
│  │  Round N: questions + proposals + info                     │      │
│  │  [Answer questions]  [Accept/reject proposals]             │      │
│  │  [Run next round]                                          │      │
│  └───────────────────────────────────────────────────────────┘      │
│                                                                      │
│  ReadinessBar: 5-dimension radar showing progress toward ready       │
└─────────────────────────┬───────────────────────────────────────────┘
                          │ POST /api/v1/backlog/{kind}/{name}/research
                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│  API: Handler.Research()                                            │
│                                                                     │
│  1. Load workshop skill for this kind                               │
│  2. Substitute variables: {{ITEM_NAME}}, {{ITEM_FOLDER}}, ...       │
│  3. Spawn agent via agent-manager.SpawnBacklog()                    │
│  4. Agent writes workshop/round-N.json and optionally plan.md       │
└─────────────────────────┬───────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Filesystem artifacts                                               │
│                                                                     │
│  {kind}/{name}/                                                     │
│  ├── spec.json                    (item metadata)                   │
│  ├── plan.md                      (primary execution artifact)      │
│  └── workshop/                                                      │
│      ├── round-1.json             (round 1 output)                  │
│      ├── round-2.json             (round 2 output)                  │
│      └── ...                                                        │
└─────────────────────────────────────────────────────────────────────┘
```

## The Workshop Loop

Each round is a self-contained iteration. The agent generates questions, proposals, and informational items based on the current state of the item and all prior rounds. The operator reviews and responds. Then the next round incorporates that feedback.

```
              ┌───────────────────────────────────────┐
              │                                       │
              ▼                                       │
       ┌──────────────┐                               │
       │  GENERATE    │  Agent produces round N       │
       │  ROUND       │  with questions, proposals,   │
       │              │  and readiness scores          │
       └──────┬───────┘                               │
              │                                       │
              ▼                                       │
       ┌──────────────┐                               │
       │  OPERATOR    │  User answers questions,      │
       │  REVIEWS     │  accepts/rejects proposals    │
       │              │                               │
       └──────┬───────┘                               │
              │                                       │
              ▼                                       │
       ┌──────────────┐     readiness < 3 on          │
       │  READINESS   │     any dimension ────────────┘
       │  CHECK       │
       └──────┬───────┘
              │ all dimensions >= 3
              ▼
       ┌──────────────┐
       │  READY       │  plan.md finalized,
       │              │  item can be queued
       └──────────────┘
```

There is no fixed number of rounds. The loop continues until readiness is achieved or the operator decides to queue manually.

## Workshop Round Format

Each round is stored as `workshop/round-N.json` with this schema:

```json
{
  "round": 1,
  "generated_at": "2026-03-20T14:30:00Z",
  "readiness": {
    "problem_clarity": 2,
    "scope_defined": 1,
    "approach_solid": 2,
    "testable": 0,
    "risk_awareness": 1
  },
  "items": [
    {
      "id": "q1",
      "type": "question",
      "text": "What authentication method should be used?",
      "context": "The spec mentions user accounts but no auth details.",
      "options": ["OAuth 2.0", "JWT tokens", "Session-based"],
      "answer": null
    },
    {
      "id": "p1",
      "type": "proposal",
      "text": "Use WebSocket instead of polling for real-time updates",
      "details": "Rationale and implementation approach...",
      "decision": "pending",
      "notes": null
    },
    {
      "id": "i1",
      "type": "info",
      "text": "The existing auth-service scenario already supports OAuth 2.0"
    }
  ],
  "plan_updates": "Optional text describing changes the agent made to plan.md"
}
```

### Item Types

| Type | Purpose | Operator Action |
|------|---------|-----------------|
| `question` | Reduce ambiguity; gather missing information | Provide an answer (free text or pick from options) |
| `proposal` | Suggest an improvement or design decision | Accept or reject (with optional notes) |
| `info` | Surface relevant context the operator should know | Read-only |

## Readiness Dimensions

Every round includes the agent's assessment of item readiness across 5 universal dimensions, each scored 0-3:

| Dimension | What it measures | Score meaning |
|-----------|-----------------|---------------|
| `problem_clarity` | Is the problem well-understood? | 0=vague, 3=crystal clear |
| `scope_defined` | Are boundaries and deliverables defined? | 0=unbounded, 3=fully scoped |
| `approach_solid` | Is the technical approach viable? | 0=no approach, 3=proven path |
| `testable` | Can success be verified? | 0=no criteria, 3=concrete test plan |
| `risk_awareness` | Are risks identified and mitigated? | 0=unknown, 3=risks addressed |

An item is considered **ready** when all 5 effective scores reach 3.

## Boost Formula

Raw readiness scores improve naturally as the agent learns more about the item. To reward iterative engagement, the system applies a **round-based boost** to dimensions that have already reached a baseline:

```
effective = raw >= 2 ? min(3, raw + floor(rounds_completed / N)) : raw
```

Where `N` (the boost divisor) varies by backlog kind:

| Kind | N | Rationale |
|------|---|-----------|
| idea | 2 | Ideas need more rounds to de-risk |
| research | 2 | Research similarly benefits from iteration |
| fix | 1 | Fixes are typically well-scoped, boost faster |
| execute | 2 | Execution tasks need careful scoping |
| chore | 1 | Chores are straightforward, boost faster |

This means a dimension with raw score 2 will reach effective score 3 after N additional rounds, even if the raw score does not change. Dimensions below 2 are not boosted (they need substantive improvement first).

## How to Trigger Workshop Rounds

### Via the UI

1. Open any backlog item from the Backlog page
2. Click the **Workshop** button in the item detail view
3. The WorkshopPanel shows the latest round's items
4. Answer questions and accept/reject proposals
5. Click **Run next round** to trigger the agent

### Via the API

```
POST /api/v1/backlog/{kind}/{name}/research
Content-Type: application/json

{
  "mode": "workshop"
}
```

The API loads the appropriate workshop prompt skill, substitutes item context variables, and spawns the agent via agent-manager.

## Item Lifecycle: Backlog to Execution

```
backlog ──workshop──▶ researching ──readiness met──▶ ready ──queue──▶ queued ──run──▶ in_progress
                           │                                                           │
                           │ (workshop loop continues                                  ▼
                           │  until ready or manual queue)                    completed / failed
                           │
                           └──▶ archived (operator decision)
```

1. **backlog**: Item exists but has not been workshopped
2. **researching**: Workshop agent is running (one or more rounds in progress)
3. **ready**: All 5 readiness dimensions are at effective score 3; `plan.md` is finalized
4. **queued**: Operator has queued the item for execution
5. **in_progress**: Execution agent is building/implementing the plan
6. **completed/failed**: Execution finished

Operators can queue items before they reach `ready` status if they judge the plan is sufficient.

## plan.md: The Primary Execution Artifact

The workshop loop's main output is `plan.md`, a structured implementation plan that serves as the specification handed to the execution agent (Generator/Improver). The agent updates `plan.md` with each round, incorporating answers to questions and accepted proposals.

The mandatory section structure, convergence patterns, quality gates, and guardrails for `plan.md` are defined by the `implementation-plan-authoring` skill (`prompt-manager skill read implementation-plan-authoring`). Workshop agents load this skill as required reading and follow it when creating or updating plans.

The execution agent reads `plan.md` as its primary input. Workshop rounds are supporting evidence, not the execution spec.

For `idea` backlog items, swarm-manager also generates a derived `handoff/` package when execution begins:

- `handoff/brief.md`
- `handoff/manifest.json`
- `handoff/source-index.json`

That package is not a separate planning surface. It is a frozen execution bridge into ecosystem-manager, regenerated from the latest finalized backlog state so downstream task creation can preserve the full context without re-reading raw workshop artifacts.

## Where Workshop Fits in the Pipeline

```
 Raw backlog item
    │
    ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  WORKSHOP    │ ──▶ │  GENERATOR   │ ──▶ │  IMPROVER    │
│  Iterative   │     │  Build       │     │  Iterate     │
│  refinement  │     │  scenario    │     │  improvements│
│  → plan.md   │     │  from plan   │     │              │
└──────────────┘     └──────────────┘     └──────────────┘
  Status: backlog      Status: queued       Status:
    → researching        → in_progress       in_progress
      → ready              → completed
```

## Implementation References

### Backend
- [CODE: api/internal/workshop/workshop.go] -- Readiness computation, round I/O, boost formula
- [CODE: api/internal/backlog/workshop.go] -- Workshop handler integration with backlog domain
- [CODE: api/internal/backlog/handler.go#Research] -- HTTP handler, prompt fetching, agent spawn
- [CODE: api/internal/backlog/maturity_summary.go] -- Maturity/readiness aggregation endpoint
- [CODE: api/internal/backlog/feedback_summary.go] -- Pending feedback aggregation endpoint

### Frontend
- [CODE: ui/src/lib/workshop-files.ts] -- Round parsing, truncation recovery, metrics
- [CODE: ui/src/types/domain.ts#WorkshopRound] -- Type definitions for rounds, items, readiness
- [CODE: ui/src/services/backlog-service.ts#research] -- API client method

### Prompts (managed by prompt-manager and cataloged in swarm-manager)
- `swarm-manager-workshop` -- Workshop rounds for `idea`, `fix`, `execute`, and `chore`
- `swarm-manager-workshop-research` -- Workshop rounds for `research`
- `swarm-manager-initialize-backlog` -- First-round bootstrap for every backlog kind
- `swarm-manager-workshop-finalize` -- Finalize rounds for non-research backlog kinds
- `swarm-manager-workshop-research-finalize` -- Finalize rounds for research backlog items
- [DOC: docs/reference/api-endpoints.md#prompts] -- Catalog and simulation endpoints
