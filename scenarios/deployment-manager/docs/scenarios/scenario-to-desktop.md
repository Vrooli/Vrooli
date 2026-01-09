# Scenario: scenario-to-desktop

## Current Status (v1)

- Generates Electron apps per scenario.
- Bundles UI production assets and Electron shell UI features (menus, notifications, file system access).
- Requires a running Tier 1 Vrooli instance; the desktop app simply proxies to it.
- Can optionally auto-bootstrap Tier 1: if `AUTO_MANAGE_TIER1` is enabled and `DEPLOYMENT_MODE` is `external-server`, the app now looks for the `vrooli` CLI, runs `vrooli setup --yes yes --skip-sudo yes`, starts `vrooli scenario start <name>` on launch, and stops it on exit.

## Short-Term Tasks

1. ✅ The generator UI, CLI, and quick-generate flows now surface `DEPLOYMENT_MODE`, `SERVER_TYPE`, Tier 1 URLs, and auto-bootstrap toggles so thin clients are configured intentionally.
2. ✅ Thin-client UX now walks builders through selecting the Tier 1 host (direct LAN or Cloudflare/app-monitor), wiring the API endpoint, and deciding whether the desktop build should run `vrooli setup/start` locally.
3. ✅ Each generated Electron wrapper streams health telemetry to `deployment-telemetry.jsonl`, and scenario-to-desktop offers `telemetry collect` to ingest those logs for deployment-manager.

## Long-Term Vision (v2+)

- Accept bundle manifests from deployment-manager describing binaries/resources/secrets to include.
- Ship a cross-platform **scenario runtime** executable that owns lifecycle (start/stop/restart, health, logs, telemetry), allocates ports from a private range, and exposes a localhost control API that Electron talks to; Electron never spawns services directly.
- Package bundleable runtime dependencies (SQLite/duckdb, in-process caches, bundled Playwright driver + Chromium, packaged models) alongside the UI + API. Heavy/shared services must be swapped before bundling.
- Include a minimal `vrooli`-compatible shim in PATH for essentials like `scenario status/port`, backed by the runtime’s registry—no global CLI install assumed.
- Provide a first-run configuration wizard for secrets flagged by the manifest (generate vs prompt vs remote) and persist them under the app data root.
- Run schema migrations/seed data for swapped stores idempotently on first run and upgrade; store data/logs under OS-specific app data dirs.
- Aggregate logs + deployment telemetry (`deployment-telemetry.jsonl`) for deployment-manager ingestion; keep upgrade-safe versioned data dirs.
- Support auto-update channels and differential patches.

## Blocking Issues

- No way to bundle heavy dependencies yet (Ollama, Postgres) — requires dependency swapping.
- Bundle manifest schema must formalize DAG, swaps, per-OS assets, env templates, health/readiness, ports, data dirs, and secrets strategy.
- Need cross-platform installers (MSI, DMG, AppImage) once bundling works.

## Milestones

1. **Telemetry Release** — Thin client instruments dependency gaps + reports to deployment-manager. *(CLI `scenario-to-desktop telemetry collect` uploads `deployment-telemetry.jsonl` today.)*
2. **Configurable Targets** — `DEPLOYMENT_MODE` toggles + wizard for connecting to Tier 1 servers. *(UI/CLI implemented; deployment-manager will read the metadata next.)*
3. **Embedded Runtime Spike** — Prototype bundling SQLite + lightweight API for picker-wheel to prove feasibility.
4. **Manifest-Driven Bundles** — Consume deployment-manager manifests and spin up background binaries/resources automatically.
5. **Installer Tier** — Generate OS-specific installers (MSI/DMG/AppImage) with updates + signing.

## Success Signals

- Picker Wheel ships as a self-contained desktop build once dependency swaps land.
- Bundles describe which services are embedded vs remote and sync that metadata back to deployment-manager.
- Install experience handles user secrets + remote server pairing without manual edits.

## References

- [Tier 2 Overview](../tiers/tier-2-desktop.md)
- [Packaging Matrix](../guides/packaging-matrix.md)
- [Picker Wheel Desktop Example](../examples/picker-wheel-desktop.md)
