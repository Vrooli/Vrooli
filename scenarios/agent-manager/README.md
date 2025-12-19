# agent-manager

Central orchestration and governance layer for running AI agents against the codebase in a controlled, reviewable, and extensible way.

## Purpose

agent-manager provides:
- **Single control plane** for all agent executions in the Vrooli ecosystem
- **Sandbox-first execution** via workspace-sandbox for safe, isolated runs
- **Policy enforcement** for approval rules, scopes, and concurrency
- **Event tracking** with append-only logs of all agent activity
- **Approval workflows** with diff review before canonical repo changes

## Quick Start

```bash
# From repo root
cd scenarios/agent-manager

# Build and install
make setup

# Start services
make start

# Run tests
make test
```

## Architecture

The codebase uses a **screaming architecture** where folder structure expresses the domain:

```
api/internal/
├── domain/           # Core entities (Task, Run, AgentProfile, Policy)
├── orchestration/    # Coordination layer - wires components together
├── adapters/         # External integration seams
│   ├── runner/       # Agent runner implementations (claude-code, codex, opencode)
│   ├── sandbox/      # workspace-sandbox integration
│   ├── event/        # Event streaming and storage
│   └── artifact/     # Diff and artifact collection
├── policy/           # Policy evaluation logic
├── repository/       # Persistence interfaces
├── handlers/         # HTTP handlers (thin presentation layer)
└── config/           # Configuration management
```

### Architectural Seams

The architecture is built around **deliberate boundaries (seams)** that enable testing and extensibility:

| Seam | Purpose | Interface |
|------|---------|-----------|
| **Runner** | Abstract agent execution | `runner.Runner` |
| **Sandbox** | Abstract isolation layer | `sandbox.Provider` |
| **Events** | Abstract event capture/storage | `event.Store`, `event.Collector` |
| **Policy** | Abstract policy decisions | `policy.Evaluator` |
| **Repository** | Abstract persistence | `*Repository` interfaces |

See [docs/SEAMS.md](docs/SEAMS.md) for detailed documentation of architectural boundaries.

### Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│                     agent-manager                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  Tasks   │  │  Runs    │  │  Events  │  │ Policies │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
│       │             │             │             │           │
│       └─────────────┴─────────────┴─────────────┘           │
│                            │                                 │
│  ┌─────────────────────────┴─────────────────────────────┐  │
│  │                  RunnerAdapter Interface               │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────┐   │  │
│  │  │ claude-code│  │   codex    │  │    opencode    │   │  │
│  │  └────────────┘  └────────────┘  └────────────────┘   │  │
│  └───────────────────────────────────────────────────────┘  │
└──────────────────────────┬──────────────────────────────────┘
                           │
               ┌───────────┴───────────┐
               │   workspace-sandbox   │
               │   (isolation layer)   │
               └───────────────────────┘
```

## Core Concepts

### AgentProfile
Defines how an agent runs: runner type, model, timeout, allowed tools.

### Task
Defines what needs to be done: title, description, scope path, context.

### Run
A concrete execution attempt linking a Task to an AgentProfile within a sandbox.

### RunEvent
Append-only event stream capturing all agent activity (logs, messages, tool calls).

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/profiles` | Create agent profile |
| GET | `/api/v1/profiles` | List agent profiles |
| POST | `/api/v1/tasks` | Create task |
| GET | `/api/v1/tasks` | List tasks |
| POST | `/api/v1/runs` | Create run |
| GET | `/api/v1/runs/:id/events` | Stream run events |
| POST | `/api/v1/runs/:id/approve` | Approve run |
| POST | `/api/v1/runs/:id/reject` | Reject run |

## CLI Commands

```bash
# Profile management
agent-manager profile create --name "default" --runner claude-code
agent-manager profile list

# Task management
agent-manager task create --title "Fix bug" --scope-path "src/"
agent-manager task list

# Run management
agent-manager run create --task <id> --profile <id>
agent-manager run logs <id>
agent-manager run diff <id>
agent-manager run approve <id>
agent-manager run reject <id>
```

## Dependencies

**Required**:
- PostgreSQL - task, run, event storage
- workspace-sandbox - isolation and diff management

**Runners** (at least one required):
- claude-code resource
- codex resource
- opencode resource

## Documentation

- [PRD.md](./PRD.md) - Product requirements and operational targets
- [docs/RESEARCH.md](./docs/RESEARCH.md) - Research and architecture decisions
- [docs/PROBLEMS.md](./docs/PROBLEMS.md) - Known issues and deferred ideas
- [docs/PROGRESS.md](./docs/PROGRESS.md) - Development progress log
- [requirements/README.md](./requirements/README.md) - Requirements registry

## Related Scenarios

| Scenario | Relationship |
|----------|--------------|
| workspace-sandbox | Required - provides isolation |
| agent-inbox | Consumer - uses for agent chat |
| ecosystem-manager | Consumer - uses for scenario generation |

## Development

```bash
# Start development servers
make dev

# Run specific tests
go test ./api/...

# Format code
make fmt

# Lint
make lint
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `API_PORT` | API server port |
| `UI_PORT` | UI server port |
| `WS_PORT` | WebSocket port |
| `DATABASE_URL` | PostgreSQL connection string |
| `WORKSPACE_SANDBOX_URL` | workspace-sandbox API URL |
