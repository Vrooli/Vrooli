## Steer focus: Storage Architecture

Prioritize **engine-independent, per-domain storage architecture** in `scenarios/{{TARGET}}/`. Schema lives next to the code that interprets it. Repository interfaces hide the engine. Cross-store ownership (SQL tables, Qdrant collections, Redis namespaces) follows the same per-domain rule. Greenfield by default; brownfield substrate appears only when production data evolution demands it.

Your goal is to make the target scenario's storage layer **production-ready and engine-portable**: any single domain can be added by creating files in one folder; any single domain can be deleted by removing one folder; any engine can be swapped behind the repository interface without business-logic churn.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve the scenario's storage architecture.

Required reading:
- `prompt-manager skill read cross-platform-readiness visited-tracker-tools`

`cross-platform-readiness` is the engine-selection authority — its §3 fitness table decides *which* store to use; this skill decides *how* to architect it. The two skills share boundaries deliberately and cross-reference each other.

---

### The programmatic counterpart: the `storage-health` scenario

This skill says *what* good storage architecture is and *why*. **`storage-health` is the *how* — the engine that measures whether a scenario actually conforms, and the gate that enforces the one non-negotiable: test-isolation safety.** What used to be the hand-rolled grep cookbook in §10 is now a real, productized validator; drive it instead of eyeballing `rg` output:

- `storage-health validate scenario {{TARGET}}` — runs every static storage analyzer (schema layout, per-domain ownership, migration hygiene, persistence-seam adoption, and **the SQL + file isolation proof**) and returns a maturity assessment with `file:line`-anchored findings + remediation. This *is* the audit; §10's greps are what it automates.
- `storage-health fix preview {{TARGET}}` / `storage-health fix apply {{TARGET}}` — preview/apply the deterministic autofixes it reports (e.g. `ENSURE_SCHEMAS_NOT_WIRED`, `DB_ROWS_NOT_CLOSED`). Idempotent: a second apply over an already-fixed tree is a no-op.
- `storage-health fleet scan` · `storage-health advisor engines` · `storage-health advisor migrations` — fleet inventory (which scenarios use which engines, isolation-readiness, backup gaps), the Postgres→SQLite fitness advisor, and migration-hygiene intelligence.
- `vrooli scenario test {{TARGET}}` **storage phase** — the same engine run as a delegated test-genie phase, positioned **before playbooks**. Its L2 verdict is the fail-closed gate: when SQL or file isolation can't be statically proven (`ROUTED_SEAMS_UNWIRED`, `FILE_ROUTED_SEAMS_UNWIRED`, `STORAGE_ISOLATION_UNVERIFIED`, or `FILE_ISOLATION_UNVERIFIED`), test-genie **refuses** destructive E2E playbooks rather than risk mutating real data. This is the regression gate your changes must keep green.

> This skill's detection has **graduated into a programmatic engine** (`programmaticHome: storage-health:storage`, dimension `storage`). Treat storage-health as the source of truth for the findings; this steer is the judgment layer over them. storage-health **superseded the five retired `scenario-auditor` DB/storage rules** (`routed_database_drivers`, `routed_database_handle_capture`, `database_backoff`, `db_rows_close`, `storage_namespace_helpers`) — don't reach for those; run the validator.

---

### 0. Why This Skill Exists

Storage problems are invisible until they cause outages, data corruption, or cross-scenario pollution. The deeper problems are *architectural*:

- **Schema scattered across a central file is shotgun surgery for adds and a deletability bug for removes.** A domain whose tables live in a project-wide `schema.sql` can never be deleted by removing its folder — the table definition stays, becomes orphaned, and is recreated on every boot.
- **Tight coupling between business logic and a single engine** prevents swapping SQLite for Postgres (or vice versa) when deployment tier changes — even though both speak SQL.
- **Missing isolation** between scenarios sharing infrastructure (one Postgres instance, one Redis cluster) creates collisions: tables overwrite, keys collide, debug sessions read each other's state.
- **Brownfield-by-default migration accretion** leaves every scenario carrying dozens of `001_…sql`, `002_…sql` files no one reads, when most scenarios never needed versioned migrations in the first place.
- **Filesystem sprawl** — runtime files written to arbitrary paths or under deploy directories — causes data loss on redeploy and breaks portability across desktop, mobile, sandboxed, or container deployments.

The architecture this skill steers toward solves all five:

- **Cohesion**: every change to a domain (add a column, add a Redis key, add a Qdrant collection) lands in one folder.
- **Locality of change**: one logical change → one diff location.
- **Deletability**: removing a domain is `rm -rf internal/<dom>/` plus a few registry edits — never an archaeology dig.
- **Bounded contexts**: each domain owns its data; cross-domain coupling is explicit (soft IDs, not hard FKs).
- **Strategy-appropriate migrations**: declarative for greenfield (with optional one-shot temp scripts for personal data preservation), versioned migrations only when real users exist.

---

### 1. Scope Boundaries

**In scope**
- Per-domain schema architecture (the canonical pattern; SQL, Qdrant, Redis, anything else)
- Repository / service abstraction so business logic doesn't know the engine
- The greenfield-vs-brownfield migration strategy (declarative + one-shot temp scripts vs versioned migrations folder)
- Cross-store ownership patterns and namespacing for isolation
- Greenfield-by-default posture; brownfield exception path
- Storage architecture audit and documentation
- Cross-domain coupling guidance (soft FKs preferred)

**Out of scope**
- *Which* storage engine to pick → `cross-platform-readiness` §3 fitness table
- Filesystem runtime contract → `cross-platform-readiness` §4 + `package:api-core/storage`
- Database query optimization, indexing strategy → performance skills
- Data modeling, domain design → domain architecture skills
- Backup, disaster recovery, DBA concerns → operational skills
- ORM / driver selection details → implementation choice

---

### 2. The Canonical Pattern: Domain-Owned Storage

Every domain owns the bits of every store it touches. Not just SQL tables — also Qdrant collections, Redis key namespaces, anything else the domain wires to. The architecture pattern is the same regardless of which engine is behind it.

```
                     DOMAIN-OWNED STORAGE
                     ───────────────────────────
internal/<dom>/schema.sql   ─┐
internal/<dom>/qdrant.go    ─┼─→ Schema() / Setup() functions
internal/<dom>/redis.go     ─┘                  │
                                                ▼
                          internal/modules/registry.go
                            (AllSchemas, AllSetups)
                                                │
                                                ▼
                  api/main.go: database.EnsureSchemas(...)
                              + qdrant/redis equivalents
                                  (called at boot)
```

Why this shape:

- **Cohesion** — schema lives next to the code that interprets it; one diff per logical change.
- **Locality of change** — adding a column = one file edit; adding a domain = one folder; removing a domain = `rm -rf` that folder.
- **Deletability** — a domain's footprint is bounded by its folder; no central files to scrub.
- **Bounded contexts** — each domain owns its data, mirroring the rest of the per-domain stack (handlers, services, types).

**Worked examples** to cross-reference rather than reinventing:
- `path:templates/scenarios/react-vite/api/internal/notes/` — canonical SQL-only example (the react-vite template).
- `path:scenarios/agent-manager/api/internal/database/` — second real-world example.
- `templates/scenarios/react-vite/api/internal/database/system.sql` — the system home pattern.

**System home for cross-cutting infrastructure.** Some bits don't belong to any one domain — postgres extensions, custom types, cross-domain views. They live in a `system.sql` (e.g., `internal/database/system.sql`), empty by default in SQLite scenarios. Tested for emptiness as a tripwire so it doesn't become a dumping ground; if you find yourself adding a `CREATE TABLE` there, ask whether the table belongs to a domain that doesn't exist yet, and create the domain first.

---

### 3. Engine Selection

Engine choice is `cross-platform-readiness`'s job — its §3 fitness table scores each store across deployment tiers and converges on the right default. Don't restate the reasoning; defer to it.

Cheat-sheet for fast decisions:

| Need | Default | Why |
|---|---|---|
| Structured / relational data (most cases) | **SQLite** via `modernc.org/sqlite` | Pure-Go, CGO-free, embedded, portable across all tiers |
| Concurrent multi-writer load + managed-DB ops at scale | **PostgreSQL** | Real concurrency story; managed-DB ecosystem; still per-domain |
| Vector embeddings / semantic search | **Qdrant** alongside relational store | Specialized vector engine; hybrid pattern (Qdrant + SQLite/Postgres) |
| Ephemeral cache / session / rate-limit | **Redis** alongside relational store | TTL-native; memory-resident |
| Large blobs (images, video, documents) | **MinIO / filesystem** alongside relational store | Metadata in DB; content in object storage |

Per-domain ownership applies to all of them. The pattern doesn't care which engine you picked.

---

### 4. Per-Domain Architecture (Worked Spec)

#### 4.1 SQL schema

Each domain ships its schema next to its code:

```
internal/<dom>/
  schema.sql      # tables, indexes, triggers this domain owns
  schema.go       # //go:embed schema.sql; func Schema() string
  types.go
  repository.go   # interface
  sqlite.go       # SQLite implementation (or postgres.go)
  service.go
```

**`schema.sql`** holds only this domain's tables. Forward-only declarative — `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`. Idempotent re-runs are required: `EnsureSchemas` is called on every boot.

> **⚠ `EnsureSchemas` only ever *creates missing tables and indexes* — it never alters the columns of a table that already exists.** Do **not** put `ALTER TABLE … ADD COLUMN` in `schema.sql`. Two reasons: (1) `ADD COLUMN IF NOT EXISTS` is valid PostgreSQL but a **syntax error in SQLite** (our default engine), and a bare `ADD COLUMN` errors with "duplicate column name" on the second boot — either form crashes boot, since `EnsureSchemas` execs the file as one statement with no per-statement tolerance. (2) Adding the column only inside `CREATE TABLE IF NOT EXISTS` **silently does nothing** on a DB that already has the table. So *any* change to an existing table's columns is a migration, not a declarative edit — see §5. (`EnsureSchemas` now runs a post-apply drift check on SQLite — `PRAGMA table_info` vs the declared columns — and **fails boot loudly** if a declared column is missing, so a forgotten migration can't ship silently.)

**`schema.go`** embeds the SQL via `//go:embed` and exports `func Schema() string`. The function is the seam; the file is its substrate.

**Handler-side re-export.** `handlers/<dom>/module.go` re-exports `Schema()`:

```go
// handlers/<dom>/module.go
func Schema() string { return internalDom.Schema() }
```

This keeps the registry's import surface narrow — it imports handler packages, not their internal peers.

**Boot wiring.** The API binary calls `database.EnsureSchemas(ctx, db, modules.AllSchemas()...)` from `path:packages/api-core/database`. The `modules.AllSchemas()` registry assembles the providers (system home first, then per-domain in stable order).

**Substrate.** The `SchemaProvider` interface, `SchemaProviderFunc` adapter, and `EnsureSchemas` function live in `packages/api-core/database/schemas.go`. `EnsureSchemas` accepts a small `SchemaExecer` interface — `*sql.DB` satisfies it, but tests don't need a real driver to exercise the helper. Empty schemas (`Schema() == ""`) skip silently so the system home can ship empty without complaint.

**System home.** `internal/database/system.sql` for cross-cutting bits (postgres extensions, custom types, cross-domain views). Empty by default in SQLite scenarios. Tested for emptiness — if you add to it, the test fails and forces a deliberate "yes, this is genuinely cross-cutting" decision.

#### 4.2 Qdrant collections

Each domain that uses vectors ships its collection setup in `internal/<dom>/qdrant.go`. Collection names are scenario-and-domain-prefixed — but the prefix is **resolved at runtime through the variant-aware helper, never hardcoded as a compile-time constant**:

```go
// internal/notes/qdrant.go
func SetupQdrant(ctx context.Context, client *qdrant.Client) error {
    collection, err := storage.Collection("notes") // "<scenario>_notes" live, "<scenario>_shadow_notes" under a shadow
    if err != nil {
        return err
    }
    return client.EnsureCollection(ctx, collection, vectorSize, distance)
}
```

Resolve the collection name once where you need it (or via a small `func CollectionName() (string, error)` wrapper) — do **not** bake it into a package-level `const`. `storage.Collection(domain)` (`packages/api-core/storage/namespace.go`) reads the lifecycle-injected `VROOLI_STORAGE_NAMESPACE`, so live and a Baseline-Modes shadow address **different** collections automatically.

> **Why (shadow isolation / Baseline Modes).** A hardcoded `const CollectionName = "<scenario>_notes_embeddings"` makes a shadow instance read and write *live's* collection — corrupting the very state the engagement exists to protect. Hardcoding the scenario slug is therefore a **maturity-ladder regression**: it routes the scenario to live-only mode in the Baseline Modes decision tree (it cannot be safely shadowed) and is flagged as a finding (§4.5). Going through the helper is the difference between a scenario that can be safely self-improved and one that can't.

`SetupQdrant` registers in `modules.AllQdrantSetups()` (or equivalent) and runs at API boot, after `EnsureSchemas` for SQL. The pattern is symmetric: providers register, applied at boot, idempotent.

When a Qdrant collection is genuinely shared across multiple domains (rare — a single embeddings collection that several domains read but no one owns), it's a system-home case. Put the setup in `internal/database/system.go` and apply the same tripwire reasoning: ask whether it actually belongs to a domain that doesn't exist yet.

#### 4.3 Redis namespacing

Each domain that uses Redis owns its key construction in `internal/<dom>/redis.go` — built through the variant-aware helper, **not** a hardcoded prefix constant:

```go
// internal/auth/redis.go
func SessionKey(id string) (string, error) {
    return storage.RedisKey("auth", "session", id) // "<scenario>:auth:session:<id>"
}

func LockKey(resource string) (string, error) {
    return storage.RedisKey("auth", "lock", resource)
}
```

Two helpers (`packages/api-core/storage/namespace.go`):
- `storage.RedisKey(domain, segments...)` — the full key, segments joined with `:`. Use this for real keys; it reproduces a mid-string dynamic token like `<scenario>:idea:<id>:research` that a flat prefix **cannot** (`RedisKey("idea", id, "research")`).
- `storage.RedisPrefix(domain)` — a terminated prefix (`<scenario>:<domain>:`) for `SCAN`/`KEYS` patterns where the full key isn't known.

Both resolve the `<scenario>[_<variant>]` root from the lifecycle-injected `VROOLI_STORAGE_NAMESPACE`. Standard key-pattern shapes (the leading `{scenario}` is the helper-supplied root — live `lpbs`, shadow `lpbs_shadow`):

| Pattern | `RedisKey(...)` call | Live example |
|---|---|---|
| `{scenario}:{domain}:session:{id}` | `RedisKey("auth", "session", id)` | `lpbs:auth:session:abc123` |
| `{scenario}:{domain}:cache:{entity}:{id}` | `RedisKey("notes", "cache", "note", id)` | `lpbs:notes:cache:note:42` |
| `{scenario}:{domain}:rate:{resource}:{id}` | `RedisKey("downloads", "rate", "api", userID)` | `lpbs:downloads:rate:api:user-7` |
| `{scenario}:{domain}:lock:{resource}` | `RedisKey("migrations", "lock", "run")` | `lpbs:migrations:lock:run` |

The scenario prefix prevents cross-scenario collisions on a shared Redis; the domain prefix prevents cross-domain collisions within one scenario; and the **variant** dimension (folded into the root by the helper) prevents a shadow from colliding with live.

> **Why (shadow isolation / Baseline Modes).** A hardcoded `const SessionPrefix = "<scenario>:auth:session:"` has no variant dimension, so a shadow instance writes into live's keyspace. Like Qdrant, this is a maturity-ladder regression that routes the scenario to live-only mode and is flagged as a finding (§4.5). Always build keys through `RedisKey`/`RedisPrefix`.

#### 4.4 The pattern as a rule, not a recipe

**Whatever store the domain touches, the domain owns the setup.** If a domain wires to a new store tomorrow (Neo4j, ClickHouse, MinIO), the new file goes in `path:internal/<dom>/`, gets a `Setup()` function, gets registered in the modules registry, gets applied at boot. Same shape, every time.

The rule's purpose is to make the architecture *uniform* so agents (and humans) don't have to invent a new pattern per engine.

#### 4.5 The variant-aware namespace rule (Redis + Qdrant)

**Never hardcode a Redis key prefix or Qdrant collection name that embeds the scenario slug.** Compose them through `storage.Collection` / `storage.RedisKey` / `storage.RedisPrefix` so the variant is resolved at runtime from the lifecycle-injected environment. This is what lets the scenario be developed and validated as a shadow while its live version keeps serving (Baseline Modes), and it is the maturity-ladder floor for every store except the variant-isolated-by-default ones (Postgres/SQLite/filesystem, which the lifecycle env scopes directly — for SQLite/data paths use `storage.ScenarioNamespace(slug)` as the `Options.ScenarioID`). The api-core helper docstrings state this contract; the scenario-auditor `storage-namespace-v1` standard (rule `storage_namespace_helpers`, surfaced through test-genie's standards battery as a `FINDING_SOURCE_STANDARDS` finding) flags any hardcoded Redis/Qdrant namespace so the EM maturity loop migrates the long tail and the Baseline-Modes decision tree can route an un-adopted writer to live mode.

---

### 5. Migration Strategy: Greenfield vs Brownfield

There are exactly two strategies. The dividing line is **whether the scenario has real users in production** — not whether the dev database has data, not whether you're about to drop a column. Greenfield covers everything before real users; brownfield begins the moment they exist.

#### Greenfield — `Schema()` only, with optional one-shot scripts for personal data

The default. The repo never gains a `migrations/` folder. `internal/<dom>/schema.sql` describes the desired clean state at all times.

**Use when:** scenario hasn't shipped to real users yet. Includes new scenarios, scenarios with only your own dev data, scenarios you've been using personally for weeks. Comfort evolving the schema is the whole point — you're allowed to change anything.

**The dividing line is the *granularity* of the change, not whether it's additive vs destructive:**

- **A whole new table or new index** → free. Add the `CREATE TABLE IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS` to `internal/<dom>/schema.sql`; the next boot's `EnsureSchemas` creates it. Done.
- **Any change to the columns of an *existing* table — add, drop, rename, or retype, no exceptions** → **always a one-shot migration script.** `EnsureSchemas` cannot do this (it only creates *missing* tables/indexes; `CREATE TABLE IF NOT EXISTS` is a no-op on a table that already exists, so a new column added there silently never lands). Two steps, every time: **(1)** edit `internal/<dom>/schema.sql` to describe the desired final shape (so a fresh DB gets it via `CREATE TABLE`), **and (2)** write a one-shot migration script that brings an *existing* DB to that shape.

> **Always migrate; never recreate.** Do **not** decide "this data is disposable, I'll just delete the DB." Agents are bad at that judgment and the cost of a wrong guess is destroyed data. Treat every existing-table column change as if the data must be preserved — write the migration. If the DB happened to be a rebuildable cache, the migration is harmlessly redundant; that's a cheap price for never having to make the call. (`EnsureSchemas` enforces this: on SQLite it drift-checks declared columns against the live table after applying and **fails boot loudly** if one is missing, naming the table and column — so a skipped migration surfaces immediately at boot instead of as a mysterious query failure later.)

The one-shot script lives in `/tmp/<scenario-slug>/migrate-<short-descriptor>.<ext>`:

```
/tmp/<scenario-slug>/migrate-add-status-column.sql
/tmp/<scenario-slug>/migrate-rename-users-table.go
/tmp/<scenario-slug>/migrate-split-orders-into-line-items.sh
```

Extension matches the script type:
- `.sql` for pure data shuffles you can express in SQL (`UPDATE … SET new_col = …`, `INSERT INTO new_table SELECT … FROM old_table`).
- `.go` for transforms needing logic (parsing fields, calling external APIs, conditional rewrites).
- `.sh` for orchestration (sqlite3/psql calls, file moves, sequenced steps).

For multi-session work where `/tmp` clears on reboot, use `~/.cache/vrooli/<scenario-slug>/` instead. Either way, the script is **personal scratch** — never committed, never tracked, discarded after the data lands cleanly.

**Pattern:**
1. Update `internal/<dom>/schema.sql` to describe the new desired state.
2. Write the migration script in the temp location. **Make it safe to re-run** — guard the change (e.g. check `PRAGMA table_info(<table>)` before `ALTER TABLE … ADD COLUMN`, since a bare re-run errors with "duplicate column name"), or treat it as strictly run-once-then-delete. Idempotence removes a footgun if the script is accidentally run twice.
3. Stop the scenario; run the script against the local DB; restart the scenario.
4. Boot's `EnsureSchemas` re-applies `schema.sql` and runs its drift check — if it passes, the existing DB now matches the declared shape and the script is no longer needed; delete it. If it still **fails boot** naming a missing column, the migration didn't cover that change — fix the script and re-run.

**Why this exists:** so you can comfortably evolve the database after starting to use a scenario locally — without polluting the codebase with version-history files that no one will read. The script migrates the existing local data into the new shape; the repo only ever describes the clean state. (When real users exist, this graduates to the versioned-migrations folder below — same idea, durable and ordered.)

#### Under a Baseline Modes engagement — the managed migration folder

The `/tmp` hand-run pattern above is for *ad-hoc personal* evolution. When a **Baseline Modes** shadow engagement is active (the safe shadow-and-promote development path — `git-control-tower baseline start`), schema migrations are not hand-run scratch: they live in the engagement's **managed per-baseline migration folder** (ordered), and the **promote machinery applies them to live**, not you. The contract:

- **Location:** the managed folder beside the engagement's restore point (not `/tmp`), so it survives and is discoverable by promote — never deleted by hand.
- **Transactional + re-runnable:** each script wraps in a transaction and is idempotent (safe to apply twice), because promote dry-runs them against a fresh copy of current live first and bounces on failure rather than half-applying.
- **Applied by promote, ordered:** the runner applies the folder's scripts in order during `baseline promote`, inside the quiesce window.
- **Shape-unchanged fast path:** no scripts authored **and** the schema fingerprint matches ⇒ promote skips all DB handling and does a code-swap + restart only; live data is untouched.

This is distinct from the one-off adoption rename below, which is a single platform-evolution event, not a per-engagement schema change.

#### One-off: adopting the variant-aware storage helpers (§4.5)

Switching an existing scenario's Redis/Qdrant namespaces from hardcoded slug constants to the `storage.Collection`/`RedisKey`/`RedisPrefix` helpers is a **rename**, and existing keys/collections use inconsistent separators (`scenario-backlog` vs `workflow_embeddings`). The shipped helper and scenario code stay greenfield (no back-compat shim); the actual reindex/rename of *existing* data is a **throwaway one-off script** under `/tmp/<scenario>/migrate-*` (the greenfield pattern above), run once with the scenario stopped, **after `data-backup-manager safety backup-now --scenario <s>` takes a pre-migration snapshot** as the safety net. If a specific reindex is non-obvious (e.g. a raw-HTTP Qdrant writer), escalate for guidance rather than guessing. Do not put adoption-rename logic in committed code.

#### Brownfield — versioned migrations folder

Real users exist. Their data is the source of truth. Schema evolution must be auditable, ordered, transactional, and applied exactly once per environment.

**Use when:** scenario has shipped to real users with persisted data. Once you cross this line, you don't go back — every schema change from this point is a versioned migration.

**Layout:**

```
internal/notes/
  migrations.go        # Migrations() embed.FS; DomainID() = "notes"
  migrations/
    001_initial.sql    # was schema.sql, renumbered as the baseline
    002_add_status.sql
    003_drop_archived_at.sql
```

The per-domain `schema.sql` is replaced by `migrations/001_initial.sql` (the baseline) the moment the scenario crosses to brownfield. From then on, every change is a new file.

**Conventions:**
- **Numbering:** prefer timestamps (e.g., `20260503T1400_add_status.sql`) to avoid merge collisions when two contributors add migration #002 the same week. Strict numeric ordering is fine for solo work.
- **Direction:** up-only by default. Down migrations are usually misleading — recovery is from backup, not reverse-migration. Don't ship them unless you have a real rollback story.
- **Granularity:** each file is one logical change. Easier to read, easier to revert by writing a corrective migration.
- **Transactions:** wrap each migration in a transaction so partial-apply doesn't corrupt state.
- **Per-domain version space:** `schema_migrations_notes` is separate from `schema_migrations_users`. Two domains can land migration #002 the same week without collision.

**The substrate for brownfield migrations is deferred — see §6.** Don't invent ad-hoc tooling.

#### Signal: which strategy am I in?

The skill *consumes* this signal from upstream — it doesn't decide it. Sources:

- **Explicit user instruction** ("this scenario hasn't shipped yet"; "we have 200 users on this").
- **Production deployment evidence** — running scenario with users, public URL, anything that says real people depend on the data.
- **`PROGRESS.md` or `STORAGE_AUDIT.md` reference** documenting the data state.

If unclear, **ask the user**. Misclassifying greenfield as brownfield burdens the codebase with unneeded migrations forever; misclassifying brownfield as greenfield risks user data on the next schema change.

---

### 6. The Deferred `MigrationProvider` Substrate

The brownfield substrate doesn't ship in `package:api-core/database` yet. Documented here so agents know what to ask for when they need it — not promising vapor; naming the seam.

**Today (the greenfield substrate, already shipped):**
- `packages/api-core/database/schemas.go` — `SchemaProvider` interface, `SchemaProviderFunc` adapter, `EnsureSchemas(ctx, db, providers...)`.

**When the first scenario crosses to brownfield** (real users with persisted data + ongoing schema evolution), build:
- `packages/api-core/database/migrations.go` — `MigrationProvider` interface (`Migrations() fs.FS`, `DomainID() string`), and a `RunMigrations(ctx, db, providers...)` helper.

**Implementation choice when we build it:** **wrap `package:pressly/goose`.** It is the de-facto Go migration library — lightweight, embeddable via `go:embed`, programmatic Go API (not just CLI), no external service dependency. Don't reinvent the migration runner; do design the per-domain integration on top of it.

**Action when an agent hits this need before the substrate exists:**
1. **Stop.** Do not implement ad-hoc migration tooling inline in the scenario.
2. **Escalate to the user.** Describe the change needed (which column, which transform).
3. **Request the substrate be built** as a separate plan.
4. The two-step (substrate first, then use it) is faster overall than a one-off scenario migration that drifts from every other scenario's approach.

This is a hard rule. One ad-hoc migration tool per scenario is exactly what `MigrationProvider` exists to prevent.

---

### 7. Engine Boundaries: Cross-Domain Coupling

**Prefer soft FKs across domain boundaries.** Store the ID, no `REFERENCES` constraint:

```sql
-- internal/orders/schema.sql
CREATE TABLE IF NOT EXISTS orders (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,           -- soft FK to users.id; no constraint
    -- ...
);
```

**Within-domain FKs are fine and encouraged** — they enforce the domain's own invariants:

```sql
-- internal/orders/schema.sql
CREATE TABLE IF NOT EXISTS order_items (
    id        TEXT PRIMARY KEY,
    order_id  TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    -- ...
);
```

**Why:** hard FKs across domains create a coupling smell — you can't delete the users domain without breaking orders' constraint, can't bootstrap orders' schema before users', can't even run `EnsureSchemas` providers in the wrong order without an error. If you genuinely need referential integrity across domains, that's signal that the two should merge into one or that a third domain should own both.

The system home (`internal/database/system.sql`) is *not* the answer. It absorbs cross-cutting infrastructure (extensions, types) — not domain coupling.

---

### 8. Repository Pattern (Engine-Neutral)

Business logic depends on repository interfaces. Concrete implementations (SQLite, Postgres, in-memory for tests) live in the same `path:internal/<dom>/` folder. Mirror the canonical layout:

```go
// internal/notes/repository.go — the interface
type Repository interface {
    Create(ctx context.Context, n Note) error
    Get(ctx context.Context, id string) (Note, error)
    List(ctx context.Context) ([]Note, error)
}

// internal/notes/sqlite.go — the production implementation
type SqliteRepository struct {
    db *sql.DB
}

func (r *SqliteRepository) Get(ctx context.Context, id string) (Note, error) {
    // SQLite-specific query
}

// internal/notes/mocks/repository.go — the test fake (co-located)
type FakeRepository struct {
    notes map[string]Note
}

// internal/notes/service.go — business logic depends on the interface
type Service struct {
    repo Repository  // any implementation
}
```

**Service consumers depend on `Service`, not on `Repository` directly.** Handlers, CLI commands, and other domains call `Service` methods. The repository is an implementation detail of the domain.

**Engine swap is free.** Want to add Postgres support later? Add `internal/notes/postgres.go` implementing `Repository`; switch the constructor in `module.go`. No business-logic edits.

**Test fakes are co-located** in `path:internal/<dom>/mocks/` — sub-package of the domain so deletion takes them with it. (See the Pass-3 worked example in `path:templates/scenarios/react-vite/api/internal/notes/mocks/`.)

---

### 9. Isolation

Three isolation concerns, each handled by its respective per-domain pattern from §4.

- **Database isolation.** SQLite scenarios get their own DB file under `package:api-core/storage`-resolved paths (cross-reference `cross-platform-readiness` §4 for the filesystem contract). Postgres scenarios use a scenario-named schema (e.g., `SET search_path TO {{TARGET}}`) declared in `service.json`'s resource block. Either way, two scenarios sharing the same DB instance never collide.
- **Redis isolation.** Per-domain key prefix `{scenario}:{domain}:` (see §4.3). The scenario prefix isolates scenarios; the domain prefix isolates domains within one scenario.
- **Qdrant isolation.** Collection names prefixed `{scenario}_{domain}_{purpose}` (see §4.2). Same two-level isolation.

If a scenario is the only consumer of its DB / Redis / Qdrant, the isolation is automatic. Apply the prefix anyway — the scenario will eventually share infrastructure with another, and the cost of fixing it later is more than the cost of doing it right now.

---

### 10. Storage Architecture Audit

Before making changes, assess `{{TARGET}}`'s current storage posture against the canonical pattern.

#### 10.1 Run the validator (don't hand-roll greps)

`storage-health` automates the entire audit — schema layout, per-domain ownership, SQL and file isolation proof, persistence hygiene, namespace adoption, migration hygiene. Run it first, read its `file:line` findings, apply the autofixes it offers, and only fall back to manual inspection for things outside its analyzer set.

```bash
# The full static storage audit, with file:line findings + remediation.
storage-health validate scenario {{TARGET}}

# Preview / apply the deterministic autofixes it reports (idempotent).
storage-health fix preview {{TARGET}}
storage-health fix apply {{TARGET}}

# Cross-scenario triage, engine-fitness advisor, migration-hygiene intelligence.
storage-health fleet scan
storage-health advisor engines
storage-health advisor migrations
```

The validator's findings map directly onto the red-flags below — it is the *mechanism* that detects them; §10.2 is the human-readable account of *what* it (and you) are looking for and *why* each matters. Reach for a manual `rg` only when you suspect something the analyzers don't yet cover, and consider filing a capability gap (`swarm-manager captures create`) so storage-health learns to catch it.

#### 10.2 Red-Flags Checklist

- [ ] No per-domain `internal/<dom>/schema.sql` files (schema is centralized somewhere)
- [ ] `path:internal/store/` or `path:internal/db/` package present with schema content (deprecated naming)
- [ ] `initialization/storage/postgres/schema.sql` exists in `.vrooli/` (resource-applied path)
- [ ] Schema files are not idempotent (missing `IF NOT EXISTS`)
- [ ] Direct SQL in handler / controller code (no repository abstraction)
- [ ] Hard cross-domain FKs (constraint-based, not soft)
- [ ] Brownfield substrate (versioned migrations folder) used in a greenfield scenario (no real users)
- [ ] Greenfield substrate used in a brownfield scenario (destructive schema change on production user data without versioned migration)
- [ ] Redis keys without scenario+domain prefix
- [ ] Qdrant collections without scenario prefix
- [ ] Hardcoded credentials anywhere
- [ ] Filesystem runtime writes bypass `package:api-core/storage` (also a `cross-platform-readiness` flag)

#### 10.3 Posture for Non-Conforming Scenarios

If `{{TARGET}}` doesn't match the canonical pattern, **strongly recommend refactoring to one of the documented patterns** for maintainability and extensibility. Surface findings in `docs/internal/STORAGE_AUDIT.md` (template in §11). Do **not** pre-emptively rewrite — refactoring is a separate decision, made per scenario, with the user.

The skill describes ideals; it does not narrate older shapes or document migration steps from them. If an agent picks up a real legacy scenario and needs migration help, escalate to the user — the answer comes from human judgment, not from the skill.

---

### 11. Document Findings

Record audit results in `scenarios/{{TARGET}}/docs/internal/STORAGE_AUDIT.md`:

```markdown
# {{TARGET}} Storage Architecture Audit

## Last Updated
[Date]

## Current Pattern
- [ ] Per-domain schema files (canonical)
- [ ] Centralized schema (refactor recommended)
- [ ] Resource-applied schema (refactor recommended)

## Migration Strategy
- [ ] Greenfield (no real users) — `Schema()` only; one-shot `/tmp/<scenario>/migrate-*.{sql,go,sh}` scripts when local data preservation needed
- [ ] Brownfield (real users with persisted data) — versioned migrations folder per domain (substrate deferred)
- Current data state: [empty / dev-only / personal use / shipped to N users]

## Architecture Status
- [ ] All domains own their SQL schema (`internal/<dom>/schema.sql` + `Schema()`)
- [ ] Repository interfaces per domain
- [ ] Business logic uses interfaces, not direct DB
- [ ] System home (`internal/database/system.sql`) is empty or contains only cross-cutting bits

## Engine Status
- [ ] SQLite via `modernc.org/sqlite` (CGO-free)  /  PostgreSQL  /  hybrid
- [ ] Qdrant collections per-domain (if used)
- [ ] Redis keys per-domain prefixed (if used)
- [ ] Filesystem writes routed through `package:api-core/storage`

## Issues Found
1. [File:line] - Issue description
2. ...

## Refactor Recommendations (if non-conforming)
1. [Highest-impact change] — Why
2. ...

## Cross-References
- `storage-health validate scenario {{TARGET}}` → the programmatic audit that produced these findings
- `cross-platform-readiness` → engine selection, filesystem contract
- `packages/api-core/database/schemas.go` → substrate
- `path:templates/scenarios/react-vite/api/internal/notes/` → canonical worked example
```

---

### 12. Memory Management with Visited Tracker

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `storage-architecture`.

---

### 13. Documentation and Memory Loop

#### 13.1 At Session Start

Read existing storage documentation:
- `scenarios/{{TARGET}}/.vrooli/service.json` — resource declarations, scenario isolation config
- `scenarios/{{TARGET}}/docs/internal/STORAGE_AUDIT.md` — prior audit findings (if exists)
- `packages/api-core/database/schemas.go` — current substrate surface
- `packages/api-core/docs/storage.md` — filesystem storage contract (cross-platform-readiness's territory)

#### 13.2 At Session End

Update `scenarios/{{TARGET}}/docs/internal/STORAGE_AUDIT.md`:
- Code is the source of truth — verify existing claims against actual code.
- Correct any inaccuracies discovered.
- Update the migration-strategy classification if the data state changed (greenfield ↔ brownfield).
- Add new architectural findings.
- Update refactor recommendations based on work completed.
- Note areas not yet audited.
- Create the `path:docs/internal/` directory if needed.

---

### **14. Output Expectations**

You may update in `scenarios/{{TARGET}}/`:
- Add per-domain `schema.sql` + `schema.go` files
- Add per-domain `qdrant.go` / `redis.go` files for non-SQL stores
- Refactor centralized schema into per-domain layout when scope permits
- Add or refine repository interfaces and implementations
- Add the modules registry (`internal/modules/registry.go`) if missing
- Namespace Redis keys and Qdrant collections per scenario+domain
- Surface findings and recommendations in `STORAGE_AUDIT.md`

You must:
- Keep `{{TARGET}}` fully functional and non-regressed
- Make all schema files idempotent (`IF NOT EXISTS` patterns)
- Place schema next to the code that interprets it (per-domain ownership)
- Pick the right migration strategy based on whether the scenario has real users (greenfield) or not (brownfield)
- Use environment variables for connection details (no hardcoded credentials)
- Route mutable filesystem state through `package:api-core/storage`
- Escalate to the user if the scenario needs brownfield versioned migrations (the substrate is deferred)

You must NOT:
- Invent ad-hoc versioned-migration tooling inline in a scenario (escalate instead)
- Centralize schema across domains (`internal/store/schema.sql`, `initialization/storage/postgres/schema.sql`, etc.)
- Pre-emptively refactor a non-conforming scenario without the user's approval
- Write SQL directly in handler / controller code
- Hardcode database credentials or connection strings
- Add hard cross-domain FKs (use soft FKs — store the ID, no constraint)
- Skip scenario+domain prefixes on Redis keys or Qdrant collections

**Avoid superficial changes that rename variables or restructure code without materially improving storage architecture.**
