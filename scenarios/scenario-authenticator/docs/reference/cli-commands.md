# CLI Commands — Scenario Authenticator

> **Current reference with explicit deferred sections.** The `auth`,
> `sessions`, and TOTP `mfa` groups are shipped from `cli/manifest.json`,
> alongside the `status` and `configure` built-ins. Federation, API keys,
> passkeys, and true multi-realm administration remain planned and are not
> claims about the current binary. Keep command names, flags, and bindings
> aligned with [`cli/manifest.json`](../../cli/manifest.json).

The scenario CLI is a **thin Go translation layer over the Connect API**.
Every command calls a single API RPC and renders the result; there is no
business logic in the CLI. If a command needs a decision the API doesn't
expose, the correct fix is to add the API endpoint — **not** to compute
it locally. The CLI never holds keys, never hashes passwords, and never
talks to SQLite or the hot-state store directly; it only speaks Connect to the API
(which is itself reached same-origin / API-to-API, never cross-origin).

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/scenario-authenticator`, and rebuilt automatically when its
sources change (cli-core's stale-detection rebuilds before any command
that touches the API).

## Source of truth: `cli/manifest.json`

The CLI's command surface (groups, commands, positionals, flags, RPC
bindings, governance metadata) is declared in
[`cli/manifest.json`](../../cli/manifest.json) and validated against
[`.vrooli/schemas/cli-manifest.schema.json`](../../../../.vrooli/schemas/cli-manifest.schema.json)
(schema id `cli-manifest/v1`). The manifest is loaded at startup by
`cliapp.LoadFromManifest`, which:

- builds each domain's `SubcommandGroup` from its manifest group
- wires each command's `binding.method` (e.g.
  `AccountsService.Login`) to a handler registered in the domain's
  `register.go` bindings map
- fails loudly on missing handlers, dead handlers, or unknown groups

Per-domain tests use `cliapp.RequireProtoServiceCoverage` to assert that
every RPC on the bound proto service either has a manifest command
binding or appears in the manifest's `omitted[]` list with a reason —
adding a new RPC without exposing it as a CLI command (or explicitly
omitting it) fails the test. This is what guarantees **full API↔CLI
parity** as the auth surface grows.

The manifest's `governance` block (`effect`, `run_eligible`,
`permissions`, and optional `requires_confirmation`) is consumed by
prompt-manager to derive action certainty automatically. Current commands
declare their actual effect and permissions; future destructive commands must
add explicit confirmation before they are exposed.

`binding.kind` is currently `connect-rpc` only. REST-exception commands
(for example a command whose request shape is a non-RPC web standard) are
appended to the loaded group outside the manifest path in the domain's
`register.go` and documented in the manifest's `omitted[]` array.

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them in
scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start scenario-authenticator` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `scenario-authenticator status`

Health check. Calls `GET /health` and renders status + dependency
details (SQLite via the storage seam, selected hot-state store). The output uses the
**operational contract**: `Status → Triage → Next Steps`.

```bash
scenario-authenticator status
scenario-authenticator status --json
```

### `scenario-authenticator configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved per
[`configuration.md`](configuration.md#cli-config-file)).

```bash
scenario-authenticator configure api_base http://localhost:15001/api/v1
scenario-authenticator configure token <token>
```

Read values back without an argument:

```bash
scenario-authenticator configure api_base
```

## Scenario commands

The current command groups are shipped and deferred by tier as described
below.

Each shipped product domain exposes its commands as a subcommand group
(`scenario-authenticator <domain> <verb>`). Every command mirrors a
single Connect RPC in [`api-endpoints.md`](api-endpoints.md). The groups
below are the current manifest-backed surface; deferred capabilities are
listed separately.

### `auth` — identity, token, machine, and scope operations (shipped)

The current commands are generated from `cli/manifest.json`; the manifest is
the source of truth for flags and bindings.

| Command | Purpose |
|---|---|
| `auth register --email <e> [--realm <r>] [--username <u>] [--password-stdin]` | Create an account and issue tokens |
| `auth login --email <e> [--realm <r>] [--password-stdin]` | Sign in and issue tokens |
| `auth change-password --access-token <t> [--password-stdin]` | Change the current password and revoke sessions |
| `auth refresh --refresh-token <t>` | Rotate a refresh token |
| `auth logout --access-token <t>` | Revoke the current access token/session |
| `auth validate --access-token <t>` | Validate a token for callers that cannot verify JWKS locally |
| `auth link-machine-account --access-token <t> --machine-id <id> [--realm <r>] [--default]` | Bind the current account to a machine principal |
| `auth issue-break-glass --access-token <t> --token-file <path> [--scopes <s,...>]` | Issue a bounded emergency capability |
| `auth grant-scope --access-token <t> --scope <s> [--principal-id <id>]` | Grant a scope within the current authority |
| `auth revoke-scope --access-token <t> --scope <s> [--principal-id <id>]` | Revoke a scope within the current authority |
| `auth list-scopes --access-token <t> [--principal-id <id>]` | List assigned scopes |

With `--password-stdin`, passwords are read from standard input; otherwise
the CLI uses its masked prompt. Passwords are never command-line arguments.
Validation lives in the API service, so weak credentials surface as typed
Connect errors rather than CLI-side checks. Login and registration do not
leak account existence.

### `sessions` — session list and revoke (shipped)

| Command | Purpose |
|---|---|
| `sessions list --access-token <t>` | List sessions for the current account |
| `sessions revoke <session-id>` | Revoke one session |
| `sessions revoke-all --access-token <t>` | Revoke every session for the current account |

### `mfa` — TOTP second factors (shipped; passkeys deferred)

TOTP enrollment and removal are shipped. WebAuthn passkeys remain planned.
The commands mirror the current `MFAService` surface.

| Command | Tier | Mirrors |
|---|---|---|
| `mfa begin-enrollment --access-token <t>` | P1 | `MFAService/BeginEnrollment` |
| `mfa confirm-enrollment --access-token <t> --enrollment-id <id> --totp-code <c>` | P1 | `MFAService/ConfirmEnrollment` |
| `mfa remove-enrollment --access-token <t>` | P1 | `MFAService/RemoveEnrollment` |

`confirm-enrollment` prints recovery codes once. Login consumes the TOTP
challenge through the account API; there is no separate CLI passkey surface.

### Deferred CLI surfaces

The CLI does not currently expose realm administration, audit queries, API
key management, federation, password recovery, passkeys, or cross-principal
identity administration. Add each command only when its API contract is
implemented and the manifest, generated bindings, tests, and this reference
can be updated together.

## Output contracts

Every scenario command renders through one of three human contracts.
Proto-backed commands should use `cliapp.RenderProtoList` or
`cliapp.RenderProtoMutation`: human consumers see the report, while
`--json` consumers receive the proto JSON response shape.

| Contract | Used by | Structure |
|---|---|---|
| **Operational** | `status`, `auth validate` | Status → Triage → Next Steps |
| **Data Retrieval** | `sessions list`, `auth list-scopes` | Summary → Results → Retrieval Hints |
| **Mutation** | `auth register/login`, `auth grant-scope`, `sessions revoke` | Result → What Changed → Next Command |

For commands that aggregate multiple API calls or produce a non-proto
report, use the `RunContext` render helpers directly (`ctx.RenderList`,
`ctx.RenderMutation`, or the operational report helpers).

## Adding a new command

For a new domain, copy the worked CRUD command group in the fenced
example below first, then replace it once your real domain is green.

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
   `func(ctx cliapp.RunContext) error`. Read values with `ctx.Flag(...)`,
   `ctx.BoolFlag(...)`, `ctx.Positional(...)`, and `ctx.JSON()`.
4. Add the handler to the bindings map in
   `cli/domains/<domain>/register.go` keyed by `"<Service>.<Method>"` so
   `cliapp.LoadFromManifest` can wire it. Missing handler or dead handler
   both fail at startup.
5. Handler implementation should:
   - Construct generated Connect clients with
     `cliapp.NewConnectHTTPClient(core)` for proto-typed operations.
   - Render proto-backed responses with `cliapp.RenderProtoList` or
     `cliapp.RenderProtoMutation`.
6. Run `make endpoints`; do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
7. Add a row to this document.
8. Add a handler test in `cli/domains/<domain>/handlers_test.go` using
   `clitest.NewTestApp` + `clitest.NewAPIServer` + `clitest.CaptureStdout`
   (see [`../internal/TESTING.md`](../internal/TESTING.md)). Driving the
   handler via `cliapp.NewTestRunContextFromArgs` against the manifest's
   schema gives the closest parity with the dispatched path.

## Command structure principles

- **Subcommand groups** (`auth login`, `sessions revoke`) over flat verbs
  (`login`, `revoke-session`). Discoverability via `--help` is the goal.
- **Positional for required, flags for optional.** `sessions revoke <id>`
  not `sessions revoke --id <id>`.
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad; "API
  unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start scenario-authenticator`" is good.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`ui-manifest.md`](ui-manifest.md) — the UI surface over the same API
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — domain ownership map
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
