# Storage Migration Guide

This document explains the migration from the legacy `skills/` storage to the new `store/` storage system.

## Overview

The prompt-manager underwent a storage architecture change to support:
- **Agents and Teams**: New entity types for organizing skills
- **Normalized Relations**: Explicit team-member relationships
- **Per-Entity Files**: Each entity has its own directory with structured files
- **Schema Validation**: JSON Schemas for runtime validation
- **Pack System**: Skills organized into precedence-ordered packs
- **Actions (proposed)**: Future executable contracts will use the same per-entity, schema-validated pattern

## Migration Status

| Component | Status | Notes |
|-----------|--------|-------|
| API wiring | Complete | `main.go` uses `store.NewFileStore()` |
| Skills data | Complete | All skills migrated to `store/skills/packs/` |
| Members → Agents | Complete | 8 members converted to agents |
| Legacy compatibility | Active | `/api/v1/members` routes still work |
| Old `skills/` dir | Deprecated | Content verified identical, can be removed |
| Actions | Proposed | Planned `store/actions/packs/{pack}/{id}/action.json` layout |

## Architecture Changes

### Old Architecture

```
skills/
├── core/
│   ├── metadata.json    # All skill metadata in one file
│   └── *.md             # Skill content files
├── local/
│   └── ...
└── data/
    └── members.json     # All members in one file
```

**Problems:**
- Single `metadata.json` per folder = merge conflicts
- Members separate from skills = no relationship modeling
- No schema validation
- No version history per skill

### New Architecture

```
store/
├── skills/packs/{pack}/{skill-id}/
│   ├── skill.json      # Skill metadata
│   ├── SKILL.md        # Skill content
│   └── history.jsonl   # Version history
├── agents/{agent-id}/
│   └── agent.json      # Agent definition
├── actions/packs/{pack}/{action-id}/  # Proposed
│   ├── action.json     # Typed executable contract
│   └── history.jsonl   # Version history
├── relations/
│   └── team-member/    # Team-agent bindings
└── schemas/            # Validation schemas
```

**Benefits:**
- Per-entity files = no merge conflicts
- Normalized relations = flexible querying
- Schema validation = early error detection
- Integrated version history

## API Changes

### New Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/agents` | GET | List all agents |
| `/api/v1/agents` | POST | Create agent |
| `/api/v1/agents/{id}` | GET | Get agent |
| `/api/v1/agents/{id}` | PUT | Update agent |
| `/api/v1/agents/{id}` | DELETE | Delete agent |

### Backward Compatibility

The `/api/v1/members` endpoints remain functional by delegating to agent handlers with format translation:

| Member Field | Agent Field |
|--------------|-------------|
| `name` | `displayName` |
| `bodyColor` | `appearance.body` |
| `headColor` | `appearance.head` |
| `accentColor` | `appearance.accent` |

## Code Changes

### Store Adapter Pattern

The skills handlers use the legacy `SkillStore` interface. An adapter bridges to the new storage:

```go
// api/skills/store_adapter.go
type StoreAdapter struct {
    fileStore *store.FileSkillStore
}

// Implements skills.SkillStore interface
func (a *StoreAdapter) GetAll() ([]Metadata, error) {
    ctx := context.Background()
    skills, err := a.fileStore.List(ctx)
    // Convert store.Skill to skills.Metadata
}
```

### Dependency Injection in main.go

```go
// Initialize new storage
fileStore := store.NewFileStore(storeDir)

// Create adapter for legacy interface
skillStoreAdapter := skills.NewStoreAdapter(fileStore.FileSkills())

// Use adapter in handlers
skillHandlers := skills.NewHandlers(skillStoreAdapter, metricsAdapter)

// New agent handlers use storage directly
agentHandlers := agents.NewHandlers(
    fileStore.Agents(),
    fileStore.Relations(),
    fileStore.Indexes(),
)
```

## Data Format Changes

### Skill Metadata

**Old format** (`metadata.json`):
```json
{
  "skills": [
    {
      "id": "debugging",
      "file": "debugging.md",
      "name": "Debugging",
      "draft": false
    }
  ]
}
```

**New format** (`skill.json`):
```json
{
  "kind": "skill",
  "schemaVersion": 1,
  "id": "debugging",
  "name": "Debugging",
  "status": "active",
  "entry": "SKILL.md",
  "revision": 1,
  "createdAt": "2026-01-15T12:00:00Z",
  "updatedAt": "2026-01-20T15:30:00Z"
}
```

### Member → Agent

**Old format** (`members.json`):
```json
{
  "members": [
    {
      "id": "agent-1",
      "name": "Agent One",
      "bodyColor": "#22C55E",
      "headColor": "#F97316",
      "accentColor": "#C7D2FE",
      "skills": ["debugging", "testing"]
    }
  ]
}
```

**New format** (`agent.json`):
```json
{
  "kind": "agent",
  "schemaVersion": 1,
  "id": "agent-1",
  "displayName": "Agent One",
  "status": "active",
  "appearance": {
    "body": "#22C55E",
    "head": "#F97316",
    "accent": "#C7D2FE"
  }
}
```

## Cleanup Plan

After confirming the migration is stable:

1. **Week 1**: Monitor for issues with new storage
2. **Week 2**: Rename `skills/` → `skills.deprecated/`
3. **Week 3+**: Delete deprecated directory if no issues

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `STORE_DIR` | `../store` | Path to new storage directory |

The old `SKILLS_DIR` variable is no longer used.

## Related Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) - Updated storage layout diagrams
- [API Reference](../reference/api-endpoints.md) - Agent endpoint documentation
- [SEAMS.md](../internal/SEAMS.md) - Testing seam documentation
