# Claude Code Resource

Anthropic Claude Code CLI for interactive and scripted development workflows.

> **Anthropic-native.** Claude Code talks to the Anthropic API directly; it does
> not route through a local model proxy. This is the one acknowledged difference
> from the other coding-agent resources — codex and opencode reach local Ollama
> models first-class, whereas claude-code is cloud/Anthropic-only. (The retired
> LiteLLM local-proxy path has been removed.)

## Intent

- Resource ID: `claude-code`
- Category: `developer-tooling`
- Driver: `external-cli`
- Portability tier: `partial`

## Use Cases

- Use Claude Code as an interactive coding agent inside local development workflows.
- Run scripted analysis or prompt-driven automation against a working tree.
- Provide a standard external CLI dependency for scenarios and operator tooling.

## Architecture

This resource uses the `external-cli` structure with Go-owned lifecycle,
configuration, installation, permissions, and hook reconciliation.

- `resource.json` is the declarative authority for install, binary probing, version checks, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Claude Code-specific Go logic when the manifest and shared control plane are not enough.
- Historical `lib/` shell behavior has been retired. The only remaining shell
  file is the explicitly allowed PreToolUse payload under `cli/internal/`.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Claude Code-specific Go code under `cli/internal/...` only where specialization is real
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
vrooli resource install claude-code

# Check that the binary is available and healthy
resource-claude-code status
```

## Coding-role policy

Claude Code owns its concrete coding-role inventory in `model-policy.json`.
Use `resource-claude-code policy validate`, `policy roles --json`, and
`policy resolve --role code.default --json` to inspect it. The response records
the concrete model, fallbacks, native policy provenance, and permission
enforcement posture; Agent Manager consumes that response at run creation but
does not copy the inventory or write Claude's native configuration.

## Permissions

`resource-claude-code permissions ...` is the canonical way to manage the bash-pattern permission rules in `~/.claude/settings.json`. The verbs are agent-gated: a detected agent caller must pass `--i-was-explicitly-authorized` to mutate the file. Read verbs (`list`, `show`, `drift-check`, `doctor`) are always allowed.

```bash
# Block git stash for Claude Code (the canonical example).
resource-claude-code permissions deny 'Bash(git stash*)'

# Inspect current state.
resource-claude-code permissions list

# Detect hand-edits to settings.json since the last Vrooli write.
resource-claude-code permissions drift-check

# Verify the installed claude binary matches the version this resource was built against.
resource-claude-code permissions doctor
```

Every `Bash(...)` deny rule is paired with the source-controlled
`.claude/.vrooli-hooks/pretooluse-bash-deny.sh` PreToolUse matcher as a
defensive backstop for the upstream `permissions.deny` enforcement bug
([anthropics/claude-code#18846](https://github.com/anthropics/claude-code/issues/18846),
[#29026](https://github.com/anthropics/claude-code/issues/29026)). The matcher
normalizes Claude's `Bash(...)` syntax and resolves `$HOME`/`~` aliases before
glob matching; it is verified with a data-only replay and a non-mutating live
probe.

For declarative automation, use `permissions plan --document desired.json --json` and `permissions reconcile --document desired.json --json`. The strict v1 document contains `schema_version`, optional `scope: "user"`, and ID-addressed `allow`/`ask`/`deny` rules with `matcher: {"kind":"bash","pattern":"..."}`. Plan never writes; reconcile is authorization-gated, preserves unmanaged settings, and reports desired/live fingerprints, native paths, changes, and the `hook_verified` enforcement posture.

Scenario-owned hooks use `resource-claude-code hooks reconcile` and
`hooks remove`, which update only the identified hook entry while preserving
unmanaged Claude settings. This is the Go replacement for the retired
`lib/hooks.sh` seam used by web-console TTS.

Pinned upstream docs: <https://code.claude.com/docs/en/permissions> (see `resource.json` → `upstream_cli`).

## Model catalog operations

`resource-claude-code models list --json` reports the quota-free `--model` alias surface and pinned evidence. `resource-claude-code models resolve --model <id> --json` returns the resource-owned canonical pricing identity. Run `resource-claude-code policy validate --against-live --json` after a retarget. Keep moving aliases as primaries and use pinned fallbacks for reproducible history; refresh provenance within the 14-day staleness budget. Policy edits remain explicitly reviewed.

## Notes

- `claude-code` is an external CLI resource, not a local daemon owned by this resource.
- Keep binary/version/install behavior declarative in `resource.json` whenever possible.
- Keep `cli/main.go` thin; do not treat it as the implementation surface for Claude Code-specific behavior.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/claude-code/docs/OPERATIONS.md) as the architecture boundary for future migrations.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
