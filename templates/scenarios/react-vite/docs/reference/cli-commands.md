# CLI Commands — {{SCENARIO_DISPLAY_NAME}}

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/{{SCENARIO_ID}}`, and rebuilt automatically when its
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
| `--auto-start` | Run `vrooli scenario start {{SCENARIO_ID}}` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `{{SCENARIO_ID}} status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
{{SCENARIO_ID}} status
{{SCENARIO_ID}} status --json
```

### `{{SCENARIO_ID}} configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
{{SCENARIO_ID}} configure api_base http://localhost:15001/api/v1
{{SCENARIO_ID}} configure token <token>
```

Read values back without an argument:

```bash
{{SCENARIO_ID}} configure api_base
```

## Scenario commands — `notes` (CRUD reference)

The `notes` domain is the canonical worked example. Copy its layout
when adding the first non-trivial domain to your scenario.

### `{{SCENARIO_ID}} notes list`

List notes, newest-first. Calls the generated Connect-RPC
`Notes/List` method. Uses the
**data-retrieval contract**: `Summary → Results → Retrieval Hints`.

```bash
{{SCENARIO_ID}} notes list
{{SCENARIO_ID}} notes list --json
```

### `{{SCENARIO_ID}} notes create --title <title> [--body <body>]`

Create a note. Calls the generated Connect-RPC `Notes/Create` method. Uses the **mutation
contract**: `Result → What Changed → Next Command`.

```bash
{{SCENARIO_ID}} notes create --title "First note" --body "Hello world"
```

`--title` is required. `--body` is optional. Validation lives in the
API service, so an empty title surfaces as an `invalid_argument`
Connect error rather than a CLI-side check.

### `{{SCENARIO_ID}} notes get <id>`

Fetch a note by id. Calls the generated Connect-RPC `Notes/Get` method.

```bash
{{SCENARIO_ID}} notes get abc123
```

A non-existent id surfaces as `not_found`; the CLI translates the
typed Connect code to an actionable error message.

### `{{SCENARIO_ID}} notes attach <id> --file <path>`

Attach a file to a note. This is the documented REST multipart
exception because the request body contains opaque bytes. The response
is proto-typed attachment metadata.

```bash
{{SCENARIO_ID}} notes attach abc123 --file ./example.png
```

## Output contracts

Every scenario command should render through one of three contracts.
Use `cliapp.RenderOperationalReport`, `RenderListReport`, or
`RenderMutationReport` so the human shape stays consistent across all
Vrooli scenarios.

| Contract | Used by | Structure |
|---|---|---|
| **Operational** | `status`, `health`, `audit`, `validate`, `doctor` | Status → Triage → Next Steps |
| **Data Retrieval** | `list`, `get`, `view`, `search` | Summary → Results → Retrieval Hints |
| **Mutation** | `create`, `update`, `delete`, `start`, `stop` | Result → What Changed → Next Command |

When a command supports `--json`, render the **same** structured
report through `cliapp.PrintReportJSON` instead of inventing a second
output shape.

## Adding a new command

1. If the command needs a new API endpoint, add it first per
   [`api-endpoints.md`](api-endpoints.md#adding-a-new-endpoint).
2. Choose a domain. New commands live in
   `cli/domains/<domain>/{register,handlers}.go`. Mirror the `notes`
   layout exactly.
3. The handler should:
   - Use `cliutil.ParseInterspersed(fs, args)` (not `fs.Parse`) so
     `notes get abc123 --json` parses correctly
   - Call `core.Get(...)` / `core.Request(...)` for `/api/v1/...`
     paths, or `core.GetRoot(...)` / `core.RequestRoot(...)` for
     root paths like `/health`
   - Mark the command with `NeedsAPI: true` so stale-checking,
     token validation, and `--auto-start` preflight all stay
     connected automatically
   - Render via `cliapp.Render*Report` (default human) and
     `cliapp.PrintReportJSON` when `--json` is set
4. Register the domain in `cli/domains/domains.go::SubcommandGroups`.
5. Update [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) so
   the endpoint's `cli_mapping` points at the new command. The CI
   gate fails if a registered command has no mapping or vice versa.
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
  `vrooli scenario start {{SCENARIO_ID}}`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
