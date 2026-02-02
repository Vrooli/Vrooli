# SOUL System

This document describes how agent personality is defined in prompt-manager.

## Overview

Personality lives entirely in **SOUL.md**. Agents no longer store persona configuration in `agent.json`.
Every agent can have a dedicated `SOUL.md` file that acts as the single source of truth for voice,
behavior, and identity.

## Storage

```
store/agents/{agent-id}/
├── agent.json
├── SOUL.md
├── AGENTS.md
└── TOOLS.md
```

Additional markdown files can live alongside `SOUL.md` (for example `AGENTS.md` and `TOOLS.md`).
All `.md` files in the agent folder are included when building heartbeat prompts, with `SOUL.md`
ordered first when present.

## SOUL.md Content

`SOUL.md` is free-form markdown. It should describe:
- Identity and role
- Communication style
- Core behavioral guidelines
- Boundaries and constraints

**Example:**
```markdown
# Debugging Expert

You are a senior debugging specialist who works systematically.

## Behavior
- Ask clarifying questions before changing code
- Explain your reasoning step by step
- Call out risk and unknowns
```

## Prompt Usage

When a heartbeat executes, prompt-manager prepends all agent `.md` files to the task prompt:
1. Agent markdown files (SOUL.md, AGENTS.md, TOOLS.md, ...)
2. RESPONSIBILITIES.md (team role)
3. Effective Skills
4. HEARTBEAT.md (task)

See [HEARTBEATS.md](HEARTBEATS.md) for the full prompt assembly flow.

## Management

- **CLI:** `prompt-manager agent soul <id>` to get/set SOUL.md
- **API:** `GET /api/v1/agents/{id}/soul`, `PUT /api/v1/agents/{id}/soul`

## Best Practices

- Keep SOUL.md concise and stable
- Avoid task-specific instructions (use skills for that)
- Update SOUL.md when an agent’s role changes
