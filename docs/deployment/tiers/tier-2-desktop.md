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

## Required Capabilities for v2

| Capability | Source Scenario | Notes |
|------------|-----------------|-------|
| Dependency DAG + resource sizing | `scenario-dependency-analyzer` | Must recurse and produce requirements (RAM, disk, GPU, offline capability). |
| Fitness scoring & swap guidance | `deployment-manager` | Enables Postgres→SQLite swaps, Ollama→OpenRouter, etc. |
| Secret stratification | `secrets-manager` | Infrastructure secrets stripped, per-install secrets generated. |
| Desktop packager | `scenario-to-desktop` | Needs to bundle UI, API, dependencies, bootstrapper, upgrade channel. |

## Roadmap

1. Extend `service.json` with `deployment.platforms.desktop` metadata (supported?, requirements, alternatives, offline flag).
2. Teach `scenario-dependency-analyzer` to compute bundle manifests (`bundle.json`) that list binaries, resources, assets per dependency.
3. Build a dependency swap UI inside `deployment-manager` so we can try e.g., SQLite instead of Postgres for desktop.
4. Update `scenario-to-desktop` to:
   - Spin up bundled resources (e.g., embed SQLite DB file, package lite inference models, etc.).
   - Offer configuration prompts for optional secrets (OpenRouter API key, etc.).
   - Ship autopatching/updater hooks.
5. Create automated installers (MSI/pkg/AppImage) driven by the bundle manifest.

## Interim Guidance

- Document tier 2 projects honestly (see [Picker Wheel Desktop](../examples/picker-wheel-desktop.md)).
- When delivering thin clients, include instructions for connecting to a Tier 1 environment.
- Track every missing dependency as a deployment-manager issue so we never repeat the surprise.
