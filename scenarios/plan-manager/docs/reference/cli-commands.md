# CLI Commands — Plan Manager

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/plan-manager`, and rebuilt automatically when its
sources change (cli-core's stale-detection rebuilds before any command
that touches the API).

## Source of truth: `cli/manifest.json`

The CLI's command surface (groups, commands, positionals, flags,
RPC bindings, governance metadata) is declared in
[`cli/manifest.json`](../../cli/manifest.json) and validated against
[`.vrooli/schemas/cli-manifest.schema.json`](../../../../.vrooli/schemas/cli-manifest.schema.json)
(schema id `cli-manifest/v1`). The manifest is loaded at startup by
`cliapp.LoadFromManifest`, which:

- builds each domain's `SubcommandGroup` from its manifest group
- wires each command's `binding.method` (e.g.
  `<Domain>Service.List<Entity>`) to a handler registered in the
  domain's `register.go` bindings map
- fails loudly on missing handlers, dead handlers, or unknown groups

Per-domain tests use `cliapp.RequireProtoServiceCoverage` to assert
that every RPC on the bound proto service either has a manifest command
binding or appears in the manifest's `omitted[]` list with a reason —
adding a new RPC without exposing it as a CLI command (or explicitly
omitting it) fails the test.

The manifest's `governance` block (`effect`, `run_eligible`,
`permissions`, `requires_confirmation`) is consumed by prompt-manager
to derive action certainty automatically; scenarios that adopt the
manifest don't need hand-classified action-safety lists.

`binding.kind` is currently `connect-rpc` only. REST-exception
commands (for example, a multipart file-upload command whose request
body carries opaque bytes) are appended to the loaded group outside the
manifest path in the domain's `register.go` and documented in the
manifest's `omitted[]` array.

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start plan-manager` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `plan-manager status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
plan-manager status
plan-manager status --json
```

### `plan-manager configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
plan-manager configure api_base http://localhost:15001/api/v1
plan-manager configure token <token>
```

Read values back without an argument:

```bash
plan-manager configure api_base
```

## Scenario commands — `<domain>`

Each product domain exposes its commands as a subcommand group
(`plan-manager <domain> <verb>`). Every command calls a single API
endpoint and renders the result through one of the three output
contracts below. Document your domain's commands here as you build
them, one row/section per command, mirroring the endpoints they call
in [`api-endpoints.md`](api-endpoints.md).

The scaffold ships one fully worked CRUD command group as a copyable
reference (see the fenced example below); `vrooli scenario detemplate
<scenario>` removes it once your real domains are green.

## Primary Agent Loops

Small agents should prefer the continue-loop commands and use the lower-level
commands only for the specific action returned by the API-owned `GuidedStep`.

| Command | RPC | Purpose |
|---|---|---|
| `plan-manager author continue <session>` | `AuthoringService.ContinueAuthoring` | Returns one recommended next authoring action across section, phase, validation-review, and finalize states. |
| `plan-manager exec continue <plan-or-execution>` | `ExecutionService.ContinueExecution` | Resumes or starts execution and returns one recommended next runner action without advancing the phase pointer. |

`plan-manager exec transition <execution> <phase> --status done` is guarded by
the execution service: it requires the last stored phase validation to be
`pass` + `fresh`, or an explicit `--validation-override-reason` for degraded or
offline completion. Prefer `exec continue` so the API recommends validation
before the done transition.

## `log` — the execution-log ledger

The `log` group is the single durable home for the typed work products an agent
produces while executing a plan: decisions, candidate findings, filed bug
reports, reusable records, and notes (DISTINCT concepts — a finding is
unvalidated, a bug report is filed to the issue tracker, a record is reusable
learning). The CLI is **flat** (`plan-manager log decision-add`, not nested
`log decision add`). Bugs and records are forwarded downstream **internally** by
Plan Manager (bug → scenario-qa, record → swarm-manager); agents never invoke an
external scenario CLI from the plan workflow.

> **Moved from `exec`.** The former `exec decision-add`, `exec finding-add`,
> `exec findings`, and `exec triage` commands (and the `RecordDecision`/
> `RecordFinding`/`ListCandidateFindings`/`TriageFinding` RPCs) were **removed**.
> Use `log decision-add`, `log finding-add`, `log list --type finding`, and
> `log update --triage` / `log promote` respectively.

| Command | RPC | Purpose |
|---|---|---|
| `log decision-add <plan-or-execution> --title <t> [--phase --detail --evidence --source-command --idempotency-key --run-id]` | `LogService.AddDecision` | Record an in-flow design decision (feeds the handoff). |
| `log finding-add <plan-or-execution> --title <t> [--severity --phase --detail --evidence --source-command --idempotency-key --run-id]` | `LogService.AddFinding` | Record a CANDIDATE finding (a possible bug; never auto-promoted). |
| `log bug-add <plan-or-execution> --title <t> [--severity --phase --detail --evidence --source-command --idempotency-key --run-id]` | `LogService.AddBug` | File a bug report; forwarded to the issue tracker (scenario-qa) through an internal seam. v1 default is a pending stub (production adapter deferred), so the entry persists `pending` and is retried via `log sync`. |
| `log record-add <plan-or-execution> --title <t> [--phase --detail --evidence --source-command --idempotency-key --run-id]` | `LogService.AddRecord` | Capture a reusable record; forwarded to Swarm Manager through an internal seam. v1 default is a pending stub (production adapter deferred), so the entry persists `pending` and is retried via `log sync`. |
| `log note-add <plan-or-execution> --title <t> [--phase --detail --evidence --source-command --idempotency-key --run-id]` | `LogService.AddNote` | Record a lightweight progress/context note (local-only). |
| `log list [<plan-or-execution>] [--phase --type --triage --sync-status]` | `LogService.ListEntries` | List ledger entries with a compact summary. `--type` = `decision\|finding\|bug_report\|record\|note`; `--triage` = `candidate\|promoted\|dismissed`; `--sync-status` = `local\|pending\|synced\|sync_failed`. |
| `log get <id>` | `LogService.GetEntry` | Get one ledger entry by id, including its downstream reference. |
| `log update <id> [--title --detail --severity --triage --add-evidence]` | `LogService.UpdateEntry` | Update mutable fields; empty/unspecified leaves a field unchanged; `--add-evidence` appends. |
| `log promote <id> --to <bug\|record> [--title --detail --severity]` | `LogService.PromoteEntry` | Promote a finding into a bug report or record, preserving the original finding (marked promoted) and linking the new entry back to it. |
| `log sync <id>` | `LogService.SyncEntry` | Retry downstream forwarding for a `pending`/`sync_failed` bug or record. |

`--idempotency-key` makes retries safe (a retry with the same key returns the
existing entry); findings/decisions also dedup by
(execution, attribution run id, type, normalized title). A failed/pending
downstream sync is never fatal — the entry stays durable and is retried with
`log sync`.

## Output contracts

Every scenario command should render through one of three human
contracts. Proto-backed commands should use `cliapp.RenderProtoList`
or `cliapp.RenderProtoMutation`: human consumers see the report, while
`--json` consumers receive the proto JSON response shape.

| Contract | Used by | Structure |
|---|---|---|
| **Operational** | `status`, `health`, `audit`, `validate`, `doctor` | Status → Triage → Next Steps |
| **Data Retrieval** | `list`, `get`, `view`, `search` | Summary → Results → Retrieval Hints |
| **Mutation** | `create`, `update`, `delete`, `start`, `stop` | Result → What Changed → Next Command |

For commands that aggregate multiple API calls or produce a
non-proto report, use the `RunContext` render helpers directly
(`ctx.RenderList`, `ctx.RenderMutation`, or the operational report
helpers).

## Adding a new command

For a new domain, copy the worked CRUD command group in the fenced
example above first, then replace it once your real domain is green.

For a command inside an existing domain:

1. If the command needs a new API endpoint (RPC), add it first per
   [`api-endpoints.md`](api-endpoints.md#adding-a-new-endpoint). The
   manifest's coverage test will fail otherwise on the next CLI build.
2. Add a command entry to the matching group in
   [`cli/manifest.json`](../../cli/manifest.json): `name`, optional
   `description`, `positionals` / `flags`, the `binding` (service +
   method), and the `governance` block (`effect`, `run_eligible`,
   `permissions`, optional `requires_confirmation`). The schema in
   `.vrooli/schemas/cli-manifest.schema.json` is authoritative.
3. Implement the handler in `cli/domains/<domain>/handlers.go` (or a
   focused sibling file) with signature
   `func(ctx cliapp.RunContext) error`. Read values with
   `ctx.Flag(...)`, `ctx.BoolFlag(...)`, `ctx.Positional(...)`, and
   `ctx.JSON()`.
4. Add the handler to the bindings map in
   `cli/domains/<domain>/register.go` keyed by `"<Service>.<Method>"`
   so `cliapp.LoadFromManifest` can wire it. Missing handler or
   dead handler both fail at startup.
5. Handler implementation should:
   - Construct generated Connect clients with
     `cliapp.NewConnectHTTPClient(core)` for proto-typed operations.
   - Use `cliapp.UploadFile` only for documented multipart REST
     exceptions (append those outside the manifest path in
     `register.go` and document them in the manifest's `omitted[]`).
   - Render proto-backed responses with `cliapp.RenderProtoList` or
     `cliapp.RenderProtoMutation`.
6. Add endpoint metadata in the API handler module and bind the method
   (or list it in `omitted[]`) in `cli/manifest.json`. Then run
   `make endpoints`; do not edit [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json)
   by hand.
7. Add a row to this document.
8. Add a handler test in
   `cli/domains/<domain>/handlers_test.go` using `clitest.NewTestApp`
   + `clitest.NewAPIServer` + `clitest.CaptureStdout` (see
   [`../internal/TESTING.md`](../internal/TESTING.md)). Driving the
   handler via `cliapp.NewTestRunContextFromArgs` against the manifest's
   schema gives the closest parity with the dispatched path.

## Command structure principles

- **Subcommand groups** (`<domain> list`, `<domain> create`) over flat
  verbs (`list-<entity>`, `create-<entity>`). Discoverability via
  `--help` is the goal.
- **Positional for required, flags for optional.** `<domain> get <id>`
  not `<domain> get --id <id>`.
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad;
  "API unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start plan-manager`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
