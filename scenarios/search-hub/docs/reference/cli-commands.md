# CLI Commands — Search Hub

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/search-hub`, and rebuilt automatically when its
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
- wires each command's `binding.method` (e.g. `RegistryService.ListProviders`)
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

`binding.kind` is `connect-rpc` for API-backed commands. The local `space`
command is registered by `spacecli` because it reads the owner-controlled
coverage denominator directly from `docs/spaces/answer-space.md`.

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start search-hub` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `search-hub status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
search-hub status
search-hub status --json
```

### `search-hub configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
search-hub configure api_base http://localhost:15001/api/v1
search-hub configure token <token>
```

Read values back without an argument:

```bash
search-hub configure api_base
```

## Scenario commands — `providers` (registry)

The `providers` domain is search-hub's first real domain: the provider
registry surface. A *provider* is one `(corpus, type)` leaf (e.g.
`cli-health.commands`); a scenario that indexes several types registers
several leaves. Each command maps to one `RegistryService` RPC.

### `search-hub providers register --descriptor <json|@path>`

Register (upsert) a provider descriptor. Calls the generated Connect-RPC
`RegistryService/RegisterProvider` method. Uses the **mutation
contract**: `Result → What Changed → Next Command`.

```bash
# Read the descriptor from a file (for example, a provider-owned fixture):
search-hub providers register --descriptor @cli-health.commands.json

# Or pass inline JSON:
search-hub providers register --descriptor '{"provider_id":"cli-health.commands","provider_group":"cli-health","bucket":"BUCKET_DO","type":"command","description":"CLI commands","endpoint":{"http_json":{"scenario_id":"cli-health","path":"/vrooli.cli_health.v1.search.SearchService/Search","method":"HTTP_METHOD_POST"}},"result_mapping":{"results_path":"results","id_field":"name","title_field":"name","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"}}'
```

The descriptor is a `ProviderDescriptor` (protojson). `--descriptor` is
required. Validation lives in the API (`internal/registry.Validate`), so
a missing field surfaces as an `invalid_argument` Connect error rather
than a CLI-side check. Registration is an **upsert** keyed by
`provider_id`.

### `search-hub providers list [--bucket DO|REUSE|KNOW|STATE] [--type <type>] [--state active|capability_gap]`

List registered providers, optionally filtered. Calls the generated
Connect-RPC `RegistryService/ListProviders` method. Uses the
**data-retrieval contract**: `Summary → Results → Retrieval Hints`.
Includes `capability_gap` stubs (tracked corpora with no endpoint yet).

```bash
search-hub providers list
search-hub providers list --bucket DO --json
search-hub providers list --state capability_gap
```

### `search-hub providers remove <provider_id>`

Deregister a provider leaf by id. Calls the generated Connect-RPC
`RegistryService/DeregisterProvider` method. Idempotent — removing a
provider that was never registered reports "nothing to remove" rather
than erroring.

```bash
search-hub providers remove cli-health.commands
```

## Scenario commands — `maturity` (fleet certification)

### `search-hub maturity scan [--fast] [--json] [--root <dir>] [--timeout <duration>] [--fleet-timeout <duration>] [--concurrency <n>] [--retries <n>]`

Scans every search-applicable scenario (those owning `.vrooli/search.json` **or**
declaring a `search`/`ai-search` service capability — the same target set as the
Test Genie `search` phase) and reports certification status.

- **`--fast`** is **inventory only** (`validation_mode: "fast"`): descriptor/corpus
  shape without live eval or latency proof. Default is **full**
  (`validation_mode: "full"`) — the certification path.
- Each result carries `applicability_reason` (`descriptor` / `capability:<token>` /
  `descriptor+capability:<token>`), a `findings` list where each finding has a
  `finding_class` of `static` (descriptor/corpus shape) or `execution` (needs live
  eval/latency), an `eval_evidence` array (`suite_id`, `freshness`, `corpus_status`,
  `last_run_id`, `recall`/`recall_target`, `met_cases`/`below_cases`,
  `latency_p95_ms`), and `repair_commands` — the concrete next commands to run.

```bash
search-hub maturity scan --json                 # full certification scan
search-hub maturity scan --fast --json          # inventory only
search-hub maturity scan --concurrency 12 --fleet-timeout 90s --json
```

`--timeout` bounds one validation RPC; `--fleet-timeout` bounds the complete
scan. The latter never discards evidence already returned: each target is
reported in deterministic scenario order with `queue_ms`, `execution_ms`,
`attempts`, and a retry outcome. An unavailable target states whether the
cause was a transport error, its validation deadline, or fleet-deadline
exhaustion. `--retries` is the number of additional attempts, still constrained
by the fleet deadline.

### Eval tiers and governance

Registered suites can be run through the provider-direct tier or the federated
tier. Both tiers use the same provider-owned corpus; the federated tier submits
through the public router and records whether the suite's provider was actually
routed. A provider's `search.json` remains the corpus source of truth, so
mutating operations require an explicit confirmation path and a declared
provider control endpoint. Preview is the default; orphan cleanup is dry-run
unless `--confirm` is supplied.

The scheduler discovers suites from the registry and records its decisions and
backoff state durably. It does not contain provider identifiers or provider-
specific policy. A new registrant therefore requires descriptor and corpus data,
not a Search Hub source change.

`maturity scan --json` exits non-zero when blocking findings remain, even though
it returns the complete per-scenario report. Use `finding_class`, `eval_evidence`,
and `repair_commands` to distinguish static contract debt from live execution
evidence that needs an owner.

### `search-hub maturity fix <scenario> [--apply] [--rule <code,…>] [--json]`

Previews (default) or applies (`--apply`) conservative **mechanical** descriptor
repairs (version, default suite ids, `tests.description`, rerank-shortlist clamp).
It never invents endpoints, ownership, class, or eval cases — those need owner
judgment and are surfaced as findings instead.

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

## Search Maturity Ladder

Search maturity uses four capability ladders: descriptor, governance, eval
performance, and operability. `L0` means evidence is unavailable or a
required contract is absent; an empty evidence set never defaults to the top
rung. Higher rungs require progressively stronger declared and observed
evidence, and required findings keep a capability below the rung they block.

`maturity scan --json` reports each scenario's current and next level,
blocking/advisory finding codes, provider availability, and eval evidence. The
report invariant is strict: advisory findings are not blockers, and a scenario
with a blocking finding is never reported as passed. Use `maturity scan --fast`
only for inventory; full validation is the certification path.

For a new domain, copy the smallest existing Search Hub command group and
keep its handlers as thin typed adapters over the API contract.

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

- **Subcommand groups** (`providers list`, `evals run`) over flat
  verbs. Discoverability via `--help` is the goal.
- **Positional for required, flags for optional.** Use a positional
  identifier for resource lookup, not a redundant `--id` flag.
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad;
  "API unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start search-hub`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
