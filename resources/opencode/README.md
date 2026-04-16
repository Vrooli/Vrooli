# OpenCode Resource

OpenCode CLI resource for terminal-first coding workflows inside Vrooli.

## Intent

- Resource ID: `opencode`
- Category: `development`
- Driver: `external-cli`
- Portability tier: `partial`

## Architecture

This resource now follows the `external-cli` template structure.

- `resource.json` is the declarative authority for install, binary probing, version checks, environment exports, freshness, and lifecycle metadata.
- `cli/` is the single binary entrypoint and command wiring surface.
- `cli/internal/` is the default home for OpenCode-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/discovery`: host binary detection and probing helpers
- `cli/internal/install`: install/bootstrap helpers
- `cli/internal/version`: version parsing and compatibility helpers
- `cli/internal/env`: environment, config-path, and runtime file helpers
- `cli/internal/auth`: provider credential validation and auth-store translation helpers

## Usage

```bash
# Install the upstream binary through the shared resource control plane
vrooli resource install opencode

# Check binary availability and health
resource-opencode status
```

## Configuration

- Mutable config and auth state should flow through canonical resource storage, not ad hoc repo-local paths.
- Keep environment exports and binary contract metadata declarative in `resource.json`.
- Only add logic to `cli/internal/env` or `cli/internal/auth` when OpenCode-specific shaping or validation is genuinely needed.

## Notes

- This resource wraps an external CLI. It should stay thin by default.
- Do not grow `cli/main.go` into a second operator framework.
- If the resource needs specialized discovery, install translation, or auth handling, place it under `cli/internal/...` rather than in shell wrappers.

## References

- [OpenCode](https://opencode.ai)
