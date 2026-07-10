# Codex Resource

OpenAI Codex CLI for local code generation and agentic engineering workflows.

## Intent

- Resource ID: `codex`
- Category: `developer-tooling`
- Driver: `external-cli`
- Portability tier: `partial`

## Use Cases

- Use Codex as an interactive or scripted coding agent in local workflows.
- Standardize Codex CLI availability for scenarios and operator tooling.
- Provide a consistent external CLI dependency for code generation and task execution.

## Architecture

This resource is being aligned to the updated `external-cli` structure.

- `resource.json` is the declarative authority for install, binary probing, version checks, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Codex-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Codex-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/discovery`: host binary detection and probing helpers
- `cli/internal/install`: install/bootstrap helpers
- `cli/internal/version`: version parsing and compatibility helpers
- `cli/internal/env`: environment and config-path helpers
- `cli/internal/auth`: auth/config validation helpers

## Usage

```bash
# Install using the declarative contract
vrooli resource install codex

# Check that the binary is available and healthy
resource-codex status
```

## Permissions

Manage Codex bash-command patterns via the `permissions` subgroup. The adapter owns a Vrooli-namespaced `[vrooli.permissions]` section in `~/.codex/config.toml` (user scope) or `~/.codex/requirements.toml` (admin scope). All other Codex-native settings (`[profiles.*]`, `sandbox_mode`, `approval_policy`, …) round-trip untouched.

```bash
# Block git stash at user scope (motivating example)
resource-codex permissions deny 'git stash *'

# Same, admin-enforced
resource-codex permissions deny --scope admin 'git stash *'

# View managed patterns
resource-codex permissions list
resource-codex permissions show --raw

# Detect drift since the last Vrooli write
resource-codex permissions drift-check

# Check version and surface the enforcement caveat
resource-codex permissions doctor
```

Mutating verbs (`deny`, `allow`, `ask`, `remove`, `reset`) refuse agent callers (detected via `cliutil.DetectCallerKind`) unless `--i-was-explicitly-authorized` is passed. Read verbs are always allowed.

**Enforcement caveat.** Unlike Claude Code (`permissions.deny` + `PreToolUse` hook) or OpenCode (`permission.bash` map), Codex does **not** today honour per-command-pattern deny/ask/allow rules natively — its policy surface is `sandbox_mode` and `approval_policy`. The `[vrooli.permissions]` section records Vrooli's intent across all coding-agent resources uniformly; for hard live enforcement use Codex's sandbox/approval primitives.

For declarative automation, use `permissions plan --scope user|admin --document desired.json --json` and `permissions reconcile --scope user|admin --document desired.json --json`. The strict v1 document contains `schema_version`, matching `scope`, and ID-addressed `allow`/`ask`/`deny` rules with `matcher: {"kind":"bash","pattern":"..."}`. Plan never writes; reconcile is authorization-gated, preserves unmanaged TOML, and reports desired/live fingerprints, native paths, changes, and the explicitly `intent_only` enforcement posture.

Upstream docs: <https://developers.openai.com/codex/permissions>.

## Notes

- `codex` is an external CLI resource, not a local daemon owned by this resource.
- Keep binary/version/install behavior declarative in `resource.json` whenever possible.
- Keep `cli/main.go` thin; do not treat it as the implementation surface for Codex-specific behavior.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/codex/docs/OPERATIONS.md) as the architecture boundary for future migrations.
