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

This resource is being aligned to the updated `external-cli` structure.

- `resource.json` is the declarative authority for install, binary probing, version checks, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Claude Code-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

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

Every `Bash(...)` deny rule is paired with a `PreToolUse` hook entry (script materialized in `~/.claude/.vrooli-hooks/`) as a defensive backstop for the upstream `permissions.deny` enforcement bug ([anthropics/claude-code#18846](https://github.com/anthropics/claude-code/issues/18846), [#29026](https://github.com/anthropics/claude-code/issues/29026)).

For declarative automation, use `permissions plan --document desired.json --json` and `permissions reconcile --document desired.json --json`. The strict v1 document contains `schema_version`, optional `scope: "user"`, and ID-addressed `allow`/`ask`/`deny` rules with `matcher: {"kind":"bash","pattern":"..."}`. Plan never writes; reconcile is authorization-gated, preserves unmanaged settings, and reports desired/live fingerprints, native paths, changes, and the `hook_backed` enforcement posture.

Pinned upstream docs: <https://code.claude.com/docs/en/permissions> (see `resource.json` → `upstream_cli`).

## Notes

- `claude-code` is an external CLI resource, not a local daemon owned by this resource.
- Keep binary/version/install behavior declarative in `resource.json` whenever possible.
- Keep `cli/main.go` thin; do not treat it as the implementation surface for Claude Code-specific behavior.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/claude-code/docs/OPERATIONS.md) as the architecture boundary for future migrations.
