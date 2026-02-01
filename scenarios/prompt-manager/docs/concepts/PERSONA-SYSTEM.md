# Persona System

This document describes the agent persona system in prompt-manager, which defines how agents present themselves and interact.

## Overview

A persona is a configuration that shapes an agent's identity, communication style, and behavioral characteristics. Personas enable agents to have consistent personalities that users and other agents can recognize and interact with predictably.

## Persona Structure

Personas are stored within the agent entity:

```json
{
  "id": "alice",
  "displayName": "Alice",
  "persona": {
    "entry": "personas/debugging-expert.md",
    "voice": "professional",
    "traits": ["methodical", "patient", "thorough"],
    "systemPromptPrefix": "You are Alice, a senior debugging specialist..."
  }
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `entry` | string | Path to detailed persona file (optional) |
| `voice` | string | Communication style identifier |
| `traits` | string[] | Behavioral characteristics |
| `systemPromptPrefix` | string | Text prepended to all prompts |

## Entry File

The `entry` field points to a detailed persona document that can include:

- Background story
- Expertise areas
- Communication preferences
- Example interactions
- Boundaries and limitations

**Example:** `store/personas/debugging-expert.md`
```markdown
# Debugging Expert Persona

## Background
You are a seasoned software engineer with 15 years of experience
specializing in systematic debugging and root cause analysis.

## Expertise Areas
- Stack trace analysis
- Memory leak detection
- Race condition identification
- Performance profiling

## Communication Style
- Always ask clarifying questions before diving into solutions
- Explain your reasoning step by step
- Acknowledge uncertainty when appropriate
- Celebrate small wins during debugging sessions

## Interaction Patterns
When presented with a bug:
1. First, understand what the expected behavior is
2. Reproduce the issue if possible
3. Form hypotheses about the cause
4. Test hypotheses systematically
5. Explain the root cause and fix

## Boundaries
- Do not make changes without explaining them
- Admit when a problem is outside your expertise
- Suggest escalation when appropriate
```

## Voice Types

The `voice` field indicates the communication style:

| Voice | Description | Example Usage |
|-------|-------------|---------------|
| `professional` | Formal, business-appropriate | Enterprise environments |
| `casual` | Friendly, conversational | Developer tools |
| `technical` | Precise, jargon-appropriate | Engineering teams |
| `supportive` | Encouraging, patient | Onboarding, education |
| `concise` | Brief, to-the-point | CLI tools, automation |

**Custom voices** can be defined by combining traits and system prompts.

## Traits

Traits are behavioral characteristics that influence how the agent operates:

### Common Traits

| Trait | Behavior |
|-------|----------|
| `methodical` | Works through problems step by step |
| `patient` | Takes time to explain, doesn't rush |
| `thorough` | Considers edge cases, validates assumptions |
| `creative` | Suggests novel approaches |
| `cautious` | Warns about risks, suggests testing |
| `efficient` | Focuses on speed and minimal steps |
| `collaborative` | Seeks input, acknowledges contributions |
| `autonomous` | Makes decisions independently |

### Trait Combinations

Traits work together to shape behavior:

```json
{
  "traits": ["methodical", "cautious", "collaborative"]
}
```

This agent would:
- Work through problems step by step (methodical)
- Warn about potential risks (cautious)
- Seek confirmation before making changes (collaborative)

## System Prompt Prefix

The `systemPromptPrefix` is prepended to every prompt sent to the LLM, establishing the agent's identity:

```json
{
  "systemPromptPrefix": "You are Alice, a senior debugging specialist. You approach problems systematically, always starting by understanding the expected behavior before investigating the actual behavior. You are patient and thorough, explaining your reasoning at each step."
}
```

### Building Effective Prefixes

**Structure:**
1. Identity statement ("You are...")
2. Primary role/expertise
3. Key behavioral traits
4. Interaction preferences

**Example:**
```
You are [Name], a [role] specializing in [expertise areas].

You are [trait 1], [trait 2], and [trait 3].

When working on tasks:
- [Behavioral guideline 1]
- [Behavioral guideline 2]
- [Behavioral guideline 3]
```

## Persona Resolution

When an agent executes a skill, the persona is applied:

```
┌─────────────────────────────────────────────────────────┐
│                 PROMPT CONSTRUCTION                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  1. Load agent persona                                  │
│     systemPromptPrefix from agent.persona               │
│                                                          │
│  2. Load persona entry (if specified)                   │
│     Additional context from entry file                  │
│                                                          │
│  3. Load skill content                                  │
│     The actual skill instructions                       │
│                                                          │
│  4. Construct final prompt:                             │
│     [systemPromptPrefix]                                │
│     [persona entry content]                             │
│     [skill content]                                     │
│     [user input]                                        │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Example Personas

### Debugging Specialist

```json
{
  "persona": {
    "voice": "technical",
    "traits": ["methodical", "thorough", "patient"],
    "systemPromptPrefix": "You are a debugging specialist. You approach bugs systematically: understand expected behavior, reproduce the issue, form hypotheses, test them methodically, and explain the root cause clearly."
  }
}
```

### Code Reviewer

```json
{
  "persona": {
    "voice": "professional",
    "traits": ["thorough", "constructive", "educational"],
    "systemPromptPrefix": "You are a code reviewer focused on both correctness and growth. You identify issues clearly, explain why they matter, and suggest improvements constructively. You celebrate good patterns when you see them."
  }
}
```

### Technical Writer

```json
{
  "persona": {
    "voice": "supportive",
    "traits": ["clear", "organized", "user-focused"],
    "systemPromptPrefix": "You are a technical writer who makes complex topics accessible. You organize information logically, use concrete examples, and always consider the reader's perspective and knowledge level."
  }
}
```

### Automation Agent

```json
{
  "persona": {
    "voice": "concise",
    "traits": ["efficient", "autonomous", "cautious"],
    "systemPromptPrefix": "You are an automation agent. You complete tasks efficiently with minimal interaction. You proceed autonomously when confident, but pause and ask for clarification when uncertain about destructive operations."
  }
}
```

## Persona Management

### Creating a Persona

Via CLI:
```bash
# Create agent with inline persona
prompt-manager agent create "Alice" \
  --voice=technical \
  --traits=methodical,thorough \
  --system-prompt="You are Alice, a debugging specialist..."

# Or update existing agent
prompt-manager agent update alice \
  --persona-entry=personas/debugging-expert.md
```

Via API:
```bash
curl -X PUT http://localhost:PORT/api/v1/agents/alice \
  -H "Content-Type: application/json" \
  -d '{
    "persona": {
      "voice": "technical",
      "traits": ["methodical", "thorough"],
      "systemPromptPrefix": "You are Alice..."
    }
  }'
```

### Persona Inheritance

Personas don't inherit from teams, but teams can recommend personas for roles:

```json
// team.json
{
  "rolePersonaRecommendations": {
    "developer": {
      "voice": "technical",
      "traits": ["efficient", "collaborative"]
    },
    "support": {
      "voice": "supportive",
      "traits": ["patient", "helpful"]
    }
  }
}
```

These are recommendations, not enforced - each agent maintains its own persona.

## Best Practices

### Do

- Keep `systemPromptPrefix` concise (under 500 characters)
- Use specific, actionable traits
- Match voice to the agent's primary audience
- Use entry files for detailed persona information
- Test personas with representative tasks

### Don't

- Overload with too many traits (3-5 is ideal)
- Include task-specific instructions in persona (that's what skills are for)
- Make personas too rigid - allow for situational adaptation
- Forget to update persona when agent's role changes

## Related Documentation

- [SWARM-MODEL.md](SWARM-MODEL.md) - Agent architecture overview
- [CAPABILITY-MATCHING.md](CAPABILITY-MATCHING.md) - Agent capabilities
- [EFFECTIVE-SKILLS.md](EFFECTIVE-SKILLS.md) - How agents get their skills
