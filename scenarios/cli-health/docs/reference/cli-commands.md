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

Manifest arguments normally resolve by name, alias, or one-level envelope
descent. Use `bind.field` only when that resolution is not sufficient. CLI
Health also checks the meaning of explicit binds: it reports competing
arguments targeting one proto field, control-vocabulary flags (`json`,
`format`, `wait`, and similar) bound into request data, presence-like required
payload fields with no argument, and binds where renaming the argument would
be enough. The first three are errors; the last is a warning. A genuine
request-data control flag or explicit decoder bind may set a non-empty
`bind_waiver` explanation, which suppresses the matching control or redundant-
bind finding. The waiver is supported on both `positionals` and `flags` and is
validated by the shared manifest schema.

`binding.kind` is currently `connect-rpc` only. REST-exception
commands, if a future domain needs them, should be appended to the
loaded group outside the manifest path in the domain's `register.go`
and declared in the manifest's top-level `exceptions[]` array
(`{ command, class, reason }`) so CLI Health's `command_architecture`
capability recognizes them as legitimate special cases. Use `omitted[]`
for proto *methods* deliberately not exposed as commands.

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
cli-health configure token "<token>"
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
marked-reference rules in `path:../../docs/reference/machine-readable-references.md`.
Use `--refresh=on-miss` when validating against a catalog that may have changed
since the last discovery pass; CLI Health performs one owner-scoped refresh
before returning an invalid command-path miss.

Examples:

```bash
# Valid current command.
cli-health command validate --command "vrooli scenario test cli-health" --policy plan

# Invalid command path with suggestions.
cli-health command validate --command "cli-health search queri docs" --policy docs

# Retry discovery once for this owner before reporting a command-path miss.
cli-health command validate --command "cli-health search query docs" --policy docs --refresh=on-miss

# Valid command path with partial argument coverage when the catalog lacks reliable argument metadata.
cli-health command validate --command "knowledge-observatory docs health cli-health --checks=refs,commands" --policy docs

# Explicit non-current reference; pass qualifiers from the marked reference.
cli-health command validate --command "future-tool launch" --policy docs --qualifiers future

# Unsupported shell expression; CLI Health does not interpret pipelines or redirects.
cli-health command validate --command "vrooli scenario test cli-health | tee out.log" --policy docs
```

### DOCS-policy placeholder validation

Under `--policy docs` (the policy knowledge-observatory dochealth uses for doc
snippets), placeholder tokens are first-class instead of shell errors. The
quoted-placeholder convention is the preferred documentation form:

- `"<name>"` — a quoted named placeholder. It matches the manifest argument
  slot it fills (positional name, flag name/alias, or bound proto field);
  a name that matches no slot yields a `placeholder_name_mismatch` warning.
- `"<a|b|c>"` — a quoted enum alternation. It must exactly equal the slot's
  effective vocabulary — the manifest flag's `values` union the bound proto
  field's enum values — or validation fails with `enum_placeholder_mismatch`
  naming the expected list.
- Literal example values are checked against descriptor-derived constraints
  where they exist (type, buf.validate numeric bounds, string lengths, uuid
  format, enum membership, manifest `values`); a violation yields
  `invalid_literal_value`. Slots without constraint metadata pass silently.
- Unquoted `<...>` groups parse leniently (they reach the same semantic checks
  as their quoted form) but emit an `unquoted_placeholder` warning whose `fix`
  field carries the byte-exact snippet with every group wrapped in double
  quotes. `knowledge-observatory docs fix-placeholders <scenario>` applies
  those fixes deterministically.
- Real shell operators — pipes outside quotes, bare redirects, command
  substitution, chaining — remain hard `unsupported_shell_syntax` errors under
  every policy. Quoting is required precisely because a model pasting a doc
  snippet verbatim must never execute a stray `<`, `>`, or `|`.

| Verdict | DOCS-policy meaning |
|---|---|
| `valid` / `argument_shape_validated` | Path, flags, placeholders, and literals all check out (style warnings may still be attached) |
| `invalid` | `enum_placeholder_mismatch`, `invalid_literal_value`, or a structural argument error |
| `unsupported` | Genuine shell syntax (unchanged from other policies) |

Non-DOCS policies (`skill`, `plan`, `action`) are unchanged: placeholders are
not interpreted and unquoted `<`/`>` remain hard errors.

```bash
# Quoted placeholders validate to argument_shape_validated.
cli-health command validate --command 'plan-manager author skill-pack "<session>" --complexity "<minor|moderate|major|architectural>"' --policy docs

# A drifted enum alternation fails, naming the expected vocabulary.
cli-health command validate --command 'plan-manager author skill-pack "<session>" --complexity "<minor|moderate>"' --policy docs
```

### Manifest `values` / `value_aliases`

The enum vocabulary a placeholder is checked against is declared once, in the
owning scenario's `cli/manifest.json` flag entry:

```json
{
  "name": "complexity",
  "values": ["minor", "moderate", "major", "architectural"],
  "value_aliases": { "low": "minor", "medium": "moderate", "high": "major" }
}
```

The runtime parser rejects out-of-vocabulary values listing the options,
generated `--help` renders the choices, and DOCS-policy validation checks doc
alternations against the same declaration — one source of truth, three
surfaces. Aliases are accepted verbatim and passed through raw; the owning
server keeps canonicalizing.

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

**`--json` is an output contract, never an operation selector.** A command
must run the same operation regardless of `--json`; the renderer is chosen at
the end. CLI Health formalizes this as the `command_architecture` maturity
capability — see [`cli-architecture-maturity.md`](cli-architecture-maturity.md)
for the rung ladder (cli-core shell → declarative → manifest-bound →
renderer-separated primitives), the exception taxonomy for legitimate special
cases (streaming, upload, passthrough, external delegation, durable runs), and
the advisory-vs-gating rollout policy.

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
   method), the `governance` block (`effect`, `run_eligible`,
   `permissions`, optional `requires_confirmation`), and an
   `architecture` block declaring the command's renderer-separated
   primitive: `{ "primitive": "proto_list" | "proto_mutation" |
   "operational" | "action" }`. Match the primitive to the renderer
   (`proto_list` ↔ `RenderProtoList`, `proto_mutation` ↔
   `RenderProtoMutation`, `operational` ↔ `RenderProtoOperational`). A
   command without an `architecture` block earns the advisory
   `arch.primitive_undeclared` maturity debt — see
   [`cli-architecture-maturity.md`](cli-architecture-maturity.md). The schema in
   `.vrooli/schemas/cli-manifest.schema.json` is authoritative.
3. Implement the handler in `cli/domains/<domain>/handlers.go` (or a
   focused sibling file) with signature
   `func(ctx cliapp.RunContext) error`. Read values with
   `ctx.Flag(...)`, `ctx.BoolFlag(...)`, `ctx.Positional(...)`, and
   `ctx.JSON()`.
4. Add the handler to the bindings map in
   `cli/domains/<domain>/register.go` keyed by `"<Service>.<Method>"`
   so `cliapp.LoadFromManifestPrimitives` can wire it. Missing handler,
   dead handler, or a primitive that contradicts the manifest's
   `architecture.primitive` all fail at startup.
5. Handler implementation should:
   - Construct generated Connect clients with
     `cliapp.NewConnectHTTPClient(core)` for proto-typed operations.
   - Build every command with a cli-core primitive and register it through
     `cliapp.LoadFromManifestPrimitives`: bind
     `cliapp.ProtoList(call, report)` /
     `cliapp.ProtoMutation(call, report)` /
     `cliapp.ProtoOperational(call, report)` (or `cliapp.ProtoListOutcome`
     for a read whose exit code is payload-derived, e.g. `validate scenario`)
     in `register.go` (see the `search`, `reindex`, `command`, and `validate`
     domains for reference). The `call`/`report` halves take a narrow
     `cliapp.OperationContext` — no `JSON()`, no renderers — so the operation
     *cannot* branch on `--json`. Only this path carries the observed primitive
     evidence that reaches **verified** L4; an inline `cliapp.RenderProtoList`
     handler is renderer-separated but carries no evidence, so it stays at
     `arch.primitive_unverified` (declared, not verified) until built with a
     primitive and captured in the committed generated artifact at
     `.vrooli/generated/cli-primitive-evidence.json`.
   - Use `cliapp.UploadFile` only for documented multipart REST
     exceptions. Special-case commands (upload, passthrough, streaming,
     external delegation, durable runs) are appended outside the manifest
     binding path in `register.go` and declared in the manifest's
     top-level `exceptions[]` (`{ command, class, reason }`) so CLI Health
     classifies them as known special cases, not unknown legacy debt.
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

- [`cli-architecture-maturity.md`](cli-architecture-maturity.md) — the command_architecture maturity ladder, exception taxonomy, and renderer-separation contract
- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
