# CLI Commands — API Health

The scenario CLI is a thin Go wrapper over the API. Every command calls one API
endpoint or shared provider operation and renders the result; there is no
validation logic in the CLI.

## Source of truth: `cli/manifest.json`

The CLI command surface is declared in
[`cli/manifest.json`](../../cli/manifest.json) and validated against the shared
CLI manifest schema. Commands that mirror Connect RPCs must bind to generated
service methods or be explicitly omitted with a reason.

## Global flags (provided by cli-core)

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Run `vrooli scenario start api-health` if the API is unreachable |
| `--json` | Emit machine-readable JSON instead of the human report |
| `--no-color` | Disable ANSI color |
| `--color` | Force-enable color |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show CLI version |

## Built-in commands (auto-provided by `cli-core`)

### `api-health status`

Health check for API Health itself.

```bash
api-health status
api-health status --json
```

### `api-health configure <key> <value>`

Persist a setting to the per-user CLI config file.

```bash
api-health configure api_base http://localhost:15001/api/v1
api-health configure token <token>
```

## Scenario commands

| Command | Domain | Purpose | Status |
|---|---|---|---|
| `api-health validate scenario <target> [--path <path>] [--include-execution]` | validation | Validate target API readiness and render capability findings. | implemented |
| `api-health validate fix-preview <target> [rule_id ...] [--path <path>]` | validation | Preview deterministic fixes without writing. | implemented |
| `api-health validate fix-apply <target> [rule_id ...] [--path <path>]` | validation | Apply explicit deterministic fixes. | implemented |
| `api-health probe health <target>` | probe | Run one bounded live health probe and render evidence. | planned |
| `api-health migration ledger` | migration | Show scenario-auditor API rule migration accounting. | planned |

## Output contracts

| Contract | Used by | Structure |
|---|---|---|
| Operational | `status`, `validate`, `probe` | Status -> Capability Summary -> Findings/Evidence -> Next Steps |
| Mutation | `fix apply` | Result -> What Changed -> Remaining Findings |
| Retrieval | `migration ledger` | Summary -> Rule Decisions -> Gaps |

## Adding a new command

1. Add the API endpoint/RPC first.
2. Add a command entry to `cli/manifest.json`.
3. Implement the handler in `cli/domains/<domain>/`.
4. Register the handler binding.
5. Add tests using cli-core test utilities.
6. Update this document.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for unreachable API
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — CLI architecture boundary
