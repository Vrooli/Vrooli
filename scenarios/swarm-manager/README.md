# Swarm Manager

Central command center for managing the Vrooli scenario ecosystem - orchestrating backlog work, scenario lifecycle, execution control, and prompt management.

## Purpose

Swarm Manager is the **staging and review layer** between agent teams and scenario execution. Agent teams in [prompt-manager](../prompt-manager/README.md) analyze codebases and produce plans (fixes, ideas, refactors), but instead of executing directly, they deposit those plans as backlog items here. This gives operators a single place to:

- **Review** all agent-generated plans before anything executes
- **Refine** plans using the built-in workshop loop and prompt catalog
- **Control Execution**: Run approved work in manual, scheduled, or YOLO mode
- **Manage Scenarios**: View, configure, and manage the lifecycle of all scenarios
- **Track Progress**: Monitor scenario health and execution runs

Think of it as a **pull request review for agent work** — agents propose, you review and refine, then approve for execution.

## Target Operating Model

Swarm Manager is the project-work ledger and operator-control surface; it is
not the runtime for programmatic agent methodology. Human-led conversations are
kept as Agent Sessions backed by Agent Manager Runs. Whenever code composes an
agent prompt and consumes a typed outcome, Swarm supplies only the typed input
snapshot and applies the typed result while Agent Manager owns the declared
Workflow's execution, validation, branching, retries, waits, and provenance.

The future-state model for intake, domain concepts, integrations, and all
transition types is documented in
[Target Operating Model](./docs/concepts/TARGET-OPERATING-MODEL.md). It is a
documentation-first migration contract; the current architecture and its narrow
workflow pilots are described separately in
[Architecture](./docs/concepts/ARCHITECTURE.md).

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

## UI Surfaces

The UI is a single **Plan board** (`/plan`) — a lens-driven workspace over
backlog, scenarios, initiatives, executions, and captures. The former standalone
**Scenarios** and **Execution** list tabs were absorbed into it; their old routes
(`/scenarios`, `/executions`, `/operations`, `/command-post`) now redirect to
`/plan`, while the detail pages (`/scenarios/{name}`, `/executions/{id}`,
`/backlog/{kind}/{name}`, `/initiatives/{name}`, `/operating-modes/{mode}`) remain
directly reachable.

- **Plan** (`/plan`) — the primary board; decisions live in its drawer.
- **Graph** (`/graph`) — the topology surface (focus is query state inside it).
- **Stats** (`/stats`) — aggregate metrics over the same workspace.
- **Records** (`/records`) — the learning-loop records browser.

## Backlog Structure

Backlog items are stored as git-tracked folders by kind:

```
ideas/
├── my-scenario-idea/
│   ├── spec.json        # Required metadata incl. plan_ref for the canonical plan-manager plan
│   ├── handoff/         # Generated at idea execution time for ecosystem-manager handoff
│   │   ├── brief.md
│   │   ├── manifest.json
│   │   └── source-index.json
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

## CLI Commands

```bash
swarm-manager backlog list [--kinds idea,research,fix,execute]
swarm-manager backlog get --kind <kind> --name <name>
swarm-manager backlog create --data '<json>'
swarm-manager backlog update --kind <kind> --name <name> --data '<json>'
swarm-manager backlog delete --kind <kind> --name <name>
swarm-manager backlog files --kind <kind> --name <name>
swarm-manager backlog file-get --kind <kind> --name <name> --path <path> [--out local-path]
swarm-manager backlog file-upload --kind <kind> --name <name> --path <path> --file <local-file>
swarm-manager backlog queue --kind <kind> --name <name> [--execute] [--force] [--mode manual|scheduled|yolo] [--delay-seconds N] [--operation generator|improver]
swarm-manager backlog research --kind <kind> --name <name> --data '<json>'
swarm-manager backlog prompt-trace --kind <kind> --name <name>

swarm-manager scenarios list [--search ... --status ... --tags ...]
swarm-manager scenarios get --name <name>
swarm-manager scenarios update --name <name> --data '<json>'
swarm-manager scenarios delete --name <name> [--archive]
swarm-manager scenarios files --name <name>
swarm-manager scenarios start --name <name>
swarm-manager scenarios stop --name <name>
swarm-manager scenarios restart --name <name>
swarm-manager scenarios spec-sync-archive --name <name> [--preset PRESET] [--paths path1,path2]

swarm-manager execution list [--status ... --mode ... --started-by ...]
swarm-manager execution get --id <execution-id>
swarm-manager execution create --kind <backlog-kind> --name <backlog-name> [--mode manual|scheduled|yolo]
swarm-manager execution policy get
swarm-manager execution policy update --mode manual|scheduled|yolo [--delay-seconds N]
swarm-manager execution start --id <execution-id>
swarm-manager execution cancel --id <execution-id>
swarm-manager execution retry --id <execution-id>
swarm-manager execution prompt-trace --id <execution-id>

swarm-manager settings get
swarm-manager settings update --data '<json>'

swarm-manager queue list
swarm-manager queue create --kind <kind>
swarm-manager queue delete --id <id>

swarm-manager agent-manager status

swarm-manager prompts catalog
swarm-manager prompts skills [--contains FILTER]
swarm-manager prompts skill-get --id <skill-id>
swarm-manager prompts skill-update --id <skill-id> --data '<json-or-@file>'
swarm-manager prompts skill-versions --id <skill-id>
swarm-manager prompts skill-revert --id <skill-id> --version <version>
swarm-manager prompts preview --id <skill-id> [--vars KEY=VALUE,...] [--with-scope]
swarm-manager prompts simulate --kind <kind> [--mode workshop|initialize|finalize] [--item-title TITLE] [--item-folder PATH]
```

Plan-backed primary executions use the bounded Agent Manager
`phased-plan-drain` workflow. Swarm pins the Plan Manager frontier and remains
the sole owner of approval and terminal result application; retry, fixup,
follow-up, and research conclusion retain their existing execution paths. The
workflow enforces the requested slice bound, makes independent review rejection
drive a reviewed same-conversation correction, and preserves blocked,
abstained, and budget-exhausted terminal outcomes. See
[the workflow seam](./docs/internal/SEAMS.md#phased-plan-workflow-pilot-seam).

`swarm-manager backlog update` uses sparse patch semantics. Omitted fields stay unchanged, empty strings clear scalar fields like `description`, and empty arrays clear list fields like `tags`, `depends_on`, or `acceptance_allow`.

CLI usage guardrail:
- Use the installed binary from `~/.vrooli/bin/swarm-manager` (or your PATH entry), not scenario-local binaries in `scenarios/swarm-manager/cli/`.
- Reinstall canonical binary with:
  ```bash
  cd scenarios/swarm-manager/cli && ./install.sh
  ```

## Integration Points

- All agent work via `agent-manager` API (never direct agent calls)
- Idea processing bridges into `ecosystem-manager` through a generated `handoff/` package plus task `notes` and `origin` metadata
- All scenario operations via `ecosystem-manager` API
- Prompt catalog inventory from swarm-manager, with skill rendering via `prompt-manager` API
- Execution run orchestration via `agent-manager` APIs

## Documentation

- [PRD.md](./PRD.md) - Product requirements and operational targets
- [docs/internal/PROGRESS.md](./docs/internal/PROGRESS.md) - Development progress log
- [docs/internal/PROBLEMS.md](./docs/internal/PROBLEMS.md) - Known issues and deferred backlog items
- [docs/guides/research-notes.md](./docs/guides/research-notes.md) - Research notes and uniqueness analysis
- [docs/guides/workshop-workflow.md](./docs/guides/workshop-workflow.md) - Universal workshop refinement loop for backlog planning
- [requirements/README.md](./requirements/README.md) - Requirement tracking modules
