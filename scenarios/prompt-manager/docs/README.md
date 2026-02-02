# prompt-manager Documentation

Welcome to the prompt-manager documentation. This scenario provides a **Skills + Agents + Teams** management system for orchestrating AI agent swarms.

## Quick Links

| Document | Description |
|----------|-------------|
| [Quick Start](QUICKSTART.md) | Get running in 5 minutes |
| [Architecture Overview](concepts/ARCHITECTURE.md) | System design and data flow |

## Core Concepts

The prompt-manager is built on three interconnected domains:

### Skills + Agents + Teams

| Concept | Document | Description |
|---------|----------|-------------|
| **Swarm Model** | [SWARM-MODEL.md](concepts/SWARM-MODEL.md) | The 3-domain architecture - how Skills, Agents, and Teams work together |
| **Relations** | [RELATIONS.md](concepts/RELATIONS.md) | Agent-skill and team-member junction records |
| **Effective Skills** | [EFFECTIVE-SKILLS.md](concepts/EFFECTIVE-SKILLS.md) | Algorithm for computing an agent's available skills |
| **SOUL System** | [PERSONA-SYSTEM.md](concepts/PERSONA-SYSTEM.md) | Agent personality defined in SOUL.md (+ optional .md files) |
| **Capability Matching** | [CAPABILITY-MATCHING.md](concepts/CAPABILITY-MATCHING.md) | Skill-to-agent matching based on capabilities |

### Visualization & Storage

| Concept | Document | Description |
|---------|----------|-------------|
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
│   ┌─────────────┐    Relations    ┌─────────────┐    Relations          │
│   │   SKILLS    │◄───────────────►│   AGENTS    │◄──────────────►       │
│   │             │                 │             │                       │
│   │  behaviors  │                 │  identities │        ┌─────────────┐│
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
- **3D visualization** - Monitor and coordinate agent swarms
- **File-based storage** - Human-readable, version-control friendly

## Getting Started

1. **New to prompt-manager?** Start with the [Quick Start Guide](QUICKSTART.md)
2. **Understanding the architecture?** Read [Swarm Model](concepts/SWARM-MODEL.md)
3. **Building integrations?** Check the [API Reference](reference/api-endpoints.md)
4. **Using the CLI?** See [CLI Commands](reference/cli-commands.md)

## Document Conventions

- `[CODE: path/to/file.go]` - References to source code
- `REQ-P0-001` - Requirement identifiers from `requirements/`
- `MOD-P0-001` - Module identifiers from `requirements/index.json`
