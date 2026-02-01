# Relations System

This document describes the normalized relations system that connects Skills, Agents, and Teams in prompt-manager.

## Overview

Relations are junction records that link entities together without embedding foreign keys directly in the entity files. This design provides:

- **Normalization**: Avoids data duplication and update anomalies
- **Query flexibility**: Find all skills for an agent, or all agents with a skill
- **Independent lifecycle**: Relations can be created/deleted without modifying core entities
- **Audit trail**: Each relation file can track its own creation/update timestamps

## Storage Structure

```
store/relations/
├── agent-skill/
│   ├── alice--debugging.json
│   ├── alice--testing.json
│   └── bob--code-review.json
└── team-member/
    ├── engineering--alice.json
    ├── engineering--bob.json
    └── marketing--charlie.json
```

**Naming Convention:** `{leftId}--{rightId}.json`
- Uses double-dash (`--`) as separator (IDs may contain single dashes)
- Left ID is the "owner" in the relationship
- Right ID is the "target" being linked

## Agent-Skill Relations

Agent-skill relations assign skills to agents with optional version pinning and enable/disable control.

### Schema

```json
{
  "agentId": "string",
  "skillId": "string",
  "pin": "string",        // Optional: version pin ("latest", "v2", specific hash)
  "enabled": true,        // Whether this relation is active
  "createdAt": "timestamp",
  "updatedAt": "timestamp"
}
```

### Example

**File:** `store/relations/agent-skill/alice--debugging.json`
```json
{
  "agentId": "alice",
  "skillId": "debugging",
  "pin": "latest",
  "enabled": true,
  "createdAt": "2025-01-15T10:00:00Z",
  "updatedAt": "2025-01-20T14:30:00Z"
}
```

### Use Cases

1. **Explicit skill assignment**: Assign specific skills beyond team role grants
2. **Version pinning**: Lock an agent to a specific skill version for stability
3. **Temporary disable**: Set `enabled: false` to temporarily remove skill without deleting relation
4. **Override team grants**: Create relation with `enabled: false` to block a role-granted skill

### Query Patterns

```go
// Get all skills for an agent
relations, _ := store.GetAgentSkillRelations(agentId)

// Get all agents using a skill
relations, _ := store.GetSkillAgentRelations(skillId)

// Check if agent has specific skill
relation, exists := store.GetAgentSkillRelation(agentId, skillId)
```

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
2. **Role assignment**: Grant roles that determine skill access
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

### Agent-Skill Relations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/agents/{id}/skills` | List agent's skill relations |
| POST | `/api/v1/agents/{id}/skills` | Create skill relation |
| PUT | `/api/v1/agents/{id}/skills/{skillId}` | Update relation (pin, enabled) |
| DELETE | `/api/v1/agents/{id}/skills/{skillId}` | Remove relation |

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
// store/indexes/agent-skills.json
{
  "alice": ["debugging", "testing", "code-review"],
  "bob": ["testing"],
  "charlie": ["documentation"]
}

// store/indexes/team-members.json
{
  "engineering": ["alice", "bob"],
  "marketing": ["charlie", "diana"]
}
```

**Important:** Indexes are generated, never hand-edited. The store layer rebuilds them on startup and keeps them synchronized on relation changes.

## Effective Skills Integration

Relations feed into the effective skills computation:

```
┌─────────────────────────────────────────────────────────┐
│              EFFECTIVE SKILLS SOURCES                    │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  1. Agent.skillPins[]                                   │
│     ↳ Direct skill assignments on agent entity          │
│                                                          │
│  2. agent-skill relations (enabled=true)                │
│     ↳ Explicit skill grants via relation files          │
│                                                          │
│  3. Team role grants                                    │
│     ↳ team-member relation → roles → skillGrantsByRole  │
│                                                          │
│  4. Subtract disabled relations (enabled=false)         │
│     ↳ Explicit exclusions override other grants         │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Best Practices

### Creating Relations

1. **Verify entities exist**: Check agent and skill/team exist before creating relation
2. **Handle duplicates**: Use upsert semantics - update if exists, create if not
3. **Update indexes**: Always rebuild affected indexes after relation changes

### Querying Relations

1. **Use indexes first**: Query from index for list operations
2. **Read file for details**: Only read relation file when you need full metadata
3. **Cache appropriately**: Relations change infrequently, cache for performance

### Deleting Relations

1. **Consider soft delete**: Set `enabled: false` instead of deleting
2. **Clean up indexes**: Remove from index files after deletion
3. **Log for audit**: Record who/when deleted for troubleshooting

## Error Handling

| Error | Cause | Resolution |
|-------|-------|------------|
| `relation_not_found` | Relation file doesn't exist | Check IDs, verify relation was created |
| `agent_not_found` | Agent ID invalid | Verify agent exists before creating relation |
| `skill_not_found` | Skill ID invalid | Verify skill exists in pack structure |
| `team_not_found` | Team ID invalid | Verify team exists before adding members |
| `duplicate_relation` | Relation already exists | Use update endpoint instead of create |

## Related Documentation

- [SWARM-MODEL.md](SWARM-MODEL.md) - Overall three-domain architecture
- [EFFECTIVE-SKILLS.md](EFFECTIVE-SKILLS.md) - How relations feed into skill computation
- [ARCHITECTURE.md](ARCHITECTURE.md) - Storage layer details
