# Relations System

This document describes the normalized relations system that connects Agents and Teams in prompt-manager.

## Overview

Relations are junction records that link entities together without embedding foreign keys directly in the entity files. This design provides:

- **Normalization**: Avoids data duplication and update anomalies
- **Query flexibility**: Find all teams for an agent, or all agents in a team
- **Independent lifecycle**: Relations can be created/deleted without modifying core entities
- **Audit trail**: Each relation file can track its own creation/update timestamps

## Storage Structure

```
store/relations/
└── team-member/
    ├── engineering--alice.json
    ├── engineering--bob.json
    └── marketing--charlie.json
```

**Naming Convention:** `{leftId}--{rightId}.json`
- Uses double-dash (`--`) as separator (IDs may contain single dashes)
- Left ID is the "owner" in the relationship
- Right ID is the "target" being linked

## Team-Member Relations

Team-member relations link agents to teams with role assignments and status tracking.

### Schema

```json
{
  "teamId": "string",
  "agentId": "string",
  "roles": ["string"],    // Role IDs within the team
  "status": "string",     // active, inactive, suspended
  "joinedAt": "timestamp",
  "updatedAt": "timestamp"
}
```

### Example

**File:** `store/relations/team-member/engineering--alice.json`
```json
{
  "teamId": "engineering",
  "agentId": "alice",
  "roles": ["lead", "developer"],
  "status": "active",
  "joinedAt": "2025-01-10T09:00:00Z",
  "updatedAt": "2025-01-25T11:00:00Z"
}
```

### Use Cases

1. **Team membership**: Track which agents belong to which teams
2. **Role assignment**: Capture responsibilities or org structure
3. **Multi-team membership**: Same agent can be in multiple teams
4. **Status management**: Track active/inactive without removing from team

### Query Patterns

```go
// Get all members of a team
relations, _ := store.GetTeamMembers(teamId)

// Get all teams an agent belongs to
relations, _ := store.GetAgentTeams(agentId)

// Get agent's roles in a specific team
relation, exists := store.GetTeamMemberRelation(teamId, agentId)
roles := relation.Roles
```

## API Endpoints

### Team-Member Relations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/teams/{id}/members` | List team members |
| POST | `/api/v1/teams/{id}/members` | Add agent to team |
| PUT | `/api/v1/teams/{id}/members/{agentId}` | Update roles/status |
| DELETE | `/api/v1/teams/{id}/members/{agentId}` | Remove from team |

## Relation Indexes

For performance, relations are indexed in `store/indexes/`:

```json
// store/indexes/team-members.json
{
  "engineering": ["alice", "bob"],
  "marketing": ["charlie", "diana"]
}
```

**Important:** Indexes are generated, never hand-edited. The store layer rebuilds them on startup and keeps them synchronized on relation changes.

## Best Practices

### Creating Relations

1. **Verify entities exist**: Check agent and team exist before creating relation
2. **Handle duplicates**: Use upsert semantics - update if exists, create if not
3. **Update indexes**: Always rebuild affected indexes after relation changes

### Querying Relations

1. **Use indexes first**: Query from index for list operations
2. **Read file for details**: Only read relation file when you need full metadata
3. **Cache appropriately**: Relations change infrequently, cache for performance

### Deleting Relations

1. **Consider soft delete**: Set `status: inactive` instead of deleting
2. **Clean up indexes**: Remove from index files after deletion
3. **Log for audit**: Record who/when deleted for troubleshooting

## Error Handling

| Error | Cause | Resolution |
|-------|-------|------------|
| `relation_not_found` | Relation file doesn't exist | Check IDs, verify relation was created |
| `agent_not_found` | Agent ID invalid | Verify agent exists before creating relation |
| `team_not_found` | Team ID invalid | Verify team exists before adding members |
| `duplicate_relation` | Relation already exists | Use update endpoint instead of create |

## Related Documentation

- [SWARM-MODEL.md](SWARM-MODEL.md) - Overall swarm architecture and Action execution layer
- [ARCHITECTURE.md](ARCHITECTURE.md) - Storage layer details
