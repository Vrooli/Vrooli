# CLI Commands — Image Tools

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/image-tools`, and rebuilt automatically when its
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
- wires each command's `binding.method` (e.g. `ModelsService.ListModels`)
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

`binding.kind` is currently `connect-rpc` only, modelling unary RPCs.
Commands that can't be expressed that way — the canonical example is
`jobs watch`, which consumes a server-streaming RPC — are appended to the
loaded group outside the manifest path in the domain's `register.go` and
documented in the manifest's `omitted[]` array.

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start image-tools` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `image-tools status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
image-tools status
image-tools status --json
```

### `image-tools configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
image-tools configure api_base http://localhost:15001/api/v1
image-tools configure token <token>
```

Read values back without an argument:

```bash
image-tools configure api_base
```

## Scenario commands — `jobs` (durable async lifecycle)

Inspect and control server-owned durable jobs. `jobs wait` is the
canonical block-once verb — it blocks server-side until the job is
terminal and returns it. Never poll `jobs get` in a loop.

### `image-tools jobs get <id>`

Get the current record for one job. Calls `JobsService/GetJob`. Uses the
**data-retrieval contract**.

```bash
image-tools jobs get <job-id>
```

### `image-tools jobs wait <id>`

Block once until the job is terminal, then print it. Calls
`JobsService/WaitJob`. A client disconnect does not affect the job.

```bash
image-tools jobs wait <job-id>
```

### `image-tools jobs list [--limit <n>]`

List recent jobs, newest first. Calls `JobsService/ListJobs`.

```bash
image-tools jobs list --limit 20
```

### `image-tools jobs cancel <id>`

Cancel a running or queued job. Calls `JobsService/CancelJob`. Uses the
**mutation contract**.

```bash
image-tools jobs cancel <job-id>
```

### `image-tools jobs watch <id>`

Stream a job's progress until it reaches a terminal state. Consumes the
server-streaming `JobsService/WatchJob` RPC, so it is appended outside
the manifest path in `register.go` and recorded in the manifest's
`omitted[]` array. `--json` emits one `ProgressEvent` per line.

```bash
image-tools jobs watch <job-id>
```

## Scenario commands — `models` (registry read + enable/disable)

Inspect the declarative model registry and toggle which models are
enabled. The seed catalog is read-only; enable/disable writes a SQLite
overlay.

### `image-tools models list [--operation <op>]`

List model registry entries, optionally filtered to one operation. Calls
`ModelsService/ListModels`.

```bash
image-tools models list
image-tools models list --operation upscale
```

### `image-tools models get <id>`

Get one model entry by id. Calls `ModelsService/GetModel`.

```bash
image-tools models get sd-1.5
```

### `image-tools models operations`

List the registry operation vocabulary. Calls
`ModelsService/ListOperations`.

```bash
image-tools models operations
```

### `image-tools models select <operation> [--override <id>]`

Preview which enabled model would run for an operation on this host
(honoring the per-op default and any override) without executing. Calls
`ModelsService/SelectModel` and surfaces the hardware-fit reason +
warnings.

```bash
image-tools models select upscale
image-tools models select upscale --override real-esrgan
```

### `image-tools models enable <id> [--disable]`

Enable (default) or disable (`--disable`) a model. Calls
`ModelsService/SetModelEnabled`, persisting the overlay. Uses the
**mutation contract**.

```bash
image-tools models enable real-esrgan
image-tools models enable real-esrgan --disable
```

### `image-tools models blocklist`

List license-encumbered models excluded from the catalog. Calls
`ModelsService/ListBlocklist`.

```bash
image-tools models blocklist
```

### `image-tools models doctor`

Diagnose catalog installability and policy integrity. Calls
`ModelsService/DoctorCatalog` and exits non-zero when enabled seed models lack
direct installable assets, an operation has no installable enabled model, or
commercial/checksum policy is incoherent.

```bash
image-tools models doctor
image-tools models doctor --json
```

## Scenario commands — `backends` (AI backend software readiness)

### `image-tools backends doctor`

Diagnose host software availability for registered inference backends and flag
enabled catalog backend families that do not yet have a runtime provider. Calls
`ModelsService/DoctorBackends` and exits non-zero when a local backend is
missing or a catalog-declared backend cannot be probed/executed yet. Hardware
fit remains reported by `models select`.

```bash
image-tools backends doctor
image-tools backends doctor --json
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

For a new domain, mirror the `jobs` / `models` command groups: a
`Register(core, manifest)` returning a `SubcommandGroup` built via
`cliapp.LoadFromManifest`, plus one handler per RPC in `handlers.go`.

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

- **Subcommand groups** (`jobs list`, `models get`) over flat
  verbs (`list-jobs`, `get-model`). Discoverability via `--help`
  is the goal.
- **Positional for required, flags for optional.** `models get <id>`
  not `models get --id <id>`.
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad;
  "API unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start image-tools`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
