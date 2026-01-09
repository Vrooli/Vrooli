# Tier 2 · Desktop Bundles

Goal: deliver a portable Windows/macOS/Linux app that contains the UI, API, dependencies, and bootstrap secrets for a single scenario (plus whatever scenarios/resources it needs).

## Current State (v1)

- `scenario-to-desktop` can generate an Electron wrapper that bundles the UI.
- The resulting `.exe` (or `.app`) is a **thin client**: it expects the API + resources to be reachable on a running Vrooli installation.
- There is **no** automated bootstrap for dependencies, secrets, or offline capability.

## Why v1 Failed

When we shipped the Picker Wheel Electron build we discovered:

1. The UI bundle worked, but it could not reach its API, because the API only ran inside the main Vrooli environment.
2. Even if we bundled the API binary, its dependencies (Postgres, Redis, etc.) were still missing.
3. The CLI + resource lifecycle (`vrooli` scripts, port registry, secrets) were assumed to exist globally.

## What v2 Bundling Must Do

- **Self-contained per platform** — Ship a single runtime binary per OS/arch plus any scenario/resource binaries; avoid shell scripts, systemd, Docker, and external tooling assumptions.
- **Manifest-driven** — Consume a `bundle.json` produced by deployment-manager + scenario-dependency-analyzer that encodes the full DAG (scenarios + resources), swaps (e.g., Postgres→SQLite), per-OS assets, env templates, port ranges, health checks, readiness signals, data directories, and secrets treatment (generate vs prompt vs remote).
- **Mandatory dependency swapping** — Heavy/shared services (Postgres, Redis, Ollama, browserless) are replaced with bundleable equivalents (SQLite/duckdb, in-process cache, packaged model files, bundled Playwright driver/Chromium) and the manifest records the chosen swap + artifacts.
- **Runtime-owned lifecycle** — A cross-platform “scenario runtime” executable supervises every bundled process, allocates ports, exposes a small control/health API, tracks logs/telemetry, and handles clean shutdown. Electron only launches the runtime and talks to it over a local channel (localhost HTTP/pipe), never spawning individual services itself.
- **Minimal vrooli shim** — Include a tiny `vrooli`-compatible CLI in PATH that proxies the runtime for essentials like `scenario status/port`; no global CLI install is expected.
- **Data + migrations** — Bundles carry schema migrations/seed data for swapped stores (e.g., SQLite). Runtime applies them idempotently on first run and upgrade.
- **Secrets hygiene** — No infrastructure secrets inside bundles. Manifest flags which secrets to generate locally, which to prompt the user for (first-run wizard), and which remain remote.
- **Logging + telemetry** — Runtime aggregates logs per service and emits deployment telemetry (`deployment-telemetry.jsonl`) for deployment-manager ingestion.
- **Upgrade safety** — Use versioned data directories under the OS app data root; runtime must survive restart/upgrade and re-run migrations as needed.

## Required Capabilities for v2

| Capability | Source Scenario | Notes |
|------------|-----------------|-------|
| Dependency DAG + resource sizing | `scenario-dependency-analyzer` | Must recurse and produce requirements (RAM, disk, GPU, offline capability). |
| Fitness scoring & swap guidance | `deployment-manager` | Enables Postgres→SQLite swaps, Ollama→OpenRouter, etc. |
| Secret stratification | `secrets-manager` | Infrastructure secrets stripped, per-install secrets generated. |
| Desktop packager | `scenario-to-desktop` | Needs to bundle UI, API, dependencies, bootstrapper, upgrade channel. |

## Installer Formats

Desktop builds now produce enterprise-ready installer formats:

| Platform | Format | File | Notes |
|----------|--------|------|-------|
| Windows | MSI | `.msi` | Enterprise-friendly, Group Policy support, silent install |
| macOS | PKG | `.pkg` | Installer package, supports signed notarization |
| Linux | AppImage | `.AppImage` | Portable, self-contained, no install required |
| Linux | DEB | `.deb` | Debian/Ubuntu package manager integration |

Build commands:
```bash
npm run dist:win    # Windows MSI
npm run dist:mac    # macOS PKG
npm run dist:linux  # Linux AppImage + DEB
npm run dist:all    # All platforms
```

## Auto-Update System

Desktop applications support automatic updates via `electron-updater`:

- **Three channels**: `dev` (internal), `beta` (testers), `stable` (production)
- **Two providers**: GitHub Releases or self-hosted generic server
- **User control**: Manual update check via Help menu
- **Telemetry**: Update events recorded for deployment-manager analytics

See [Auto-Update Channel Design](../guides/auto-updates.md) for full configuration details.

**Quick setup (GitHub Releases):**
```json
{
  "update_config": {
    "channel": "stable",
    "provider": "github",
    "github": {"owner": "your-org", "repo": "your-app-desktop"},
    "auto_check": true
  }
}
```

## Roadmap

1. Extend `service.json` with `deployment.platforms.desktop` metadata (supported?, requirements, alternatives, offline flag).
2. Teach `scenario-dependency-analyzer` to compute bundle manifests (`bundle.json`) that list binaries, resources, assets per dependency.
3. Build a dependency swap UI inside `deployment-manager` so we can try e.g., SQLite instead of Postgres for desktop.
4. Update `scenario-to-desktop` to:
   - Spin up bundled resources (e.g., embed SQLite DB file, package lite inference models, etc.).
   - Offer configuration prompts for optional secrets (OpenRouter API key, etc.).
5. ~~Create automated installers (MSI/pkg/AppImage) driven by the bundle manifest.~~ ✅ Done - see Installer Formats above.
6. ~~Implement auto-update channel design.~~ ✅ Done - see Auto-Update System above.

## Interim Guidance

- Document tier 2 projects honestly (see [Picker Wheel Desktop](../examples/picker-wheel-desktop.md)).
- When delivering thin clients, include instructions for connecting to a Tier 1 environment.
- Track every missing dependency as a deployment-manager issue so we never repeat the surprise.
- When experimenting with bundled automation, embed the Playwright driver + browser: start it from Electron main with `PLAYWRIGHT_DRIVER_PORT=0`, read the chosen port, set `PLAYWRIGHT_DRIVER_URL`/`ENGINE=playwright` for the bundled API, and reuse Electron’s Chromium via `PLAYWRIGHT_CHROMIUM_PATH` or ship `@playwright/browser-chromium` to stay offline.
