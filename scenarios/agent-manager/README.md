# agent-manager

Central orchestration and governance layer for running AI agents against the codebase in a controlled, reviewable, and extensible way.

## Purpose

agent-manager provides:
- **Single control plane** for all agent executions in the Vrooli ecosystem
- **Sandbox-first execution** via workspace-sandbox for safe, isolated runs
- **Policy enforcement** for approval rules, scopes, and concurrency
- **Portable role policy** with validated catalogs, immutable run snapshots, and explainable fallback
- **Event tracking** with append-only logs of all agent activity
- **Approval workflows** with diff review before canonical repo changes

## Choose the right primitive

Agent Manager has two deliberately different integration surfaces:

| Situation | Use | Owner of the prompt and interpretation |
| --- | --- | --- |
| A person is having a conversation with an agent. | **Run** (often owned by a consumer's session). | The person. |
| Code builds typed input and needs a typed result to continue. | Declared **Workflow**. | The workflow declaration. |

The distinction is independent of the number of agent turns: a single
programmatic agent turn is still a workflow. A workflow uses Runs as its
execution substrate, but adds a declared input/output contract, pinned prompt
provenance, validation, budgets, durable routing, and inspectable execution
history. Consumer code should retain only its two domain adapters: create a
bounded input snapshot and authorize/apply the typed terminal result. Prompt
assembly, result parsing, retries, loops, branches, and waits belong in the
workflow declaration.

Read [Workflow adoption](docs/guides/workflow-adoption.md) before adding an
agent integration, and [scenario declarations](docs/reference/scenario-declarations.md)
for the declaration schema and reconciliation commands.

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
├── rolepolicy/       # Portable-role catalog, atomic activation, and resource-owned resolution snapshots
├── permissionpolicy/ # Desired portable permissions, projection plans, and reconciliation audit evidence
├── conformance/      # Read-only Test Genie validation of consumer configuration
├── orchestration/    # Coordination layer - wires components together
├── adapters/         # External integration seams
│   ├── runner/       # Generic runner pipeline + claude-code, codex, opencode, and grok codecs
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

See [docs/internal/SEAMS.md](docs/internal/SEAMS.md) for detailed documentation of architectural boundaries.

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
Defines how an agent runs: portable `roleRef`, timeout, and allowed tools. A
role resolves once through resource-owned policies into the immutable candidate
snapshot stored with each new run.

### Task
Defines what needs to be done: title, description, scope path, context.

### Run
A concrete execution attempt linking a Task to an AgentProfile within a sandbox.

### RunEvent
Append-only event stream capturing all agent activity (logs, messages, tool calls).

### Execution mode
Every run executes in one of two modes, orthogonal to sandboxed/in-place run mode:

- **`codec_pipe`** (default) — agent-manager launches the agent CLI as its own
  child process in `--print`/`exec --json` mode and decodes the stdout stream
  through the runner codec. This is the mode used for all headless and protected
  runs.
- **`interactive`** — agent-manager creates a [web-console](../web-console)
  (tmux) session and launches the *real* interactive agent CLI inside it, then
  observes the run by tailing the agent-owned on-disk transcript with the same
  codec transcript parsers. A human can watch and type into the very same
  session while the run proceeds; the run-detail UI deep-links to it. Supported
  for **claude, codex, grok** (opencode is descoped — it has no tailable
  transcript file). Completion is a transcript **terminal marker** plus a
  turn-boundary idle-debounce (interactive CLIs stay alive between turns), not
  process exit. Follow-up turns (**Continue**) are typed into the live session;
  **Stop** interrupts and tears the session down. Interactive mode is rejected
  for **protected** runs — see [docs/PROTECTED_MODE_RUNNERS.md](docs/PROTECTED_MODE_RUNNERS.md).

  Opt in per run with `--execution-mode interactive` on `run create`. The full
  architecture (plug-in seam, per-agent transcript contract, launch/seed flow,
  recovery) is documented in
  [docs/interactive-runner-design.md](docs/interactive-runner-design.md); the
  restart-recovery story it shares with codec-pipe runs is in
  [docs/runner-transcript-recovery.md](docs/runner-transcript-recovery.md).

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/profiles` | Create agent profile |
| POST | `/api/v1/profiles/ensure` | Ensure profile by key (create if missing) |
| GET | `/api/v1/profiles` | List agent profiles |
| POST | `/api/v1/tasks` | Create task |
| GET | `/api/v1/tasks` | List tasks |
| POST | `/api/v1/runs` | Create run |
| GET | `/api/v1/runs/:id/events` | Stream run events |
| POST | `/api/v1/runs/:id/approve` | Approve run |
| POST | `/api/v1/runs/:id/reject` | Reject run |
| GET | `/api/v1/role-policy/status` | Inspect readiness, path, and active catalog digest |
| GET | `/api/v1/role-policy/catalog` | Inspect the active portable-role catalog projection |
| POST | `/api/v1/role-policy/validate` | Validate declared catalog state without activation |
| POST | `/api/v1/role-policy/reload` | Atomically activate a valid declared catalog |

## CLI Commands

```bash
# Profile management
agent-manager profile create --name "default" --role-ref code.default
agent-manager profile list

# Task management
agent-manager task create --title "Fix bug" --scope-path "src/"
agent-manager task list

# Run management
agent-manager run create --task <id> --profile <id>
agent-manager run create --task <id> --profile <id> --execution-mode interactive  # run inside a live web-console session
agent-manager run logs <id>
agent-manager run diff <id>
agent-manager run approve <id>
agent-manager run reject <id>

# Role-policy operations (declared catalog remains Git-managed)
agent-manager role-policy status
agent-manager role-policy catalog
agent-manager role-policy validate
agent-manager role-policy reload
agent-manager role-policy explain profile <id>
agent-manager role-policy explain run <id>

# Desired global permission intent (resources retain native-file ownership)
agent-manager permission-policy status
agent-manager permission-policy validate
agent-manager permission-policy plan
agent-manager permission-policy doctor
agent-manager permission-policy reconcile --i-was-explicitly-authorized

# Scenario-owned profile sources
agent-manager profile reconcile-scenario --scenario <scenario> --dry-run
```

See [docs/reference/configuration.md](docs/reference/configuration.md#role-policy-catalog)
for catalog update, validation, rollback, and failure-recovery guidance.

Note: Claude Code often uses one turn for tool use and another for tool results. If you need a final assistant message (for example, a "DONE" response), set `max_turns >= 3` on the profile.

## Dependencies

**Required**:
- SQLite - task, run, event, profile, and policy storage (embedded, no external service needed)
- workspace-sandbox - isolation and diff management

**Runners** (at least one required):
- claude-code resource
- codex resource
- opencode resource
- grok resource

## Documentation

- [PRD.md](./PRD.md) - Product requirements and operational targets
- [docs/interactive-runner-design.md](./docs/interactive-runner-design.md) - Interactive execution mode: seam, per-agent transcript contract, launch/recovery
- [docs/runner-transcript-recovery.md](./docs/runner-transcript-recovery.md) - Durable transcript + restart recovery (codec-pipe and interactive)
- [docs/RESEARCH.md](./docs/RESEARCH.md) - Research and architecture decisions
- [docs/internal/PROBLEMS.md](./docs/internal/PROBLEMS.md) - Known issues and deferred ideas
- [docs/internal/PROGRESS.md](./docs/internal/PROGRESS.md) - Development progress log
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
| `AM_SQLITE_PATH` | Direct path to SQLite database file (highest priority) |
| `DATABASE_URL` | SQLite path via `file:` protocol (e.g. `file:/path/to/db`) |
| `WORKSPACE_SANDBOX_URL` | workspace-sandbox API URL |
| `AGENT_MANAGER_ROLE_POLICY_CATALOG_PATH` | Optional path override for the declared role-policy catalog |
