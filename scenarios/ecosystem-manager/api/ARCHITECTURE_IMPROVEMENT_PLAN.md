# Ecosystem Manager API - Architecture Improvement Plan

## Executive Summary

The ecosystem-manager API has grown organically and now shows signs of architectural debt. While the `autosteer/` package demonstrates excellent design principles, the `queue/` package has become a monolith centered around a God Object (`Processor`). This plan addresses screaming architecture violations, boundary issues, duplicated code, and testability concerns.

---

## Current Architecture Analysis

### Package Structure (Current)
```
api/pkg/
├── agentmanager/      # Agent execution delegation
├── api/dto/           # Data transfer objects
├── autosteer/         # ✅ Well-designed: profiles, orchestration, metrics
├── discovery/         # Resource/scenario discovery
├── handlers/          # HTTP handlers (mixed concerns)
├── insights/          # Insight types only
├── internal/          # Small utilities (paths, slices, timeutil)
├── prompts/           # Prompt assembly
├── queue/             # ❌ MONOLITH: processor, execution, history, insights, rate limiting
├── ratelimit/         # Rate limit detection
├── recycler/          # Task recycling
├── server/            # HTTP server setup
├── settings/          # Settings management
├── summarizer/        # Text summarization
├── systemlog/         # Logging
├── tasks/             # Task types and storage
└── websocket/         # WebSocket management
```

### Screaming Architecture Violations

**Problem**: The folder structure screams "technical layers" not "domain concepts."

A stranger looking at `api/pkg/` would learn:
- "There's a queue" ❌
- "There are handlers" ❌
- "There are tasks" (vague) ❌

They should learn:
- "This manages automation loops" ✅
- "This handles steering and profiles" ✅
- "This executes tasks via agents" ✅

---

## Critical Issues Identified

### 1. God Object: `Processor` (CRITICAL)

**Location**: `queue/processor.go`

**Evidence**:
- 27 fields in the struct
- 86 methods spread across 23 files
- Files referencing Processor: `execution.go`, `execution_history.go`, `insights.go`, `processor_agents.go`, `processor_diagnostics.go`, `processor_logs.go`, `processor_ratelimit.go`, `processor_reconcile.go`, `processor_registry.go`, etc.

**Impact**:
- Impossible to test in isolation
- Changes ripple unpredictably
- New developers can't understand entry points

### 2. Duplicated Code

**`cooldownRemaining` function**:
- `tasks/lifecycle.go:431-441`
- `recycler/recycler.go:494-504`

Comment in lifecycle.go admits it:
```go
// cooldownRemaining is duplicated from recycler for now to keep lifecycle self-contained.
```

### 3. Deprecated/Legacy Code

**`taskExecution.cmd` field** (`queue/processor.go:27`):
```go
cmd *exec.Cmd // Deprecated: only used for legacy fallback, nil when using agent-manager
```

Also deprecated functions in `tasks/lifecycle.go:335,349`:
```go
// Deprecated: prefer ApplyTransition directly to maintain single-path behavior.
func ApplyManual...

// Deprecated: prefer ApplyTransition directly.
func ApplyRecycler...
```

### 4. Boundary Violations

**handlers → queue coupling** (`handlers/types.go`):
```go
import "github.com/ecosystem-manager/api/pkg/queue"

type ProcessesListResponse struct {
    Processes []queue.ProcessInfo `json:"processes"`  // Leaking internal type
```

**Two handler packages**:
- `handlers/` - general handlers
- `autosteer/handlers.go` - autosteer-specific handlers

### 5. Fallback Implementation Pattern

Multiple methods have this anti-pattern:
```go
func (em *ExecutionManager) GetExecutionDir(taskID, executionID string) string {
    if em.historyManager != nil {
        return em.historyManager.GetExecutionDir(taskID, executionID)
    }
    return filepath.Join(em.taskLogsDir, taskID, "executions", executionID)  // Duplicated logic
}
```

This creates:
- Maintenance burden (logic in two places)
- Testing complexity (must test both paths)
- Unclear ownership

### 6. Large Files (> 500 lines)

| File | Lines | Concern |
|------|-------|---------|
| `queue/execution_manager.go` | 1126 | Mixed execution lifecycle + Claude config + artifact saving |
| `queue/insight_manager.go` | 921 | Could split into generator + storage |
| `queue/processor.go` | 862 | Core loop + state + concurrency |
| `handlers/tasks.go` | 929 | Too many task-related endpoints |
| `queue/insights_generator.go` | 569 | Complex insight generation |
| `autosteer/metrics_refactor.go` | 612 | Refactor metrics collection |
| `autosteer/metrics_security.go` | 638 | Security metrics |

### 7. Testing Seams

**Good** (autosteer/):
- Clean interfaces (`ProfileServiceAPI`, `ExecutionEngineAPI`, `HistoryServiceAPI`)
- Interface assertions at compile time
- Mock implementations (`repositories_mock.go`)

**Poor** (queue/):
- `Processor` is difficult to mock
- Fallback implementations complicate testing
- No clear seams for unit testing individual behaviors

---

## Improvement Plan

### Phase 1: Cleanup (Low Risk, High Value) ✅ COMPLETED

#### 1.1 Remove Deprecated Code
- [x] ~~Remove `taskExecution.cmd` field and related `pid()` method~~ DEFERRED: Still used by legacy execution paths
- [x] Remove deprecated `StartPending` and `StopActive` wrapper functions from lifecycle.go
- [x] Refactored `processor.go` to use `ApplyTransition` directly

#### 1.2 Eliminate Code Duplication
- [x] Extract `cooldownRemaining` to `tasks/cooldown.go` as exported `CooldownRemaining` function
- [x] Update both `lifecycle.go` and `recycler.go` to use shared function

#### 1.3 Fix Boundary Violations
- [x] Remove queue import from handlers/types.go
- [x] Update handlers to use inline maps/types instead of queue types in responses
- [ ] DEFERRED: Move remaining queue types to DTOs (requires ProcessorAPI interface changes)

#### 1.4 Eliminate Fallback Implementations
- [x] Assessed: `HistoryManager` is always provided in production (created in NewProcessor)
- [ ] DEFERRED: Remove fallback logic once Phase 2 testing confirms no impact

### Phase 2: Decompose God Object (Medium Risk)

#### 2.1 Extract Execution Domain
```
queue/
├── processor.go          # ONLY scheduling loop + concurrency control
├── execution/
│   ├── manager.go        # Task execution lifecycle (FROM execution_manager.go)
│   ├── registry.go       # Running task tracking (FROM execution_registry.go)
│   └── history.go        # Execution history (FROM execution_history.go)
```

#### 2.2 Extract Insights Domain
```
insights/                  # MOVE from queue/
├── types.go              # Already exists
├── applier.go            # Already exists
├── generator.go          # FROM queue/insights_generator.go
├── manager.go            # FROM queue/insight_manager.go
└── storage.go            # New: insight persistence
```

#### 2.3 Extract Rate Limiting
```
ratelimit/
├── detector.go           # Already exists
├── limiter.go            # FROM queue/rate_limiter.go
└── pause_manager.go      # FROM processor_ratelimit.go logic
```

#### 2.4 Slim Down Processor
Target: Processor should have ~10 fields and ~20 methods:
- Start/Stop/Pause control
- Slot management
- Task scheduling loop
- Delegation to extracted components

### Phase 3: Screaming Architecture Refactor (Higher Risk)

#### 3.1 Rename to Domain-Focused Structure
```
api/pkg/
├── automation/           # RENAME from queue/
│   ├── loop.go          # Main automation loop (scheduling)
│   ├── slots.go         # Concurrency slot management
│   └── lifecycle.go     # Start/stop/pause
├── execution/            # NEW domain package
│   ├── manager.go       # Execute individual tasks
│   ├── registry.go      # Track running executions
│   ├── history.go       # Execution history
│   └── artifacts.go     # Logs, outputs, prompts
├── steering/             # KEEP autosteer/, rename for clarity
│   ├── profiles/        # Profile CRUD
│   ├── orchestration/   # Phase coordination
│   ├── metrics/         # Metrics collection
│   └── enhancement/     # Prompt enhancement
├── insights/             # Already domain-focused
├── recycling/            # RENAME from recycler/
├── tasks/                # Task domain (storage, types, lifecycle)
├── discovery/            # Resource/scenario discovery
├── settings/             # Configuration
├── agents/               # RENAME from agentmanager/
└── infrastructure/       # Technical concerns
    ├── http/            # FROM handlers/ + server/
    ├── websocket/       # Already exists
    ├── logging/         # FROM systemlog/
    └── paths/           # FROM internal/paths
```

#### 3.2 Consolidate Handlers
- [ ] Move autosteer handlers into unified `infrastructure/http/` package
- [ ] Group handlers by domain, not by HTTP concern
- [ ] Create handler registry for clean route setup

### Phase 4: Testing Infrastructure

#### 4.1 Add Integration Points
- [ ] Define clear interfaces at package boundaries
- [ ] Create test doubles for each domain
- [ ] Add integration test suite for cross-domain flows

#### 4.2 Improve Unit Testability
- [ ] Each extracted component should be testable in isolation
- [ ] Use constructor injection consistently
- [ ] Remove global state dependencies

---

## Priority Matrix

| Task | Risk | Value | Order |
|------|------|-------|-------|
| Remove deprecated code | Low | Medium | 1 |
| Fix code duplication | Low | Medium | 2 |
| Fix boundary violations | Low | High | 3 |
| Eliminate fallbacks | Low | Medium | 4 |
| Extract Insights domain | Medium | High | 5 |
| Extract Execution domain | Medium | High | 6 |
| Slim down Processor | Medium | Critical | 7 |
| Screaming architecture refactor | High | High | 8 |
| Handler consolidation | Medium | Medium | 9 |

---

## Success Metrics

After completing this plan:

1. **Processor** has ≤10 fields and ≤20 methods
2. **No file** exceeds 500 lines (excluding tests)
3. **Zero** duplicated functions
4. **Zero** deprecated code markers
5. **handlers/** has no imports from domain packages (only DTOs)
6. **New developer** can understand system from folder names alone
7. **Unit test coverage** ≥80% for extracted components

---

## Appendix: File Size Audit

### Files Requiring Attention (>400 lines, non-test)

```
queue/execution_manager.go     1126 lines  → Split into 3-4 files
queue/insight_manager.go        921 lines  → Move to insights/
handlers/tasks.go               929 lines  → Split by concern
queue/processor.go              862 lines  → Slim to ~300 lines
queue/insights_generator.go     569 lines  → Move to insights/
handlers/insights.go            518 lines  → Consider splitting
autosteer/metrics_security.go   638 lines  → Keep (domain complexity)
autosteer/metrics_refactor.go   612 lines  → Keep (domain complexity)
autosteer/templates.go          581 lines  → Keep (template definitions)
```

---

## Notes

- The `autosteer/` package is a **good reference** for how other packages should be structured
- The extraction phases can be done incrementally without breaking changes
- Each phase should include updating tests and documentation
- Consider feature flags for gradual rollout of refactored code paths
