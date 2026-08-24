# CLI Commands — React Component Library

The scenario CLI is a thin Go wrapper over the API. Every command
calls a single API endpoint and renders the result; there is no
business logic in the CLI. If a command needs to make a decision the
API doesn't expose, the correct fix is to add the API endpoint —
**not** to compute it locally.

The CLI binary is built from `cli/`, installed by `make setup` to
`~/.vrooli/bin/react-component-library`, and rebuilt automatically when its
sources change (cli-core's stale-detection rebuilds before any command
that touches the API).

For environment-variable precedence and CLI config-file shape, see
[`configuration.md`](configuration.md).

## Global flags (provided by cli-core)

Every command supports the following flags. **Do not reimplement them
in scenario commands.**

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start react-component-library` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color (also respects the `NO_COLOR` env var) |
| `--color` | Force-enable color (overrides terminal detection) |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show the CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `react-component-library status`

Health check. Calls `GET /health` and renders status + dependency
details. The output uses the **operational contract**:
`Status → Triage → Next Steps`.

```bash
react-component-library status
react-component-library status --json
```

### `react-component-library configure <key> <value>`

Persist a setting to the per-user CLI config file (location resolved
per [`configuration.md`](configuration.md#cli-config-file)).

```bash
react-component-library configure api_base http://localhost:15001/api/v1
react-component-library configure token "<token>"
```

Read values back without an argument:

```bash
react-component-library configure api_base
```

## Scenario commands — `components`

The `components` group is the authoring and registry surface for the
Git-tracked library source tree.

```bash
react-component-library components init header --library-id react-component-library:header --display-name Header --description "Scenario header" --tags layout,navigation --version 0.1.0

react-component-library components version-create "<component-id>" 0.2.0-beta.1 --draft true
react-component-library components manifest-update "<component-id>" --latest-version 1.0.0
react-component-library components index
react-component-library components list --match header --tags layout,navigation
react-component-library components get "<component-id>"
react-component-library components get-by-library-id react-component-library:Header
react-component-library components content-get "<component-id>"
react-component-library components content-set "<component-id>" ./Header.tsx --expected-sha256 "<sha>"
react-component-library components versions "<component-id>"
react-component-library components show-version "<component-id>" 0.1.0
react-component-library components ingest web-console ui/src/components/DrawerShell.tsx drawer-shell --slot ui-pattern
```

`init`, `version-create`, and `manifest-update` mutate
`library/components/<slug>/component.json` and
`library/components/<slug>/versions/<version>/*.tsx`, then re-index the
registry. SQLite remains the indexed registry and adoption ledger; it is
not the canonical component source.

`ingest` creates an indexed baseline and a draft from a real scenario TSX
source file. It returns origin provenance and de-scenario-ification findings;
fix those findings and run catalog conformance plus preview rendering before
promoting the draft.

## Scenario commands — `versions`

Version cleanup is deliberately plan-first. Released versions are eligible
only when they are not latest or draft, have no adoption or dependency
references, and satisfy the optional age filter. The ledger row and audit
history remain after retirement; only the version source folder is reclaimed.

```bash
react-component-library versions plan-cleanup --component-id "react-component-library:Button" --older-than-days 30
react-component-library versions cleanup --component-id "react-component-library:Button" --older-than-days 30 --plan-hash "sha256:example-plan-hash" --confirm true
react-component-library versions plan-cleanup --older-than-days 90
react-component-library versions cleanup-draft "react-component-library:Button" --older-than-days 14
react-component-library versions cleanup-draft "react-component-library:Button" --older-than-days 14 --confirm true
```

`plan-cleanup` is read-only and returns blocked items plus an exact plan hash.
`cleanup` is also a dry run unless `--confirm` and the matching plan hash are
provided. Draft cleanup is separate because drafts are mutable work-in-progress
and are removed from the manifest rather than retained as released history.

## Scenario commands — `adoptions`

```bash
react-component-library adoptions apply "<component-id>" web-console ui/src/components/adopted/Button.tsx
react-component-library adoptions apply "<component-id>" web-console ui/src/components/Button.tsx --replace-existing true --confirm-overwrite true
react-component-library adoptions suggest --scenario web-console --limit 10
```

Apply validates dependencies and design style at the server. Replacement mode
is explicit and reports direct import sites; confirmation is required when the
existing source body differs from the selected library version. Suggestions
are ranked from real UI inventory and include their matching surface,
style-fit, and dependency reasons.

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

For a new domain, copy the notes command group first, then replace it
once your real domain is green.

For a command inside an existing domain:

1. If the command needs a new API endpoint, add it first per
   [`api-endpoints.md`](api-endpoints.md#adding-a-new-endpoint).
2. Add the command to `cli/domains/<domain>/register.go`.
3. Implement its handler in `cli/domains/<domain>/handlers.go` or a
   focused sibling file.
4. The handler should:
   - Declare flags and positionals in `cliapp.ArgSchema`; cli-core
     uses the schema for parsing and help output.
   - Implement `RunCtx func(ctx cliapp.RunContext) error`, then read
     values with `ctx.Flag(...)`, `ctx.Positional(...)`, and
     `ctx.JSON()`.
   - Construct generated Connect clients with
     `cliapp.NewConnectHTTPClient(core)` for proto-typed operations.
   - Use `cliapp.UploadFile` only for documented multipart REST
     exceptions.
   - Mark the command with `NeedsAPI: true` so stale-checking,
     token validation, and `--auto-start` preflight all stay
     connected automatically
   - Render proto-backed responses with `cliapp.RenderProtoList` or
     `cliapp.RenderProtoMutation`.
5. Add endpoint metadata in the API handler module and add a matching
   row to `api/cmd/gen-endpoints/cli_commands_seed.json`. Then run
   `make endpoints`; do not edit [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json)
   by hand.
6. Add a row to this document.
7. Add a handler test in
   `cli/domains/<domain>/handlers_test.go` using `clitest.NewTestApp`
   + `clitest.NewAPIServer` + `clitest.CaptureStdout` (see
   [`../internal/TESTING.md`](../internal/TESTING.md)).

## Command structure principles

- **Subcommand groups** (`notes list`, `notes create`) over flat
  verbs (`list-notes`, `create-note`). Discoverability via `--help`
  is the goal.
- **Positional for required, flags for optional.** `notes get <id>`
  not `notes get --id <id>`.
- **One command per API endpoint.** If you find yourself making two
  endpoint calls, the API is missing a use-case.
- **Error messages must be actionable.** "API unreachable" is bad;
  "API unreachable at http://localhost:15001 — try `--auto-start` or
  `vrooli scenario start react-component-library`" is good.

## Catalog graph commands

The catalog group exposes the dependency projection without performing graph
math in the CLI:

- `react-component-library catalog graph <asset-id>` reports direct
  dependencies, closure bands, direct dependents, and blast radius. Add
  `--json` for the generated response.
- `react-component-library catalog graph-reconcile` compares catalog,
  manifest, and TypeScript import edges and reports one typed verdict per
  asset. It never repairs manifests.
- `react-component-library catalog ports <asset-id>` reports unmet port-facet
  capabilities, demanding closure assets, and candidate satisfiers.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for "API unreachable", auth, stale binary
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#inside-the-cli-thin-wrapper-domain-organized) — CLI architecture
