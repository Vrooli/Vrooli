# CLI Commands — Flow Verifier

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/flow-verifier`, and rebuilt automatically when its
sources change (cli-core's stale-detection rebuilds before any command
that touches the API).

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start flow-verifier` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `flow-verifier status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
flow-verifier status
flow-verifier status --json
```

### `flow-verifier configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
flow-verifier configure api_base http://localhost:15001/api/v1
flow-verifier configure token <token>
```

Read values back without an argument:

```bash
flow-verifier configure api_base
```

## Scenario commands — `notes` (CRUD reference)

The `notes` domain is the canonical worked example. Copy its layout
when adding the first non-trivial domain to your scenario.

### `flow-verifier notes list`

List notes, newest-first. Calls the generated Connect-RPC
`Notes/List` method. Uses the
**data-retrieval contract**: `Summary → Results → Retrieval Hints`.

```bash
flow-verifier notes list
flow-verifier notes list --json
```

### `flow-verifier notes create --title <title> [--body <body>]`

Create a note. Calls the generated Connect-RPC `Notes/Create` method. Uses the **mutation
contract**: `Result → What Changed → Next Command`.

```bash
flow-verifier notes create --title "First note" --body "Hello world"
```

`--title` is required. `--body` is optional. Validation lives in the
API service, so an empty title surfaces as an `invalid_argument`
Connect error rather than a CLI-side check.

### `flow-verifier notes get <id>`

Fetch a note by id. Calls the generated Connect-RPC `Notes/Get` method.

```bash
flow-verifier notes get abc123
```

A non-existent id surfaces as `not_found`; the CLI translates the
typed Connect code to an actionable error message.

### `flow-verifier notes attach <id> --file <path>`

Attach a file to a note. This is the documented REST multipart
exception because the request body contains opaque bytes. The response
is proto-typed attachment metadata.

```bash
flow-verifier notes attach abc123 --file ./example.png
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

For a new domain, copy the notes command group first, then replace it
once your real domain is green.

For a command inside an existing domain:

1. If the command needs a new API endpoint, add it first per
   [`api-endpoints.md`](api-endpoints.md#adding-a-new-endpoint).
2. Add the command to `cli/domains/<domain>/register.go`.
3. Implement its handler in `cli/domains/<domain>/handlers.go` or a
   focused sibling file.
4. The handler should:
   - Declare flags and positionals in `cliapp.ArgSchema`; cli-core
     uses the schema for parsing and help output.
   - Implement `RunCtx func(ctx cliapp.RunContext) error`, then read
     values with `ctx.Flag(...)`, `ctx.Positional(...)`, and
     `ctx.JSON()`.
   - Construct generated Connect clients with
     `cliapp.NewConnectHTTPClient(core)` for proto-typed operations.
   - Use `cliapp.UploadFile` only for documented multipart REST
     exceptions.
   - Mark the command with `NeedsAPI: true` so stale-checking,
     token validation, and `--auto-start` preflight all stay
     connected automatically
   - Render proto-backed responses with `cliapp.RenderProtoList` or
     `cliapp.RenderProtoMutation`.
5. Add endpoint metadata in the API handler module and add a matching
   row to `api/cmd/gen-endpoints/cli_commands_seed.json`. Then run
   `make endpoints`; do not edit [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json)
   by hand.
6. Add a row to this document.
7. Add a handler test in
   `cli/domains/<domain>/handlers_test.go` using `clitest.NewTestApp`
   + `clitest.NewAPIServer` + `clitest.CaptureStdout` (see
   [`../internal/TESTING.md`](../internal/TESTING.md)).

## Command structure principles

- **Subcommand groups** (`notes list`, `notes create`) over flat
  verbs (`list-notes`, `create-note`). Discoverability via `--help`
  is the goal.
- **Positional for required, flags for optional.** `notes get <id>`
  not `notes get --id <id>`.
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad;
  "API unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start flow-verifier`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
