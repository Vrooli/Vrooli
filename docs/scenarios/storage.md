# Scenario Storage

This page defines the canonical storage policy for scenarios at the platform level.

It exists to make the target architecture explicit rather than leaving it implied across implementation examples, audits, and prompt-manager skills.

## Current Rule

Scenario runtime state must not be stored under the scenario source tree.

In particular:

- do not treat `path:scenarios/<name>/` as a runtime data root
- do not add new mutable runtime state under `path:scenarios/<name>/data/`
- do not rely on repo-local `../data/...` paths for scenario runtime state

Scenario source trees are deployable inputs. Mutable runtime state belongs outside the repo.

## Three File Categories

Scenario files should be classified into three categories:

- repo metadata
  - structural files that define scenario identity or repo-contract-facing layout
  - example: `path:scenarios/<name>/.vrooli/service.json`
- tracked scenario-authored assets
  - files intentionally edited by humans or scenario UIs and committed to git as shared defaults, policies, plans, or authored content
  - examples: `config/`, `policy/`, `initiatives/`, `ideas/`, `research/`, `requirements/`, `docs/`
- runtime mutable state
  - operational data created or changed while the scenario runs and not intended to be shared through git
  - examples: queues, runs, checkpoints, lockfiles, telemetry, local databases, caches

If a file is edited through the UI but the intended result is a shared, reviewable change to scenario behavior, treat it as a tracked scenario-authored asset, not runtime state.

## Canonical Runtime Storage Contract

For mutable filesystem state, scenarios should use:

- `github.com/vrooli/api-core/storage`

This is the canonical runtime storage seam for scenarios.

It provides:

- profile-aware storage roots
- classed storage directories
- safe relative path resolution
- atomic file write helpers
- a cross-platform path policy that keeps mutable state out of source trees

## Storage Classes

Scenarios should use the `package:api-core/storage` classes intentionally:

- `config`
  - durable operator/user-managed configuration
- `data`
  - primary mutable application data
- `cache`
  - rebuildable artifacts safe to evict
- `logs`
  - diagnostics and operational logs
- `state`
  - checkpoints, locks, transient runtime state

At runtime this resolves to class-scoped directories like:

- `<config-root>/vrooli/<scenario>/...`
- `<data-root>/vrooli/<scenario>/...`
- `<cache-root>/vrooli/<scenario>/...`
- `<logs-root>/vrooli/<scenario>/...`
- `<state-root>/vrooli/<scenario>/...`

See [packages/api-core/docs/storage.md](/home/matthalloran8/Vrooli/packages/api-core/docs/storage.md) for the package-level contract.

Tracked source-tree `config/` or `policy/` files are different from `package:api-core/storage` `config` class files:

- source-tree `config/` or `policy/`
  - versioned scenario defaults or policy authored in git
- storage `config` class
  - local operator or user-managed mutable configuration outside the repo

## Tracked Scenario Assets

Not all mutable-looking files are runtime state.

Some files are intentionally edited through tooling or scenario UIs and are meant to be committed to git as shared defaults, policies, or authored content. These are tracked scenario-authored assets.

These belong in explicit scenario directories such as:

- `config/`
- `policy/`
- `initiatives/`
- `ideas/`
- `research/`
- `requirements/`
- `docs/`

Do not place these in `.vrooli/` unless they are true repo or manifest metadata.

Do not place these in `package:api-core/storage` unless they are local runtime state rather than shared source.

## Structured Persistence

Filesystem runtime storage is only one part of scenario storage.

Scenarios should also follow these rules:

- resource-backed persistence is declared in `path:scenarios/<name>/.vrooli/service.json`
- every filesystem class a scenario writes (`config`, `data`, `cache`, `logs`, or `state`) has an explicit `storage.entries` declaration, including an empty declaration when it writes none
- every storage entry states `regenerable: true` or `regenerable: false`; omission is a schema error
- every regenerable entry carries a workload-derived `budget.max_age` or `budget.max_bytes`; do not copy the current measured size into the ceiling
- schema and seed assets live under `api/internal/<domain>/`
- scenarios should prefer resource-injected environment variables instead of hard-coded connection details for **networked resources**; the embedded SQLite path is the exception and is resolved from the scenario id (see [SQLite: A Scenario Names Itself](#sqlite-a-scenario-names-itself))
- scenario-private database/file layout details should be documented in scenario-local docs when they matter

Typical examples:

- PostgreSQL schema init in `api/internal/<domain>/storage/postgres/schema.sql`
- scenario dependency declaration in `.vrooli/service.json`
- SQLite database resolved through `storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "<name>"})` rather than from an environment variable

## SQLite: A Scenario Names Itself

A scenario opens its SQLite database by naming **itself**. It does not read a
path from the environment.

```go
dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "notification-hub"})
```

or, when the connection goes through `package:api-core/database`, one field
replaces the whole step:

```go
db, err := database.Open(ctx, database.Config{
    Driver:   database.DriverSQLite,
    Scenario: "notification-hub",
})
```

The resolved path is a pure function of the scenario's own slug and its
variant-aware namespace. There is no per-scenario path input left for another
process to supply.

### Why the environment is not an input

Scenarios used to resolve their path from `SQLITE_PATH`, falling back to
`SQLITE_DB`, **above** their own identity. A variable that does not identify a
scenario cannot be scoped to one, so any process that exported it redirected the
database of every scenario it went on to start. The supervisor scenario declared
`SQLITE_PATH` in its manifest and restarted sick scenarios by exec'ing the CLI;
each restarted child inherited the value and opened the supervisor's file.

Twelve scenarios ended up sharing one 9.35 GB database behind a single writer
lock, including Test Genie's run ledger and the plan corpus. Every layer was
individually reasonable — a manifest declaring where its data lives, a
supervisor restarting sick scenarios, a scenario honouring an explicit override
— and no scenario's own tests could see it, because the defect lives in process
environment inheritance rather than in any code path a test exercises.

### What the environment may still supply

Two variables are read, and both are safe for the same reason: they are
**scenario-agnostic**. Each names a namespace or a tree, not one scenario's
file, so every scenario beneath them still resolves to its own separate path.
Inheriting one isolates; it cannot collide.

| Variable | Effect | Why it is safe to inherit |
| --- | --- | --- |
| `VROOLI_STORAGE_NAMESPACE` | Scopes storage to `<scenario>` or `<scenario>_<variant>` | Carries the variant, not a path; the seam additionally rejects a root that names a different scenario |
| `VROOLI_STORAGE_ROOT` | Redirects every storage class root under one tree | Names a tree; each scenario still resolves its own path within it |

`VROOLI_STORAGE_ROOT` is how a test harness leases a scenario isolated storage,
and it outranks the lifecycle-assigned data directory precisely so a scenario
under test never runs against production data.

### Explicit paths

A test, migration, or diagnostic that needs to name a file passes it as a
function **argument**:

```go
dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
```

An override that lives in the environment is an override that gets inherited; an
argument cannot leak into a child process. For reading a database this scenario
does not own — a census, a backup audit — use `storage.SQLiteReadOnlyDSNAt`,
which cannot write, take a lock, or create the file.

### Tuning

The canonical pragma set lives in exactly one place. A scenario with a genuine
reason to differ states it in typed form rather than assembling its own string:

```go
storage.SQLiteTuning{PageSizeBytes: 4096, MMapSizeBytes: 268435456}
```

Before consolidation, sixty-plus scenarios each carried a private copy of the
pragma string and they had drifted into four implementations — including one
still written in the pre-`_pragma` grammar that the driver ignores, so that
scenario had never actually run in WAL mode despite a comment saying it did.

### Enforcement

`storage-manager validate` runs the `isolation.database-path-from-environment`
analyzer over every scenario. It fails a generic database-path read and a
hand-rolled SQLite DSN, and a fleet-wide test asserts the whole repository stays
clean. A scenario-scoped variable (`BAS_SQLITE_PATH`, `PLAYBOOKS_SQLITE_DSN`) is
not flagged: its name carries its owner, so it cannot silently capture a
sibling.

## Skills And Authority

The prompt-manager skills are useful implementation steers, not the canonical policy layer.

Relevant skills include:

- `storage-steer`
- `cross-platform-readiness`

Those skills already steer agents toward:

- declaring storage dependencies in `service.json`
- using environment-driven configuration for **resource** connection details (Postgres, Redis, Qdrant), which the lifecycle scopes per variant
- using `package:api-core/storage` for mutable filesystem state, including the SQLite database path, which is resolved from the scenario id rather than from the environment
- treating deploy directories as disposable

That guidance is aligned with this document, but this page is the canonical cross-scenario documentation layer.

## Anti-Patterns

These are legacy or non-target patterns for scenarios:

- mutable writes to `./data`, `./state`, or similar scenario-local folders
- mutable writes to `../data/...`
- hard-coded absolute paths such as `$HOME/...` or `/tmp/...` for durable state
- hand-rolled `DATA_DIR` traversal logic when `package:api-core/storage` is available
- reading a database path from a **generic** environment variable (`SQLITE_PATH`, `SQLITE_DB`, or a `file:` `DATABASE_URL`), which lets any process that exports it redirect a sibling scenario's database
- assembling a SQLite DSN by hand instead of going through `package:api-core/storage`
- storing real runtime state under bundle/deploy/app target directories

## Relationship To The Repo Contract

The repo contract defines canonical source-tree layout such as:

- `path:scenarios/<name>/.vrooli/service.json`
- `path:scenarios/<name>/api`
- `path:scenarios/<name>/ui`
- `path:scenarios/<name>/api/internal/<domain>/schema.sql`

It does **not** define scenario-private runtime data layout.

That separation is intentional:

- repo contract: future-state source layout and shared structural semantics
- this document: runtime storage policy for scenarios

## Transitional Reality

Some scenarios still contain repo-local runtime storage patterns today.

Those should be treated as migration targets, not architecture authority.

When migrating a scenario:

- move mutable filesystem state to `package:api-core/storage`
- keep source-tree assets under `api/internal/<domain>/`, `docs/`, `requirements/`, or other canonical source paths
- add or update `docs/internal/STORAGE_AUDIT.md` when storage behavior is important enough to audit explicitly

## Top-Level `data/`

The top-level repo `data/` folder is legacy/transitional from the perspective of scenario runtime storage.

New scenario work should not depend on it.
