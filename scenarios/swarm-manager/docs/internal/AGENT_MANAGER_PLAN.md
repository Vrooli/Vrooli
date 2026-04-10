# Agent-Manager Integration Plan (Swarm Manager)

> Superseded (2026-02-14): This plan reflects a pre-greenfield design that included recommendation workflows.
>
> Current implementation direction is documented in `docs/internal/GREENFIELD_EXECUTION_CONTROL_PLAN.md` and centers on backlog + execution control without a Swarm Manager recommendation subsystem.

## Legacy Purpose

Provide a concrete, implementation-ready plan to upgrade swarm-manager’s agent-manager
integration to match the mature patterns used in `scenario-to-cloud` and `test-genie`.
This plan assumes a fresh chat agent with no prior context; it includes the current
state, target architecture, file-by-file actions, testing seams, and acceptance
criteria.

---

## 1) Current State (Swarm Manager)

### What exists today
- **Low-level HTTP client only**: `api/internal/agentmanager/client.go` can create tasks + runs
  for **idea research**.
- **Ideas handler calls the client directly**: `api/internal/ideas/handler.go` uses
  `agentmanager.Client` in `Research()`.
- **Tests exist for the client and ideas research seam**:
  `api/internal/agentmanager/client_test.go`, `api/internal/ideas/handler_test.go`.

### What’s missing
- **No AgentService layer** (profile management, run tagging, availability checks).
- **No profile initialization** at API startup.
- **No agent-manager status endpoint**.
- **No recommendation → agent run workflow**.
- Requirements `OT-P0-009` still not complete (workflow integration pending).

---

## 2) Reference Patterns to Copy

### `scenario-to-cloud`
- **Client + Service split**:
  - Client: `scenarios/scenario-to-cloud/api/agentmanager/client.go`
  - Service: `scenarios/scenario-to-cloud/api/agentmanager/service.go`
- **Service responsibilities**:
  - `Initialize()` with `EnsureProfile`
  - `IsEnabled()` / `IsAvailable()` / `ResolveURL()`
  - Run tagging + orchestration
- **Tests**: `api/agentmanager/service_test.go`

### `test-genie`
- **Service provides orchestration** (batch spawn, concurrency limits, tagging)
- **HTTP handlers call the service**, not the client
- **Status/WS endpoints** for agent-manager availability

**Key takeaway:** All domain logic should speak to an **AgentService interface**; HTTP/proto
details live in the integration package.

---

## 3) Target Architecture (Screaming Architecture)

### Design principles
1. **Domain modules should not know HTTP/proto/discovery**.
2. **Integration layer hides agent-manager API details**.
3. **Handlers depend on interfaces, not concrete clients**.
4. **Profile config is centralized and testable**.
5. **All side-effects run through agent-manager**.

### Proposed modules

```
api/internal/agentmanager/
  client.go        # HTTP + proto adapter (discovery, protojson)
  service.go       # domain-level orchestration (profiles, tagging, spawn)
  service_test.go  # profile defaults + buildProfile tests
```

### Domain-facing interface (example)

```
type Service interface {
  IsEnabled() bool
  IsAvailable(ctx context.Context) bool
  ResolveURL(ctx context.Context) (string, error)

  SpawnResearch(ctx context.Context, req ResearchRequest) (ResearchResult, error)
  SpawnRecommendation(ctx context.Context, req RecommendationRequest) (RecommendationResult, error)
}
```

Handlers (`ideas`, `recommendations`) should accept this **interface**, enabling mocks
without HTTP.

---

## 4) Implementation Plan (Phased)

### Phase A — Create AgentService layer
**Goal:** introduce a proper integration boundary.

1) **Add `service.go` in `api/internal/agentmanager/`**
   - Follow the structure of `scenario-to-cloud` / `test-genie`:
     - `AgentServiceConfig` (ProfileName, ProfileKey, Timeout, Enabled)
     - `Initialize(ctx, profileConfig)`
     - `IsEnabled`, `IsAvailable`, `ResolveURL`
     - `buildProfile(cfg)` + `DefaultProfileConfig()`
   - Profile config should include:
     - Runner type + model preset
     - Max turns + timeout
     - Allowed tools (safe defaults)
     - Permission flags (for research, likely `RequiresApproval = true` by default)

2) **Keep and evolve `client.go`**
   - Consider aligning with proto-based client (like other scenarios).
   - For minimal change, keep existing request format but wrap it in service.

3) **Add `service_test.go`**
   - Match patterns in `scenario-to-cloud/api/agentmanager/service_test.go`:
     - `TestDefaultProfileConfig`
     - `TestBuildProfile`

---

### Phase B — Replace direct client usage in Ideas handler
**Goal:** ideas package uses the service seam (not HTTP).

1) Update handler struct:
   - Replace `agentClient agentmanager.Client` with `agentService agentmanager.Service`.
2) Update constructor:
   - `NewHandlerWithClients(ideasDir, ecosystemClient, agentService)`
3) Update `Research()`:
   - Call `agentService.SpawnResearch(...)`
4) Update tests:
   - Replace `mockAgentClient` with `mockAgentService`
   - Ensure request fields are verified (title, scope path, prompt, tag).

---

### Phase C — Add recommendation run workflow
**Goal:** recommendations can spawn agent runs (manual or YOLO).

1) Extend `Recommendation` model:
   - `TaskID`, `RunID`, `StartedAt`, `StartedBy`, `AutoApproved` (optional)
   - Keep backward-compatible JSON normalization.

2) Add endpoint:
   - `POST /api/v1/recommendations/{id}/start`
   - Behavior:
     - Check settings: allow only in `suggestions` or `yolo` mode.
     - If already started, return 409 or no-op (decide).
     - Spawn via `agentService.SpawnRecommendation(...)`.
     - Persist TaskID/RunID + status update to `approved` if YOLO.

3) Update recommendations handler tests:
   - Start success
   - Start when agent-manager unavailable -> 503
   - Start when already started -> 409 (if chosen)

4) UI support (optional in this phase):
   - Add “Start” button on Recommendations page for pending items.

---

### Phase D — Wire service initialization + status endpoint
**Goal:** consistent system-level availability.

1) In `api/main.go`:
   - Instantiate `AgentService` with config.
   - Call `Initialize()` on startup (log warning on failure, non-fatal).

2) Add status endpoint:
   - `GET /api/v1/agent-manager/status`
   - Response: `enabled`, `available`, `url`, `profileId`

---

### Phase E — Requirements + docs updates
**Goal:** close OT-P0-009.

1) Update `requirements/09-agent-manager-integration/module.json`
   - Add validation refs to:
     - `service_test.go` tests
     - handler tests for research + recommendation start

2) Update `docs/internal/SEAMS.md`
   - Add “agent-manager integration service” seam
   - Clarify responsibilities (handlers use service, not client)

---

## 5) Open Decisions (resolve before coding)

1) **Profile defaults**
   - Runner type (Codex vs Claude Code)
   - Allowed tools set
   - Permission model (`RequiresApproval` true/false)

2) **Tagging schema**
   - Proposed: `swarm-manager:idea:{idea}` and `swarm-manager:rec:{recId}`

3) **Recommendation start semantics**
   - If already started: 409 vs idempotent return
   - Status transitions on start (`pending → approved` in YOLO only)

4) **Inline config overrides**
   - Should UI allow model/runner overrides per research/recommendation?

---

## 6) Acceptance Criteria

### Functional
- Ideas research uses agent-manager via `AgentService`.
- Recommendations can spawn agent runs (manual + YOLO).
- Agent-manager profile is ensured at startup.
- Agent-manager availability is visible via status endpoint.

### Architectural
- Domain packages do not import HTTP/proto APIs.
- Agent integration logic lives entirely under `internal/agentmanager`.
- Tests rely on seams (mock service, not HTTP).

### Requirements
- `OT-P0-009` marked complete with passing validations.

---

## 7) Suggested Order of Execution (for a fresh agent)

1) Add `agentmanager/service.go` + tests.
2) Switch ideas handler to use service seam + update tests.
3) Add recommendation “start” endpoint + model fields + tests.
4) Wire service init + status endpoint in `api/main.go`.
5) Update requirements + docs.

---

## 8) File Checklist

**New**
- `api/internal/agentmanager/service.go`
- `api/internal/agentmanager/service_test.go`

**Modify**
- `api/internal/ideas/handler.go`
- `api/internal/ideas/handler_test.go`
- `api/internal/recommendations/handler.go`
- `api/internal/recommendations/handler_test.go`
- `api/main.go`
- `requirements/09-agent-manager-integration/module.json`
- `docs/internal/SEAMS.md`

---

## 9) Guardrails

- Do not add new dependencies.
- Keep JSON formats backward compatible.
- All agent spawning MUST route through agent-manager.
- Maintain filesystem-only persistence.
