# Capability Matching

This document describes the capability system in prompt-manager, which enables intelligent skill-to-agent matching based on declared capabilities.

## Overview

Capabilities are a contract system that declares:
- What an **agent provides** - abilities the agent has
- What an **agent requires** - abilities the agent needs from its environment
- What a **skill requires** - abilities an agent must have to use the skill

This enables:
- Automatic skill filtering based on agent capabilities
- Validation before skill assignment
- Discovery of which agents can perform specific tasks

## Capability Structure

### Agent Capabilities

Agents declare capabilities with verbs indicating what actions they can perform:

```json
{
  "id": "alice",
  "capabilities": {
    "provides": [
      {"capabilityId": "code-analysis", "verbs": ["read", "analyze", "annotate"]},
      {"capabilityId": "debugging", "verbs": ["diagnose", "fix", "explain"]},
      {"capabilityId": "testing", "verbs": ["write", "execute"]}
    ],
    "requires": [
      {"capabilityId": "file-access", "verbs": ["read", "write"]},
      {"capabilityId": "terminal", "verbs": ["execute"]}
    ]
  }
}
```

### Skill Requirements

Skills declare what capabilities an agent must provide to use the skill:

```json
{
  "id": "debugging",
  "requires": {
    "capabilities": ["code-analysis", "file-access"]
  }
}
```

## Matching Algorithm

### Basic Match

A skill can be used by an agent if the agent provides all required capabilities:

```
CanUseSkill(agent, skill) -> bool:
    for cap in skill.requires.capabilities:
        if cap not in agent.capabilities.provides:
            return false
    return true
```

### Verb-Level Match

For more granular control, match at the verb level:

```
CanUseSkillWithVerbs(agent, skill, requiredVerbs) -> bool:
    for cap, verbs in requiredVerbs:
        agentCap = findCapability(agent.capabilities.provides, cap)
        if agentCap is nil:
            return false
        for verb in verbs:
            if verb not in agentCap.verbs:
                return false
    return true
```

### Example Match

```
Agent alice:
  provides: [{code-analysis: [read, analyze]}, {file-access: [read]}]

Skill debugging:
  requires.capabilities: [code-analysis, file-access]

Match check:
  - code-analysis: alice provides [read, analyze] ✓
  - file-access: alice provides [read] ✓

Result: alice CAN use debugging skill
```

```
Agent bob:
  provides: [{documentation: [write]}]

Skill debugging:
  requires.capabilities: [code-analysis, file-access]

Match check:
  - code-analysis: bob does NOT provide ✗

Result: bob CANNOT use debugging skill
```

## Common Capabilities

### Code-Related

| Capability | Common Verbs | Description |
|------------|--------------|-------------|
| `code-analysis` | read, analyze, annotate | Understand code structure |
| `code-modification` | edit, refactor, format | Change existing code |
| `code-generation` | write, template, scaffold | Create new code |
| `testing` | write, execute, validate | Test operations |
| `debugging` | diagnose, trace, fix | Debug operations |

### System-Related

| Capability | Common Verbs | Description |
|------------|--------------|-------------|
| `file-access` | read, write, delete | File system operations |
| `terminal` | execute, stream | Command execution |
| `network` | fetch, request, stream | Network operations |
| `database` | query, insert, update | Database operations |

### Communication-Related

| Capability | Common Verbs | Description |
|------------|--------------|-------------|
| `messaging` | send, receive, broadcast | Message passing |
| `notification` | alert, notify, escalate | Alert systems |
| `logging` | info, warn, error | Log generation |

## API Usage

### Filter Skills by Agent Capabilities

```bash
# Get only skills the agent can use
GET /api/v1/agents/{agentId}/available-skills
```

Response:
```json
{
  "agentId": "alice",
  "availableSkills": [
    {
      "skill": {"id": "debugging", "name": "Debugging Expert"},
      "matchedCapabilities": ["code-analysis", "file-access"],
      "missingCapabilities": []
    }
  ],
  "unavailableSkills": [
    {
      "skill": {"id": "deployment", "name": "Deployment Manager"},
      "matchedCapabilities": ["terminal"],
      "missingCapabilities": ["kubernetes", "docker"]
    }
  ]
}
```

### Find Agents for a Skill

```bash
# Get agents that can use a specific skill
GET /api/v1/skills/{skillId}/capable-agents
```

Response:
```json
{
  "skillId": "debugging",
  "requiredCapabilities": ["code-analysis", "file-access"],
  "capableAgents": [
    {"id": "alice", "displayName": "Alice", "matchScore": 1.0},
    {"id": "bob", "displayName": "Bob", "matchScore": 0.8}
  ]
}
```

### Validate Assignment

```bash
# Check if a skill assignment is valid
POST /api/v1/validate/skill-assignment
{
  "agentId": "alice",
  "skillId": "debugging"
}
```

Response:
```json
{
  "valid": true,
  "agent": {"id": "alice"},
  "skill": {"id": "debugging"},
  "matchedCapabilities": ["code-analysis", "file-access"],
  "missingCapabilities": [],
  "warnings": []
}
```

## Capability Inheritance

Capabilities can be hierarchical:

```
code-modification
├── code-editing     (subset)
├── code-refactoring (subset)
└── code-formatting  (subset)
```

If an agent provides `code-modification`, it implicitly provides all sub-capabilities.

### Defining Hierarchies

```json
// store/capabilities/hierarchy.json
{
  "code-modification": {
    "includes": ["code-editing", "code-refactoring", "code-formatting"]
  },
  "file-access": {
    "includes": ["file-read", "file-write", "file-delete"]
  }
}
```

### Resolution

```
Agent provides: [code-modification]
Skill requires: [code-editing]

Resolution:
  - code-editing is included in code-modification
  - Match: ✓
```

## Capability Discovery

### List All Capabilities

```bash
GET /api/v1/capabilities
```

Returns all known capabilities with descriptions and common verbs.

### Suggest Capabilities

```bash
POST /api/v1/capabilities/suggest
{
  "skillContent": "This skill helps debug memory leaks..."
}
```

Returns suggested capabilities based on skill content analysis.

## Best Practices

### For Skill Authors

1. **Be specific**: List only truly required capabilities
2. **Use standard names**: Check existing capabilities before creating new ones
3. **Document requirements**: Explain why each capability is needed
4. **Test with minimal agents**: Verify skill works with just the required capabilities

### For Agent Operators

1. **Start minimal**: Add capabilities as needed
2. **Use verbs precisely**: Only grant verbs the agent actually needs
3. **Review periodically**: Remove capabilities no longer used
4. **Document custom capabilities**: Keep a registry of organization-specific capabilities

### For System Designers

1. **Define a capability taxonomy**: Establish hierarchies upfront
2. **Version capabilities**: Use semantic versioning for capability definitions
3. **Provide discovery**: Make it easy to find what capabilities exist
4. **Validate early**: Check capability matches before assignment, not at runtime

## Error Messages

| Error | Meaning | Resolution |
|-------|---------|------------|
| `capability_not_found` | Referenced capability doesn't exist | Check capability name spelling |
| `verb_not_found` | Verb not valid for capability | Check valid verbs for capability |
| `insufficient_capabilities` | Agent lacks required capabilities | Add missing capabilities to agent |
| `circular_hierarchy` | Capability hierarchy has a cycle | Fix hierarchy definition |

## Example: Complete Flow

```
1. Define capabilities
   - code-analysis: [read, analyze]
   - debugging: [diagnose, fix]
   - file-access: [read, write]

2. Create agent with capabilities
   Agent alice:
     provides: [code-analysis, debugging, file-access]
     requires: [terminal]

3. Create skill with requirements
   Skill "advanced-debugging":
     requires.capabilities: [code-analysis, debugging]

4. Validate and assign
   POST /api/v1/validate/skill-assignment
     agentId: alice
     skillId: advanced-debugging

   Response: {valid: true, matchedCapabilities: [code-analysis, debugging]}

5. Use skill
   Agent alice now has "advanced-debugging" in effective skills
```

## Related Documentation

- [SWARM-MODEL.md](SWARM-MODEL.md) - Overall architecture
- [EFFECTIVE-SKILLS.md](EFFECTIVE-SKILLS.md) - Skill computation
- [PERSONA-SYSTEM.md](PERSONA-SYSTEM.md) - Agent identity
