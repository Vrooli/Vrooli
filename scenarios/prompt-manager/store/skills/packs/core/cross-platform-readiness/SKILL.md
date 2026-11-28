---
name: "cross-platform-readiness"
description: "Prepares scenarios for deployment beyond Tier 1 (local stack) by eliminating environment variable dependencies, replacing non-portable resources, adopting api-core/storage for runtime filesystem state, and ensuring build/runtime portability across desktop, mobile, cloud, and enterprise tiers."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer"]
  tags: ["portability","desktop","mobile","deployment","environment-variables","resources","filesystem","api-core-storage","sqlite","bundling","electron","cross-compilation"]
  status: "active"
  revision: 30
  createdAt: "2026-02-03T14:00:00Z"
  updatedAt: "2026-05-04T02:48:26Z"
  requires:
    scenarios: ["prompt-manager", "vrooli"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "vrooli"]
  origin:
    kind: "authored"
---
## Steer focus: Cross-Platform Readiness

Prioritize **preparing scenarios for deployment beyond Tier 1** (local Vrooli stack) in `scenarios/{{TARGET}}/`. This skill steers toward eliminating assumptions that break on desktop, mobile, cloud, or enterprise deployments.

Your goal is to ensure `{{TARGET}}` can be **bundled, distributed, and run** on platforms where the Vrooli monorepo, lifecycle system, and shared resources are not available. Changes should make the scenario self-contained and adaptable to different runtime contexts.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve the scenario's portability.

Required reading:
- `prompt-manager skill read storage-steer visited-tracker-tools`

`storage-steer` is the storage-architecture authority — this skill picks *which* engine each tier needs; `storage-steer` decides *how* to architect it (per-domain schema, repository pattern, migration tier). The two skills share boundaries deliberately and cross-reference each other.

Optional reading:
- `prompt-manager skill read brand-manager` (draft — branding validation for deployment readiness)

---

### 0. Why This Skill Exists

Scenarios built for Tier 1 (local Vrooli stack) assume things that fail elsewhere: `VROOLI_ROOT` and `SCENARIO_ROOT` set (a desktop Go binary crashes on "env var required"), `~/.vrooli/` paths (wrong on Windows, sandboxed on mobile), PostgreSQL, Redis or an always-on Ollama present, arbitrary filesystem writes (blocked by macOS notarization and iOS), Unix path separators, `localhost` always working, fixed ports free, and Vrooli lifecycle service discovery.

| Tier | Platform | Key constraint |
|---|---|---|
| 1 | Local stack | Full Vrooli, all resources |
| 2 | Desktop (Electron) | Self-contained bundle, no external dependencies |
| 3 | Mobile (iOS/Android) | Strict sandboxing, no server processes |
| 4 | Cloud/SaaS | Container isolation, managed services |
| 5 | Enterprise/Appliance | Air-gapped, custom resource mapping |

### 1. Scope Boundaries

In scope: environment variable patterns (fallbacks, detection flags), resource dependency analysis and swap recommendations, runtime filesystem portability via `package:api-core/storage`, build configuration (CGO, static binaries, cross-compilation), network patterns (localhost variants, ports), postgres-to-SQLite portability, secret classification. Out of scope: tier-specific UI (Electron IPC, mobile native), code signing and distribution (`scenario-to-desktop`), storage architecture (`storage-steer`), performance, bundle creation (`deployment-manager`).

---

### 2. Environment Variable Portability

The lifecycle injects variables that do not exist in bundles: `VROOLI_ROOT` and `SCENARIO_ROOT` are not set, `VROOLI_DATA` becomes `APP_DATA_DIR` (Electron sets it), `API_PORT` and the like must be injected by the wrapper, and `POSTGRES_*` is unavailable (use SQLite).

Pattern: resolve the data root as `APP_DATA_DIR`, else `VROOLI_DATA`, else `~/.vrooli/data`; detect desktop with `VROOLI_DESKTOP_MODE == "true"` and read bundled assets from `BUNDLE_ROOT` there, `SCENARIO_ROOT` otherwise. Never hard-require monorepo variables (`requireEnv("VROOLI_ROOT")`). When Go code needs canonical monorepo layout, use `packages/repo-contract-go` instead of joining `VROOLI_ROOT` with `scenarios/...`.

Ports: `service.json` declares `"ports": {"api": {"env_var": "API_PORT", "port": 18700}}`; Electron `main.ts` injects them (scenario-to-desktop generates it, so you do not write it). Scenario code reads them with a sensible default (`getEnvInt("API_PORT", 18700)`), never `requireEnv("API_PORT")` and never a hardcoded `:8080`.

---

### 3. Resource Dependency Portability

Fitness by tier (>= 0.9 ship-ready, < 0.6 needs an alternative; `scenarios/deployment-manager/docs/guides/fitness-scoring.md` has the criteria): PostgreSQL 0.3 desktop, 0.0 mobile, 0.95 cloud, so use SQLite; Redis 0.5, 0.3, 0.9, so an in-memory cache; Ollama 0.0, 0.0, 0.95, so OpenRouter fallback; browser-automation-studio 0.6, 0.0, 0.9, so bundled Playwright; Neo4j 0.6, 0.0, 0.95, so SQLite plus JSON; MinIO 0.7, 0.4, 0.9, so filesystem plus cloud; Qdrant 0.5, 0.2, 0.9, so SQLite FTS.

Ollama is the special case: its value is being free and local, so do not replace it; make it optional with OpenRouter as the fallback where it is unavailable.

**SQLite.** Use the pure-Go driver `modernc.org/sqlite` (`sql.Open("sqlite", path)`, compatible with `CGO_ENABLED=0`), never `github.com/mattn/go-sqlite3`. Put the file under `<data root>/{{TARGET}}/database.db`, embedded in the scenario rather than as a standalone SQLite resource. `storage-steer` covers the architecture after the engine choice (per-domain embedded schema applied by `database.EnsureSchemas`, repository interfaces, Qdrant and Redis namespaces).

**Full replacement versus runtime swap.** Steer toward full replacement: remove the non-portable resource when the portable one covers what you actually use (simple CRUD and per-user workflow storage move to SQLite). Swap at runtime only when the non-portable resource offers something you actually need that the alternative cannot (multi-user concurrent writes in a SaaS that also ships a desktop build; advanced queries with no workaround; Ollama's free local inference). Two backends cost complexity, tests and bugs.

**Document the strategy** in `service.json` or the bundle manifest: full replacement notes the embedded SQLite path (`<APP>_SQLITE_PATH` from `${SCENARIO_DATA_DIR}`) in `notes.storage_strategy`; runtime swap keeps the resource `required: false` and records the selection logic (for example a `STORAGE_BACKEND` env var defaulting to SQLite when postgres is unavailable); an optional resource with fallback lists both under `capabilities.ai_providers` with `ai_fallback`.

---

### 4. Filesystem Portability

Adopt `github.com/vrooli/api-core/storage` (`packages/api-core/docs/storage.md`) for runtime storage instead of custom path logic: it encapsulates OS-aware roots (`auto`, `desktop`, `mobile`, `vps`), separates mutable state from disposable deploy directories, and provides traversal-safe joins and atomic writes. Pattern: `storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})`, `storage.EnsureAllDirs(resolver, storage.Options{ScenarioID: "{{TARGET}}"}, 0)`, `resolver.Path(opts, storage.ClassState, "runtime.json")`, `storage.WriteFileAtomic(path, payload, storage.DefaultFilePerm)`.

Anti-patterns: hardcoded absolute paths (`$HOME`, `os.TempDir()`, `C:\`), scenario-local mutable writes (`./data`, `./state`) under app or deploy targets, hand-rolled `DATA_DIR` or traversal checks. Tracked scenario-authored source (edited through a tool but meant as a shared reviewable change) belongs in the repo under an explicit directory such as `config/` or `policy/`, not in `package:api-core/storage`, and `.vrooli/` is for repo and manifest metadata, not a bucket for checked-in configuration.

### 5. Build Portability

Desktop bundles need static binaries: build with `CGO_ENABLED=0` for each of linux, darwin and windows (`GOOS`/`GOARCH`). Audit for blockers with `rg "mattn/go-sqlite3|// #cgo|import \"C\""` and `cd scenarios/{{TARGET}}/api && CGO_ENABLED=0 go build ./...`. For the UI, `rg "localhost:[0-9]+" ui/src` should find only configurable references, and every `process.env.NODE_ENV` check needs a fallback.

### 6. Network Portability

Accept both `http://localhost:<port>` and `http://127.0.0.1:<port>` in CORS allowed origins. Do not assume a port is free: configurable with a default, never a bare listen on `:8080`. Mark offline behavior in `service.json` `capabilities.offline_capable` and `offline_limitations` (for example "AI features require OpenRouter API").

### 7. Secret Classification

Infrastructure secrets (DB password, provider keys) are not bundleable (reference them in the manifest only). Per-install generated secrets (JWT secret, session key) are bundleable: generate on first launch (`generator: random_base64_32`). User-prompted secrets (a user's API key) are bundleable: prompt in the UI and store locally. Remote-fetch secrets (license key) are partial and need the network. The bundle manifest lists each as `{name, class, generator|prompt|note}`.

---

### 8. Audit

Before changing anything, run `rg` over `scenarios/{{TARGET}}`:
- Env: `requireEnv`, `os.Getenv("VROOLI_ROOT`, `os.Getenv("SCENARIO_ROOT`, `process.env.VROOLI_ROOT`; `os.Getenv(` lines with no fallback.
- Build: CGO imports and the static-build test above.
- Filesystem: adoption (`storage.NewResolver`, `EnsureAllDirs`, `WriteFileAtomic`) against direct writes (`os.WriteFile`, `ioutil.WriteFile`, `os.Create(`, `os.OpenFile(`) and ad hoc path policy (`DATA_DIR`, `/tmp`, `/home/`, `/Users/`).
- Network: fixed ports (`:8080`, `:3000`, `:5000`) with no env override; resources via `jq '.dependencies.resources' .vrooli/service.json`.

Red flags: `VROOLI_ROOT` without a `VROOLI_DESKTOP_MODE` check, `SCENARIO_ROOT` without a `BUNDLE_ROOT` fallback, `VROOLI_DATA` without an `APP_DATA_DIR` fallback, port variables with no default; PostgreSQL, Redis or Ollama with no portable alternative or fallback, or runtime-swap complexity where full replacement would do; `go-sqlite3` or CGO imports, or a failing `CGO_ENABLED=0` build; runtime writes that bypass `package:api-core/storage`, mutable files under deploy directories, a custom `DATA_DIR` policy; fixed ports, CORS without `127.0.0.1`; infrastructure secrets required at runtime; deployment-readiness branding gaps (display name, SVG and raster logo, multi-size favicon, a WCAG-validated color system, typography).

Record durable findings in the scenario's `docs/ARCHITECTURE.md` or `docs/internal/PROBLEMS.md` (target tiers, env variable status, resource strategy per resource with fitness, build and network status, issues with `file:line`, required changes per tier); no standalone `PORTABILITY_AUDIT.md` (a legacy one may be read as input). At session start read `service.json`, those documents, `packages/api-core/README.md` and `packages/api-core/docs/storage.md`; at session end verify claims against code, correct inaccuracies and update tier compatibility and the swaps made or still needed. Track visited files with `visited-tracker-tools` (location `scenarios/{{TARGET}}`, tag `cross-platform-readiness`).

---

### **9. Output Expectations**

You may add env fallback chains and `VROOLI_DESKTOP_MODE` handling, swap database drivers (postgres to SQLite with `modernc.org/sqlite`), add resource abstraction interfaces for swappable backends, set `offline_capable` and limitations in `service.json`, adopt `package:api-core/storage`, accept both localhost forms in CORS, and update build scripts for static builds.

You must preserve all Tier 1 functionality and testing, use `modernc.org/sqlite`, keep `CGO_ENABLED=0 go build` working, route mutable runtime files through `package:api-core/storage`, document resource swap limitations in the manifest, and record durable changes in the canonical architecture or problem document.

You must not remove support for Vrooli lifecycle environment variables, hardcode runtime storage roots, store mutable files under deploy directories, add tier-specific code without feature detection, remove a resource dependency without an alternative, or make superficial renames.
