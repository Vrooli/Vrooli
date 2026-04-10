# Implementation Plan: Meta-Orchestration as First-Class Swarm Manager Feature

## 1. Purpose

Make meta-orchestration (high-level vision → initiatives + backlog items) a persistent, resumable, and attributable feature within swarm-manager. Currently, meta-orchestration runs as a one-shot Claude Code conversation skill with no persistence beyond a static `orchestration-summary.md`. This plan adds session storage, API endpoints, UI integration, and attribution linking so that planning conversations are preserved, resumable, and traceable to the items they produce.

## 2. Required Reading

```bash
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read scenario-generation
prompt-manager skill read swarm-manager-meta-orchestrator
prompt-manager skill read swarm-manager-backlog-tools
prompt-manager skill read api-steer storage-steer
prompt-manager skill read cli-steer seam-discovery-and-enforcement
prompt-manager skill read react-coherence ux
```

## 3. Greenfield Declaration

This is a **new feature** built on existing infrastructure. No backward compatibility shims or migration paths are needed. Existing capture and clarification flows remain unchanged.

## 4. Problem Statement

When planning the vrooli-events and notification-hub-greenfield initiatives, the meta-orchestrator skill produced 9 backlog items across 2 initiatives. However:
- The planning conversation was lost after the session — only `orchestration-summary.md` survived
- Executing agents lacked context about WHY decisions were made, causing drift
- There's no way to resume a planning session when priorities change
- Created items have no back-reference to their planning session
- The planning UX is disconnected from the swarm-manager UI

## 5. Scope

### In Scope
- New `OrchestrationSession` entity with folder-per-session JSON storage under `.vrooli/orchestration-sessions/{id}/`
- API endpoints for CRUD + continue/resume/fork
- Ollama-powered title generation (reusing agent-inbox pattern)
- "Planning" tab in sidebar with session list and conversation view
- Icon toggle on capture input for quick-capture vs meta-orchestration mode
- Conversation panel adapted from existing ClarificationPanel/ClarificationMessages
- Attribution linking (`meta_orchestration_id` + `meta_orchestration_context` on items/initiatives)
- ContinueRun with fallback to new spawn with injected conversation history
- Resume/fork capability for existing sessions

### Out of Scope
- Modifying the meta-orchestrator skill itself (it works well as-is)
- Real-time collaboration (multi-user on same session)
- Session branching/tree (fork creates a new linear session)
- Migration of existing orchestration-summary.md files
- Shared Ollama library between scenarios (each scenario maintains its own client)

## 6. Current Technical Context

### Key Files
- **Capture UI**: `ui/src/components/capture/quick-capture-input.tsx` — textarea with `onOpenForm` escape hatch
- **Chat UI**: `ui/src/components/backlog/clarification-panel.tsx` + `clarification-messages.tsx` — full chat UX with polling, staleness, markdown
- **Chat Hook**: `ui/src/hooks/useClarificationThread.ts` — 3s polling, staleness detection, message management
- **Polling Hook**: `ui/src/hooks/useStorePolling.ts` — generic interval polling with enabled/disabled control
- **Agent spawning**: `api/internal/agentmanager/service.go` — `SpawnBacklog()`, `ContinueRun()`
- **Activity tracking**: `api/internal/agentactivity/service.go` — wraps spawning with tracking records
- **Identity**: `api/internal/identity/provenance.go` — X-Agent-Identity-Token for attribution chain
- **Captures API**: `api/internal/captures/create.go` — multipart creation + agent classification
- **Meta-orchestrator skill**: `prompt-manager/store/skills/packs/core/swarm-manager-meta-orchestrator/SKILL.md`
- **Sidebar tabs**: `ui/src/surfaces/graph/components/sidebar/types.ts` — 5 tabs: activity, backlog, captures, initiatives, executions
- **Graph workspace**: `ui/src/surfaces/graph/components/GraphWorkspace.tsx` — HUD-like floating controls, detail pages as overlays
- **API routes**: `api/main.go` lines 102-132 — `registerXxxRoutes()` pattern with Handler.RegisterRoutes()
- **Backlog handler**: `api/internal/backlog/handler.go` — Handler struct with dependency injection and RegisterRoutes

### Storage Patterns
All swarm-manager persistence is JSON files on disk:
- Folder-per-entity: `{rootDir}/{kind}/{name}/spec.json` for backlog items
- Flat JSON: `.vrooli/agent-activities.json`, `.vrooli/execution-runs.json`
- Initiative files: `.vrooli/initiatives/{name}/initiative.json`

### Ollama Title Generation Pattern (from agent-inbox)
- File: `scenarios/agent-inbox/api/handlers/chat_naming.go` + `integrations/ollama.go`
- Model: `llama3.1:8b` (configurable via `OLLAMA_NAMING_MODEL`)
- Temperature: 0.3, MaxTokens: 20, Timeout: 2s (opportunistic)
- Prompt: "Generate a very short, descriptive title (3-6 words max)..."
- Fallback: "New Conversation" if Ollama unavailable
- Triggered after first message, not blocking session creation

### ContinueRun Pattern
- `service.go:ContinueRun()` → `client.go` POSTs to `/api/v1/runs/{runID}/continue` with `{"runId", "message"}`
- Simple pass-through to agent-manager; run must still be alive

## 7. Target End State

1. User clicks meta-orchestration icon toggle in capture area → conversation panel opens in Planning tab
2. Agent with meta-orchestrator skill engages in multi-turn planning conversation
3. Ollama generates a session title from the first few messages (non-blocking, fallback to "New Planning Session")
4. Full conversation is stored in `.vrooli/orchestration-sessions/{id}/session.json` with all messages
5. When agent creates items/initiatives, they carry `meta_orchestration_id` and a brief `meta_orchestration_context` string
6. User can browse past sessions in the Planning sidebar tab, resume them, or fork from any point
7. Resume tries ContinueRun first; if the run is stale, spawns a new agent with conversation history injected
8. UI shows attribution: "Created from planning session: {title}" on backlog items

## 8. Implementation Strategy

### Phase 1: Session Storage & Store (Backend)
**Goal**: Persistent session CRUD on disk.

- Define `OrchestrationSession` struct:
  ```go
  type OrchestrationSession struct {
      ID           string                `json:"id"`           // ULID
      Title        string                `json:"title"`        // Ollama-generated or fallback
      Status       string                `json:"status"`       // "active", "completed", "archived"
      Messages     []ConversationMessage `json:"messages"`
      CreatedItems []CreatedItemRef      `json:"created_items"`
      RunID        string                `json:"run_id"`       // current agent-manager run
      TaskID       string                `json:"task_id"`
      ForkedFrom   string                `json:"forked_from,omitempty"`
      CreatedAt    string                `json:"created_at"`
      UpdatedAt    string                `json:"updated_at"`
  }
  ```
- Storage path: `.vrooli/orchestration-sessions/{id}/session.json`
- Implement `OrchestrationStore` with: `Create`, `Get`, `List`, `Update`, `Delete`
- Follow existing store patterns (file locking, atomic writes)
- New package: `api/internal/orchestration/`

### Phase 2: Ollama Title Generation
**Goal**: Auto-generate session titles from conversation content.

- Port agent-inbox's Ollama integration pattern into swarm-manager:
  - New file: `api/internal/orchestration/naming.go`
  - Ollama client: `api/internal/orchestration/ollama.go` (or shared integrations package)
  - Config: `OLLAMA_NAMING_MODEL` (default `llama3.1:8b`), temperature 0.3, max 20 tokens
- Trigger: After first agent response, fire-and-forget goroutine
- Build conversation summary (max 10 messages, 200 chars each) → Ollama prompt → update session title
- Fallback: "New Planning Session" if Ollama times out (2s) or unavailable
- Non-blocking: session creation returns immediately with fallback title

### Phase 3: API Endpoints
**Goal**: REST API for session lifecycle.

- New handler: `api/internal/orchestration/handler.go`
- Endpoints:
  - `POST /api/v1/orchestration-sessions` — create session, spawn agent with meta-orchestrator skill, return session with fallback title
  - `GET /api/v1/orchestration-sessions` — list sessions (sorted by updated_at desc)
  - `GET /api/v1/orchestration-sessions/{id}` — get session with messages
  - `POST /api/v1/orchestration-sessions/{id}/continue` — append user message, ContinueRun (or fallback spawn), return updated session
  - `POST /api/v1/orchestration-sessions/{id}/fork` — create new session copying messages up to current point, spawn fresh agent with history
  - `PATCH /api/v1/orchestration-sessions/{id}` — update status (complete, archive)
  - `DELETE /api/v1/orchestration-sessions/{id}` — delete session and folder
- Register routes in `main.go` following existing `registerXxxRoutes()` pattern
- Inject `AgentService` and `OrchestrationStore` into handler

### Phase 4: Attribution Linking
**Goal**: Link created items/initiatives back to their planning session.

- Add `meta_orchestration_id` and `meta_orchestration_context` fields to backlog item spec and initiative spec
- Pass `VROOLI_META_ORCHESTRATION_ID` environment variable to spawned meta-orchestrator agent
- Meta-orchestrator skill reads env var and passes to `swarm-manager backlog batch-create` / `swarm-manager initiatives create`
- When agent creates items, the `/continue` handler polls for new items and updates session's `created_items` list
- API list endpoints gain `?orchestration_id=` filter parameter
- CLI: `swarm-manager backlog update` supports the new fields via `--data`

### Phase 5: UI — Planning Tab & Conversation Panel
**Goal**: Full UI for managing orchestration sessions.

- Add "planning" to sidebar tabs in `ui/src/surfaces/graph/components/sidebar/types.ts`
- Create sidebar tab component: `ui/src/surfaces/graph/components/sidebar/PlanningTab.tsx`
  - Session list with title, status badge, created date, item count
  - "New Session" button
  - Click session → open conversation view
- Create `OrchestrationPanel` component adapted from `ClarificationPanel`:
  - `ui/src/components/orchestration/orchestration-panel.tsx` — conversation UI
  - `ui/src/components/orchestration/orchestration-messages.tsx` — message thread (adapt from clarification-messages)
  - Show created items inline as cards when agent reports them
- Create Zustand store: `ui/src/stores/orchestration-store.ts`
  - Session list, active session, messages, polling state
  - localStorage persistence for draft messages
- Create service: `ui/src/services/orchestration/index.ts`
  - API client methods matching the endpoints
- Create hook: `ui/src/hooks/useOrchestrationThread.ts`
  - Adapt `useClarificationThread` pattern: polling, staleness, message management
- Add icon toggle to `quick-capture-input.tsx`:
  - New prop or internal state for mode ("capture" vs "orchestration")
  - In orchestration mode, submit opens Planning tab and creates session instead of capture

### Phase 6: Resume & Fork UX
**Goal**: Allow users to continue or branch planning sessions.

- Resume button on session detail → calls `/continue` endpoint
  - If ContinueRun succeeds: append message, resume polling
  - If ContinueRun fails (stale run): API spawns new agent with full message history as prompt context, updates `run_id`
- Fork button on session detail → calls `/fork` endpoint
  - Creates new session with messages copied, new agent spawned with history
  - Original session unchanged, new session linked via `forked_from`
- History injection format for fallback/fork: messages formatted as `User: ...\nAssistant: ...` blocks, prepended to the meta-orchestrator skill prompt

### Phase 7: Final Cleanup
- End-to-end testing
- `vrooli scenario restart swarm-manager`

## 9. Contract Decisions

### Session Storage Contract
```
.vrooli/orchestration-sessions/
  {ulid}/
    session.json     # OrchestrationSession
```

### API Contracts
```
POST   /api/v1/orchestration-sessions           → 201 { session }
GET    /api/v1/orchestration-sessions            → 200 { sessions: [] }
GET    /api/v1/orchestration-sessions/{id}       → 200 { session }
POST   /api/v1/orchestration-sessions/{id}/continue → 200 { session }
POST   /api/v1/orchestration-sessions/{id}/fork  → 201 { session }
PATCH  /api/v1/orchestration-sessions/{id}       → 200 { session }
DELETE /api/v1/orchestration-sessions/{id}       → 204
```

### Data Model
```go
type OrchestrationSession struct {
    ID           string                `json:"id"`
    Title        string                `json:"title"`
    Status       string                `json:"status"` // "active", "completed", "archived"
    Messages     []ConversationMessage `json:"messages"`
    CreatedItems []CreatedItemRef      `json:"created_items"`
    RunID        string                `json:"run_id"`
    TaskID       string                `json:"task_id"`
    ForkedFrom   string                `json:"forked_from,omitempty"`
    CreatedAt    string                `json:"created_at"`
    UpdatedAt    string                `json:"updated_at"`
}

type ConversationMessage struct {
    Role          string   `json:"role"` // "user", "assistant"
    Content       string   `json:"content"`
    AttachmentIDs []string `json:"attachment_ids,omitempty"`
    Timestamp     string   `json:"timestamp"`
}

type CreatedItemRef struct {
    Kind string `json:"kind"`     // "idea", "fix", "execute", "chore"
    Name string `json:"name"`
    Type string `json:"type"`     // "backlog" or "initiative"
}
```

### Attribution Fields (on backlog items and initiatives)
```go
MetaOrchestrationID      string `json:"meta_orchestration_id,omitempty"`
MetaOrchestrationContext string `json:"meta_orchestration_context,omitempty"`
```

## 10. Testing Plan

### Unit Tests
- **Store tests** (`orchestration/store_test.go`):
  - Create session → verify file written to correct path
  - Get session → verify round-trip serialization
  - List sessions → verify sorted by updated_at desc
  - Update session → verify atomic write (no corruption on concurrent access)
  - Delete session → verify folder removed
- **Handler tests** (`orchestration/handler_test.go`):
  - POST create → spawns agent, returns session with 201
  - GET list → returns sessions sorted correctly
  - GET by ID → returns full session with messages
  - POST continue → appends message, calls ContinueRun
  - POST continue with stale run → falls back to new spawn with history
  - POST fork → creates new session with copied messages
  - PATCH → updates status
  - DELETE → removes session
- **Naming tests** (`orchestration/naming_test.go`):
  - Ollama available → returns generated title
  - Ollama timeout → returns fallback title
  - Ollama unavailable → returns fallback title
  - Summary builder → truncates long messages, limits message count
- **Attribution tests**:
  - Created item carries `meta_orchestration_id` when env var set
  - Filter by `?orchestration_id=` returns correct items

### Integration Tests
- Create session → send message → agent responds → verify messages stored
- Create session → agent creates backlog item → verify `meta_orchestration_id` on item
- Create session → let run go stale → resume → verify new agent spawned with history
- Fork session → verify new session has message history, original unchanged

### UI Component Tests
- OrchestrationPanel renders messages correctly
- Mode toggle switches capture input behavior
- Session list renders with correct sorting
- Created items appear inline in conversation

## 11. Rollout/Validation Checklist

- [ ] Session store creates/reads/updates/deletes correctly
- [ ] API endpoints return correct responses with proper status codes
- [ ] Agent spawned with meta-orchestrator skill and correct env vars
- [ ] ContinueRun works for multi-turn conversation
- [ ] Stale run detection triggers fallback spawn with history injection
- [ ] Created items carry `meta_orchestration_id` and `meta_orchestration_context`
- [ ] Ollama generates titles (or gracefully falls back)
- [ ] "Planning" tab appears in sidebar with session list
- [ ] Conversation panel shows messages with markdown rendering
- [ ] Mode toggle on capture input switches between capture and orchestration
- [ ] Resume loads prior messages and continues the run
- [ ] Fork creates new session with conversation history
- [ ] Attribution visible on backlog items in UI ("From planning session: X")
- [ ] `vrooli scenario restart swarm-manager` succeeds

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Long conversations exceed agent context window | Medium | High | Store summary alongside full messages; fallback spawn uses summary for very long sessions (>50 messages) |
| ContinueRun to stale agent fails | Medium | Medium | Automatic fallback: detect 404/timeout, spawn new agent with injected history |
| Ollama unavailable for title generation | Low | Low | Non-blocking with "New Planning Session" fallback; title can be regenerated later |
| Message storage grows large for long sessions | Low | Low | Paginate message retrieval in GET endpoint; lazy-load older messages in UI |
| Concurrent session modifications (race condition) | Low | Medium | File locking on session.json writes (follow existing store patterns) |
| Attribution linking breaks if meta-orchestrator skill doesn't propagate ID | Medium | High | Integration test verifying end-to-end attribution; skill reads VROOLI_META_ORCHESTRATION_ID env var |
| Agent-manager unavailable when creating session | Low | High | Return error to user with retry option; session saved in "pending" state |
| History injection for long conversations produces oversized prompts | Low | Medium | Cap injected history at last N messages + summary of earlier messages |

## 13. Non-goals / Prohibited Patterns

- Do NOT modify the existing capture flow — meta-orchestration is a new parallel path
- Do NOT introduce SQLite or any new storage backend — use existing JSON file patterns
- Do NOT add real-time websocket communication — polling is sufficient (matches existing patterns)
- Do NOT add session versioning/branching — fork creates a flat new session
- Do NOT create a shared Ollama library across scenarios — each scenario maintains its own integration
- Do NOT modify the meta-orchestrator skill to be session-aware — the API layer handles persistence, the skill just does planning

## 14. Definition of Done

1. A user can start a meta-orchestration session from the capture area icon toggle
2. The conversation is stored persistently in `.vrooli/orchestration-sessions/{id}/session.json`
3. Ollama generates a title after the first exchange (with graceful fallback)
4. Created items/initiatives carry `meta_orchestration_id` and `meta_orchestration_context`
5. A user can browse sessions in the Planning sidebar tab
6. A user can resume an existing session (ContinueRun or fallback spawn)
7. A user can fork a session to explore a different direction
8. Attribution is visible in the UI on items created by orchestration
9. All API endpoints and store have unit tests
10. Integration test verifies end-to-end attribution flow
11. `vrooli scenario restart swarm-manager` succeeds cleanly
