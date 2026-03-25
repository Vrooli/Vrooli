# API Refactor — Implementation Plan

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

## 1. Purpose

Refactor the agent-manager API for improved structure, maintainability, and consistency. The primary driver is that several core files have grown monolithic (handlers.go: 2705 lines, service.go: 2909 lines, types.go: 1566 lines, main.go: 626 lines) making them difficult to navigate, review, and extend.

## 2. Problem Statement

The agent-manager API has organically accumulated functionality across a growing set of domains (profiles, tasks, runs, investigations, recommendations, settings, pricing, tools). As features were added, new handler methods, service methods, and types were appended to existing files rather than split into domain-cohesive modules.

**Symptoms:**
- `handlers/handlers.go` contains ~45 handler methods across 6+ domains in a single 2700-line file
- `orchestration/service.go` contains ~2900 lines mixing profile, task, run, investigation, recommendation, and settings logic
- `domain/types.go` holds all domain entities (1566 lines) without domain separation
- `main.go` contains a 264-line `createOrchestrator()` function doing all dependency wiring
- Route registration is split inconsistently: some routes in `RegisterRoutes()`, some in `main.go.setupRoutes()`

**Impact:**
- Merge conflicts when multiple features touch the same large files
- Difficult to reason about blast radius of changes
- Long file navigation for code review
- Hard to identify domain boundaries for future extraction

## 3. Scope

### In Scope
- Split monolithic Go source files into domain-cohesive modules
- Consistent file organization within each package
- Preserve all existing APIs, behavior, and test coverage
- Maintain backward compatibility (no API changes)

### Out of Scope
- Changing the API contract or HTTP routes
- Adding new features or endpoints
- Database schema changes
- Modifying the proto definitions
- Performance optimization
- Changing the adapter layer (runners, sandbox, etc.)

## 4. Current Technical Context

### Key Files / Components

| File | Lines | Role | Issue |
|------|-------|------|-------|
| `internal/handlers/handlers.go` | 2705 | All HTTP handlers | Monolithic — 45+ methods across 6 domains |
| `internal/orchestration/service.go` | 2909 | Core orchestration | Monolithic — profile/task/run/investigation/settings/recommendation |
| `internal/domain/types.go` | 1566 | Domain entities | All entities in one file |
| `main.go` | 626 | Server bootstrap + wiring | 264-line createOrchestrator, route setup split |
| `internal/handlers/handlers_test.go` | 1896 | Handler tests | Monolithic test file |

### Existing Structure (Good Patterns to Preserve)

- Clean adapter pattern: `internal/adapters/{runner,sandbox,event,recommendation,artifact}/`
- Interface-driven repositories: `internal/repository/interface.go`
- Separate pricing package: `internal/pricing/`
- Config package with clear concerns: `internal/config/`
- Domain validation separated: `internal/domain/validation.go`
- Proto conversion isolated: `internal/protoconv/`
- Good test helpers: `internal/testutil/`

### Package Dependency Flow
```
main.go → handlers → orchestration → domain
                   → protoconv     → adapters/*
                   → storage       → repository
```

## 5. Target End State

### handlers/ Package Split

```
internal/handlers/
├── handler.go          # Handler struct, New(), shared helpers, middleware
├── handler_profile.go  # CreateProfile, GetProfile, ListProfiles, UpdateProfile, DeleteProfile, EnsureProfile
├── handler_task.go     # CreateTask, GetTask, ListTasks, UpdateTask, CancelTask, DeleteTask
├── handler_run.go      # CreateRun, GetRun, ListRuns, DeleteRun, StopRun, ContinueRun, etc.
├── handler_investigation.go  # CreateInvestigationRun, CreateInvestigationApplyRun, recommendations
├── handler_settings.go # InvestigationSettings, OrchestrationSettings, ValidatePath
├── handler_runner.go   # GetRunnerStatus, ProbeRunner, GetRunnerModels, UpdateRunnerModels
├── handler_admin.go    # PurgeData, Health
├── handler_upload.go   # UploadAttachment, ServeUpload
├── pricing.go          # (already separate — keep)
├── stats.go            # (already separate — keep)
├── tools.go            # (already separate — keep)
├── websocket.go        # (already separate — keep)
├── routes.go           # RegisterRoutes consolidated from handlers + main.go
└── *_test.go           # Tests split to match source files
```

### orchestration/ Package Split

```
internal/orchestration/
├── service.go          # Service interface, Orchestrator struct, New(), config, shared internals
├── service_profile.go  # Profile CRUD operations
├── service_task.go     # Task CRUD operations
├── service_run.go      # Run CRUD, stop, continue, resume operations
├── service_investigation.go  # (already separate — investigation.go)
├── service_approval.go # (already separate — approval.go)
├── service_settings.go # Settings operations (if any exist in service.go)
├── run_executor.go     # (already separate)
├── run_actions.go      # (already separate)
├── recommendation*.go  # (already separate)
├── reconciler*.go      # (already separate)
├── terminator*.go      # (already separate)
├── stats.go            # (already separate)
└── *_test.go           # Tests split to match source files
```

### domain/ Package Split

```
internal/domain/
├── types.go            # Shared types (enums, small value objects)
├── profile.go          # AgentProfile entity
├── task.go             # Task entity
├── run.go              # Run, RunEvent, RunProgress entities
├── sandbox.go          # SandboxConfig, SandboxResult entities
├── investigation.go    # (already separate)
├── recommendation.go   # (already separate)
├── decisions.go        # (already separate)
├── errors.go           # (already separate)
├── validation.go       # (already separate)
├── tools.go            # (already separate)
└── *_test.go           # Tests split to match source files
```

### main.go Simplification

Extract `createOrchestrator()` to a dedicated bootstrap package or file:
```
main.go                 # Slim: Config, Server struct, main(), setupRoutes()
wire.go                 # createOrchestrator() and orchestratorDeps
middleware.go           # corsMiddleware, loggingMiddleware, isOriginAllowed, getAllowedOrigins
```

## 6. Implementation Strategy

### Phase 1: Split handlers.go (Highest Impact)
1. Create handler_*.go files by domain
2. Move each handler method to its domain file
3. Keep shared helpers (validateProto, writeJSON, parseUUID, etc.) in handler.go
4. Move RegisterRoutes to routes.go
5. Split handlers_test.go to match

### Phase 2: Split orchestration/service.go
1. Create service_profile.go, service_task.go, service_run.go
2. Move method groups to their files
3. Keep Service interface, Orchestrator struct, and New() in service.go
4. Split service_test.go to match

### Phase 3: Split domain/types.go
1. Create profile.go, task.go, run.go, sandbox.go
2. Move entity definitions and associated methods
3. Keep shared enums and small types in types.go
4. Split types_test.go to match

### Phase 4: Clean up main.go
1. Extract createOrchestrator to wire.go
2. Extract middleware to middleware.go
3. Consolidate route setup (move remaining routes from setupRoutes into handlers.RegisterRoutes)

## 7. Contract Decisions

- **No API contract changes**: All HTTP routes, request/response formats, and WebSocket protocol remain identical
- **No package boundary changes**: All splits are within existing packages (same `package` declarations)
- **No interface changes**: The `orchestration.Service` interface stays in service.go unchanged
- **Method signatures preserved**: All public and private method signatures remain identical

## 8. Testing Plan

- **Compilation check**: `go build ./...` must pass after each phase
- **Full test suite**: `go test ./... -timeout 300s` must pass after each phase
- **No test changes required**: Since we're only moving code between files in the same package, no import paths change
- **Verification**: `wc -l` on refactored files to confirm distribution

## 9. Rollout / Validation Checklist

- [ ] Phase 1 complete: handlers.go split, tests pass
- [ ] Phase 2 complete: service.go split, tests pass
- [ ] Phase 3 complete: types.go split, tests pass
- [ ] Phase 4 complete: main.go cleaned up, tests pass
- [ ] Final: `go build ./...` && `go test ./... -timeout 300s` pass
- [ ] Final: `gofumpt -l .` shows no formatting issues
- [ ] Final: No file exceeds ~800 lines (guideline, not hard limit)

## 10. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Git merge conflicts with in-flight work | Medium | Medium | Coordinate timing; do as single atomic commit per phase |
| Missed method move (compilation error) | Low | Low | `go build ./...` catches immediately |
| Test file split breaks test helpers | Low | Low | Helpers stay in original file or test_helpers_test.go |
| IDE/tooling confusion from many files | Low | Low | Consistent naming convention (handler_*.go, service_*.go) |

## 11. Non-goals / Prohibited Patterns

- Do NOT change any function signatures or behavior
- Do NOT add new abstractions, interfaces, or indirection layers
- Do NOT rename packages, types, or methods
- Do NOT add compatibility shims or deprecated aliases
- Do NOT "improve" code while moving it — pure mechanical split only
- Do NOT change import paths (all splits are within-package)

## 12. Definition of Done

1. All Go files in handlers/, orchestration/, domain/, and root package are split into domain-cohesive modules
2. No single source file exceeds ~800 lines (tests may be larger)
3. `go build ./...` passes
4. `go test ./... -timeout 300s` passes with identical results
5. `gofumpt -l .` shows no formatting issues
6. All existing API endpoints work identically (no behavioral changes)
