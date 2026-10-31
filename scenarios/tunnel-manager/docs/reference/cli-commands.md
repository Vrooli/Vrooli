# CLI Commands — Tunnel Manager

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/tunnel-manager`, and rebuilt automatically when its
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
| `--auto-start` | Run `vrooli scenario start tunnel-manager` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `tunnel-manager status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
tunnel-manager status
tunnel-manager status --json
```

### `tunnel-manager configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
tunnel-manager configure api_base http://localhost:15001/api/v1
tunnel-manager configure token <token>
```

Read values back without an argument:

```bash
tunnel-manager configure api_base
```

## Scenario commands — `<domain>`

Each product domain exposes its commands as a subcommand group
(`tunnel-manager <domain> <verb>`). Every command calls a single API
endpoint and renders the result through one of the three output
contracts below, mirroring the endpoints in
[`api-endpoints.md`](api-endpoints.md).

## Tunnel Manager commands

The command groups below are implemented in `cli/manifest.json` and
validated against the proto descriptors. Every command emits proto-typed
`--json` through the global flag above.

### `tunnel-manager tunnel status`

Tunnel health overview. Operational contract (Status → Triage → Next Steps).

### `tunnel-manager routes <list|get|create|update|delete>`

List the exposure manifest (SSOT) — subdomain, scenario, domain, local
port, tier, lease, enabled. Data-retrieval contract.

| Flag | Purpose |
|---|---|
| `--tier <core\|leased>` | Filter by tier. |

### `tunnel-manager exposure expose <scenario>`

Request **leased** exposure of a scenario. Ensures a route (using the scenario
name as the subdomain), ensures the scenario is running on a *fixed* UI port
(delegated to `internal/lifecycle` via a stop+start cycle if needed), pins a
fixed port via structure-health if the scenario only declared a range, and
requests ingress + DNS. Mutation contract.

If the scenario used a port *range*, `expose` will assign a concrete fixed port,
update its `service.json`, and cycle the process so the tunnel target is stable.
The success report will say `port_assigned` and suggest verification.

| Flag | Purpose |
|---|---|
| `--ttl-seconds <seconds>` | Lease lifetime (default ≈ 1 week). |
| `--requested-by <id>` | Record the requester (operator or scenario). |

### `tunnel-manager exposure <leases|extend|revoke|list|check|reconcile>`

Manage leased exposure.

| Subcommand | Purpose | Key args/flags |
|---|---|---|
| `exposure leases` | List active/expired/revoked leases. | `--status <active\|expired\|revoked>` |
| `exposure extend <lease_id>` | Extend a lease's TTL. | `--ttl-seconds <seconds>` |
| `exposure revoke <lease_id>` | Revoke a lease early and tear down ingress. | — |
| `exposure list` | List reconciled exposure state. | — |
| `exposure check <scenario>` | Check whether a scenario is exposed. | — |
| `exposure reconcile` | Re-derive CORE routes and reap expired leases. | — |

### `tunnel-manager probes <run|history|classify>`

Run internal (local port) and external (public URL) liveness probes for
exposed routes, list probe history, or classify latest probe pairs.
Operational contract.

| Flag | Purpose |
|---|---|
| `history --subdomain <name>` | Filter history to a single route subdomain. |
| `history --limit <n>` | Limit returned history rows. |

### `tunnel-manager audit run`

Port-compliance findings: exposed scenarios must declare a fixed UI port
in `service.json` matching the manifest. Operational contract.

### `tunnel-manager recovery <state|events|run>`

Inspect recovery state (backoff / circuit breaker) and the recovery event
log; optionally trigger a recovery attempt. Operational contract.

| Flag | Purpose |
|---|---|
| `events --limit <n>` | Limit returned recovery events. |
| `run --force true` | Bypass the circuit breaker for a manual recovery attempt. |

### `tunnel-manager config <get|credentials-status|credentials-set|credentials-clear|sync|mode>`

Manage Cloudflare ingress and mode.

| Subcommand | Purpose | Key args/flags |
|---|---|---|
| `config get` | Show browser-safe configuration readiness. | — |
| `config credentials-status` | Show Cloudflare credential presence/source metadata without printing secrets; values come from the credential authority. | — |
| `config credentials-set` | Store write-only Cloudflare credentials in the canonical Vrooli credential authority; read API tokens from stdin so they never appear in argv. | `--account-id`, `--tunnel-id`, `--api-token-stdin` |
| `config credentials-clear` | Clear Cloudflare credentials from the credential authority. | `--field account_id\|tunnel_id\|api_token\|all` |
| `config sync` | Reconcile ingress (remote API or local `config.yml`) with the manifest. | — |
| `config mode` | Switch and migrate ingress mode. | `--target remote\|local` |

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
  `vrooli scenario start tunnel-manager`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
