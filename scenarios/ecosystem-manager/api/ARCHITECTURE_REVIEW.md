# Ecosystem Manager API: Architectural Review & Improvement Plan

## Executive Summary

The ecosystem-manager API has grown into a sophisticated system with ~10,400 lines in the `queue` package alone. While the **autosteer** package demonstrates excellent architectural discipline (clean interfaces, SRP components, good testing seams), other parts of the codebase have accumulated technical debt that obscures the system's intent and creates maintenance burden.

---

## Part 1: Screaming Architecture Analysis

### What the Current Structure "Screams"

```
pkg/
├── agentmanager/    → "I manage agents"         ✓ Clear
├── api/dto/         → "I transfer data"         ✓ Clear
├── autosteer/       → "I steer automation"      ✓ Clear, well-structured
├── discovery/       → "I discover resources"    ✓ Clear
├── handlers/        → "I handle HTTP"           ✗ Generic, tells nothing about domain
├── insights/        → "I provide insights"      ✗ Orphaned types, logic lives elsewhere
├── internal/        → "I'm internal utils"      ✓ Appropriate
├── prompts/         → "I assemble prompts"      ✓ Clear
├── queue/           → "I am... everything?"     ✗ GOD PACKAGE - 35 files, 10k+ lines
├── ratelimit/       → "I detect rate limits"    ✓ Clear, focused
├── recycler/        → "I recycle tasks"         ✓ Clear
├── server/          → "I wire things together"  ✓ Appropriate composition root
├── settings/        → "I manage settings"       ✓ Clear
├── summarizer/      → "I summarize output"      ✓ Clear, focused
├── systemlog/       → "I log system events"     ✓ Clear
├── tasks/           → "I am tasks + shared kernel" ✗ Mixed responsibilities
└── websocket/       → "I manage websockets"     ✓ Clear
```

### Core Problem: The `queue` Package

The `queue` package is a **God Package** that violates screaming architecture:

| File | Lines | Actual Responsibility |
|------|-------|----------------------|
| `execution_manager.go` | 1,126 | Task execution orchestration |
| `execution.go` | 1,166 | **Legacy** task execution (duplicate!) |
| `insight_manager.go` | 921 | Insight generation & management |
| `processor.go` | 859 | Queue processing loop |
| `insights_generator.go` | 569 | More insight generation |
| `insights.go` | 523 | Even more insights |
| `execution_history.go` | 354 | Execution history storage |
| ... | ... | ... |

**What "queue" should scream:** "I process a queue of pending tasks"
**What it actually does:** Execution, insights, history, rate limiting, diagnostics, logging, Auto Steer integration, agent management, concurrency control, timeouts, and more.

---

## Part 2: Boundary of Responsibility Issues

### 2.1 The Processor God Object

`Processor` has **50+ methods** spanning multiple concerns:

```go
// Queue management (appropriate)
func (qp *Processor) ProcessQueue()
func (qp *Processor) Start() / Stop() / Pause() / Resume()

// Execution (should be separate)
func (qp *Processor) executeTask()
func (qp *Processor) executeTaskLegacy()  // DUPLICATE
func (qp *Processor) callClaudeCode()

// History (should be separate)
func (qp *Processor) LoadExecutionHistory()
func (qp *Processor) savePromptToHistory()
func (qp *Processor) saveOutputToHistory()

// Insights (should be separate)
func (qp *Processor) SaveInsightReport()
func (qp *Processor) getInsightDir()

// Agent registry (should be separate)
func (qp *Processor) registerExecution()
func (qp *Processor) unregisterExecution()
```

### 2.2 Cross-Package Type Leakage

Types are defined in wrong packages:

| Type | Current Location | Should Be |
|------|-----------------|-----------|
| `ClaudeCodeRequest/Response` | `tasks` | `agentmanager` |
| `ResourceInfo/ScenarioInfo` | `tasks` | `discovery` |
| `ExecutionHistory` | `queue` | `execution` (new) |
| `InsightReport` | `insights` | ✓ Correct |

### 2.3 Inconsistent Handler Organization

```
handlers/           → Generic HTTP handlers
autosteer/handlers  → AutoSteer-specific handlers (inconsistent!)
```

The `autosteer` package correctly co-locates its handlers with domain logic. Other domains should follow this pattern.

---

## Part 3: Legacy & Migration Artifacts

### 3.1 Active Legacy Code to Remove

| Location | Issue | Impact |
|----------|-------|--------|
| `queue/execution.go:149-450` | `executeTaskLegacy()` - 300+ lines | Duplicate execution path |
| `tasks/lifecycle.go:25-26` | Deprecated `Manual`, `ForceOverride` fields | API confusion |
| `discovery/scenarios.go:224` | "Legacy filesystem functions" comment | Dead code |
| `handlers/tasks_execution.go:134` | Legacy timestamped log parsing | Maintenance burden |
| `handlers/tasks_prompts.go:63` | "Legacy behavior" comment | Technical debt marker |

### 3.2 Deprecated Code Patterns

```go
// tasks/lifecycle.go - These should be removed
Manual        bool // deprecated: use IntentManual instead
ForceOverride bool // deprecated: use IntentReconcile when override is needed

// queue/processor.go:27
cmd *exec.Cmd // Deprecated: only used for legacy fallback
```

### 3.3 Migration Artifacts

```go
// autosteer/db_init.go:163 - Old migration code
return fmt.Errorf("failed to drop legacy execution feedback foreign key: %w", err)

// summarizer/summarizer_test.go:45 - Legacy test case name
func TestParseResultLegacyPartialMapsToSomeProgress(t *testing.T)
```

---

## Part 4: Testing Seam Analysis

### 4.1 Good Examples (autosteer package)

```go
// interfaces.go - Clean interface segregation
type ConditionEvaluatorAPI interface { ... }
type PhaseCoordinatorAPI interface { ... }
type IterationEvaluatorAPI interface { ... }
type PromptEnhancerAPI interface { ... }

// Compile-time interface assertions
var _ ConditionEvaluatorAPI = (*ConditionEvaluator)(nil)
```

### 4.2 Missing Interfaces

| Component | Has Interface? | Testability |
|-----------|---------------|-------------|
| `Processor` | ❌ No | Hard to mock |
| `ExecutionManager` | ❌ No | Hard to mock |
| `HistoryManager` | ❌ Partial | Some testability |
| `InsightManager` | ❌ No | Hard to mock |
| `Storage` | ✅ `StorageAPI` | Good |
| `Recycler` | ❌ No | Hard to mock |

### 4.3 Tight Coupling Examples

```go
// queue/processor.go - Direct struct dependency
type Processor struct {
    storage           *tasks.Storage        // Should be tasks.StorageAPI
    executionManager  *ExecutionManager     // No interface
    insightManager    *InsightManager       // No interface
    historyManager    *HistoryManager       // No interface
}
```

---

## Part 5: Duplication Analysis

### 5.1 Insight Code Fragmentation

**4 files, 2000+ lines doing insight-related work:**

```
pkg/insights/types.go        (110 lines) - Type definitions only
pkg/insights/applier.go      (281 lines) - Applies suggestions
pkg/queue/insights.go        (523 lines) - Fallback implementations
pkg/queue/insights_generator.go (569 lines) - Generation logic
pkg/queue/insight_manager.go (921 lines) - Management logic
```

**Solution:** Consolidate into `pkg/insights/` with proper interfaces.

### 5.2 Execution Code Duplication

**Two parallel execution paths:**

```
queue/execution.go:149 → executeTaskLegacy() - Old path
queue/execution_manager.go  → ExecuteTask() - New path
```

Both contain:
- Prompt assembly
- Auto Steer initialization
- Claude Code invocation
- Result processing
- History saving

**Solution:** Remove legacy path entirely.

---

## Part 6: Improvement Plan

### Phase 1: Remove Legacy Code (Low Risk, High Impact)

**Goal:** Reduce code surface area by ~1500 lines

1. **Remove `executeTaskLegacy()`** (queue/execution.go:149-450)
   - Verify ExecutionManager is used for all executions
   - Remove fallback path and dead code

2. **Remove deprecated lifecycle fields**
   - Remove `Manual`, `ForceOverride` from `tasks.TransitionRequest`
   - Update all callers to use `Intent` enum

3. **Clean up migration artifacts**
   - Remove legacy FK drop code from db_init.go
   - Remove legacy log parsing from handlers

### Phase 2: Extract Execution Domain (Medium Risk, High Impact)

**Goal:** Make architecture scream "I execute tasks"

```
pkg/execution/                    # NEW PACKAGE
├── types.go                      # ExecutionHistory, ExecutionResult
├── interfaces.go                 # ExecutorAPI, HistoryRepositoryAPI
├── manager.go                    # Core execution orchestration
├── history.go                    # History persistence
├── claude_executor.go            # Claude Code specific execution
└── manager_test.go
```

**Migration Steps:**
1. Create `pkg/execution/` package
2. Move `ExecutionManager` from queue → execution
3. Move `ExecutionHistory` types
4. Extract history methods from Processor
5. Define `ExecutorAPI` interface for testability
6. Update Processor to use interface

### Phase 3: Consolidate Insights (Medium Risk, Medium Impact)

**Goal:** Single source of truth for insights

```
pkg/insights/                     # CONSOLIDATED
├── types.go                      # ✓ Already exists
├── applier.go                    # ✓ Already exists
├── interfaces.go                 # NEW: InsightGeneratorAPI, InsightRepositoryAPI
├── generator.go                  # Merged from queue/insights_generator.go
├── manager.go                    # Merged from queue/insight_manager.go
├── repository.go                 # Storage layer
└── generator_test.go
```

### Phase 4: Slim Down Processor (Medium Risk, High Impact)

**Goal:** Processor only processes the queue

After Phase 2 & 3, Processor should only have:

```go
type Processor struct {
    storage      tasks.StorageAPI
    executor     execution.ExecutorAPI  // Interface
    insights     insights.ManagerAPI    // Interface
    concurrency  ConcurrencyManager
    rateLimit    RateLimitManager
}

// Core methods only
func (p *Processor) Start()
func (p *Processor) Stop()
func (p *Processor) ProcessQueue()
func (p *Processor) Wake()
```

### Phase 5: Relocate Misplaced Types (Low Risk, Low Impact)

1. Move `ClaudeCodeRequest/Response` → `agentmanager`
2. Move `ResourceInfo/ScenarioInfo` → `discovery`
3. Ensure `tasks` package only contains task-related types

### Phase 6: Unify Handler Organization (Low Risk, Medium Impact)

**Option A:** Move all handlers into domain packages (like autosteer)
```
autosteer/handlers.go    ✓ Already correct
execution/handlers.go    # NEW
insights/handlers.go     # NEW
tasks/handlers.go        # Move from handlers/tasks.go
```

**Option B:** Keep `handlers/` but organize by subdomain
```
handlers/
├── autosteer/     # Move from autosteer/handlers.go
├── execution/
├── insights/
├── tasks/
└── common.go
```

**Recommendation:** Option A - co-locate handlers with domain logic

---

## Implementation Priority

| Phase | Risk | Impact | LOC Change | Recommended Order |
|-------|------|--------|------------|-------------------|
| 1. Remove Legacy | Low | High | -1500 | **First** |
| 2. Extract Execution | Medium | High | ~+500, -800 | Second |
| 3. Consolidate Insights | Medium | Medium | ~+200, -600 | Third |
| 4. Slim Processor | Medium | High | -500 | Fourth (after 2,3) |
| 5. Relocate Types | Low | Low | ~0 | Fifth |
| 6. Unify Handlers | Low | Medium | ~0 | Last |

---

## Success Metrics

After implementation, the architecture should:

1. **Scream its intent:**
   - `execution/` → "I execute tasks"
   - `insights/` → "I generate insights"
   - `queue/` → "I process the queue"

2. **Have clear boundaries:**
   - No package with 10k+ lines
   - No struct with 50+ methods
   - Types defined in their natural domain

3. **Be testable:**
   - All major components have interfaces
   - Compile-time interface assertions
   - No tight coupling to concrete types

4. **Have no legacy code:**
   - No `// deprecated` comments
   - No `// legacy` code paths
   - No migration artifacts

---

## Appendix: Files to Touch

### Phase 1 Files
- `queue/execution.go` (delete legacy function)
- `tasks/lifecycle.go` (remove deprecated fields)
- `autosteer/db_init.go` (remove migration code)
- `handlers/tasks_execution.go` (remove legacy parsing)

### Phase 2 Files (New Package)
- Create `pkg/execution/`
- Modify `queue/processor.go`
- Modify `queue/execution_manager.go` → move
- Modify `queue/execution_history.go` → move

### Phase 3 Files
- Modify `pkg/insights/`
- Delete `queue/insights.go`
- Delete `queue/insights_generator.go`
- Delete `queue/insight_manager.go`
