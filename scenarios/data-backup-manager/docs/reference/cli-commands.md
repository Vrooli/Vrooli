# CLI Commands — Data Backup Manager

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/data-backup-manager`, and rebuilt automatically when its
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
commands (the canonical example is `notes attach`, which uses
multipart upload) are appended to the loaded group outside the manifest
path in the domain's `register.go` and documented in the manifest's
`omitted[]` array.

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start data-backup-manager` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `data-backup-manager status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
data-backup-manager status
data-backup-manager status --json
```

### `data-backup-manager configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
data-backup-manager configure api_base http://localhost:15001/api/v1
data-backup-manager configure token <token>
```

Read values back without an argument:

```bash
data-backup-manager configure api_base
```

## Scenario commands — backup management (planned contract)

> **Status — planned.** The command groups in this section describe the
> **intended** CLI surface for the locked design and are pending
> implementation. They follow the rules above: one command per API
> method, declared in `cli/manifest.json`, thin handlers, no local
> business logic. Flag names are indicative; precise flags firm up with
> the proto messages. The `status` command and the `notes` worked
> example below are the only commands present in the template today.

The CLI serves two audiences: **scenarios** self-register the state
they own (typically at their own lifecycle), and **operators** manage
destinations, plans, runs, and restores. The vocabulary is Target,
Destination, Plan, Run, Restore.

### `data-backup-manager targets ...`

Self-registration and catalog inspection.

```bash
# Scenario self-registration (idempotent upsert, owner+name keyed)
data-backup-manager targets register --owner prompt-manager \
  --name store-teams --kind filesystem --locator store/teams
data-backup-manager targets deregister --owner prompt-manager --name store-teams

# Operator inspection
data-backup-manager targets list
data-backup-manager targets get prompt-manager/store-teams
```

`--kind` is one of `filesystem`, `sqlite`, `postgres`, `redis`,
`qdrant`, `object-storage`. Filesystem locators are storage-root-relative
for portability.

### `data-backup-manager destinations ...`

Manage backup destinations (kopia repositories). Encryption is always
on; passphrases and access keys come from the `vault` resource, never
flags or config.

```bash
data-backup-manager destinations create --name nightly-local \
  --backend filesystem --path /var/backups/nightly --cap 50GiB
data-backup-manager destinations create --name offsite \
  --backend s3 --bucket vrooli-backups --endpoint "$MINIO_ENDPOINT" --cap 200GiB
data-backup-manager destinations list          # shows usage vs cap
data-backup-manager destinations get offsite
data-backup-manager destinations check offsite # reachability dry-run (P1)
```

A destination is rejected if its path would fall under the storage root
it is meant to protect (separate-root rule). The cap defaults to
alert + block.

### `data-backup-manager plans ...`

Bind targets to destinations with a schedule and retention.

```bash
data-backup-manager plans create --name nightly \
  --target prompt-manager/store-teams \
  --destination nightly-local \
  --schedule "0 3 * * *" --keep-daily 7 --keep-weekly 4
data-backup-manager plans list
data-backup-manager plans get nightly
data-backup-manager plans delete nightly
```

A target may be bound to several plans (e.g. daily-to-local and
weekly-to-offsite).

### `data-backup-manager runs ...`

Trigger and inspect plan executions.

```bash
data-backup-manager runs start --plan nightly   # on-demand; scheduler triggers the same path
data-backup-manager runs list
data-backup-manager runs get <run-id>            # per-target outcomes + snapshot refs
```

### `data-backup-manager restores ...`

Restore a target, or verify it can be restored.

```bash
# Verify (test-restore to scratch + checksum) — the gate before removing data from git
data-backup-manager restores verify \
  --target prompt-manager/store-teams --destination nightly-local

# Restore to a chosen location
data-backup-manager restores start \
  --target prompt-manager/store-teams --destination nightly-local \
  --to /restore/store-teams
data-backup-manager restores list
```

## Scenario commands — `notes` (CRUD reference)

The `notes` domain is the canonical worked example. Copy its layout
when adding the first non-trivial domain to your scenario.

### `data-backup-manager notes list`

List notes, newest-first. Calls the generated Connect-RPC
`Notes/List` method. Uses the
**data-retrieval contract**: `Summary → Results → Retrieval Hints`.

```bash
data-backup-manager notes list
data-backup-manager notes list --json
```

### `data-backup-manager notes create --title <title> [--body <body>]`

Create a note. Calls the generated Connect-RPC `Notes/Create` method. Uses the **mutation
contract**: `Result → What Changed → Next Command`.

```bash
data-backup-manager notes create --title "First note" --body "Hello world"
```

`--title` is required. `--body` is optional. Validation lives in the
API service, so an empty title surfaces as an `invalid_argument`
Connect error rather than a CLI-side check.

### `data-backup-manager notes get <id>`

Fetch a note by id. Calls the generated Connect-RPC `Notes/Get` method.

```bash
data-backup-manager notes get abc123
```

A non-existent id surfaces as `not_found`; the CLI translates the
typed Connect code to an actionable error message.

### `data-backup-manager notes attach <id> --file <path>`

Attach a file to a note. This is the documented REST multipart
exception because the request body contains opaque bytes. The response
is proto-typed attachment metadata.

```bash
data-backup-manager notes attach abc123 --file ./example.png
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

- **Subcommand groups** (`notes list`, `notes create`) over flat
  verbs (`list-notes`, `create-note`). Discoverability via `--help`
  is the goal.
- **Positional for required, flags for optional.** `notes get <id>`
  not `notes get --id <id>`.
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad;
  "API unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start data-backup-manager`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
