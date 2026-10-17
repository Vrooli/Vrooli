# CLI Commands — Unit Health

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/unit-health`, and rebuilt automatically when its
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
- wires each command's `binding.method` (`ValidationService.ValidateScenario`)
  to a handler registered in the domain's `register.go` bindings map
- fails loudly on missing handlers, dead handlers, or unknown groups

Per-domain tests use `cliapp.RequireProtoServiceCoverage` to assert
that every RPC on the bound proto service either has a manifest command
binding or appears in the manifest's `omitted[]` list with a reason —
adding a new RPC without exposing it as a CLI command (or explicitly
omitting it) fails the test.

The manifest's `governance` block (`effect`, `run_eligible`,
`permissions`, `requires_confirmation`) is consumed by prompt-manager
and Program Runtime to derive action certainty and the governed
binding `unit-health/validate/scenario`; the scenario does not keep a
hand-classified action-safety list.

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start unit-health` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `unit-health status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
unit-health status
unit-health status --json
```

### `unit-health configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
unit-health configure api_base http://localhost:15001/api/v1
unit-health configure token "<token>"
```

Read values back without an argument:

```bash
unit-health configure api_base
```

## Scenario commands — `validate`

The `validate` group is the scenario's whole command surface. It binds
to `ValidationService.ValidateScenario`.

### `unit-health validate scenario <scenario>`

Assess a scenario's test surface: discovery, per-workspace test plan,
canonical-framework and coverage-config projection checks, test
architecture, advisory test-quality rule results, requirement
traceability, and the six-capability local maturity assessment. The
human report is the default and is the agent workflow; `--json` is for
programmatic consumers such as Test Genie's `unit` phase and the
`unit-health.setpoint-read` program.

```bash
unit-health validate scenario unit-health
unit-health validate scenario unit-health --execution
unit-health validate scenario unit-health --workspace api --execution
unit-health validate scenario unit-health --execution --fast-test-only
unit-health validate scenario unit-health --json
```

| Argument | Meaning |
|---|---|
| `<scenario>` | Scenario slug. Optional when `--path` is set. |
| `--path <dir>` | Validate a filesystem path instead of a registered scenario. |
| `--workspace <id>` | Restrict validation to one workspace id (repeatable). |
| `--execution` | Execute the planned test commands under bounded limits. Without it the response is a plan: the maturity level still reflects static required findings, but coverage, execution, and reliability findings are absent and `evidence_stages.executed` reads `not_requested`. |
| `--fast-test-only` | With `--execution`, run only the fast test command and omit coverage artifacts. |
| `--json` | Emit the `ValidateScenarioResponse` proto JSON. |

The CLI flag `--execution` maps to the request field `include_execution`.
API messages name the CLI form so a reader of the human report can act on
them directly.

Exit status is nonzero when `counts.errors > 0`. The report uses the
**operational contract**: summary, workspaces and plan, projection
checks, findings grouped by capability, coverage, test-quality results,
and next steps.

### `unit-health mutation pilot <scenario>`

Run a bounded owner-side mutation pilot for a Go package. Unit Health applies
one `go/ast` mutation per disposable copy below the scenario cache directory and
runs the owning test through the bounded executor. The shared checkout is never
the mutation target.

```bash
unit-health mutation pilot unit-health --package ./internal/testquality/... --seed pilot-1
unit-health mutation pilot unit-health --package ./internal/testquality/... --operators boundary --max-mutants 20 --seed pilot-1 --json
```

The supported operators are `boundary`, `negate-condition`, and
`return-constant`. Receipts use the closed vocabulary `killed`, `survived`,
`invalid`, `equivalent`, `out_of_contract`, `infrastructure_failure`, and
`unknown`; only valid killed/survived mutants contribute to `kill_rate`.

The comparable Unit Health pilot floor is 0.95, derived from two 20-mutant
runs in the improve skill. A surviving valid mutant is a review signal, not a
defect verdict.

Reading the result without over-claiming is the `unit-health` usage
skill's job (`prompt-manager skill read unit-health`); the maturity
vocabulary is in [`maturity.md`](maturity.md) and the rule vocabulary in
[`test-quality-rules.md`](test-quality-rules.md).

### `unit-health calibrate run`

Compare the committed calibration corpus with the owner adapters through
`ValidationService.RunCalibration`. The command is the governed CLI surface and
the `unit-health/calibrate/run` read binding used by `unit-health.setpoint-read`.

```bash
unit-health calibrate run --partition development
unit-health calibrate run --partition reviewed-holdout --holdout assertion-observation-go-v1 --rule assertion-observation
unit-health calibrate corpus
```

The development read reports exact matches against implemented cases. The
reviewed holdout reports false positives, false negatives, unknowns, and the
catalog decision state; it never promotes a rule automatically. `calibrate corpus`
reports the bounded implemented, retired, and specified inventory.

The developer-only `test-quality-reference` generator remains test-fixture tooling,
not a CLI command; it is not a sensor or an authorization path.

## Output contracts

Every scenario command should render through one of three human
contracts. Proto-backed commands should use `cliapp.RenderProtoList`
or `cliapp.RenderProtoMutation`: human consumers see the report, while
`--json` consumers receive the proto JSON response shape.

| Contract | Used by | Structure |
|---|---|---|
| **Operational** | `status`, `validate` | Status → Triage → Next Steps |
| **Data Retrieval** | `list`, `get`, `view`, `search` | Summary → Results → Retrieval Hints |
| **Mutation** | `create`, `update`, `delete`, `start`, `stop` | Result → What Changed → Next Command |

For commands that aggregate multiple API calls or produce a
non-proto report, use the `RunContext` render helpers directly
(`ctx.RenderList`, `ctx.RenderMutation`, or the operational report
helpers).

## Adding a new command

Every command is one RPC. A tool that today runs through `go run`
(the calibration comparison above) becomes a command only by first
becoming an RPC.

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
5. Render proto-backed responses with `cliapp.RenderProtoList` or
   `cliapp.RenderProtoMutation`; construct generated Connect clients
   with `cliapp.NewConnectHTTPClient(core)`.
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

- **Subcommand groups** (`validate scenario`) over flat verbs.
  Discoverability via `--help` is the goal.
- **Positional for required, flags for optional.** `validate scenario <name>`
  not `validate scenario --name <name>`.
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad;
  "API unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start unit-health`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`maturity.md`](maturity.md) — how to read the assessment
- [`test-quality-rules.md`](test-quality-rules.md) — the advisory rule catalog
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
