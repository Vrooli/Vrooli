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
- **None** - Swarm Manager is filesystem-only for single-user local ops

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
swarm-manager ideas list [--status STATUS]
swarm-manager ideas create NAME --title TITLE
swarm-manager ideas queue ID

swarm-manager scenarios list
swarm-manager scenarios show ID
swarm-manager scenarios delete ID [--archive]

swarm-manager recommendations list
swarm-manager recommendations approve ID

swarm-manager settings show
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
