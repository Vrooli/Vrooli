# prompt-manager Documentation

Welcome to the prompt-manager documentation. This scenario provides a **Skills + Agents + Teams** management system for orchestrating AI agent swarms, with **Actions** for deterministic executable operations.

## Quick Links

| Document | Description |
|----------|-------------|
| [Quick Start](QUICKSTART.md) | Get running in 5 minutes |
| [Architecture Overview](concepts/ARCHITECTURE.md) | System design and data flow |

## Core Concepts

The prompt-manager is built on three coordination domains and one execution domain:

### Skills + Agents + Teams + Actions

| Concept | Document | Description |
|---------|----------|-------------|
| **Swarm Model** | [SWARM-MODEL.md](concepts/SWARM-MODEL.md) | How Skills, Agents, Teams, and Actions work together |
| **Actions** | [ACTIONS.md](concepts/ACTIONS.md) | Executable wrapper entity for deterministic Vrooli-controlled CLI operations |
| **Memory Promotion** | [MEMORY-PROMOTION.md](concepts/MEMORY-PROMOTION.md) | How typed observations graduate into Plan of Record, Skills, Actions, CLIs, or backlog |
| **Relations** | [RELATIONS.md](concepts/RELATIONS.md) | Team-member junction records |
| **SOUL System** | [PERSONA-SYSTEM.md](concepts/PERSONA-SYSTEM.md) | Agent personality defined in SOUL.md (+ optional .md files) |
| **Capability Matching** | [CAPABILITY-MATCHING.md](concepts/CAPABILITY-MATCHING.md) | Skill-to-agent matching based on capabilities |

### Visualization & Analysis

| Concept | Document | Description |
|---------|----------|-------------|
| **Relationship Graph** | [GRAPH.md](concepts/GRAPH.md) | Dependency graph mapping connections between teams, agents, skills, and CLIs |
| **3D World** | [3D-WORLD-ARCHITECTURE.md](concepts/3D-WORLD-ARCHITECTURE.md) | React Three Fiber visualization for agents |
| **Store Migration** | [STORE-MIGRATION.md](concepts/STORE-MIGRATION.md) | Migration from legacy storage to per-entity files |

## Reference

| Document | Description |
|----------|-------------|
| [API Endpoints](reference/api-endpoints.md) | Complete REST API documentation |
| [CLI Commands](reference/cli-commands.md) | All CLI commands and options |
| [Configuration](reference/configuration.md) | Environment variables and settings |

## Internal

Development documentation for contributors:

| Document | Description |
|----------|-------------|
| [Testing Seams](internal/SEAMS.md) | Interface boundaries for testing |
| [Progress](internal/PROGRESS.md) | Development milestones and ADRs |
| [Problems](internal/PROBLEMS.md) | Known issues and technical debt |

## Architecture at a Glance

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         prompt-manager                                   │
│                                                                          │
│   ┌─────────────┐    Guidance     ┌─────────────┐    Relations          │
│   │   SKILLS    │◄───────────────►│   AGENTS    │◄──────────────►       │
│   │             │                 │             │                       │
│   │  judgment   │                 │  identities│        ┌─────────────┐│
│   │  with packs │                 │  + souls   │        │    TEAMS    ││
│   └─────────────┘                 └─────────────┘        │             ││
│         │                               │                │ coordination││
│         │                               │                │ + roles     ││
│         ▼                               ▼                └─────────────┘│
│   ┌──────────────────────────────────────────┐                          │
│   │             EFFECTIVE SKILLS              │                          │
│   │   pins + relations + team role grants    │                          │
│   └──────────────────────────────────────────┘                          │
│                                                                          │
│   ┌──────────────────────────────────────────┐                          │
│   │          ACTIONS (PROPOSED)               │                          │
│   │ typed execution over Vrooli-owned CLIs   │                          │
│   └──────────────────────────────────────────┘                          │
│                                                                          │
│   ┌──────────────────────────────────────────┐                          │
│   │              3D WORLD UI                  │                          │
│   │   React Three Fiber visualization        │                          │
│   └──────────────────────────────────────────┘                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Key Features

- **Pack-based skill organization** - Core, local, and drafts with precedence ordering
- **Agent .md files** - Personality and operating guidance (SOUL.md is primary)
- **Team role grants** - Automatic skill assignment based on team membership
- **Effective skills computation** - Unified skill set from all sources
- **Relationship graph** - Visualize dependencies, detect orphans and cycles
- **3D visualization** - Monitor and coordinate agent swarms
- **File-based storage** - Human-readable, version-control friendly
- **Proposed Actions layer** - Typed wrappers over Vrooli-controlled CLI operations

## Entity Ontology

Use this ontology when deciding where persistent knowledge belongs:

```text
Knowledge = typed observations
Plan of Record = accepted durable truth
Skill = reusable judgment/process guidance
Action = typed executable operation
CLI = implementation of behavior
Backlog = unbuilt or broken behavior
```

Short classifier:

```text
If it says what is true -> Plan of Record.
If it says how to decide -> Skill.
If it says what to run -> Action.
If it says how it works -> CLI implementation.
If it says what is missing -> Backlog/capability-work.
If it is unverified or one-off -> typed knowledge under the most specific topic.
```

See [Memory Promotion](concepts/MEMORY-PROMOTION.md) for the full decision tree and [Actions](concepts/ACTIONS.md) for the proposed executable entity contract.

## Getting Started

1. **New to prompt-manager?** Start with the [Quick Start Guide](QUICKSTART.md)
2. **Understanding the architecture?** Read [Swarm Model](concepts/SWARM-MODEL.md)
3. **Building integrations?** Check the [API Reference](reference/api-endpoints.md)
4. **Using the CLI?** See [CLI Commands](reference/cli-commands.md)

## Document Conventions

- `[CODE: path/to/file.go]` - References to source code
- `REQ-P0-001` - Requirement identifiers from `requirements/`
- `MOD-P0-001` - Module identifiers from `requirements/index.json`
