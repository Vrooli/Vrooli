# Swarm Manager

Central command center for managing the Vrooli scenario ecosystem - orchestrating idea backlogs, scenario lifecycle, autonomous recommendations, and self-improvement insights.

## Purpose

Swarm Manager is the primary interface for humans and agents to:
- **Manage Ideas**: Create, organize, and queue scenario ideas for implementation
- **Control Scenarios**: View, configure, and manage the lifecycle of all scenarios
- **Configure Recommendations**: Set up autonomous improvement recommendations (off/suggestions/yolo)
- **Track Progress**: Monitor scenario health, completeness, and test results

## Architecture

```
swarm-manager/
├── api/           # Go API (Gin framework)
├── cli/           # Go CLI (Cobra)
├── ui/            # React + Vite + TypeScript
├── ideas/         # Git-tracked idea backlog (folder-per-idea)
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

1. **Ideas** - Todo-list style backlog, click for details with file tree + preview
2. **Scenarios** - Grid of scenario cards with search/filter, click for lifecycle controls
3. **Recommendations** - Pending recommendations + engine mode controls
4. **Settings** - Theme, engine configs, integration status

## Idea Backlog Structure

Ideas are stored as git-tracked folders in `ideas/`:

```
ideas/
├── my-scenario-idea/
│   ├── spec.json        # Required: name, title, description, status, priority
│   ├── notes.md         # Optional context
│   ├── mockup.png       # Optional visuals
│   └── research/        # Optional supporting files
```

## Dependencies

### Required Resources
- **None** - Filesystem persistence is used for ideas, settings, queue, and recommendations. Scenario inventory is sourced from the Vrooli CLI.

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
swarm-manager ideas list
swarm-manager ideas get <name>
swarm-manager ideas create '<json>'
swarm-manager ideas update <name> '<json>'
swarm-manager ideas delete <name>
swarm-manager ideas queue <name> [operation]
swarm-manager ideas research <name> '<json>'

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
- [docs/PROBLEMS.md](./docs/PROBLEMS.md) - Known issues and deferred ideas
- [docs/RESEARCH.md](./docs/RESEARCH.md) - Research notes and uniqueness analysis
- [requirements/README.md](./requirements/README.md) - Requirement tracking modules
