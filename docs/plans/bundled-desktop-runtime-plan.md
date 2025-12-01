# Bundled Desktop Runtime Plan

Goal: deliver true offline/portable desktop bundles (UI + API + resources + secrets bootstrap) without relying on Bash/systemd/Docker. This plan aligns the four key scenarios so future agents can implement in parallel.

## Shared Context and Constraints
- Per-platform artifacts: one Electron app + one runtime binary per OS/arch; no external deps assumed.
- Manifest-driven: `bundle.json` from deployment-manager + scenario-dependency-analyzer governs everything (services, swaps, assets, env, ports, secrets, health checks, data dirs, migrations).
- Bundled resources must be “bundle-safe” (e.g., SQLite/duckdb, embedded caches, packaged model files, bundled Playwright driver/Chromium). Heavy infra (Postgres, Redis, Ollama, browserless) require swaps recorded in the manifest.
- Runtime is the supervisor: owns port allocation, process start/stop, readiness, telemetry, logs, and exposes a control API; Electron only launches runtime and loads UI when ready.
- Secrets: no infra secrets in bundles; per-install secrets are generated or prompted on first run and persisted locally.
- Minimal CLI shim: only mirrors essential `vrooli` commands (status/ports/log tail) backed by the runtime’s control API and registry.

## scenario-to-desktop
- Implement runtime packaging
  - Embed per-OS runtime binary and all service/resource binaries/assets referenced in `bundle.json`.
  - Package UI bundle and wire renderer to local runtime API; block UI launch until runtime reports ready.
  - Choose IPC (localhost HTTP/UDS/named pipe) + auth token stored in app data.
- Manifest consumption
  - Read `bundle.json`, stage binaries/assets, place data/log roots in user data dirs (`%APPDATA%` / `~/Library/Application Support/` / `~/.config/`).
  - Validate manifest (schema + completeness) pre-build; fail fast with actionable errors.
- Runtime control UX
  - First-run wizard to collect/prompt user secrets and config declared in manifest; pass to runtime via control API.
  - Health panel to surface readiness, port map, log tail, and telemetry upload (to scenario-to-desktop API/deployment-manager).
  - Shutdown request on app exit; restart hooks for crash/upgrade.
- Minimal vrooli shim
  - Ship a tiny CLI that proxies to runtime control endpoints for `scenario status/port/logs` equivalents.
  - Update templates and docs to discourage broader CLI expectations.
- Telemetry
  - Emit deployment telemetry (`app_start`, `ready`, `dependency_unreachable`, `restart`, `shutdown`) from runtime to `deployment-telemetry.jsonl`; expose upload from Electron UI/CLI.

## deployment-manager
- Bundle creation workflow
  - UI flow to select target scenario + tier=desktop, drive dependency analysis, choose swaps, and export signed `bundle.json`.
  - Secrets planning step (classify: generated/prompt/remote) with per-tier rules; produce instructions for first-run wizard.
  - Fitness/scoring surface to highlight blockers (GPU-required models, heavy DBs) and propose swaps.
- Manifest schema owner
  - Define and publish `bundle.json` schema (services/resources, binaries/paths, env templates, ports/port ranges, health checks, readiness probes, data dirs, migrations, secrets, IPC token requirements, telemetry endpoints).
  - Versioned schema with compatibility notes for scenario-to-desktop runtime.
- Telemetry ingestion
  - Ingest `deployment-telemetry.jsonl` from built apps; visualize failures (dependency_unreachable, swap_missing_asset) to drive new swap rules.

## scenario-dependency-analyzer
- DAG + swap planning
  - Compute full dependency graph (scenarios + resources) and annotate bundle fitness (disk/RAM/CPU/GPU/offline).
  - Propose swaps to bundle-safe alternatives; emit chosen/available options into manifest fields deployment-manager can render.
- Bundle manifest generation
  - Produce `bundle.json` skeleton: binaries/assets per OS/arch, env templates, default port ranges, start order, health checks, readiness signals, migration hooks, data directories.
  - Mark resources needing artifacts (SQLite seed, model blobs, Playwright driver/Chromium) and their source/download/build steps for deployment-manager to fulfill before packaging.
- Port/health metadata
  - Standardize per-service health check definitions and readiness criteria so runtime can act deterministically.

## secrets-manager
- Secret classification for bundles
  - Extend schema to mark secrets as `infrastructure` (never bundled), `per_install_generated`, `user_prompt`, `remote_fetch`.
  - Provide generation hooks and storage guidance for per-install secrets (e.g., write to app data with correct permissions).
- Bundle export integration
  - API/UI for deployment-manager to fetch secret plans for a target scenario/tier and render first-run instructions.
  - Ensure manifests include placeholders and validation rules (format, length, required/optional) for runtime to enforce.
- Compliance/guardrails
  - Prevent infra secrets from entering `bundle.json`; add audit logs for bundle exports; surface warnings for secrets lacking a per-install strategy.

## Open Decisions (to settle early)
- IPC choice and auth model between Electron and runtime (localhost HTTP vs UDS/pipe; token storage and rotation).
- Minimal CLI command set and UX error handling when commands exceed supported scope.
- Data/migration contract: how runtime discovers and runs migrations, and how upgrades are versioned.
- Handling GPU-optional workloads: manifest flags for “degrade to CPU” vs “fail if no GPU.”
- Packaging of headless automation: standardizing bundled Playwright driver/Chromium paths and env vars (`PLAYWRIGHT_DRIVER_URL`, `PLAYWRIGHT_CHROMIUM_PATH`).

## Decisions Locked
- IPC: Localhost HTTP control API with random auth token bound to loopback.
- Auth: Persist bearer token in app data; rotate on reinstall/major update if needed.
- CLI shim: Only `status`, `port <service|scenario>`, `logs [--tail]`, `health` mapped to runtime control API.
- Migrations: Manifest lists migration command + version; runtime runs idempotent migrations on start/upgrade.
- GPU handling: Manifest flag per service (`gpu_required`, `gpu_optional_with_cpu_fallback`, `gpu_optional_but_warn`); runtime enforces.
- Playwright bundling: Use existing `resources/playwright` driver; bundle driver + node_modules + Chromium (or point `PLAYWRIGHT_CHROMIUM_PATH` to Electron’s); runtime starts driver directly (no shell wrappers) with env from manifest and records `PLAYWRIGHT_DRIVER_URL`/`ENGINE=playwright`.
- Control API surface: Expose `/healthz`, `/readyz`, `/ports`, `/logs/tail`, `/shutdown` for Electron UX + debug.
- Token storage: App data directory alongside telemetry.
- Ports: Manifest-specified ranges; runtime allocates sequentially and exposes map.
- Assets/swaps: Hard fail with clear error/telemetry if required asset is missing; downloads happen before packaging, not at runtime.
