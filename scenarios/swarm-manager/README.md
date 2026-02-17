# Swarm Manager

Central command center for managing the Vrooli scenario ecosystem - orchestrating backlog work, scenario lifecycle, execution control, and prompt management.

## Purpose

Swarm Manager is the **staging and review layer** between agent teams and scenario execution. Agent teams in [prompt-manager](../prompt-manager/) analyze codebases and produce plans (fixes, ideas, refactors), but instead of executing directly, they deposit those plans as backlog items here. This gives operators a single place to:

- **Review** all agent-generated plans before anything executes
- **Refine** plans using the built-in Idea Agent (clarify → suggest → enhance)
- **Control Execution**: Run approved work in manual, scheduled, or YOLO mode
- **Manage Scenarios**: View, configure, and manage the lifecycle of all scenarios
- **Track Progress**: Monitor scenario health and execution runs

Think of it as a **pull request review for agent work** — agents propose, you review and refine, then approve for execution.

## Architecture

```
swarm-manager/
├── api/           # Go API (gorilla/mux + api-core)
├── cli/           # Go CLI (cli-core ScenarioApp)
├── ui/            # React + Vite + TypeScript
├── ideas/         # Git-tracked backlog (idea)
├── research/      # Git-tracked backlog (research)
├── fix/           # Git-tracked backlog (fix)
├── execute/       # Git-tracked backlog (execute)
├── requirements/  # Requirement tracking modules
└── docs/          # Quick start, concepts/reference, and internal docs
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
3. **Execution** - Pending/scheduled, running, completed, and failed runs
4. **Prompts** - View, edit, preview, simulate, and version prompt-manager skills
5. **Settings** - Theme, execution policy, and insights configuration

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
- **None** - Filesystem persistence is used for backlog, settings, queue, and execution runs. Scenario inventory is sourced from the Vrooli CLI.

### Required Scenarios
- **agent-manager** - Spawning agents for automated work
- **ecosystem-manager** - Scenario initialization and improvement
- **prompt-manager** - Prompt skill resolution, preview, simulate, and versioning

### Optional Scenarios (P1)
- knowledge-observatory, visited-tracker, scenario-completeness-scoring
- app-issue-tracker, test-genie

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
swarm-manager backlog files <kind> <name>
swarm-manager backlog file get <kind> <name> <path> [--out local-path]
swarm-manager backlog file upload <kind> <name> <local-file> [--path backlog/subdir]
swarm-manager backlog queue <kind> <name> [--mode manual|scheduled|yolo] [--delay-seconds N] [--operation generator|improver]
swarm-manager backlog research <kind> <name> '<json>'
swarm-manager backlog convert <kind> <name> <target-kind> [target-name]
swarm-manager backlog prompt-trace <kind> <name>

swarm-manager scenarios list [--search ... --status ... --tags ...]
swarm-manager scenarios get <name>
swarm-manager scenarios update <name> '<json>'
swarm-manager scenarios delete <name> [--archive]
swarm-manager scenarios files <name>
swarm-manager scenarios start <name>
swarm-manager scenarios stop <name>
swarm-manager scenarios restart <name>
swarm-manager scenarios spec-sync-archive <name> [--preset PRESET] [--paths path1,path2]

swarm-manager execution list [--status ... --mode ... --started-by ...]
swarm-manager execution get <execution-id>
swarm-manager execution create <backlog-kind> <backlog-name> [--mode manual|scheduled|yolo]
swarm-manager execution policy get
swarm-manager execution policy update --mode manual|scheduled|yolo [--delay-seconds N]
swarm-manager execution start <execution-id>
swarm-manager execution cancel <execution-id>
swarm-manager execution retry <execution-id>
swarm-manager execution prompt-trace <execution-id>

swarm-manager settings get
swarm-manager settings update '<json>'

swarm-manager queue list
swarm-manager queue create <kind> '<payload-json>'
swarm-manager queue delete <id>

swarm-manager agent-manager status

swarm-manager prompts map
swarm-manager prompts skills [--contains FILTER]
swarm-manager prompts skill-get <skill-id>
swarm-manager prompts skill-update <skill-id> '<json-or-@file>'
swarm-manager prompts skill-versions <skill-id>
swarm-manager prompts skill-revert <skill-id> <version>
swarm-manager prompts preview <skill-id> [--vars KEY=VALUE,...] [--with-scope]
swarm-manager prompts simulate <kind> [--mode MODE] [--operation OP] [--item-title TITLE] [--item-folder PATH]
```

## Integration Points

- All agent work via `agent-manager` API (never direct agent calls)
- All scenario operations via `ecosystem-manager` API
- Prompt skill resolution, preview, and simulation via `prompt-manager` API
- Execution run orchestration via `agent-manager` APIs

## Documentation

- [PRD.md](./PRD.md) - Product requirements and operational targets
- [docs/internal/PROGRESS.md](./docs/internal/PROGRESS.md) - Development progress log
- [docs/internal/PROBLEMS.md](./docs/internal/PROBLEMS.md) - Known issues and deferred backlog items
- [docs/guides/research-notes.md](./docs/guides/research-notes.md) - Research notes and uniqueness analysis
- [docs/guides/idea-agent-workflow.md](./docs/guides/idea-agent-workflow.md) - Idea Agent 3-phase pipeline (clarify → suggest → enhance)
- [requirements/README.md](./requirements/README.md) - Requirement tracking modules
