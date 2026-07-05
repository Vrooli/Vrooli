# CLI Commands — Experience Manager

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/experience-manager`, and rebuilt automatically when its
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
| `--auto-start` | Run `vrooli scenario start experience-manager` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `experience-manager status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
experience-manager status
experience-manager status --json
```

### `experience-manager configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
experience-manager configure api_base http://localhost:15001/api/v1
experience-manager configure token <token>
```

Read values back without an argument:

```bash
experience-manager configure api_base
```

## Scenario commands — `<domain>`

Each product domain exposes its commands as a subcommand group
(`experience-manager <domain> <verb>`). Every command calls a single API
endpoint and renders the result through one of the three output
contracts below. Document your domain's commands here as you build
them, one row/section per command, mirroring the endpoints they call
in [`api-endpoints.md`](api-endpoints.md).

The scaffold ships one fully worked CRUD command group as a copyable
reference (see the fenced example below); `vrooli scenario detemplate
<scenario>` removes it once your real domains are green.

### `experience-manager spec validate <scenario>`

Parse and validate a scenario's `experience/` contract against
`scenario-experience-spec/v1`, including index parity, cross-document
references, PRD operational-target references, tier semantics, and binding
integrity.

```bash
experience-manager spec validate experience-manager
experience-manager spec validate web-console --json
experience-manager spec validate custom --path /path/to/scenario
```

### `experience-manager spec fleet`

Compute experience-spec coverage and debt across all scenarios, sorted
worst-first. The sweep is live and does not read a persisted cache.

```bash
experience-manager spec fleet
experience-manager spec fleet --json
```

### `experience-manager spec attest <scenario> <page> <claim>`

Append manual evidence for a manual-tier claim. The ledger is append-only:
refreshing evidence creates a new row with a fresh expiry instead of mutating
the previous attestation.

```bash
experience-manager spec attest experience-manager fleet debt-table-perceivable \
  --author operator --rationale "Reviewed against current design" \
  --expires-at 2027-01-01T00:00:00Z
```

### `experience-manager spec list <scenario>`

List pages and journeys declared in a scenario's `experience/index.json`.

```bash
experience-manager spec list experience-manager
experience-manager spec list custom --path /path/to/scenario --json
```

### `experience-manager spec show <scenario> <page>`

Render one page spec document.

```bash
experience-manager spec show experience-manager studio
```

### `experience-manager spec render <scenario> <page>`

Render one page spec into deterministic workshop HTML and write the artifact
under `coverage/wireframes/<scenario>/`.

```bash
experience-manager spec render experience-manager studio
experience-manager spec render business-health matrix --mode image
```

`--mode image` is allowed, but degrades to wireframe output when `image-tools`
is not available.

### `experience-manager spec compare-variants <scenario> <page>`

Render page-form variants side by side as deterministic workshop HTML. The
`--file` payload may be either a JSON array of `SpecVariant` objects or an
object with a `variants` array.

```bash
experience-manager spec compare-variants experience-manager studio --file /tmp/studio-variants.json
experience-manager spec compare-variants business-health matrix --mode image --file /tmp/matrix-variants.json
```

### `experience-manager spec promote-variant <scenario> <page>`

Promote one reviewed variant into the target `experience/` page. The selected
variant is written only after the parser-clean preview succeeds, and the command
returns the resulting diffs plus validation status.

```bash
experience-manager spec promote-variant experience-manager studio --file /tmp/selected-studio-variant.json
```

### `experience-manager spec suggest-bindings <scenario> <page>`

Suggest page bindings from existing spec bindings and the latest stored
reconciliation evidence.

```bash
experience-manager spec suggest-bindings web-console home --limit 20
```

### `experience-manager spec scaffold <scenario>`

Derive BAS case stubs from active page specs. Use `--dry-run` to preview the
same deterministic changes without writing files.

```bash
experience-manager spec scaffold experience-manager --dry-run
experience-manager spec scaffold custom --path /path/to/scenario
```

### `experience-manager fix preview|apply <scenario>`

Preview or apply deterministic remediations through the shared
`ScenarioValidationService` fix contract. Use `--rules` to target one rule.
The BAS scaffold rule is `experience.case_scaffold`.

```bash
experience-manager fix preview experience-manager --rules experience.case_scaffold
experience-manager fix apply custom --path /path/to/scenario --rules experience.binding_orphan
```

### `experience-manager author start <scenario>`

Start a persisted authoring session.

```bash
experience-manager author start experience-manager
```

### `experience-manager author submit <session>`

Submit a typed page form into a session. Use `--file` for scripted PageForm
JSON, or flags for compact manual entry.

```bash
experience-manager author submit expauth-abc123 --file /tmp/page-form.json
```

### `experience-manager author preview|apply|discard <session>`

Preview computes diffs plus parser validation without writing the target tree.
Apply writes only after the preview has zero parser error findings. Discard
removes the persisted session.

```bash
experience-manager author preview expauth-abc123
experience-manager author apply expauth-abc123
experience-manager author discard expauth-abc123
```

### `experience-manager provider validate <scenario>`

Render the same parser-backed validation through the shared
`ScenarioValidationService` contract consumed by Test Genie.

```bash
experience-manager provider validate experience-manager
experience-manager provider validate experience-manager --json
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
  `vrooli scenario start experience-manager`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
