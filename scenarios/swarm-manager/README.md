# Swarm Manager

Central command center for managing the Vrooli scenario ecosystem - orchestrating backlog work, scenario lifecycle, autonomous recommendations, and self-improvement insights.

## Purpose

Swarm Manager is the primary interface for humans and agents to:
- **Manage Backlog**: Create, organize, and queue research, ideas, fixes, and execution tasks
- **Control Scenarios**: View, configure, and manage the lifecycle of all scenarios
- **Configure Recommendations**: Set up autonomous improvement recommendations (off/suggestions/yolo)
- **Track Progress**: Monitor scenario health, completeness, and test results

## Architecture

```
swarm-manager/
├── api/           # Go API (Gin framework)
├── cli/           # Go CLI (Cobra)
├── ui/            # React + Vite + TypeScript
├── ideas/         # Git-tracked backlog (idea)
├── research/      # Git-tracked backlog (research)
├── fix/           # Git-tracked backlog (fix)
├── execute/       # Git-tracked backlog (execute)
├── requirements/  # Requirement tracking modules
└── docs/          # PROGRESS.md, PROBLEMS.md, RESEARCH.md
```

## Quick Start

```bash
# Navigate to scenario
cd scenarios/swarm-manager

# Setup (build API, CLI, UI)
make setup

# Start development servers
make start

# Run tests
make test

# View logs
make logs

# Stop services
make stop
```

## UI Tabs

1. **Backlog** - Tabbed backlog for research, ideas, fixes, and execution
2. **Scenarios** - Grid of scenario cards with search/filter, click for lifecycle controls
3. **Recommendations** - Pending recommendations + engine mode controls
4. **Settings** - Theme, engine configs, integration status

## Backlog Structure

Backlog items are stored as git-tracked folders by kind:

```
ideas/
├── my-scenario-idea/
│   ├── spec.json        # Required: name, title, description, status, priority
│   ├── notes.md         # Optional context
│   ├── mockup.png       # Optional visuals
│   └── research/        # Optional supporting files
research/
├── discovery-pass/
│   ├── spec.json
│   └── research/
│       └── summary.md
fix/
├── bugfix-auth-timeout/
│   └── spec.json
execute/
├── rollout-plan/
│   └── spec.json
```

## Dependencies

### Required Resources
- **None** - Filesystem persistence is used for backlog, settings, queue, and recommendations. Scenario inventory is sourced from the Vrooli CLI.

### Required Scenarios
- **agent-manager** - Spawning agents for automated work
- **ecosystem-manager** - Scenario initialization and improvement

### Optional Scenarios (P1)
- knowledge-observatory, visited-tracker, scenario-completeness-scoring
- app-issue-tracker, test-genie, prompt-manager

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `API_PORT` | Go API server port |
| `UI_PORT` | React UI port |
| `WS_PORT` | WebSocket for real-time updates |

## CLI Commands

```bash
swarm-manager backlog list [--kinds idea,research,fix,execute]
swarm-manager backlog get <kind> <name>
swarm-manager backlog create '<json>'
swarm-manager backlog update <kind> <name> '<json>'
swarm-manager backlog delete <kind> <name>
swarm-manager backlog queue <kind> <name> [operation]
swarm-manager backlog research <kind> <name> '<json>'
swarm-manager backlog convert <kind> <name> '<json>'

swarm-manager scenarios list [--search ... --status ... --tags ...]
swarm-manager scenarios get <name>
swarm-manager scenarios update <name> '<json>'
swarm-manager scenarios delete <name> [--archive]

swarm-manager recommendations list [--status ... --scenario ... --type ...]
swarm-manager recommendations refresh
swarm-manager recommendations create '<json>'
swarm-manager recommendations update <id> <status>

swarm-manager settings get
swarm-manager settings update '<json>'

swarm-manager queue list
swarm-manager queue create <kind> '<payload-json>'
swarm-manager queue delete <id>
```

## Integration Points

- All agent work via `agent-manager` API (never direct agent calls)
- All scenario operations via `ecosystem-manager` API
- Recommendation engine queries dependent scenario APIs for data sources

## Documentation

- [PRD.md](./PRD.md) - Product requirements and operational targets
- [docs/PROGRESS.md](./docs/PROGRESS.md) - Development progress log
- [docs/PROBLEMS.md](./docs/PROBLEMS.md) - Known issues and deferred backlog items
- [docs/RESEARCH.md](./docs/RESEARCH.md) - Research notes and uniqueness analysis
- [requirements/README.md](./requirements/README.md) - Requirement tracking modules
