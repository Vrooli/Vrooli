# CLI Commands — Ecosystem Manager

The `ecosystem-manager` CLI is a thin Go wrapper over the REST API. Each
command calls one or more API endpoints and renders the result; there is
no business logic in the CLI. If a command needs a decision the API
doesn't expose, add the API endpoint — don't compute it locally.

The CLI is built from [CODE: cli], installed by `make setup` /
`bash ./cli/install.sh` to `~/.vrooli/bin/ecosystem-manager`, and
rebuilt when its sources change. It is scaffolded on `cli-core`
(`cliapp.NewStandardScenarioApp`, see [CODE: cli/app.go]) with
`APIPrefix: /api` and anonymous access enabled.

> **Pre-proto scenario.** Unlike current Vrooli scenarios, this CLI has
> no `cli/manifest.json` and no Connect-RPC bindings — commands are
> defined imperatively in Go and call REST endpoints directly. There is
> therefore no manifest-driven RPC-coverage test; keep the CLI and
> [`api-endpoints.md`](api-endpoints.md) in sync by hand. See the
> drift note in [`../internal/COHERENCE-NOTES.md`](../internal/COHERENCE-NOTES.md).

Command groups are assembled in [CODE: cli/domains/domains.go] from the
per-group files: [CODE: cli/tasks/commands.go],
[CODE: cli/steer/commands.go], [CODE: cli/queue/commands.go],
[CODE: cli/logs/commands.go].

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start ecosystem-manager` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects `NO_COLOR`) |
| `--color` | Force-enable color |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands

Auto-provided by `cli-core`.

| Command | Purpose |
|---|---|
| `ecosystem-manager status` | Health check — calls `GET /health`, renders status + dependencies |
| `ecosystem-manager configure <key> [value]` | Read/write a per-user CLI config value |
| `ecosystem-manager --version` | Print the CLI version |
| `ecosystem-manager --help` | Show usage |

```bash
ecosystem-manager status --json
ecosystem-manager configure api_base http://localhost:30500/api
```

## Scenario commands

Each group is a single top-level command that dispatches on its first
positional argument (subcommand verb). Run `<group>` with no subcommand
to print usage.

### `task` — manage generation/improvement tasks

[CODE: cli/tasks/commands.go]

| Invocation | Purpose | Endpoint(s) |
|---|---|---|
| `task add [resource\|scenario] <name>` | Create a generation task | `POST /api/tasks` |
| `task improve [resource\|scenario] <name>` | Create an improvement task | `POST /api/tasks` |
| `task list` (`ls`) | List tasks (`--status`, `--type`) | `GET /api/tasks` |
| `task show <id>` (`get`) | Show task detail | `GET /api/tasks/{id}` |
| `task status <id>` | Update task status | `PUT /api/tasks/{id}/status` |
| `task delete <id>` (`rm`) | Delete a task | `DELETE /api/tasks/{id}` |

```bash
ecosystem-manager task add scenario my-app --steer-profile balanced
ecosystem-manager task improve scenario my-app --steer-profile production-ready
ecosystem-manager task list --status pending --type scenario
ecosystem-manager task show <task-id> --json
```

### `steer` — auto-steer profiles & templates

[CODE: cli/steer/commands.go]

| Invocation | Purpose | Endpoint(s) |
|---|---|---|
| `steer profiles` (`ls`) | List auto-steer profiles | `GET /api/auto-steer/profiles` |
| `steer templates` | List built-in templates | `GET /api/auto-steer/templates` |
| `steer show <id>` (`get`) | Show profile detail | `GET /api/auto-steer/profiles/{id}` |

```bash
ecosystem-manager steer profiles
ecosystem-manager steer templates --json
ecosystem-manager steer show balanced
```

### `queue` — queue processor control

[CODE: cli/queue/commands.go]

| Invocation | Purpose | Endpoint(s) |
|---|---|---|
| `queue status` | Show queue status | `GET /api/queue/status` |
| `queue start` | Start the queue processor | `POST /api/queue/start` |
| `queue stop` | Stop the queue processor | `POST /api/queue/stop` |

```bash
ecosystem-manager queue status --json
ecosystem-manager queue start
```

### `logs` — API logs

[CODE: cli/logs/commands.go]

| Invocation | Purpose | Endpoint(s) |
|---|---|---|
| `logs` | View ecosystem-manager API logs | `GET /api/logs` |

```bash
ecosystem-manager logs
ecosystem-manager logs --json
```

## Output contracts

Commands render through the `cli-core` operational / data-retrieval /
mutation contracts (see [CODE: cli/internal/format/mutation.go]). Human
consumers see the structured report; `--json` consumers receive the raw
JSON response.

| Contract | Used by | Structure |
|---|---|---|
| **Operational** | `status` | Status → Triage → Next Steps |
| **Data Retrieval** | `task list/show`, `steer profiles/templates/show`, `queue status`, `logs` | Summary → Results → Hints |
| **Mutation** | `task add/improve/status/delete`, `queue start/stop` | Result → What Changed → Next Command |

## Adding a new command

1. If the command needs a new endpoint, add it first per
   [`api-endpoints.md`](api-endpoints.md#adding-a-new-endpoint).
2. Add the subcommand verb to the relevant group file
   ([CODE: cli/tasks/commands.go] etc.): extend the dispatch switch,
   implement the handler, and update the group's `Usage` help text.
3. For a brand-new group, create `cli/<group>/commands.go` and register
   it in [CODE: cli/domains/domains.go].
4. Read flags/positionals with the `cliapp.RunContext` helpers; call the
   API through the shared HTTP client; render via the
   `cli/internal/format` helpers.
5. Add a handler test (see [CODE: cli/tasks/commands_test.go]).
6. Update this document and keep
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) in sync.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — endpoints these commands call
- [`configuration.md`](configuration.md) — env-var precedence and CLI config file
- [`ui-manifest.md`](ui-manifest.md) — UI structure (pre-adoption status)
- [`../internal/COHERENCE-NOTES.md`](../internal/COHERENCE-NOTES.md) — REST-vs-proto drift note
- [`../guides/getting-started.md`](../guides/getting-started.md) — first-run walkthrough
