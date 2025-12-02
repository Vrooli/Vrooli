# Bundled Desktop Runtime Plan

Goal: deliver true offline/portable desktop bundles (UI + API + resources + secrets bootstrap) without relying on Bash/systemd/Docker. This plan aligns the four key scenarios so future agents can implement in parallel.

## Shared Context and Constraints
- Per-platform artifacts: one Electron app + one runtime binary per OS/arch; no external deps assumed.
- Manifest-driven: `bundle.json` from deployment-manager + scenario-dependency-analyzer governs everything (services, swaps, assets, env, ports, secrets, health checks, data dirs, migrations).
- Bundled resources must be “bundle-safe” (e.g., SQLite/duckdb, embedded caches, packaged model files, bundled Playwright driver/Chromium). Heavy infra (Postgres, Redis, Ollama, browserless) require swaps recorded in the manifest.
- Runtime is the supervisor: owns port allocation, process start/stop, readiness, telemetry, logs, and exposes a control API; Electron only launches runtime and loads UI when ready.
- Secrets: no infra secrets in bundles; per-install secrets are generated or prompted on first run and persisted locally.
- Minimal CLI shim: only mirrors essential `vrooli` commands (status/ports/log tail) backed by the runtime’s control API and registry.

## Implementation Contracts (concrete, to avoid guessing)
- Manifest schema (to be formalized in JSON Schema v0.1)
  - Field types, required/optional, defaults.
  - Provide two full examples under `docs/deployment/examples/manifests/`: `desktop-happy.json` (UI+API+SQLite, no GPU) and `desktop-playwright.json` (adds Playwright driver, Chromium vendored or Electron path reuse, GPU optional flag).
  - Validate in deployment-manager; scenario-to-desktop consumes only validated manifests.
- Runtime control API (baseline spec)
  - Routes:
    - `GET /healthz` → `{status: "ok"|"degraded"|"failed"}`.
    - `GET /readyz` → `{ready: bool, details: {serviceId: {ready: bool, reason?: string}}}`.
    - `GET /ports` → `{services: {serviceId: {portName: number}}, apiBase?: string}`.
    - `GET /logs/tail?serviceId=&lines=` → text/JSON stream of last N lines.
    - `POST /shutdown` → `{status: "stopping"}`.
    - `POST /secrets` → accepts `{secrets: {id: value}}` for first-run/user-prompt entries.
    - `GET /telemetry` → returns path/status for `deployment-telemetry.jsonl`.
  - Auth: bearer token in header; bind to loopback. Token persisted in app data.
  - Log format: per-service log files under app data log dir; tail endpoint reads from there.
  - Telemetry record shape (JSONL): `{"ts": "...", "event": "app_start|ready|dependency_unreachable|restart|shutdown|asset_missing|migration_failed", "service_id": "...", "details": {...}}`.
- Secrets storage/hand-off
  - Store per-install secrets in app data under `secrets.json` (0600 perms). Infra secrets never appear.
  - Runtime injects secrets as env vars or temp files per manifest instructions (`secrets[].target`: `env` | `file:path`).
  - First-run wizard collects `user_prompt` secrets; runtime generates `per_install_generated`; both persisted; subsequent launches reuse without prompting unless validation fails.
- Migrations/versioning
  - Manifest carries `app.version` and per-service `migrations` with `version` and `command`.
  - Runtime tracks applied versions in app data (e.g., `migrations.json`), runs missing ones in order; on failure, stop startup and emit telemetry `migration_failed`.
  - No rollback automation initially; document manual recovery guidance.
- Test plan locations
  - scenario-to-desktop: integration tests under `scenarios/scenario-to-desktop/test/` using sample manifests; e2e harness to launch runtime and assert ready/logs/ports/telemetry.
  - deployment-manager: schema validation + export tests under its test suite; fixtures using sample manifests.
  - scenario-dependency-analyzer: unit tests for DAG/swaps and manifest skeleton generation.
  - secrets-manager: tests for classification, generation hooks, and manifest export blocking infra secrets.

## Interfaces and Handoffs
- scenario-dependency-analyzer → deployment-manager: DAG + fitness + swap options + manifest skeleton (with service/resource metadata, health, migrations, ports).
- secrets-manager → deployment-manager: per-install secret plan (class, generation rules, prompt schema, storage guidance) per tier.
- deployment-manager → scenario-to-desktop: versioned `bundle.json` (validated, signed) with all binaries/assets resolved and secrets plan embedded as placeholders/validation rules.
- scenario-to-desktop → deployment-manager: telemetry uploads (`deployment-telemetry.jsonl`) and build status for visibility/feedback loops.

## Draft Manifest Schema (v0.1 outline)
- `schema_version`: string.
- `app`: name, version, description, tier (`desktop`), electron/runtime versions pinned.
- `services`: array of entries with:
  - `id`, `type` (`api-binary` | `ui-bundle` | `worker` | `resource`), `os`/`arch` binary paths, `args`, `env` (templated), `cwd`, `data_dirs`, `log_dir`.
  - `ports`: `preferred_range`, `requires_socket` (bool), exported names.
  - `health`: `type` (`http`, `tcp`, `command`), `path`/`port`/`command`, `timeout`, `interval`, `retries`.
  - `readiness`: condition (`health_success` | `log_match` | `port_open`).
  - `dependencies`: IDs for startup ordering.
  - `migrations`: list with `version`, `command`, `args`, `env`, `run_on` (`first_install` | `upgrade` | `always`).
  - `gpu`: `requirement` (`required` | `optional_with_cpu_fallback` | `optional_but_warn`).
  - `assets`: bundled files/blobs required pre-run (checksums, sizes).
- `swaps`: original resource → bundled alternative (e.g., Postgres → SQLite), rationale, limits.
- `secrets`: entries with `id`, `class` (`infrastructure` | `per_install_generated` | `user_prompt` | `remote_fetch`), `format`/validation, storage hint, prompt copy.
- `ipc`: control API host/port, auth token placeholder, allowed callers.
- `telemetry`: local path for `deployment-telemetry.jsonl`, endpoints for upload.
- `ports`: global ranges and exclusions.
- `playwright`: driver entry (`node driver/server.js`), env (`PLAYWRIGHT_DRIVER_PORT=0`, `PLAYWRIGHT_CHROMIUM_PATH` optional), assets (vendored Chromium path), readiness check.

## Acceptance Criteria by Scenario
- scenario-to-desktop
  - Consumes a sample `bundle.json`, packages runtime + services, launches runtime, waits for readiness, serves UI, and CLI shim returns ports/status/logs.
  - First-run wizard prompts for `user_prompt` secrets; generated secrets persisted; rerun is non-interactive.
  - Telemetry emitted and uploadable; missing asset or health failure blocks launch with clear UI.
- deployment-manager
  - Generates `bundle.json` for a sample scenario with swaps applied; validates against schema; refuses export if assets/secrets unresolved.
  - Renders secret plan UI; exports prompt/generation rules into manifest.
  - Ingests telemetry and surfaces failures (dependency_unreachable, swap_missing_asset).
- scenario-dependency-analyzer
  - Produces DAG + fitness + swap suggestions; emits manifest skeleton with health/migration hints and port ranges.
  - Flags non-bundle-safe resources and suggests alternatives.
- secrets-manager
  - Returns per-install secret plan per tier; blocks infra secrets from manifest; provides validation rules consumed by runtime/UI.

## Work Breakdown (parallelizable)
- Shared
  - Finalize `bundle.json` schema v0.1 and sample manifests (desktop happy path, GPU optional, Playwright).
- scenario-to-desktop
  - Implement runtime launcher + control API client; wire UI/CLI to control surface.
  - Build first-run wizard + health/log/telemetry panels.
  - Package Playwright driver + Chromium or Electron-path reuse; set env.
  - CLI shim implementation + docs.
- deployment-manager
  - Schema validation and manifest export; swap selection UI; secret plan integration.
  - Telemetry ingestion UI with filters for common failures.
- scenario-dependency-analyzer
  - DAG + fitness scoring with swap suggestions; manifest skeleton generator; health/migration metadata extraction.
- secrets-manager
  - Secret classification extension; generation hooks; API for bundle export; validation rule definitions.

## Shared Artifacts and Samples
- Location for sample manifests: `docs/deployment/examples/manifests/` (baseline SQLite + API bundle, and Playwright-enabled variant seeded).
- Reusable assets: bundled SQLite seed example, small model blob example, Playwright driver + Chromium vendored path notes.
- Reference test scenario: choose an existing lightweight scenario (e.g., picker-wheel) for end-to-end fixture.

## Risks and Assumptions
- Size constraints: model/browser assets may bloat installers; need size budgets and warnings.
- GPU availability variance; must honor `gpu_optional_*` flags and degrade gracefully.
- Electron/Playwright version pinning to avoid Chromium mismatches.
- Asset licensing for bundled models/browsers.
- Windows/macOS sandbox quirks (path lengths, gatekeeper) need validation.

## Doc Alignment
- Keep aligned with: `docs/deployment/tiers/tier-2-desktop.md`, `docs/deployment/guides/packaging-matrix.md`, `docs/deployment/guides/secrets-management.md`, `docs/deployment/examples/picker-wheel-desktop.md`, `resources/playwright/README.md`.

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

## Progress
**Done**
- [x] Schema + samples: Added `docs/deployment/bundle-schema.desktop.v0.1.json` and sample manifests (`desktop-happy.json`, `desktop-playwright.json`) to anchor validation and fixtures.
- [x] Runtime core: Supervisor loads manifests, allocates ports, runs migrations, enforces readiness/health, persists secrets/migrations, records telemetry, and exposes control API + CLI shim; Playwright env defaults + asset size checks added.
- [x] Runtime wiring to product UX: Bundled mode now consumes real manifests end-to-end (UI/CLI accept `bundle_manifest_path`, generator packages bundles/runtime, Electron launches the bundled runtime instead of blocking, and dry-run defaults to real service startup).

**Missing / To Do**
- [x] Manifest assembly: deployment-manager now generates desktop `bundle.json` from scenario-dependency-analyzer bundle skeletons (v0.1) and merges secrets plans; exposed via `/api/v1/bundles/assemble`.
- [x] Secrets plan + UX: secrets-manager exports bundle-ready secret plans per tier (including fallback when discovery fails); Electron bundled launches now gate on the runtime control API, prompt for `user_prompt` secrets before readiness checks, POST them to `/secrets`, and fail fast with clear messaging if required secrets are withheld.
- [ ] Telemetry/logs UX: Electron UI should render `/readyz`/`/ports`/`/logs` and automate telemetry upload; deployment-manager UI should ingest and surface bundle telemetry failures.
- [ ] Packaging completeness polish: scrub remaining exe/dmg references, confirm MSI/PKG/AppImage flows, and document/update the updater channel design (still pending).
- [ ] GPU visibility/tests: surface GPU availability/requirement status in UI and add runtime tests covering gpu_required/optional paths.
