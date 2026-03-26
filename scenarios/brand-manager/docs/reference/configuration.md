# Configuration

Brand Manager exposes a small, intentional set of tunable levers. All levers have sane defaults that work out of the box. Override via environment variables when you need to adapt to different workloads or environments.

## Control Surface Overview

| Group | Lever | Env Var | Default | Impact |
|-------|-------|---------|---------|--------|
| Database | SQLite path | `BM_SQLITE_PATH` | `~/.vrooli/brand-manager/brand-manager.db` | Where brand data lives |
| Database | Busy timeout | `BM_BUSY_TIMEOUT_MS` | `10000` (10s) | Higher = more tolerant of contention |
| Database | Cache size | `BM_CACHE_SIZE_KB` | `2000` (~2 MB) | Higher = faster reads, more memory |
| API | Default list limit | `BM_DEFAULT_LIST_LIMIT` | `100` | Items returned when no `?limit=` |
| API | Max list limit | `BM_MAX_LIST_LIMIT` | `1000` | Absolute ceiling for `?limit=` |
| Contrast | AA normal threshold | `BM_CONTRAST_AA_NORMAL` | `4.5` | WCAG AA for normal text |
| Contrast | AA large threshold | `BM_CONTRAST_AA_LARGE` | `3.0` | WCAG AA for large text |
| Contrast | Display precision | `BM_CONTRAST_PRECISION` | `2` | Decimal places in ratio output |

## Database Configuration

### SQLite Path

The database path is resolved in priority order:

1. `BM_SQLITE_PATH` (primary, scenario-specific)
2. `SQLITE_PATH` (shared fallback)
3. `SQLITE_DB` (shared fallback)
4. Default: `~/.vrooli/brand-manager/brand-manager.db`

See [CODE: api/config/config.go#Load] for the resolution chain.

### SQLite Pragmas

The database connection applies these pragmas for performance (see [CODE: api/config/config.go#DSN]):

| Pragma | Value | Tunable? | Purpose |
|--------|-------|----------|---------|
| `foreign_keys` | ON | No | Enforce referential integrity (non-negotiable) |
| `journal_mode` | WAL | No | Write-ahead logging for concurrent reads |
| `busy_timeout` | `BM_BUSY_TIMEOUT_MS` (default 10000ms) | **Yes** | Wait before returning SQLITE_BUSY |
| `cache_size` | `-BM_CACHE_SIZE_KB` (default -2000, ~2 MB) | **Yes** | In-memory page cache |
| `synchronous` | NORMAL | No | Balance durability and speed |
| `temp_store` | MEMORY | No | Keep temp tables in RAM |

**Why `busy_timeout` is tunable:** Under high write contention (many concurrent API requests), increasing this avoids SQLITE_BUSY errors. Under low load, the default 10s is generous.

**Why `cache_size` is tunable:** Memory-constrained environments (e.g., small VPS) may want to reduce this; high-traffic deployments benefit from more cache.

**Why `journal_mode`, `synchronous`, and `foreign_keys` are NOT tunable:** These affect data integrity and correctness. Changing them could cause data loss or constraint violations.

### Connection Limits

`MaxOpenConns` is fixed at 1. SQLite's single-writer architecture means additional connections add overhead without benefit. This is intentionally not exposed.

## API Configuration

### List Pagination

| Lever | Env Var | Default | Min | Max | Impact |
|-------|---------|---------|-----|-----|--------|
| Default list limit | `BM_DEFAULT_LIST_LIMIT` | 100 | 1 | — | Applied when caller omits `?limit=` |
| Max list limit | `BM_MAX_LIST_LIMIT` | 1000 | ≥ default | — | Caps `?limit=` to prevent unbounded queries |

When a caller provides `?limit=5000` and `BM_MAX_LIST_LIMIT=1000`, the response silently caps at 1000 items.

### API Version

The version reported at `/health` is set at build time (`1.0.0`). It is not currently tunable via environment — change it in `config.Default()` when releasing a new version.

## WCAG Contrast Configuration

| Lever | Env Var | Default | Min | Max | Use Case |
|-------|---------|---------|-----|-----|----------|
| AA normal text | `BM_CONTRAST_AA_NORMAL` | 4.5 | 1.0 | — | Override to 7.0 for AAA testing |
| AA large text | `BM_CONTRAST_AA_LARGE` | 3.0 | 1.0 | — | Override to 4.5 for AAA testing |
| Display precision | `BM_CONTRAST_PRECISION` | 2 | 0 | 6 | More decimals = more precise ratios |

**Why these are tunable:** WCAG AA (4.5/3.0) are the spec defaults. Some organizations require AAA compliance (7.0/4.5). The threshold levers let operators enforce stricter or custom standards without code changes.

**Guardrails:** Thresholds below 1.0 are clamped to 1.0 (the mathematical minimum for a contrast ratio). Precision above 6 is capped to prevent floating-point noise.

## UI Constants

UI tunable values live in [CODE: ui/src/config/constants.ts]:

| Constant | Default | Purpose |
|----------|---------|---------|
| `HEALTH_CHECK_RETRY` | 2 | How many times to retry a failed health check |
| `HEALTH_CHECK_INTERVAL_MS` | 30,000 (30s) | Polling interval for API health |
| `WCAG_AA_NORMAL` | 4.5 | Display threshold for contrast badges |
| `WCAG_AA_LARGE` | 3.0 | Display threshold for contrast badges |
| `DEFAULT_PAGE_SIZE` | 20 | Brands per page (future pagination) |

These are compile-time constants. To change them, edit the file and rebuild the UI.

## CLI Configuration

The CLI resolves the API base URL from multiple sources:

1. `--api-base` flag
2. `API_BASE_URL` / `VITE_API_BASE_URL` environment variables
3. Vrooli port detection (`vrooli scenario port brand-manager API_PORT`)

See [CODE: cli/app.go#NewApp] for the full resolution chain.

## Service Configuration

The service lifecycle is configured via [CODE: .vrooli/service.json]:

- **Ports**: API, UI, and WebSocket ports allocated from Vrooli's port ranges
- **Health checks**: HTTP checks on `/health` for both API and UI
- **Resources**: SQLite (embedded, no daemon required)
- **Lifecycle**: Setup builds Go API and React UI; develop starts both servers

## What Is Intentionally NOT Configurable

| Setting | Value | Reason |
|---------|-------|--------|
| SQLite `journal_mode` | WAL | Changing from WAL risks data corruption under concurrent access |
| SQLite `foreign_keys` | ON | Disabling would allow orphaned records |
| SQLite `synchronous` | NORMAL | FULL is too slow; OFF risks data loss on crash |
| SQLite `MaxOpenConns` | 1 | SQLite single-writer; more connections add overhead, not throughput |
| API version string | `1.0.0` | Compile-time constant; versioned with releases |
| Initial brand version | 1 | Semantic: first version is always 1 |
| WCAG color pairing set | 5 pairings | Defined by brand color semantics (text/primary/accent on background/surface) |
