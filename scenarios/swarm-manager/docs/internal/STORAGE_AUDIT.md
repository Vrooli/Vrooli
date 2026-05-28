# Storage Audit — swarm-manager

Authoritative map of where swarm-manager's runtime data lives after the storage
migration (see `docs/plans/swarm-manager-storage-migration-plan.md`). Update this
file whenever a domain's storage class or base changes.

## Principle

Scenario runtime/domain data MUST NOT live under the scenario source tree
(`docs/scenarios/storage.md`). All mutable data resolves through the
`github.com/vrooli/api-core/storage` seam, wrapped by
`api/internal/runtimepaths` (`DataPath` / `CachePath` / `StatePath`). The
user-profile default for that seam is the operator runtime home (`~/.vrooli`);
the on-disk layout is `~/.vrooli/<class>/vrooli/swarm-manager/...`.

The scenario source root (`scenarios/swarm-manager/`) keeps only **code**
(`api/`, `cli/`, `ui/`, `bas/`), **shipped defaults** (`config/`,
`initialization/`, `requirements/`, `docs/`), and **metadata** (`.vrooli/`). A
fresh checkout carries zero domain data; the running scenario recreates and
serves it from storage (locked by `TestFreshCheckout_ServesDomainDataFromStorage`).

## Domain → class → location

| Domain | Class | Resolved base | Owning store | Backed up? |
|---|---|---|---|---|
| Backlog items (`ideas`, `execute`, `fix`, `chore`, `research`) | data | `DataPath("")` → `~/.vrooli/data/vrooli/swarm-manager/<kind>/` | `internal/backlog` (`FileStore.rootDir` = data base) | yes |
| Initiatives | data | `DataPath("initiatives")` | `internal/initiatives` | yes |
| Agent sessions | data | `DataPath("agent-sessions")` | `internal/agentsessions` | yes |
| Captures | **cache** | `CachePath("captures")` | `internal/captures`, `internal/graph` capture adapter | **no** (disposable) |
| Event log DB (`events.db`) | data | `DataPath(...)` | event log | yes (via data base) |
| Queue / agent-activities / execution-runs / circuit-breaker | state | `StatePath(...)` | `internal/queue`, `internal/agentactivity`, `internal/execution` | n/a (transient) |
| Review decision records (`<kind>/<name>/review/decisions/`) | data | co-located under the backlog item (data base) | `internal/backlog` retry/review-decide | yes (with item) |

## Source-root concerns (NOT data — stay on the scenario root)

These legitimately address the source tree / repo and are passed the
`scenarioRoot` (or its parent `scenarios/` dir), never a data base:

- `config/settings.json` — shipped default settings (`registerSettingsRoutes`).
- Glob validation (`internal/backlog/validate_globs.go`) — resolves the **repo
  root** from `scenarioRoot` to count matching repo files.
- Sibling-scenario existence checks (`internal/execution` preflight via
  `ServiceConfig.ScenariosDir = filepath.Dir(scenarioRoot)`).
- Scenario source manifests (`internal/sessioncontext` `resolveScenario` via
  `scenariosDir`).

The `backlog.Handler` carries both a `dataDir` (data base) and a `scenarioRoot`
(repo source) to keep these distinct; `execution.Service` carries both `RootDir`
(data base for item folders) and `ScenariosDir` (repo source for sibling checks).

## Backup

`internal/backup.EnsureBackupTargets` self-registers one filesystem backup target
(`owner=swarm-manager`, `name=domain-data`, `kind=filesystem`,
`locator=DataPath("")`) with `data-backup-manager` at boot (idempotent,
best-effort, non-fatal). Captures are excluded (cache class). Destinations and
plans are operator configuration; a verified-restore gate must pass before any
git removal of the formerly-tracked domain folders.
