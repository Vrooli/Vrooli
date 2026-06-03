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
- wires each command's `binding.method` (e.g. `TargetsService.ListTargets`)
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

`binding.kind` is currently `connect-rpc` only. REST-exception commands
would be appended to the loaded group outside the manifest path in the
domain's `register.go` and documented in the manifest's `omitted[]`
array.

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

## Scenario Commands — Backup Management

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
data-backup-manager targets get <target-id>
```

`--kind` is one of `filesystem`, `sqlite`, `postgres`, `redis`,
`qdrant`, `object-storage`.

### `data-backup-manager destinations ...`

Manage backup destinations (kopia repositories). Encryption is always
on; passphrases and access keys come from the `vault` resource, never
flags or config.

```bash
# --name must be slug-safe (lowercase, digits, hyphens) — it is also the kopia
# repository name. --location is the operator-facing bundle root for filesystem.
data-backup-manager destinations create --name nightly-local \
  --backend filesystem --location /var/backups/nightly --cap-bytes 53687091200
data-backup-manager destinations list          # shows bundle root, repository path, usage vs cap
data-backup-manager destinations get <destination-id>
data-backup-manager destinations usage <destination-id>
data-backup-manager destinations update --id <destination-id> --cap-policy alert-block

# Delete is precise about what it removes:
data-backup-manager destinations delete --id <destination-id>                       # catalog row only
data-backup-manager destinations delete --id <destination-id> --delete-repository true
#   ^ also removes LOCAL kopia metadata/config/cache + vault secret refs.
#     The encrypted repository bytes on the backend are NEVER deleted by DBM.

# Read-only drive readiness before creating a filesystem destination.
data-backup-manager destinations readiness --location /media/user/USB --json

# Non-mutating preparation planning. Execution stays separate and guarded.
data-backup-manager destinations prepare-plan --location /media/user/USB \
  --action create-subdir --subdir vrooli-backups --json
data-backup-manager destinations prepare-execute --plan-json '<plan-json>' \
  --confirm '<exact phrase>' --dry-run true
```

For a filesystem backend, `create` builds a self-describing **bundle** at the
bundle root (`location`): `README.txt`, `RECOVERY.txt`,
`vrooli-backup-destination.json`, plus the vanilla kopia repository nested under
`repositories/<slug>.kopia` (shown as `repository_location`). The create output
lists the bundle root, the nested repository path, and the metadata files
written.

A destination is rejected if its path would fall under the storage root
it is meant to protect (separate-root rule), if its name is not slug-safe, or if
the bundle root is already a kopia repository at its root. The cap defaults to
alert + block. For removable drives, run `destinations readiness` first
and prefer the recommended `vrooli-backups` subdirectory instead of a
raw mount root. `prepare-execute` is marked destructive governance and is
not prompt-manager runnable; real non-dry-run execution requires the
server-side plan identity checks and confirmation phrase to pass.

### `data-backup-manager discovery ...`

Onboarding suggestions, discovered read-only from the local environment.
Use these on a fresh install to see what's worth protecting and where to
back it up, instead of wiring the five nouns by hand.

```bash
data-backup-manager discovery targets        # well-known ~/.vrooli runtime state worth protecting
data-backup-manager discovery destinations   # mounted volumes / plugged-in drives to back up to
data-backup-manager discovery dismiss --id <id>   # hide a suggestion permanently
```

There is intentionally **no `accept` command**. Each suggestion prints the
exact values to enable it — accepting one is just the existing
`targets register …` (using the suggestion's owner/name/kind/locator) or
`destinations create …` (using its location, `--backend filesystem`). This
keeps a single source of truth and reuses their validation (separate-root
rule, encryption-on). Suggestions are derived every call and filtered
against the live catalog + dismissals, so once you register/create one it
stops appearing. A destination suggestion flagged `separate-root=UNSAFE`
overlaps protected data and `destinations create` would reject it.

### `data-backup-manager coverage ...`

First-real-backup readiness. Where `discovery` lists raw suggestions, `coverage`
is the decision surface: it composes discovery with the targets/plans/runs/
restores catalogs into one report and bulk-registers the recommended
non-sensitive defaults — so a plan protects *all* known non-regenerable durable
state, not just one self-registered target.

```bash
data-backup-manager coverage report                       # registered/recommended/sensitive/planned/backed-up/verified
data-backup-manager coverage accept-defaults --dry-run    # preview — registers nothing
data-backup-manager coverage accept-defaults              # register non-sensitive discovered durable targets
data-backup-manager coverage accept-defaults --include-sensitive  # also register credential/token targets (deliberate)
```

Sensitive credential/token suggestions are **never** registered by default;
they require the explicit `--include-sensitive` opt-in. Acceptance is idempotent
(discovery already excludes registered targets) and reads no file contents —
registration stores locators only. Per-item failures are reported, never
swallowed.

### `data-backup-manager plans ...`

Bind targets to destinations with a schedule and retention.

```bash
data-backup-manager plans create --name nightly \
  --targets <target-id> \
  --destinations <destination-id> \
  --schedule "24h" --keep-latest 7
data-backup-manager plans list
data-backup-manager plans get <plan-id>
data-backup-manager plans update --id <plan-id> --name nightly \
  --targets <target-id> --destinations <destination-id> --schedule "24h"
data-backup-manager plans delete --id <plan-id>
```

A target may be bound to several plans (e.g. daily-to-local and
weekly-to-offsite).

**Coverage guard.** `plans create` / `plans update` are blocked
(`failed_precondition`) while any non-sensitive recommended target is still
unregistered, so a plan cannot silently omit default coverage. Run
`coverage accept-defaults` first, or pass `--allow-incomplete-coverage` to
proceed deliberately with a narrow plan. Sensitive unregistered suggestions warn
but never block.

### `data-backup-manager runs ...`

Trigger and inspect plan executions.

```bash
data-backup-manager runs trigger --plan <plan-id>   # on-demand; scheduler triggers the same path
data-backup-manager runs list
data-backup-manager runs get <run-id>            # per-target outcomes + snapshot refs
data-backup-manager runs status                  # current catalog targets by default
data-backup-manager runs browse --destination <destination-id> --snapshot <snapshot-id>
```

### `data-backup-manager restores ...`

Restore a target, or verify it can be restored.

```bash
# Verify (test-restore to scratch + checksum) — the gate before removing data from git
data-backup-manager restores verify \
  --target <target-id> --destination <destination-id> --snapshot <snapshot-id>

# Restore to a chosen location
data-backup-manager restores restore \
  --target <target-id> --destination <destination-id> \
  --snapshot <snapshot-id> --location /restore/store-teams
data-backup-manager restores list
```

### Audit a snapshot (generic inventory proof)

```bash
# Restore to scratch + capture live + compare by generic inventory.
# --no-content-hash / --no-sqlite-checks trade proof strength for speed on huge trees.
data-backup-manager audits run \
  --target <target-id> --destination <destination-id> --snapshot <snapshot-id>

data-backup-manager audits get <audit-id>
data-backup-manager audits list --target <target-id>
```

The audit prints an operator verdict (PASSED / DIFF / DIFF-drift / FAILED),
the generic mismatches, and the live-vs-snapshot inventory evidence. It is
scenario-agnostic and read-only on the live target.

### Gated backup proof

Run the disposable canary proof when you need operational evidence that the
installed manager can back up, verify, restore, and byte-compare a filesystem
target through the public CLI:

```bash
DBM_E2E_BACKUP=1 ./scripts/prove-backup-restore.sh
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

For a new domain, copy the closest existing command group first, then
replace the service binding and handlers once the new API endpoint is
green.

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

- **Subcommand groups** (`targets list`, `plans create`) over flat
  verbs (`list-targets`, `create-plan`). Discoverability via `--help`
  is the goal.
- **Positional for required, flags for optional.** `targets get <id>`
  not `targets get --id <id>`.
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
