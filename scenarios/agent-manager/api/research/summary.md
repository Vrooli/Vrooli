# Research Summary: Run Mapper with Fog/Total Progress

## Overview

This backlog item proposes adding a **codebase mapper** feature to agent-manager that tracks agent exploration progress using a "fog of war" metaphor, where unexplored areas remain hidden until the agent visits them.

## Current State

### Existing Progress Tracking

Agent-manager already has progress tracking infrastructure:

1. **Run Progress** (`domain/types.go:1284-1306`):
   - `RunProgress` struct with `Phase`, `PercentComplete`, `CurrentAction`
   - `ElapsedTime`, `EstimatedRemaining`, `LastUpdate` fields
   - `PhaseToProgress()` converts phases to percentage (0-100%)

2. **Progress Events** (`domain/types.go:830-858`):
   - `ProgressEventData` includes phase, percent, action, turns, tokens, elapsed time
   - Events broadcast via WebSocket for real-time UI updates

3. **Heartbeat System** (`orchestration/run_executor.go:417-480`):
   - Regular heartbeat updates during execution
   - Phase tracking with checkpoint persistence

### Limitations

Current progress tracking is:
- **Phase-based only**: Tracks which execution phase (initializing, executing, etc.)
- **No codebase awareness**: Doesn't know what files/directories agent has explored
- **Linear percentage**: Progress is a single 0-100% value based on execution phase
- **No "fog of war"**: No concept of known vs unknown areas

## Proposed Feature: Codebase Mapper

### Concept

A **mapper** that tracks which parts of the codebase an agent has explored during a run:

| Term | Definition |
|------|------------|
| **Fog** | Unexplored areas of the codebase (files/dirs not yet accessed) |
| **Explored** | Files/directories the agent has read or modified |
| **Total** | The full scope of the codebase or task scope |
| **Progress** | Ratio of explored to total (with fog indicating uncertainty) |

### Use Cases

1. **Investigation runs**: Track what the agent has examined during debugging
2. **Scope validation**: Ensure agent stays within allowed paths
3. **Resume context**: Know what agent has already seen for resumption
4. **User visibility**: Show what percentage of relevant codebase was analyzed

### Implementation Approach

#### Option A: File-Level Tracking (Recommended)

Track individual file accesses from tool events:

```go
type CodebaseMap struct {
    ExploredFiles    map[string]time.Time  // file path -> first access time
    ModifiedFiles    map[string]time.Time  // file path -> first modification
    TotalScopeFiles  int                   // total files in scope
    ExploredPercent  float64               // explored / total
}
```

**Pros**: Simple, leverages existing RunEvent stream
**Cons**: Requires knowing total scope file count upfront

#### Option B: Directory-Tree Tracking

Track exploration as a tree with fog at directory level:

```go
type DirectoryNode struct {
    Path       string
    Status     ExplorationStatus  // fog, partial, explored
    Children   []*DirectoryNode
    FileCount  int
    ExploredCount int
}
```

**Pros**: Hierarchical view, better fog metaphor
**Cons**: More complex, may not match how agents work

#### Option C: Hybrid Tracking

Combine file-level precision with directory-level aggregation:

```go
type MapperState struct {
    Files       map[string]FileStatus
    Directories map[string]DirSummary
    TotalScope  ScopeStats
    FogPercent  float64  // (total - explored) / total
}
```

## Data Sources

Mapper would extract exploration data from existing events:

1. **Tool Call Events** (`EventTypeToolCall`):
   - `Read` tool → file explored
   - `Glob`/`Grep` tools → directory searched
   - `Write`/`Edit` tools → file modified

2. **Tool Result Events** (`EventTypeToolResult`):
   - Success/failure of file access
   - Files listed in search results

3. **Message Events** (`EventTypeMessage`):
   - Agent reasoning may reference files

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Scope calculation overhead** | High latency on large repos | Lazy scope calculation, caching |
| **File path normalization** | Inconsistent tracking | Standardize all paths through helper |
| **Event stream parsing** | Brittle if tool format changes | Abstract tool-specific parsing |
| **Storage growth** | Large maps for big repos | Bloom filter for "seen" checks, periodic compaction |
| **Privacy concerns** | Exposing file structure | Respect allowed/denied paths from profile |

## Recommended Next Actions

1. **Define scope calculation**: How do we determine "total" files in scope?
   - Option: Use task's `ScopePath` and enumerate files
   - Option: Compute on-demand from first tool events

2. **Extend ProgressEventData**: Add mapper fields without breaking existing consumers:
   ```go
   type ProgressEventData struct {
       // ... existing fields ...
       MapperState  *MapperStateData `json:"mapperState,omitempty"`
   }
   ```

3. **Add event processor**: Subscribe to tool events and update map:
   ```go
   type MapperProcessor interface {
       OnToolCall(event *ToolCallEventData)
       OnToolResult(event *ToolResultEventData)
       GetState() *MapperState
   }
   ```

4. **Integrate with run executor**: Wire mapper into execution flow
5. **Add API endpoint**: Expose `/runs/{id}/map` for querying current map state
6. **UI visualization**: Render fog/explored areas (tree view or heatmap)

## Effort Estimate

| Component | Complexity | Notes |
|-----------|------------|-------|
| Domain types | Low | New structs, validation |
| Event processor | Medium | Parse tool events, update state |
| Scope calculator | Medium | File enumeration, caching |
| API endpoints | Low | Standard CRUD pattern |
| WebSocket integration | Low | Extend existing progress events |
| UI rendering | Medium-High | Tree visualization with fog |

**Total**: Medium complexity feature, estimated 2-3 sprints

## Related Work

- **Investigation runs** already track "what the agent looked at" conceptually
- **Recommendation extraction** parses event streams for insights
- **Sandbox diff** tracks what was modified (but not what was read)
