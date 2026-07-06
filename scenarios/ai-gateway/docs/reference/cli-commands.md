# CLI Commands — AI Gateway

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/ai-gateway`, and rebuilt automatically when its
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
| `--auto-start` | Run `vrooli scenario start ai-gateway` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `ai-gateway status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
ai-gateway status
ai-gateway status --json
```

### `ai-gateway configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
ai-gateway configure api_base http://localhost:15001/api/v1
ai-gateway configure token TOKEN_VALUE
```

Read values back without an argument:

```bash
ai-gateway configure api_base
```

## Scenario Commands

Each product domain exposes its commands as a subcommand group
(`ai-gateway <domain> <verb>`). Every command calls a single API
endpoint and renders the result through one of the three output
contracts below. Document your domain's commands here as you build
them, one row/section per command, mirroring the endpoints they call
in [`api-endpoints.md`](api-endpoints.md).

AI Gateway's generated example command group has been removed. The
current product-owned command surface is:

| Command | RPC | Purpose |
|---|---|---|
| `gateway validate --role <role>` | `GatewayService.ValidateGatewayRequest` | Validate provider-neutral request metadata before routing. |
| `inventory roles [--provider <provider>]` | `InventoryService.ListProviderRoles` | Read resource-owned provider role policy. |
| `inventory smoke --provider <provider>` | `InventoryService.SmokeProvider` | Run bounded provider smoke diagnostics through the resource seam. |
| `routing preview --role <role>` | `RoutingService.PreviewRoute` | Explain selected/rejected candidates without inference side effects. |
| `routing execute --role <role> --input <text>` | `RoutingService.ExecuteRoute` | Execute through the selected resource command and persist redacted evidence. |
| `routing evidence-list [--scenario <scenario>]` | `RoutingService.ListRouteEvidence` | List recent metadata-only route evidence. |
| `routing evidence-show <event-id>` | `RoutingService.GetRouteEvidence` | Inspect one route evidence event. |
| `conformance scan --scenario <scenario>` | `ConformanceService.ScanScenario` | Run AI Gateway's native scanner and migration recommendations. |
| `validation validate --scenario <scenario>` | `ScenarioValidationService.ValidateScenario` | Exercise the shared Test Genie provider contract. |
| `validation preview-fix --scenario <scenario>` | `ScenarioValidationService.PreviewFix` | Preview deterministic conformance fixes, currently guidance-only. |
| `validation apply-fix --scenario <scenario>` | `ScenarioValidationService.ApplyFix` | Call the explicit apply path, currently an API no-op until safe migrations exist. |

Common gateway request flags on `gateway validate`, `routing preview`,
and `routing execute`:

| Flag | Values / meaning |
|---|---|
| `--kind` | `text`, `embedding`, or `extract` (default `text`) |
| `--role` | Provider-neutral role, for example `chat.default` or `embedding.default` |
| `--profile` | `local-only`, `local-first`, `remote-only`, `quality-first`, `cheap-first`, or `privacy-sensitive` |
| `--privacy` | `public`, `internal`, `confidential`, or `secret` |
| `--operation`, `--scenario`, `--request-id` | Metadata labels stored in route evidence |
| `--timeout-ms`, `--max-cost-usd`, `--max-output-tokens` | Caller constraints passed to the API |

Every command supports `--json`; proto-backed commands emit the
proto JSON response shape in that mode.

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

For a new domain, follow the CLI steer pattern for generated Connect
clients and bind commands to the product RPCs in `cli/manifest.json`.

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
  `vrooli scenario start ai-gateway`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
