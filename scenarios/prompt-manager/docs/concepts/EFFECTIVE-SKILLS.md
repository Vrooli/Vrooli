# Effective Skills Computation

This document describes the algorithm for computing an agent's effective skill set - the complete list of skills available to an agent at runtime.

## Overview

An agent's effective skills come from multiple sources:
1. **Skill pins** - Direct assignments on the agent entity
2. **Agent-skill relations** - Explicit grants via relation files
3. **Team role grants** - Skills granted by team membership and roles

The effective skills API resolves all sources and returns a unified skill set.

## Algorithm

### Input

```
ComputeEffectiveSkills(agentId: string, teamId?: string) -> SkillSet
```

### Steps

```
┌─────────────────────────────────────────────────────────────────────┐
│                    EFFECTIVE SKILLS ALGORITHM                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Step 1: Load agent                                                 │
│  ────────────────                                                   │
│  agent = AgentStore.Get(agentId)                                    │
│  if !agent: return error("agent_not_found")                         │
│                                                                      │
│  Step 2: Collect skill pins                                         │
│  ─────────────────────────                                          │
│  skills = {}                                                        │
│  for pin in agent.skillPins:                                        │
│      skills[pin.skillId] = {                                        │
│          source: "pin",                                             │
│          version: pin.version                                       │
│      }                                                              │
│                                                                      │
│  Step 3: Collect agent-skill relations                              │
│  ────────────────────────────────────                               │
│  relations = RelationStore.GetAgentSkillRelations(agentId)          │
│  for rel in relations:                                              │
│      if rel.enabled:                                                │
│          skills[rel.skillId] = {                                    │
│              source: "relation",                                    │
│              version: rel.pin                                       │
│          }                                                          │
│      else:                                                          │
│          // Disabled relation = explicit exclusion                  │
│          delete skills[rel.skillId]                                 │
│                                                                      │
│  Step 4: Add team role grants (if teamId provided)                  │
│  ─────────────────────────────────────────────────                  │
│  if teamId:                                                         │
│      team = TeamStore.Get(teamId)                                   │
│      membership = RelationStore.GetTeamMemberRelation(teamId, agentId) │
│      if membership && membership.status == "active":                │
│          for role in membership.roles:                              │
│              grantedSkills = team.defaults.skillGrantsByRole[role]  │
│              for skillId in grantedSkills:                          │
│                  if skillId not in skills:  // Don't override       │
│                      skills[skillId] = {                            │
│                          source: "role:" + role,                    │
│                          version: "latest"                          │
│                      }                                              │
│                                                                      │
│  Step 5: Resolve skill metadata                                     │
│  ─────────────────────────────                                      │
│  result = []                                                        │
│  for skillId, info in skills:                                       │
│      skill = SkillStore.Get(skillId)                                │
│      if skill:                                                      │
│          result.append({                                            │
│              skill: skill,                                          │
│              source: info.source,                                   │
│              version: info.version                                  │
│          })                                                         │
│                                                                      │
│  return result                                                      │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Pseudocode

```go
func ComputeEffectiveSkills(agentId string, teamId *string) ([]EffectiveSkill, error) {
    // Step 1: Load agent
    agent, err := agentStore.Get(agentId)
    if err != nil {
        return nil, fmt.Errorf("agent_not_found: %s", agentId)
    }

    // Track skills with their source
    skills := make(map[string]SkillGrant)

    // Step 2: Add skill pins (highest priority)
    for _, pin := range agent.SkillPins {
        skills[pin.SkillId] = SkillGrant{
            Source:  "pin",
            Version: pin.Version,
        }
    }

    // Step 3: Process agent-skill relations
    relations, _ := relationStore.GetAgentSkillRelations(agentId)
    for _, rel := range relations {
        if rel.Enabled {
            // Only add if not already pinned (pins have priority)
            if _, exists := skills[rel.SkillId]; !exists {
                skills[rel.SkillId] = SkillGrant{
                    Source:  "relation",
                    Version: rel.Pin,
                }
            }
        } else {
            // Disabled relation = explicit exclusion
            delete(skills, rel.SkillId)
        }
    }

    // Step 4: Add team role grants (if teamId provided)
    if teamId != nil {
        team, err := teamStore.Get(*teamId)
        if err == nil {
            membership, _ := relationStore.GetTeamMemberRelation(*teamId, agentId)
            if membership != nil && membership.Status == "active" {
                for _, role := range membership.Roles {
                    grantedSkills := team.Defaults.SkillGrantsByRole[role]
                    for _, skillId := range grantedSkills {
                        // Don't override existing grants
                        if _, exists := skills[skillId]; !exists {
                            skills[skillId] = SkillGrant{
                                Source:  "role:" + role,
                                Version: "latest",
                            }
                        }
                    }
                }
            }
        }
    }

    // Step 5: Resolve skill metadata
    var result []EffectiveSkill
    for skillId, grant := range skills {
        skill, err := skillStore.Get(skillId)
        if err == nil {
            result = append(result, EffectiveSkill{
                Skill:   skill,
                Source:  grant.Source,
                Version: grant.Version,
            })
        }
    }

    return result, nil
}
```

## Priority Order

When the same skill appears from multiple sources, the following priority determines which grant is used:

| Priority | Source | Example |
|----------|--------|---------|
| 1 (highest) | Disabled relation | `enabled: false` blocks all other grants |
| 2 | Skill pin | `agent.skillPins[{skillId: "debugging"}]` |
| 3 | Agent-skill relation | `relations/agent-skill/alice--debugging.json` |
| 4 (lowest) | Team role grant | `team.defaults.skillGrantsByRole.developer` |

**Key rule:** More specific grants take priority over general grants.

## API Usage

### Endpoint

```
GET /api/v1/agents/{agentId}/effective-skills?teamId={teamId}
```

### Response

```json
{
  "agentId": "alice",
  "teamId": "engineering",
  "effectiveSkills": [
    {
      "skill": {
        "id": "debugging",
        "name": "Debugging Expert",
        "description": "Systematic debugging approach"
      },
      "source": "pin",
      "version": "latest"
    },
    {
      "skill": {
        "id": "testing",
        "name": "Test Writer",
        "description": "Write comprehensive tests"
      },
      "source": "relation",
      "version": "v2"
    },
    {
      "skill": {
        "id": "code-review",
        "name": "Code Reviewer",
        "description": "Review code for quality"
      },
      "source": "role:developer",
      "version": "latest"
    }
  ],
  "computedAt": "2025-01-30T10:30:00Z"
}
```

### CLI Usage

```bash
# Get effective skills without team context
prompt-manager agent effective-skills alice

# Get effective skills with team context
prompt-manager agent effective-skills alice --team=engineering

# Output as JSON
prompt-manager agent effective-skills alice --team=engineering --json
```

## Examples

### Example 1: Agent with Pins Only

```
Agent: alice
  skillPins: [debugging, testing]

Team: (none specified)

Effective Skills:
  - debugging (source: pin)
  - testing (source: pin)
```

### Example 2: Agent with Relations

```
Agent: alice
  skillPins: [debugging]

Relations:
  agent-skill/alice--testing.json: {enabled: true}
  agent-skill/alice--code-review.json: {enabled: true}

Team: (none specified)

Effective Skills:
  - debugging (source: pin)
  - testing (source: relation)
  - code-review (source: relation)
```

### Example 3: Agent with Team Role Grants

```
Agent: alice
  skillPins: [debugging]

Relations:
  team-member/engineering--alice.json: {roles: ["developer"]}

Team: engineering
  defaults.skillGrantsByRole:
    developer: [testing, code-review, documentation]

Effective Skills:
  - debugging (source: pin)
  - testing (source: role:developer)
  - code-review (source: role:developer)
  - documentation (source: role:developer)
```

### Example 4: Disabled Relation Override

```
Agent: alice
  skillPins: []

Relations:
  agent-skill/alice--debugging.json: {enabled: false}  # Explicit block
  team-member/engineering--alice.json: {roles: ["developer"]}

Team: engineering
  defaults.skillGrantsByRole:
    developer: [debugging, testing]  # Would normally grant debugging

Effective Skills:
  - testing (source: role:developer)
  # debugging is BLOCKED by disabled relation, even though role grants it
```

## Performance Considerations

### Caching

The effective skills computation is designed to be fast (<50ms target), but can be cached:

```go
// Cache key includes both agent and team
cacheKey := fmt.Sprintf("effective-skills:%s:%s", agentId, teamId)

// Cache TTL: 5 minutes (skills change infrequently)
cached, exists := cache.Get(cacheKey)
if exists {
    return cached, nil
}

// Compute and cache
result := computeEffectiveSkills(agentId, teamId)
cache.Set(cacheKey, result, 5*time.Minute)
```

### Cache Invalidation

Invalidate cache when:
- Agent's skillPins change
- Agent-skill relation created/updated/deleted
- Team-member relation roles change
- Team's skillGrantsByRole change

### Index Usage

Use relation indexes for fast lookups:
```go
// Fast: Uses index
skillIds := index.GetAgentSkills(agentId)

// Slower: Scans relation files
relations := store.ScanAgentSkillRelations(agentId)
```

## Error Handling

| Error | Cause | Resolution |
|-------|-------|------------|
| `agent_not_found` | Invalid agentId | Verify agent exists |
| `team_not_found` | Invalid teamId | Verify team exists (ignored, computation continues) |
| `skill_not_found` | Skill referenced in relation doesn't exist | Logged as warning, skill excluded |
| `invalid_role` | Role in membership doesn't exist in team | Logged as warning, role skipped |

## Related Documentation

- [SWARM-MODEL.md](SWARM-MODEL.md) - Overall architecture
- [RELATIONS.md](RELATIONS.md) - Relation system details
- [CAPABILITY-MATCHING.md](CAPABILITY-MATCHING.md) - Capability-based skill filtering
