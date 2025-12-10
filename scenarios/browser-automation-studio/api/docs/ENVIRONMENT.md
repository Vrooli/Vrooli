## Database backends

- Postgres (default): set `DATABASE_URL` or `POSTGRES_HOST/PORT/USER/PASSWORD/DB`. Leave `BAS_DB_BACKEND` unset or `postgres`.
- SQLite (desktop-friendly):
  - `BAS_DB_BACKEND=sqlite`
  - Optional path: `BAS_SQLITE_PATH=/abs/path/to/bas.db` or `DATABASE_URL=file:/abs/path/to/bas.db`
  - Defaults to `~/.vrooli/data/sqlite/databases/browser-automation-studio.db` and applies resource-style pragmas (WAL, busy_timeout, cache_size, page_size, synchronous=NORMAL, temp_store=MEMORY, mmap).
  - Best-effort runs `resource-sqlite manage install` to prep directories. Safe to ignore if the CLI isn’t present.
  - Skip sqlite smoke tests with `BAS_SKIP_SQLITE_TESTS=true` (they’re opt-in and light).
  - Go tests pick the backend from `BAS_TEST_BACKEND` when set, otherwise from `BAS_DB_BACKEND` (default postgres). Set `BAS_TEST_BACKEND=sqlite` to force SQLite and skip docker in CI/local runs (uses a temp DB and dialect-aware resets).

Other relevant envs:
- `BAS_SKIP_DEMO_SEED=true` to skip demo data during tests/CI.
- `SQLITE_DATABASE_PATH` to override the sqlite resource default root (falls back to `VROOLI_DATA` or `~/.vrooli/data/sqlite/databases`).
