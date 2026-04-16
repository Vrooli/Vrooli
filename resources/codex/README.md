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

## Notes

- `codex` is an external CLI resource, not a local daemon owned by this resource.
- Keep binary/version/install behavior declarative in `resource.json` whenever possible.
- Keep `cli/main.go` thin; do not treat it as the implementation surface for Codex-specific behavior.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/codex/docs/OPERATIONS.md) as the architecture boundary for future migrations.
