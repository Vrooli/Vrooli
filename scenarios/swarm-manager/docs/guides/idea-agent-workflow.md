# Idea Agent Workflow

The Idea Agent is a 3-phase AI refinement pipeline that transforms raw backlog ideas into actionable specifications. It runs within the backlog system and coordinates through filesystem artifacts.

## End-to-End Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│  BacklogDetailsPage                                                 │
│  User clicks "Idea Agent" button on an idea backlog item            │
│  ┌───────────────────────────────────────────────────────────┐      │
│  │  backlog-agent-dialog                                     │      │
│  │  ○ Clarify   ○ Suggest   ○ Enhance                        │      │
│  │  [Optional additional context]                            │      │
│  │  [Run Agent]                                              │      │
│  └───────────────────────────────────────────────────────────┘      │
└─────────────────────────┬───────────────────────────────────────────┘
                          │ POST /api/v1/backlog/{kind}/{name}/research
                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│  API: Handler.Research()                                            │
│                                                                     │
│  1. Parse mode (clarify | suggest | enhance)                        │
│  2. Load prompt template: prompts/workflow/{mode}.md                │
│  3. Substitute variables: {{ITEM_NAME}}, {{ITEM_FOLDER}}, ...       │
│  4. Spawn agent via agent-manager.SpawnBacklog()                    │
└─────────────────────────┬───────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Agent writes to filesystem                                         │
│                                                                     │
│  ideas/{name}/                                                      │
│  ├── spec.json                  (item metadata)                     │
│  ├── clarify/questions.json     (phase 1 output)                    │
│  ├── suggest/suggestions.json   (phase 2 output)                    │
│  └── enhance/summary.md         (phase 3 output)                   │
└─────────────────────────────────────────────────────────────────────┘
```

## The Three Phases

The phases are designed to run sequentially, but users can skip or re-run any phase.

```
 ┌──────────┐         ┌──────────┐         ┌──────────┐
 │ CLARIFY  │ ──────▶ │ SUGGEST  │ ──────▶ │ ENHANCE  │
 │          │         │          │         │          │
 │ Generate │  user   │ Propose  │  user   │ Synthesize│
 │ questions│ answers │ improve- │ accepts/│ final    │
 │          │ them    │ ments    │ rejects │ plan     │
 └──────────┘         └──────────┘         └──────────┘
      │                    │                    │
      ▼                    ▼                    ▼
 questions.json       suggestions.json      summary.md
```

After clarify, the user can choose to go directly to enhance (skipping suggest).

### Phase 1: Clarify

**Purpose**: Generate targeted questions to reduce ambiguity before implementation.

- **Prompt**: [DOC: prompts/workflow/clarify.md]
- **Output**: `clarify/questions.json`
- **UI**: [CODE: ui/src/components/backlog/idea-clarify-panel.tsx] — renders questions, collects answers

**Output schema** (`clarify/questions.json`):

```json
{
  "questions": [{
    "id": "q1",
    "question": "What authentication method should be used?",
    "category": "users|technical|scope|constraints|integration",
    "importance": "critical|important|nice-to-have",
    "options": ["OAuth 2.0", "JWT tokens", "Session-based"],
    "answer": ""
  }],
  "generated_at": "2024-01-15T10:30:00Z",
  "max_questions": 10
}
```

**Constraints**: 5-7 questions (max 10). Must preserve existing Q&A if re-run.

**User interaction**: After answering, user selects next mode:
- "Suggest" — generate improvements based on answers
- "Enhance" — skip suggestions, go directly to refined plan

### Phase 2: Suggest

**Purpose**: Propose improvements and alternative approaches for review.

- **Prompt**: [DOC: prompts/workflow/suggest.md]
- **Input**: `clarify/questions.json` (with answers, if available)
- **Output**: `suggest/suggestions.json`
- **UI**: [CODE: ui/src/components/backlog/idea-suggestions-panel.tsx] — renders suggestions with accept/reject controls

**Output schema** (`suggest/suggestions.json`):

```json
{
  "suggestions": [{
    "id": "s1",
    "suggestion": "Use WebSocket instead of polling",
    "details": "Rationale and implementation notes...",
    "category": "architecture|ux|scope|risk|opportunity",
    "impact": "high|medium|low",
    "status": "pending|accepted|rejected",
    "rejection_reason": ""
  }],
  "generated_at": "2024-01-15T10:30:00Z",
  "max_suggestions": 7
}
```

**Constraints**: Max 7 suggestions (quality over quantity). Must preserve prior decisions if re-run.

**User interaction**: Accept or reject each suggestion, then trigger enhance.

### Phase 3: Enhance

**Purpose**: Synthesize answered clarifications and accepted suggestions into a refined, actionable plan.

- **Prompt**: [DOC: prompts/workflow/enhance.md]
- **Input**: `clarify/questions.json` (with answers) + `suggest/suggestions.json` (with decisions)
- **Output**: `enhance/summary.md` (+ optional `spec.json` updates)
- **UI**: Rendered as markdown in the backlog details page

**Output structure** (`enhance/summary.md`):
1. Enhanced description incorporating clarifications
2. Clarifications Applied table
3. Suggestions Integrated section
4. Scope boundaries (included/excluded)
5. Implementation notes
6. Success criteria
7. Ready-for-processing checklist

## How Prompts Are Loaded

```
prompt-manager skill store
(packs/core/swarm-manager-{mode}-idea/)
         │
         ▼
 promptmanager.Client.ReadSkill(skillID, variables, withScope)
         │                              ← per-request via api-core discovery
         ▼
 POST /api/v1/skills/read              ← variable substitution by prompt-manager
         │
         ▼
 Final prompt string with item context
         │
         ▼
 agent-manager.SpawnBacklog(...)       ← Purpose: "research"
```

Prompts are managed as prompt-manager skills (e.g., `swarm-manager-clarify-idea`, `swarm-manager-suggest-idea`, `swarm-manager-enhance-idea`). Template variables (`{{ITEM_NAME}}`, `{{ITEM_TITLE}}`, etc.) are substituted by prompt-manager's skill read API.

The `promptmanager.Client` resolves the prompt-manager URL via `api-core/discovery` on each request.

## Where Idea Agent Fits

The Idea Agent is the first of three agent types in the backlog pipeline:

```
 Raw idea
    │
    ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  IDEA AGENT  │ ──▶ │  GENERATOR   │ ──▶ │  IMPROVER    │
│  Refine idea │     │  Build       │     │  Iterate     │
│  into spec   │     │  scenario    │     │  improvements│
└──────────────┘     └──────────────┘     └──────────────┘
  Status: backlog      Status: queued       Status:
    → researching        → in_progress       in_progress
      → ready              → completed
```

Once enhanced, the idea becomes `ready` status and can be queued for the Generator agent.

## Backlog Item Status Lifecycle

```
backlog → researching → ready → queued → in_progress → completed
                                                      → archived
```

The `researching` status is set while any Idea Agent phase is running.

## Implementation References

### Backend
- [CODE: api/internal/backlog/handler.go#Research] — HTTP handler, prompt fetching, agent spawn
- [CODE: api/internal/promptmanager/client.go] — prompt-manager API client

### Frontend
- [CODE: ui/src/pages/BacklogDetailsPage.tsx] — orchestrates agent mutations and followup chains
- [CODE: ui/src/components/backlog/backlog-agent-dialog.tsx] — mode selection dialog
- [CODE: ui/src/components/backlog/idea-clarify-panel.tsx] — clarify Q&A interaction
- [CODE: ui/src/components/backlog/idea-suggestions-panel.tsx] — suggestion review interaction
- [CODE: ui/src/lib/idea-agent-files.ts] — file parsing/serialization utilities
- [CODE: ui/src/services/backlog-service.ts#research] — API client method
- [CODE: ui/src/types/domain.ts#IdeaAgentMode] — type definitions

### Prompts (managed by prompt-manager)
- `swarm-manager-clarify-idea` — clarify phase skill
- `swarm-manager-suggest-idea` — suggest phase skill
- `swarm-manager-enhance-idea` — enhance phase skill
- [DOC: docs/guides/research-notes.md#references] — prompt-manager and integration references
