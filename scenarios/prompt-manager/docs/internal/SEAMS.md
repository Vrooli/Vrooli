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
| `AISearchIndexer` | `skills` | AI search index updates |

### Search Service Seam

The `search.Service` provides a dedicated seam for text and content search, isolating
search behavior from HTTP handlers and the UI:

```
HTTP handler (/search/skills, /search/skills/content)
  ↓
search.Service (Search / SearchContent)
  ↓
skills.SkillStore (GetAll + GetContent)
```

`SearchContent` is the authoritative boundary for line-level matching behavior
(case sensitivity, whole word matching, regex parsing), which makes it testable
without touching the HTTP layer or filesystem.

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

The `skills` package cannot import `aisearch` (would cause circular import: aisearch needs skills).

**Solution:** Post-initialization setter injection

```go
// In main.go
skillHandlers := skills.NewHandlers(skillStoreAdapter, metricsAdapter)
skillHandlers.SetAIIndexer(aiSearchService)  // Called after aisearch.Service is initialized
```

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

Heartbeat execution uses two explicit seams:

- **Scheduler → Executor**: `heartbeat.Scheduler` depends on the `HeartbeatExecutor` interface, allowing
  tests to substitute fake executors to validate scheduling behavior without running agent-manager.
- **Scheduler → Config Store**: `heartbeat.Scheduler` depends on `HeartbeatConfigStore` to resolve
  per-member profile keys and enabled state at run time.

Member cleanup is centralized in the team handlers:

- **Member cleanup**: `teams.Handlers.cleanupMemberData` unschedules heartbeats and deletes
  `store/teams/{team-id}/members/{agent-id}/` via the file store when available.
  Tests can assert scheduler calls while keeping file I/O isolated to temporary directories.

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

## Related Documentation

- [ARCHITECTURE.md](../concepts/ARCHITECTURE.md) - System architecture overview
- [RELATIONS.md](../concepts/RELATIONS.md) - Relation system details
- [api/TESTING_GUIDE.md](../../api/TESTING_GUIDE.md) - API testing patterns
