# CLI Commands — Scenario Authenticator

> **Current reference with explicit deferred sections.** The `auth` and
> `sessions` groups are shipped from `cli/manifest.json`, alongside the
> `status` and `configure` built-ins. Sections for MFA, federation, API keys,
> and true multi-realm administration remain planned and are not claims about
> the current binary. Keep command names, flags, and bindings aligned with
> [`cli/manifest.json`](../../cli/manifest.json).

The scenario CLI is a **thin Go translation layer over the Connect API**.
Every command calls a single API RPC and renders the result; there is no
business logic in the CLI. If a command needs a decision the API doesn't
expose, the correct fix is to add the API endpoint — **not** to compute
it locally. The CLI never holds keys, never hashes passwords, and never
talks to SQLite or Redis directly; it only speaks Connect to the API
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
  `IdentityService.Login`) to a handler registered in the domain's
  `register.go` bindings map
- fails loudly on missing handlers, dead handlers, or unknown groups

Per-domain tests use `cliapp.RequireProtoServiceCoverage` to assert that
every RPC on the bound proto service either has a manifest command
binding or appears in the manifest's `omitted[]` list with a reason —
adding a new RPC without exposing it as a CLI command (or explicitly
omitting it) fails the test. This is what guarantees **full API↔CLI
parity** as the auth surface grows.

The manifest's `governance` block (`effect`, `run_eligible`,
`permissions`, `requires_confirmation`) is consumed by prompt-manager to
derive action certainty automatically. For an IdP this matters: mutating
commands like `realm delete`, `session revoke`, and `apikey revoke`
carry `requires_confirmation` and constrained `permissions` so agents
cannot run destructive identity operations unattended.

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
details (SQLite via the storage seam, Redis). The output uses the
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
below are the target parity surface; tiers track the PRD operational
targets.

### `auth` — identity + token lifecycle (shipped)

The end-user authentication lifecycle. Mirrors `IdentityService`,
`TokensService`, and `SessionsService`.

| Command | Tier | Mirrors | Contract |
|---|---|---|---|
| `auth register --email <e> --password <p> [--realm <r>] [--username <u>]` | P0 | `IdentityService/Register` | Mutation |
| `auth login --email <e> --password <p> [--realm <r>]` | P0 | `IdentityService/Login` | Mutation |
| `auth whoami` | P0 | `IdentityService/GetCurrentUser` | Data retrieval |
| `auth refresh --refresh-token <t>` | P0 | `TokensService/Refresh` | Mutation |
| `auth logout [--all]` | P0 | `SessionsService/Logout` / `RevokeAllSessions` | Mutation |
| `auth reset-request --email <e> [--realm <r>]` | P1 | `IdentityService/RequestPasswordReset` | Mutation |
| `auth reset-complete --token <t> --password <p>` | P1 | `IdentityService/CompletePasswordReset` | Mutation |
| `auth verify-email --token <t>` | P1 | `IdentityService/VerifyEmail` | Mutation |

```bash
scenario-authenticator auth register --email dev@example.com --password 's3cret!'
scenario-authenticator auth login --email dev@example.com --password 's3cret!'
scenario-authenticator auth whoami
scenario-authenticator auth logout --all
```

Validation lives in the API service, so a weak password surfaces as an
`invalid_argument` Connect error rather than a CLI-side check. Login and
registration relay faithful reasons **without** leaking account
existence.

### `token` — token + JWKS inspection (P0)

Token diagnostics. Mirrors `TokensService` plus the JWKS REST edge.

| Command | Tier | Mirrors | Contract |
|---|---|---|---|
| `token validate --token <t> [--aud <realm>]` | P0 | `TokensService/Validate` | Operational |
| `token jwks` | P0 | `GET /.well-known/jwks.json` | Data retrieval |

```bash
scenario-authenticator token jwks
scenario-authenticator token validate --token "$ACCESS_TOKEN"
```

`token validate` is a diagnostic — RPs verify tokens locally against
JWKS and never call this on the hot path.

### `sessions` — session list + revoke (shipped)

Server-tracked sessions. Mirrors `SessionsService`.

| Command | Tier | Mirrors | Contract |
|---|---|---|---|
| `session list [--user-id <id>] [--scope all] [--limit <n>]` | P0 | `SessionsService/ListSessions` | Data retrieval |
| `session revoke <session-id>` | P0 | `SessionsService/RevokeSession` (and the carried-over REST `DELETE /api/v1/sessions/{id}`) | Mutation |

```bash
scenario-authenticator session list
scenario-authenticator session revoke 9f3c…
```

`--user-id` / `--scope all` require admin; a non-admin request surfaces
as a `permission_denied` Connect error.

### `realm` — tenant management (P0 read → P1 CRUD)

The tenant boundary. Mirrors `RealmsService`.

| Command | Tier | Mirrors | Contract |
|---|---|---|---|
| `realm get <id>` | P0 | `RealmsService/GetRealm` | Data retrieval |
| `realm list` | P1 | `RealmsService/ListRealms` | Data retrieval |
| `realm create --slug <s> --name <n> [...]` | P1 | `RealmsService/CreateRealm` | Mutation |
| `realm update <id> [...]` | P1 | `RealmsService/UpdateRealm` | Mutation |
| `realm delete <id>` | P1 | `RealmsService/DeleteRealm` | Mutation (confirmation-gated) |

```bash
scenario-authenticator realm get default
scenario-authenticator realm create --slug acme --name "Acme Corp"
```

`realm delete` carries `requires_confirmation`; the default realm cannot
be deleted (`failed_precondition`).

### `role` / `scope` — authorization definitions (P0 → P1)

Realm role and scope definitions + assignment. Mirrors
`AuthorizationService`. Enforcement of fine-grained "can-they" stays with
the Relying Party.

| Command | Tier | Mirrors |
|---|---|---|
| `role list` | P0 | `AuthorizationService/ListRoles` |
| `role assign --user <id> --role <r>` | P0 | `AuthorizationService/AssignRole` |
| `role revoke --user <id> --role <r>` | P0 | `AuthorizationService/RevokeRole` |
| `role create --name <n>` | P1 | `AuthorizationService/CreateRole` |
| `role delete <name>` | P1 | `AuthorizationService/DeleteRole` |
| `scope list` | P1 | `AuthorizationService/ListScopes` |
| `scope create --name <n>` | P1 | `AuthorizationService/CreateScope` |
| `scope assign [...]` | P1 | `AuthorizationService/AssignScope` |

### `audit` — security event query (P0)

Query the append-only audit log. Mirrors `AuditService`. Uses the
**operational contract**.

| Command | Tier | Mirrors |
|---|---|---|
| `audit query [--user-id <id>] [--action <a>] [--since <ts>] [--limit <n>]` | P0 | `AuditService/QueryEvents` |

```bash
scenario-authenticator audit query --action user.login.failed --limit 50
```

### `mfa` — second factors (P1)

TOTP and WebAuthn passkeys. Mirrors `MfaService`.

| Command | Tier | Mirrors |
|---|---|---|
| `mfa enroll-totp` | P1 | `MfaService/EnrollTotp` |
| `mfa activate-totp --code <c>` | P1 | `MfaService/ActivateTotp` |
| `mfa verify --challenge <id> --code <c>` | P1 | `MfaService/VerifyChallenge` |
| `mfa disable-totp` | P1 | `MfaService/DisableTotp` |
| `mfa passkey register` | P1 | `MfaService/RegisterPasskey` |
| `mfa passkey list` | P1 | `MfaService/ListPasskeys` |
| `mfa passkey remove <id>` | P1 | `MfaService/RemovePasskey` |

`activate-totp` prints recovery codes once; passkey assertion is
primarily a browser flow (the CLI assists with registration options).

### `apikey` — machine principals (P1)

Hashed API keys + client-credentials grant. Mirrors `ApiKeysService`.

| Command | Tier | Mirrors |
|---|---|---|
| `apikey create --name <n> [--scope <s>...] [--expires-in <days>]` | P1 | `ApiKeysService/CreateApiKey` |
| `apikey list` | P1 | `ApiKeysService/ListApiKeys` |
| `apikey revoke <id>` | P1 | `ApiKeysService/RevokeApiKey` (confirmation-gated) |
| `apikey token --api-key <k> [--realm <r>]` | P1 | `ApiKeysService/IssueClientToken` |

```bash
scenario-authenticator apikey create --name ci-bot --scope read:own --expires-in 90
```

The plaintext key is shown **once** on `create`; `list` never returns it.

### `oauth` — social federation (P1)

Inbound OAuth2/OIDC social sign-in. Mirrors `FederationService`. The
provider callback itself is a REST web standard (no CLI command).

| Command | Tier | Mirrors |
|---|---|---|
| `oauth providers [--realm <r>]` | P1 | `FederationService/ListProviders` |
| `oauth start --provider <p> [--realm <r>]` | P1 | `FederationService/StartOAuth` |

```bash
scenario-authenticator oauth providers
scenario-authenticator oauth start --provider google
```

`oauth start` prints the upstream authorization URL to open in a browser;
the redirect lands on the REST callback, not the CLI.

## Output contracts

Every scenario command renders through one of three human contracts.
Proto-backed commands should use `cliapp.RenderProtoList` or
`cliapp.RenderProtoMutation`: human consumers see the report, while
`--json` consumers receive the proto JSON response shape.

| Contract | Used by | Structure |
|---|---|---|
| **Operational** | `status`, `audit query`, `token validate` | Status → Triage → Next Steps |
| **Data Retrieval** | `session list`, `realm get`, `apikey list`, `auth whoami` | Summary → Results → Retrieval Hints |
| **Mutation** | `auth register/login`, `realm create`, `session revoke`, `apikey revoke` | Result → What Changed → Next Command |

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

- **Subcommand groups** (`auth login`, `session revoke`) over flat verbs
  (`login`, `revoke-session`). Discoverability via `--help` is the goal.
- **Positional for required, flags for optional.** `session revoke <id>`
  not `session revoke --id <id>`.
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
