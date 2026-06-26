# CLI Commands — CLI Health

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/cli-health`, and rebuilt automatically when its
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
- wires each command's `binding.method` (e.g. `NotesService.ListNotes`)
  to a handler registered in the domain's `register.go` bindings map
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
commands, if a future domain needs them, should be appended to the
loaded group outside the manifest path in the domain's `register.go`
and documented in the manifest's `omitted[]` array.

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start cli-health` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `cli-health status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
cli-health status
cli-health status --json
```

### `cli-health configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
cli-health configure api_base http://localhost:15001/api/v1
cli-health configure token <token>
```

Read values back without an argument:

```bash
cli-health configure api_base
```

## Command reference validation

`cli-health command validate` validates Vrooli-owned command references without
executing the referenced command. It tokenizes the command text, rejects shell
operators and expansions, checks the command path against CLI Health's catalog,
and validates argument shape when manifest metadata is available.

```bash
cli-health command validate --command "knowledge-observatory docs health cli-health --checks=refs,commands" --policy docs
cli-health command validate --command "vrooli scenario test cli-health" --policy plan --json
```

Results distinguish valid, invalid, partial, skipped, unknown, and unsupported
references. A partial result means the command path exists, but reliable argument
metadata was unavailable. Qualifiers such as `cli[future]:...`, `cli[old]:...`,
`cli[external]:...`, and `cli[literal]:...` are interpreted through the shared
marked-reference rules in `path:docs/reference/machine-readable-references.md`.

Examples:

```bash
# Valid current command.
cli-health command validate --command "vrooli scenario test cli-health" --policy plan

# Invalid command path with suggestions.
cli-health command validate --command "cli-health search queri docs" --policy docs

# Valid command path with partial argument coverage when the catalog lacks reliable argument metadata.
cli-health command validate --command "knowledge-observatory docs health cli-health --checks=refs,commands" --policy docs

# Explicit non-current reference; pass qualifiers from the marked reference.
cli-health command validate --command "future-tool launch" --policy docs --qualifiers future

# Unsupported shell expression; CLI Health does not interpret pipelines or redirects.
cli-health command validate --command "vrooli scenario test cli-health | tee out.log" --policy docs
```

## Scenario commands

### `cli-health search query <text>`

Search the indexed command catalog.

```bash
cli-health search query "docs health"
cli-health search query "docs health" --mode text --limit 5
```

### `cli-health search status`

Show command-search backend availability.

```bash
cli-health search status
cli-health search status --json
```

### `cli-health reindex run`

Rebuild command-search corpus entries from manifests and help output.

```bash
cli-health reindex run --dry-run
cli-health reindex run --scenario prompt-manager
```

### `cli-health reindex status <job_id>`

Poll an in-flight reindex job.

```bash
cli-health reindex status job-123
```

### `cli-health reindex cancel <job_id>`

Request cooperative cancellation for an in-flight reindex job.

```bash
cli-health reindex cancel job-123
```

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

For a new domain, copy the smallest current manifest-backed command
group that has the same read/write shape, then replace it once your
real domain is green.

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
6. Add endpoint metadata in the API handler module and add a matching
   row to `api/cmd/gen-endpoints/cli_commands_seed.json`. Then run
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

- **Subcommand groups** (`search query`, `reindex run`) over flat
  verbs (`search-query`, `run-reindex`). Discoverability via `--help`
  is the goal.
- **Positional for required, flags for optional.** `reindex status <job_id>`
  not `reindex status --job-id <job_id>`.
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad;
  "API unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start cli-health`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
