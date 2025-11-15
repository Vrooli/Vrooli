# Scenario: scenario-to-desktop

## Current Status (v1)

- Generates Electron apps per scenario.
- Bundles UI production assets and Electron shell UI features (menus, notifications, file system access).
- **Does not** bundle API servers, resources, or Vrooli CLI dependencies.
- Requires a running Tier 1 Vrooli instance; the desktop app simply proxies to it.

## Short-Term Tasks

1. Update templates/config so desktop apps expose a `DEPLOYMENT_MODE` (e.g., `external-server`, `cloud-api`, `bundled`).
2. Document how to point thin clients at Tier 1 servers (host selection, authentication, Cloudflare tunnel URLs).
3. Emit telemetry about missing dependencies so deployment-manager can reason about gaps.

## Long-Term Vision (v2+)

- Accept bundle manifests from deployment-manager describing binaries/resources/secrets to include.
- Package runtime dependencies (SQLite, lightweight vector DBs, offline models) alongside the UI + API.
- Run local services via background processes or embedded servers launched by the Electron main process.
- Provide first-run configuration wizard for user secrets (OpenRouter key, remote server selection, etc.).
- Support auto-update channels and differential patches.

## Blocking Issues

- No way to bundle heavy dependencies yet (Ollama, Postgres) — requires dependency swapping.
- Secrets must be re-architected to avoid leaking infrastructure credentials.
- Need cross-platform installers (MSI, DMG, AppImage) once bundling works.

## References

- [Tier 2 Overview](../tiers/tier-2-desktop.md)
- [Packaging Matrix](../guides/packaging-matrix.md)
- [Picker Wheel Desktop Example](../examples/picker-wheel-desktop.md)
