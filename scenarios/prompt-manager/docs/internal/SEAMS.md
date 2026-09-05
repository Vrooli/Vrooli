# Testing Seams

This document describes the testing seams in the prompt-manager scenario - deliberate boundaries where behavior can be substituted for testing.

## Overview

The prompt-manager uses interface-based design to create clear testing seams. Each domain handler depends on interfaces rather than concrete types, enabling mock implementations for unit testing.

## Normalized package layout

The API now follows the scenario layout contract:

```text
api/
├── handlers/<domain>/   # composition-facing transport facade
└── internal/<domain>/   # domain behavior, ports, persistence adapters, tests
```

The phase-11 move was deliberately behavior-preserving. Each
`handlers/<domain>/facade.go` re-exports only the composition-root surface from
the moved implementation package. This keeps every unexported invariant and
the exact REST behavior intact while making the dependency direction visible.
The proto migration slices replace these compatibility facades with thin
Connect adapters and progressively narrow their exports; new business logic
must not be added to a facade.

Database schema providers live under `internal/database/<domain>` rather than
inside a domain package. That separation prevents schema bootstrap from
creating domain import cycles (for example, skills uses metrics while the
metrics integration test applies the skills schema).

Tests moved with their implementation packages. Repository-relative fixtures
were adjusted for the additional `internal/` path component, without changing
their asserted behavior. The canonical before/after comparison is package
outcome plus the invariant counts of 393 implementation/test Go files and 172
test files; the normalized tree adds 16 facade files and removes no tests.

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
│  │  SkillStore │ AgentStore │ TeamStore │  LLMClient   │   │
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
| `VariantStore` | `store` | CRUD operations for skill variants (A/B testing) |
| `ExperimentStore` | `store` | Experiment lifecycle, opaque outcome JSONL storage |
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
| `TopicStoreReader` | `aisearch` | Read-only topic access for discover pipeline |
| `BudgetConfigProvider` | `aisearch` | Budget tier config for discover budgeting |
| `DiscoverFilterConfigProvider` | `aisearch` | Discover exclusion filter config (drafts/ids/modes/tags) |
| `DiscoverRankingConfigProvider` | `aisearch` | Ranking levers (topic gate, high-confidence bar, caps) for the block-aware discover composition; faked via `MockRankingConfigProvider` in tests |
| `SemanticActionSearcher` | `actions` | Semantic similarity for `action create` dedup previews (adapts aisearch; faked in tests) |
| `DiscoveryMissStore` | `aisearch` | Discovery-miss telemetry sink/source (impl: `store.DiscoveryMissStore`; faked in tests) |
| `DiscoveryCallStore` | `aisearch` | Per-call discovery telemetry sink/source — records EVERY discover call, not just misses (impl: `store.DiscoveryCallStore`; faked in tests) |

### Search Service Seams

The `search` package provides dedicated seams for text and content search across all
entity types, isolating search behavior from Connect handlers and consumers:

```
Connect handler (`SearchService` deterministic methods)
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
  StopRun, EnsureProfile, ReconcileScenarioProfiles, Health). The concrete `*AgentManagerClient` satisfies this interface
  and has a `testBaseURL` field for httptest-based integration tests.
- **Scheduler → Executor**: `heartbeat.Scheduler` depends on the `HeartbeatExecutor` interface, allowing
  tests to substitute fake executors to validate scheduling behavior without running agent-manager.
- **Scheduler → Config Store**: `heartbeat.Scheduler` depends on `HeartbeatConfigStore` to resolve
  per-member profile keys and enabled state at run time.
- **Heartbeat Control Store** (`heartbeat/control.go`): persisted runtime-data state for global/per-team
  auto-pause policy, manual pause/resume state, and last operator engagement. `Scheduler.Schedule`,
  cron fire execution, and manual trigger handlers consult this gate before starting new work. The
  gate never mutates `heartbeat.json.enabled`; resume reschedules enabled configs separately.
- **Structured Engagement Detection**: decision updates and heartbeat control/config mutations parse
  `X-Vrooli-Attribution` and record engagement only for `kind=operator-direct`. Agent-member and
  writer-skill activity never resets the idle clock.

### Scenario-owned Profile Reconciliation

Prompt Manager declares its heartbeat profiles in `.vrooli/agent-profiles/`
and registers both files in its Agent Manager dependency configuration.
`Scheduler.Start()` calls `ReconcileScenarioProfiles("prompt-manager")`; the
consumer never sends inline profile defaults, runner types, models, or policy
references.

`CreateRun` sends only the reconciled `profile_key`. `EnsureProfile` remains a
read-only key-to-ID lookup for run-list filtering, so a missing profile is a
clear startup/reconciliation failure rather than an opportunity for a consumer
to create one. `client_contract_test.go` pins both request shapes as key-only.

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

### CreateRun Profile Reference

[CODE: api/heartbeat/executor.go, api/heartbeat/client.go]

The `CreateRunRequest.ProfileRef` contains only the stable, scenario-owned
profile key. Agent Manager resolves the profile's portable `roleRef` through
resource-owned policy and records the concrete runner/model only in the
immutable execution snapshot. `TestExecute_CreateRunUsesDeclaredProfile`
prevents inline defaults from returning.

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

## Budget Config

The discover system uses a `BudgetConfigProvider` interface (`aisearch/budget_config.go`)
to decouple budget lookup from file I/O:

```go
type BudgetConfigProvider interface {
    Get(ctx context.Context) (BudgetConfig, error)
}
```

**Production:** `BudgetConfigStore` reads/writes `store/config/budgets.json` with
`sync.RWMutex` protection. Falls back to `DefaultBudgetConfig()` when no file exists.

**Testing:** `MockBudgetConfigProvider` returns custom budget values, allowing Discover()
tests to verify budget calculation with arbitrary tier configs without touching the filesystem.

**Handlers:** `GetBudgetConfig`/`PutBudgetConfig` on `aisearch.Handlers` follow the same
pattern as `graph.Handlers.GetHealthConfig`/`PutHealthConfig`.

---

## Discovery Telemetry

The discover pipeline has two parallel telemetry seams, both interfaces in the
`aisearch` package whose production impls live in `store` and whose fakes are
injected in tests:

```go
type DiscoveryMissStore interface { // unmet queries only (top score < 0.45)
    Append(miss store.DiscoveryMiss) error
    ReadSince(window time.Duration) ([]store.DiscoveryMiss, error)
}

type DiscoveryCallStore interface { // EVERY discover call
    Append(call store.DiscoveryCall) error
    ReadSince(window time.Duration) ([]store.DiscoveryCall, error)
}
```

**Why two.** The miss store answers "what are agents looking for that we don't
have?" (mined as capability-work alpha). The call store answers "is the
threshold/budget tuned right?" — it records every call (scores, budget status,
trimmed count, optional clip count) so a call that returns relevant-but-clipped
results is visible. The miss store alone is blind to that case.

**Production:** `store.DiscoveryMissStore` / `store.DiscoveryCallStore` write
bounded, time-windowed JSONL (`discovery-misses.jsonl` / `discovery-calls.jsonl`)
under the runtime-data root — separate files so miss-mining semantics stay
clean. Wired in `main.go` via `SetDiscoveryMissStore` / `SetDiscoveryCallStore`.

**Testing:** `fakeMissStore` / `fakeCallStore` capture appended records and
return canned read responses, so `recordDiscoveryMiss`, `recordDiscoveryCall`,
`DiscoveryGaps`, and `DiscoveryMetrics` are tested without a filesystem home.
Both recorders are guarded (nil store = no-op) and log-and-continue on error so
telemetry never fails the discover response.

See [discovery-pipeline.md](../reference/discovery-pipeline.md) for the full
pipeline, threshold/budget semantics, and tuning rubric.

---

## World Service

`api/internal/world` persists the world's operator config (`<config>/world/config.json`)
and per-scene layout overrides (`world/layout-<scene>.json`) through `store.LoadJSON` /
`store.SaveJSON`, and fans swarm signals out through `world.Hub`. The hub is the
`heartbeat.RunObserver` installed on the run registry; a `ScheduleWatcher` polls the
cron scheduler for upcoming heartbeats.

**Testing:** `NewStore(dir)` takes a temp directory; `NewHub(ring, depth, source)` takes a
`SnapshotSource` fake; the Connect handler is exercised through `httptest` with the
generated client (`handlers/world/connect_handler_test.go`).

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

### Skills, actions, and tags Connect handlers

The `handlers/{skills,actions,tags}` packages own the generated transport
adapters for slice 1. Public REST registrations for these operations have been
removed. The generated clients used by the CLI, UI, and scenario consumers are
therefore the only supported network transport.

The domains still contain mature behavior in their existing `net/http`
handlers. `handlers/transportbridge` is a deliberately narrow in-process seam:
it constructs a request, invokes that behavior without a loopback network
call, maps HTTP failures to Connect codes, and decodes successful JSON with
unknown protobuf fields rejected. This keeps the migration behavior-preserving
while later domain-service extraction proceeds. It is not a second public
transport, and new business logic must not be added to the bridge.

Slice 1 exposes 25 RPC methods: 16 skills (including five variant operations),
seven actions, and two tags. The CLI declares 22 bindings because variant get,
variant update, and usage recording have no CLI commands. The retired surface contained 26 REST route
registrations; action preview and create are represented by the single typed
`AuthorAction` operation with an explicit `apply` field.

### Search, AI search, and discovery Connect handlers

Slice 2 applies the same behavior-preserving transport seam to
`handlers/search` and `handlers/aisearch`. `SearchService` owns six
deterministic entity/content queries, `AISearchService` owns four semantic
queries plus status and reconciliation, and `DiscoveryService` owns unified
discovery, gaps, metrics, and skill-usage reporting. Together they expose 18
typed methods and replace 18 public REST registrations.

The CLI has 11 exact governed bindings from this slice: the five search flats,
four discovery/reporting flats, plus `agent search` and `team search`.
Deterministic and content-search methods remain real generated-client paths
behind those commands' `--text` and `--content` modes and carry explicit
manifest omission reasons because they are not separate commands.

All discovered consumers moved with the cutover: the prompt-manager UI uses
generated TypeScript clients, Agent Inbox uses the generated Go AI-search
client, and Search Hub's declarative descriptors target Connect-JSON
procedure paths. The old `net/http` handlers are reachable only through the
in-process compatibility adapter described above; they are not public routes.

### Graph Connect Handler (proto transport seam) {#graph-connect-handler}

`graph.NewConnectMount(indexStore)` builds prompt-manager's first proto/Connect
contract — `vrooli.prompt_manager.v1.graph.GraphService.GetHealthScores` — and
returns the `(procedurePath, http.Handler)` pair mounted on the existing
gorilla/mux router via `connectx.RegisterServices` in `api/main.go`. The handler
(`api/graph/connect_handler.go`) owns no domain logic: it delegates to the same
`graphIndexProvider` the REST `GET /api/v1/graph/health` handler reads and maps
the domain `HealthScore`/`HealthMessage` onto their proto wire shapes.

This is **additive**: the legacy REST route stays live; prompt-manager's own
CLI/UI are not migrated. The contract exists because
meta-optimization-manager's Guide numerator consumes it as a typed
`GraphServiceClient` over a discovery-resolved base URL (replacing a
`prompt-manager graph health --json` CLI shell-out). Full proto/Connect adoption
of the other graph + prompt-manager domains — plus the `gen-endpoints` /
`endpoints.json` drift gate — is deferred to a dedicated adoption plan
(see [PROBLEMS.md](PROBLEMS.md)).

**Files**: [CODE: api/graph/connect_handler.go], [CODE: api/main.go] (mount),
[CODE: packages/proto/schemas/prompt-manager/v1/graph/graph.proto] (contract).

## Related Documentation

- [ARCHITECTURE.md](../concepts/ARCHITECTURE.md) - System architecture overview
- [RELATIONS.md](../concepts/RELATIONS.md) - Relation system details
- [api/TESTING_GUIDE.md](../../api/TESTING_GUIDE.md) - API testing patterns
