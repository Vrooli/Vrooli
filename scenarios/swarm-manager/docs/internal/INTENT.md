# Scenario Intent: Swarm Manager

This document explains the purpose, design rationale, and intended behavior of the
Swarm Manager scenario. It answers the question: "Why does this exist, what is it
for, and when should it change?"

## Purpose

Swarm Manager is the **central command center** for the Vrooli scenario ecosystem.
It provides a unified interface for:

1. **Backlog Management** - Track research, ideas, fixes, and execution tasks from
   initial capture to ready-for-processing
2. **Scenario Catalog** - View and manage all deployed scenarios with status,
   priority, and completeness information
3. **Recommendations** - Surface system-generated improvement suggestions with
   approve/reject workflows
4. **Settings** - Configure system behavior including recommendation engine and
   insights features

## Design Philosophy

### Single Source of Truth

Swarm Manager does NOT implement business logic directly. Instead, it:
- Delegates agent work to `agent-manager`
- Delegates scenario operations to `ecosystem-manager`
- Integrates with other scenarios for specialized features (test-genie, etc.)

This design prevents duplication and ensures all scenario operations go through
proper lifecycle management.

### Git-Tracked Backlog

Backlog items are stored as folders in `scenarios/swarm-manager/{ideas,research,fix,execute}/`
with a `spec.json` file and optional context files. This design:
- Provides version control for backlog evolution
- Allows offline editing and review
- Integrates with existing Git workflows

### Three Deployment Surfaces

The scenario provides three ways to interact:

1. **UI (React)** - Visual management for operators and developers
2. **API (Go/Gin)** - Programmatic access for automation and integrations
3. **CLI (Go/Cobra)** - Terminal access for scripts and quick checks

All three surfaces share the same domain logic and provide consistent behavior.

## Module Responsibilities

### UI Components

| Module | Responsibility | NOT Responsible For | Code Reference |
|--------|---------------|---------------------|----------------|
| `pages/` | Page-level layout and user interaction | Data fetching (uses services) | [CODE: ui/src/pages/BacklogPage.tsx] |
| `services/` | API communication, request/response handling | HTTP implementation (uses api-client) | [CODE: ui/src/services/backlog-service.ts] |
| `lib/api-client.ts` | HTTP transport, error handling | Domain logic, UI state | [CODE: ui/src/lib/api-client.ts] |
| `config/` | Tunable settings with validated defaults | Implementation details | [CODE: ui/src/config/index.ts] |
| `types/` | Domain type definitions and constants | Presentation logic | [CODE: ui/src/types/domain.ts] |

### API Components

| Module | Responsibility | NOT Responsible For | Code Reference |
|--------|---------------|---------------------|----------------|
| `main.go` | Server setup, routes, filesystem persistence for backlog/settings/queue/recommendations; CLI-sourced scenario inventory | Business logic (delegated) | [CODE: api/main.go] |
| Health handler | Readiness/liveness checks | Domain operations | [CODE: api/main.go:69] |

### CLI Components

| Module | Responsibility | NOT Responsible For | Code Reference |
|--------|---------------|---------------------|----------------|
| `app.go` | Command registration, thin API wrappers | HTTP implementation (uses cli-core) | [CODE: cli/app.go] |
| `main.go` | Entry point | Application logic | [CODE: cli/main.go] |

## Key Flows

### Flow 1: Viewing the Backlog

```
User → BacklogPage → backlogService.list() → ApiClient.get() → API /backlog → Filesystem
                                                                  ↓
User ← BacklogPage (render) ← useQuery ← Promise<BacklogItem[]> ←──────────┘
```

**Key invariant**: If the API fails, the UI shows ErrorState (not empty state).

### Flow 2: Checking Scenario Status

```
User → scenariosService.list() → ApiClient.get() → API /scenarios → Vrooli CLI + completeness CLI
                                                                     ↓
User ← ScenariosPage (render) ← useQuery ← Promise<Scenario[]> ←─────┘
```

**Key invariant**: Scenarios always show current status (running/stopped/error).

### Flow 3: CLI Health Check

```
User → `swarm-manager status` → resolveV1Endpoint → HTTP GET /api/v1/health
                                                                  ↓
User ← formatted output ← healthResponse ← JSON ←─────────────────┘
```

**Key invariant**: CLI never exposes raw stack traces to users.

## When to Modify This Scenario

| Scenario | Action | Code Reference |
|----------|--------|----------------|
| Need to add new backlog status | Update `BacklogStatus` type in `types/domain.ts` | [CODE: ui/src/types/domain.ts#BacklogStatus] |
| Need to add new scenario field | Update `Scenario` type, API response, and UI | [CODE: ui/src/types/domain.ts#Scenario] |
| Need to change error behavior | Update `error-utils.ts` and `ERROR-SEMANTICS.md` | [CODE: ui/src/lib/error-utils.ts] |
| Need to add API endpoint | Add route in `api/main.go`, update services | [CODE: api/main.go#setupRoutes] |
| Need to add CLI command | Add to `registerCommands()` in `cli/app.go` | [CODE: cli/app.go#registerCommands] |
| Need to change retry behavior | Update `config/index.ts` | [CODE: ui/src/config/index.ts#dataFetchingConfig] |

## What NOT to Modify Here

| Need | Where to Go Instead |
|------|---------------------|
| Spawn agents | Use `agent-manager` scenario |
| Initialize scenarios | Use `ecosystem-manager` scenario |
| Run tests | Use `test-genie` scenario |
| Track issues | Use `app-issue-tracker` scenario |
| Scan PROBLEMS.md | Use `knowledge-observatory` scenario |

## Current Implementation Status

| Component | Status | Notes |
|-----------|--------|-------|
| UI - Backlog Page | Wired | List + create dialog + details + queue/research/convert |
| UI - Scenarios Page | Wired | List, filters, metadata updates, delete |
| UI - Recommendations Page | Wired | List + refresh + update status + empty states |
| UI - Settings Page | Wired | Settings persistence via API |
| API | Active | Backlog/scenarios/settings/recommendations/queue + health |
| CLI | Wired | Backlog + scenarios + recommendations + settings + queue |

For detailed progress, see `docs/PROGRESS.md`.

## Change Log

| Date | Author | Changes |
|------|--------|---------|
| 2026-01-28 | Claude Opus 4.5 | Initial intent documentation (Phase 7) |
