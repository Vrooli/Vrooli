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
| `TeamStore` | `store` | CRUD operations for teams |
| `RelationStore` | `store` | Agent-skill and team-member relations |
| `IndexStore` | `store` | Generated index management |
| `MetricsService` | `skills` | Usage tracking |
| `AISearchIndexer` | `skills` | AI search index updates |

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

### Testing Implications

1. **Unit tests** can mock at the `SkillStore` level (adapter interface)
2. **Integration tests** should test the full adapter → FileStore chain
3. When debugging, determine whether the issue is in the adapter translation or underlying store

## Circular Dependency Handling

The `skills` package cannot import `aisearch` (would cause circular import: aisearch needs skills).

**Solution:** Post-initialization setter injection

```go
// In main.go
skillHandlers := skills.NewHandlers(skillStoreAdapter, metricsAdapter)
skillHandlers.SetAIIndexer(aiSearchService)  // Called after aisearch.Service is initialized
```

**Testing Impact:** Mock tests must call `SetAIIndexer()` or the async indexing code path won't be exercised.

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
- [api/TESTING_GUIDE.md](../../api/TESTING_GUIDE.md) - API testing patterns
