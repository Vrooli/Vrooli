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

This resource uses the updated `external-cli` structure.

- `resource.json` is the declarative authority for install, binary probing, version checks, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Codex-specific Go logic when the manifest and shared control plane are not enough.
- Historical shell behavior has been retired; lifecycle and configuration
  behavior lives in the shared control plane and typed Go packages.

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

## Coding-role policy

Codex owns its concrete coding-role inventory in `model-policy.json`. Use
`resource-codex policy validate`, `policy roles --json`, and `policy resolve
--role code.default --json` to inspect it. The response records the concrete
model, fallbacks, policy provenance, and the intentionally `intent_only`
permission posture; Agent Manager consumes that response at run creation but
does not duplicate the inventory or write Codex configuration.

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

**Enforcement caveat.** Codex's native `sandbox_mode` and `approval_policy` remain the authoritative controls; the `[vrooli.permissions]` section is a uniform policy projection rather than a native pattern matcher. Vrooli also projects `~/.codex/hooks.json` with a `PreToolUse` command hook when deny rules exist. The CLI reports this as `hook_unverified` until a live canary proves the installed Codex version fires and honors the hook; do not treat hook-file presence as sandbox enforcement.

For declarative automation, use `permissions plan --scope user|admin --document desired.json --json` and `permissions reconcile --scope user|admin --document desired.json --json`. The strict v1 document contains `schema_version`, matching `scope`, and ID-addressed `allow`/`ask`/`deny` rules with `matcher: {"kind":"bash","pattern":"..."}`. Plan never writes; reconcile is authorization-gated, preserves unmanaged TOML, and reports desired/live fingerprints, native paths, changes, and the `hook_unverified` enforcement posture.

Upstream docs: <https://developers.openai.com/codex/permissions>.

## Model catalog operations

`resource-codex models list --json` reads the Codex model cache without invoking a model. `resource-codex models resolve --model <id> --json` returns the resource-owned canonical pricing identity. Run `resource-codex policy validate --against-live --json` after a retarget. `observed_at` has a 14-day budget; aliases should remain runner-facing while pinned fallbacks must be refreshed from the same live evidence. Policy edits are reviewed explicitly and are never made by the drift safeguard.

## Notes

- `codex` is an external CLI resource, not a local daemon owned by this resource.
- Keep binary/version/install behavior declarative in `resource.json` whenever possible.
- Keep `cli/main.go` thin; do not treat it as the implementation surface for Codex-specific behavior.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/codex/docs/OPERATIONS.md) as the architecture boundary for future migrations.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
