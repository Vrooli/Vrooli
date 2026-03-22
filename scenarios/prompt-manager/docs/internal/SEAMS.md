# Testing Seams

This document describes the testing seams in the prompt-manager scenario - deliberate boundaries where behavior can be substituted for testing.

## Overview

The prompt-manager uses interface-based design to create clear testing seams. Each domain handler depends on interfaces rather than concrete types, enabling mock implementations for unit testing.

## API Seams

### Handler Layer

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Handlers                             │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐           │
│  │ skills  │ │ agents  │ │  teams  │ │ testing │ ...       │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘           │
│       │          │          │          │                    │
│  ┌────┴──────────┴──────────┴──────────┴───────────────┐   │
│  │                   Interfaces                         │   │
│  │  SkillStore │ AgentStore │ TeamStore │ OllamaClient │   │
│  └────┬──────────┴──────────┴──────────┴──────┬────────┘   │
└───────┼─────────────────────────────────────────┼───────────┘
        │ Production                              │ Testing
        ▼                                         ▼
┌───────────────┐                        ┌───────────────┐
│ FileStore     │                        │ MockStore     │
│ (real impl)   │                        │ (test impl)   │
└───────────────┘                        └───────────────┘
```

### Key Interfaces

| Interface | Package | Purpose |
|-----------|---------|---------|
| `SkillStore` | `store` | CRUD operations for skills |
| `AgentStore` | `store` | CRUD operations for agents |
| `TeamStore` | `store` | CRUD operations for teams, roles, org charts, and inboxes |
| `RelationStore` | `store` | Team-member relations |
| `IndexStore` | `store` | Generated index management |
| `MetricsService` | `skills` | Usage tracking |
| `AISearchIndexer` | `skills` | AI search index updates for skills |
| `AIAgentIndexer` | `agents` | AI search index updates for agents |
| `AITeamIndexer` | `teams` | AI search index updates for teams |
| `AgentFileReader` | `search` | Content search across agent files |
| `TeamFileReader` | `search` | Content search across team shared files |
| `AgentStoreReader` | `aisearch` | Read-only agent access for AI indexing |
| `AgentSoulReader` | `aisearch` | SOUL.md content for agent embeddings |
| `TeamStoreReader` | `aisearch` | Read-only team access for AI indexing |
| `TeamRelReader` | `aisearch` | Team member relations for team embeddings |

### Search Service Seams

The `search` package provides dedicated seams for text and content search across all
entity types, isolating search behavior from HTTP handlers and the UI:

```
HTTP handler (/search/skills, /search/agents, /search/teams)
  ↓
search.Service / AgentSearchService / TeamSearchService
  ↓
SkillStore / AgentStore / TeamStore + file reader interfaces
```

**Agent Content Search** uses the `AgentFileReader` interface:
```go
type AgentFileReader interface {
    ListFiles(ctx context.Context, id string) ([]store.AgentFileEntry, error)
    ReadFile(ctx context.Context, id, path string) (string, error)
}
```

**Team Content Search** uses the `TeamFileReader` interface:
```go
type TeamFileReader interface {
    ListSharedFiles(ctx context.Context, id string) ([]store.TeamFileEntry, error)
    ReadSharedFile(ctx context.Context, id, path string) (string, error)
}
```

`SearchContent` (on all three services) is the authoritative boundary for line-level
matching behavior (case sensitivity, whole word matching, regex parsing), which makes
it testable without touching the HTTP layer or filesystem.

## Store Adapter Layer

The skills domain uses a `StoreAdapter` to bridge the legacy handler interface with the new file-based store:

```
┌────────────────────────────────────────────────────────┐
│              Skills Handler                             │
│  calls methods like LoadMetadata(), GetContent()       │
└────────────────────┬───────────────────────────────────┘
                     │ SkillStore interface
                     ▼
┌────────────────────────────────────────────────────────┐
│              StoreAdapter                               │
│  Converts folder-based calls to pack-based storage     │
│  ~120 lines of translation logic                       │
└────────────────────┬───────────────────────────────────┘
                     │ FileSkillStore interface
                     ▼
┌────────────────────────────────────────────────────────┐
│              FileSkillStore                             │
│  Per-skill directory structure:                        │
│  store/skills/packs/{pack}/{skill-id}/                 │
│    ├── skill.json                                      │
│    ├── SKILL.md                                        │
│    └── history.jsonl                                   │
└────────────────────────────────────────────────────────┘
```

### Change Detection Seam

The `StoreAdapter.SaveMetadata()` function includes a change detection seam that prevents
spurious updates when skill metadata hasn't actually changed. This is critical because
`FileSkillStore.Update()` always:
1. Increments the `revision` counter
2. Appends to `history.jsonl`
3. Writes `skill.json`

Without change detection, saving all skills in a pack (even unchanged ones) would cause
all skills to get revision increments and history entries.

**Implementation:**
- `metadataChanged(old, new Metadata) bool` - compares meaningful fields
- `stringSliceEqual(a, b []string) bool` - compares slice contents
- `stringPtrEqual(a, b *string) bool` - nil-safe pointer comparison

**Fields Compared:** Name, Description, Modes, Tags, Icon, Draft, TargetToolID, DefaultScope
**Fields NOT Compared:** ID (identity), File (derived), CreatedAt/UpdatedAt (timestamps)

**Testing:** The `MockSkillStore.UpdateCalls` slice tracks which skill IDs had `Update()`
called on them, allowing tests to verify that only changed skills are updated.

### Testing Implications

1. **Unit tests** can mock at the `SkillStore` level (adapter interface)
2. **Integration tests** should test the full adapter → FileStore chain
3. When debugging, determine whether the issue is in the adapter translation or underlying store
4. **Change detection tests** verify that `UpdateCalls` only contains IDs of actually-changed skills

## Circular Dependency Handling

The `skills`, `agents`, and `teams` packages cannot import `aisearch` (would cause circular
import: aisearch depends on these packages for store access).

**Solution:** Post-initialization setter injection

```go
// In main.go
skillHandlers.SetAIIndexer(aiSearchService)
agentHandlers.SetAIIndexer(aiSearchService)  // AIAgentIndexer interface
teamHandlers.SetAIIndexer(aiSearchService)   // AITeamIndexer interface
```

Each handler package defines its own narrow indexer interface:
- `skills.AISearchIndexer` — `IndexSkill`, `DeleteFromIndex`
- `agents.AIAgentIndexer` — `IndexAgent`, `DeleteAgentFromIndex`
- `teams.AITeamIndexer` — `IndexTeam`, `DeleteTeamFromIndex`

**Testing Impact:** Mock tests must call `SetAIIndexer()` or the async indexing code path won't be exercised.

## Org Chart + Inbox Seams

**Org chart validation** lives in `api/teams/org_chart.go` and is exercised through the Teams handlers. The
validation relies on the `RelationStore` seam (team membership) so tests can control membership without touching
the filesystem.

**Team inbox persistence** uses the `TeamStore` seam (`GetInbox`/`SetInbox`) with a dedicated file per member:

```
store/teams/{team-id}/members/{agent-id}/inbox.json
```

Mock stores can supply inbox content directly for unit tests without requiring file I/O.

## Heartbeat Seams

Heartbeat execution uses three explicit seams:

- **AgentClient interface** (`agent_client_iface.go`): `Executor`, `Scheduler`, `Handlers`, and
  `RunRegistry` depend on the `AgentClient` interface instead of the concrete `*AgentManagerClient`.
  Tests substitute `mockAgentClient` (in `mock_agent_client_test.go`) with configurable responses
  and error injection for all agent-manager API calls (CreateTask, CreateRun, GetRun, WaitForRun,
  StopRun, EnsureProfile, Health). The concrete `*AgentManagerClient` satisfies this interface
  and has a `testBaseURL` field for httptest-based integration tests.
- **Scheduler → Executor**: `heartbeat.Scheduler` depends on the `HeartbeatExecutor` interface, allowing
  tests to substitute fake executors to validate scheduling behavior without running agent-manager.
- **Scheduler → Config Store**: `heartbeat.Scheduler` depends on `HeartbeatConfigStore` to resolve
  per-member profile keys and enabled state at run time.

### Timeout Serialization Seam

Go's `time.Duration` serializes as nanosecond integers (e.g., `600000000000` for 10 minutes) via
`encoding/json`. Agent-manager uses `google.protobuf.Duration` which expects the protojson canonical
string format (e.g., `"600s"`). With `protojson.UnmarshalOptions{DiscardUnknown: false}`, the
nanosecond integer causes `EnsureProfile` to fail at startup, which then causes every `CreateRun`
to fail with "profile not found".

**Fix:** `AgentProfile.Timeout` is typed as `string` (not `time.Duration`). The `DurationToProtojson()`
helper converts Go durations to protojson format. This is validated by:
- `TestEnsureProfileRequest_ProtojsonCompatibility` — verifies timeout is a JSON string, not a number
- `TestEnsureProfileRequest_ProtojsonRoundTrip` — validates the full payload structure
- `TestDurationToProtojson` — unit tests for the helper function

### Contract Test Coverage

Cross-scenario payloads (prompt-manager → agent-manager) are validated by contract tests in
`client_contract_test.go`. These tests verify:
- All field names are snake_case (protojson UseProtoNames)
- Duration fields use protojson string format
- Enum fields use proto enum name strings (e.g., `RUNNER_TYPE_CODEX`)
- No unknown fields that would be rejected by `DiscardUnknown=false`

### EnsureProfile → CreateRun Dependency Chain

`Scheduler.Start()` calls `ensureProfile()` to create the heartbeat profile in agent-manager.
If this fails (e.g., due to a serialization bug), the profile is never created, and all subsequent
`CreateRun` calls fail with "profile not found". The `TestEnsureProfileFailure_CausesCreateRunProfileNotFound`
integration test validates this failure chain end-to-end.

Member cleanup is centralized in the team handlers:

- **Member cleanup**: `teams.Handlers.cleanupMemberData` unschedules heartbeats and deletes
  `store/teams/{team-id}/members/{agent-id}/` via the file store when available.
  Tests can assert scheduler calls while keeping file I/O isolated to temporary directories.

## Team Execution Seams

The team execution system introduces two seams for serialized per-team heartbeat execution:

### TeamExecutionManager Interface

[CODE: api/heartbeat/team_execution.go]

```go
type TeamExecutionManager interface {
    Enqueue(ctx context.Context, teamID, agentID, profileKey string) (*EnqueueResult, error)
    Status(teamID string) TeamExecutionStatus
}
```

The `Scheduler` and `Handlers` depend on this interface rather than the concrete `TeamExecutionStore`.
Tests can inject a mock implementation that tracks enqueue calls and returns predetermined results
without requiring a real executor or queue.

### Queue Persistence

[CODE: api/heartbeat/team_execution.go, api/heartbeat/team_execution_store.go]

Queue state is persisted to `{storeDir}/team-queue-{teamID}.json` following the same pattern as
`RunRegistry` (`run_registry.go`). The `TeamExecutionStore` accepts a `persistDir` parameter, so
tests point at `t.TempDir()` to isolate queue file I/O.

**Testing Impact:**
- Unit tests mock `TeamExecutionManager` to verify handler/scheduler behavior without real queue state
- Integration tests use `t.TempDir()` for persistence and verify recover-after-restart scenarios
- The `captureExecutor` pattern from `scheduler_test.go` is reused for queue dequeue tests

### Context Isolation

[CODE: api/heartbeat/team_execution.go]

`TeamExecutionContext.Enqueue` launches executor goroutines with `context.Background()` instead of
the caller's context. This is critical because the caller is typically an HTTP handler that returns
a 202 response immediately — by the time the executor calls `CreateTask`/`CreateRun` on
agent-manager, the request context would already be cancelled. Both the immediate-start path (line 99)
and the queued-dequeue path (line 153) use `context.Background()` for this reason.

**Testing:** `TestEnqueue_ExecutorUsesDetachedContext` verifies this by cancelling the caller context
immediately after Enqueue and asserting the executor receives a live context.

### HandoffExtractor

**Interface**: `heartbeat.HandoffExtractor`
**Purpose**: Extracts structured handoff content from raw run event data.
**Implementations**:
- `SentinelExtractor` (primary) — scans the last assistant message for a `## HANDOFF` markdown header
- `ChainExtractor` — composes multiple extractors in priority order; returns first non-empty result

**Adding a fallback strategy**: To add an LLM-based fallback (e.g., via Ollama), implement `HandoffExtractor.Extract()` and compose it with `ChainExtractor`:
```go
chain := NewChainExtractor(
    NewSentinelExtractor(),     // fast, no API call
    NewOllamaExtractor(client), // slower, uses LLM
)
```
Pass the chain to `NewExecutor()` as the `handoffExtractor` parameter.

### Executor Completion Callback

[CODE: api/heartbeat/executor.go]

The `Executor` has an `OnComplete func(teamID, agentID string)` field that the `TeamExecutionStore`
sets during initialization. When `waitForCompletion()` finishes, it calls `OnComplete` to trigger
queue dequeue. Tests can set this callback to verify completion notification behavior.

### Execute Failure → Queue Cleanup

[CODE: api/heartbeat/team_execution.go]

When `Enqueue` starts a goroutine to call `executor.Execute()`, the goroutine must call
`OnMemberComplete(agentID)` if Execute returns an error. Without this, the team's `running`
field stays set forever, blocking all future heartbeats for the team (409 "already queued").
This was a real production bug: `CreateRun` failed with "profile not found", but the queue
state was never cleaned up. The same pattern applies to both the immediate-start goroutine
and the queued-dequeue goroutine in `OnMemberComplete`.

**Testing:** `TestEnqueue_ExecuteFailure_ClearsRunningState` uses a `failingExecutor` to verify
the state is cleared and re-enqueue succeeds.

### CreateRun ProfileRef.Defaults

[CODE: api/heartbeat/executor.go, api/heartbeat/client.go]

The `CreateRunRequest.ProfileRef` must always include `Defaults` (a full `AgentProfile`).
Agent-manager's `resolveRunConfig` calls `EnsureProfile` internally, and if the profile
doesn't exist and no defaults are provided, it returns a validation error. The `EnsureProfile`
call at scheduler startup is non-fatal (logs warning, continues), so the profile may not
exist when `CreateRun` is called. Including defaults makes `CreateRun` self-sufficient.

`BuildDefaultProfile(profileKey)` is the exported constructor for the default heartbeat
profile, shared between `Scheduler.ensureProfile()` and `Executor.Execute()`.

**Testing:** `TestExecute_CreateRunIncludesDefaults` verifies the ProfileRef always has
non-nil Defaults with required fields populated.

---

## Interop Seams

The `interop` package provides a tool-agnostic conversion layer for bidirectional team
config translation between prompt-manager and external tools (currently Claude Code).

### Converter Interface

```go
type Converter interface {
    ToolID() string
    FromPMTeam(snapshot *PMTeamSnapshot) (*ToolTeamConfig, error)
    ToPMTeam(config *ToolTeamConfig) (*PMTeamImport, error)
    FormatSpawnPrompt(config *ToolTeamConfig, ctx SpawnContext) (string, error)
}
```

The `ClaudeCodeConverter` implements this interface. Tests verify roundtrip conversion
(PM → CC → PM) preserves essential data without requiring filesystem access.

### CC Config Reader Seam

The import handler (`teams.Handlers.ImportClaudeCode`) uses a `readCCConfig` function
field to read Claude Code team configs from disk. In production, `defaultReadCCConfig`
reads from `~/.claude/teams/{name}/config.json`. Tests inject a function that returns
in-memory JSON, enabling full handler testing without filesystem dependencies.

### CC Team Directory Listing Seam

The `ListAvailableCCTeams` handler uses a `listCCTeamDirs` function field to enumerate
Claude Code teams on disk. In production, `defaultListCCTeamDirs` reads subdirectories
of `~/.claude/teams/` and parses each `config.json` to count members. Tests inject a
function that returns in-memory team lists, keeping handler tests filesystem-independent.

### teamDocReader Interface

The export handler uses a local `teamDocReader` interface instead of asserting to
`*store.FileTeamStore`:

```go
type teamDocReader interface {
    GetResponsibilities(ctx context.Context, teamID, agentID string) (string, error)
    GetHeartbeatInstructions(ctx context.Context, teamID, agentID string) (string, error)
}
```

`MockTeamStore` in tests implements this interface, making the export handler fully
testable with mock stores.

### ParseCCConfig

`interop.ParseCCConfig(data []byte, teamNameFallback string)` centralizes CC config
JSON parsing, eliminating duplication between the import handler and the interop package.
Tests cover valid configs, fallback team names, invalid JSON, and empty members.

## Graph Seams

The graph system uses interface-based design for testability at three layers.

### graphBuilder Interface

[CODE: api/graph/index.go]

`GraphIndexStore` depends on the `graphBuilder` interface rather than the concrete `Builder`:

```go
type graphBuilder interface {
    Build(ctx context.Context) (Graph, error)
}
```

Tests can inject a mock builder that returns a predetermined graph without scanning the filesystem.

### graphIndexProvider Interface

[CODE: api/graph/handlers.go]

HTTP handlers depend on `graphIndexProvider`, not `GraphIndexStore` directly:

```go
type graphIndexProvider interface {
    Get(ctx context.Context) (*GraphIndex, error)
    Regenerate(ctx context.Context) error
}
```

Handler tests inject a mock provider returning static graph data, testing HTTP behavior without any filesystem or builder dependencies.

### GraphInvalidator Interface

[CODE: api/graph/models.go]

The `GraphInvalidator` interface is injected into skill and agent handlers:

```go
type GraphInvalidator interface {
    Invalidate()
}
```

CRUD handlers call `Invalidate()` after mutations. Tests verify invalidation calls without touching the index store.

### Builder Source Interfaces

[CODE: api/graph/builder.go]

The `Builder` depends on three narrow interfaces for node collection:

```go
type agentNodeSource interface {
    List(ctx context.Context) ([]store.Agent, error)
}
type teamNodeSource interface {
    List(ctx context.Context) ([]store.Team, error)
}
type skillNodeSource interface {
    List(ctx context.Context) ([]store.Skill, error)
}
```

Tests inject stubs returning predetermined node lists, isolating the build pipeline from filesystem stores.

### graphScanner Interface

[CODE: api/graph/builder.go]

The `Builder` depends on `graphScanner` for edge extraction:

```go
type graphScanner interface {
    ScanAll(ctx context.Context) ([]Edge, error)
}
```

Tests inject a mock scanner returning predetermined edges, so builder tests focus on node collection and health scoring without scanning real files.

### codeDetector Interface

[CODE: api/graph/scanner.go]

The `Scanner` depends on a `codeDetector` interface for code-reference detection rather than the concrete `*CLIDetector`:

```go
type codeDetector interface {
    Detect(content string) []CodeReference
}
```

`*CLIDetector` satisfies this interface implicitly. Tests can inject a `stubCodeDetector` that returns predetermined `CodeReference` slices, isolating scanner edge-creation logic from regex matching.

**Filtering policy:** `CodeScenarioCLI`, `CodeExternalTool`, and `CodeScript` references produce `code-usage` edges with the `Category` field set on each edge. `CodeAPICall` is intentionally excluded (documentation, not tool invocation). `prompt-manager skill read` commands are excluded since they're Skill→Skill relations handled as `EdgeCLIRead`. This is tested by `TestCodeUsageEdgesFromContent_AllowedCategories`, `TestScanAll_APICallExcluded`, and `TestCodeUsageEdges_SkipsCLIRead`.

### Index Persistence Seam

`GraphIndexStore` accepts a `storeDir` parameter. Tests point at a temp directory, isolating index file I/O from the real store.

---

## World Scale Config

The world-scale system uses a simple file-based config (`store/world-scale.json`) with
GET/PUT handlers in `api/worldscale/handlers.go`. The handlers use `store.LoadJSON` and
`store.SaveJSON` — the same seams as other file-based stores.

**Testing:** Handlers accept `storeDir string`, so tests can point at a temp directory.
No interfaces needed — the seam is the filesystem path.

## UI Seams

### Service Layer

```typescript
// Services depend on api.ts for HTTP calls
// Can be mocked at the api level

api.ts           // Low-level HTTP client
  ↓
skillService.ts  // Domain logic + caching
agentService.ts  // Domain logic + caching
  ↓
hooks/           // React Query hooks
  ↓
components/      // UI components
```

Content search follows the same seam: UI components call `searchSkillContent`
in `skillService.ts`, which delegates to the API client and returns structured
match data for rendering.

### Cache Layer

The `lib/cache.ts` module provides a generic cache manager that services use:

```typescript
const skillsCache = createCacheManager<Skill[]>()
```

For testing, the cache can be bypassed by passing `forceRefresh = true` to getters.

## Recommended Test Patterns

### Handler Unit Tests

```go
// Create mock stores
agentStore := NewMockAgentStore()
relationStore := &MockRelationStore{}
indexStore := &MockIndexStore{}

// Initialize handlers with mocks
handlers := agents.NewHandlers(agentStore, relationStore, indexStore)

// Test HTTP handling
req := httptest.NewRequest("GET", "/agents", nil)
w := httptest.NewRecorder()
handlers.List(w, req)
```

### UI Service Tests

```typescript
// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    getAgents: vi.fn().mockResolvedValue([/* test data */]),
  },
}))

// Test service behavior
const agents = await agentService.getAgents()
expect(agents).toHaveLength(1)
```

## Decision Approval Seams

### X-Caller-ID Header (Caller Type Seam)

The `X-Caller-ID` HTTP header identifies who is making a decision status update. In production:
- Agents send their agent ID (e.g., `agent-1`)
- The UI sends nothing or `"ui-user"`

In tests, set this header to simulate agent vs human callers:
```go
req.Header.Set("X-Caller-ID", "agent-1") // simulate agent caller
// or omit header to simulate human caller
```

**File**: [CODE: api/heartbeat/handlers.go#checkApprovalEnforcement]

### DecisionMode (Behavior Toggle Seam)

The `Team.DecisionMode` field (`"yolo"` or `"approval"`) controls whether approval enforcement is active. In tests, create teams with the desired mode:
```go
teamStore.Create(ctx, &store.Team{
    ID: "team-test", DecisionMode: "approval",
})
```

**File**: [CODE: api/store/models.go#Team]

## Related Documentation

- [ARCHITECTURE.md](../concepts/ARCHITECTURE.md) - System architecture overview
- [RELATIONS.md](../concepts/RELATIONS.md) - Relation system details
- [api/TESTING_GUIDE.md](../../api/TESTING_GUIDE.md) - API testing patterns
