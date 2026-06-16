# Configuration

## Lifecycle Configuration

Tidiness Manager should be started through Vrooli lifecycle or the scenario Makefile. Lifecycle assigns ports and injects resource configuration.

```bash
cd scenarios/tidiness-manager
make start
make status
```

## Environment

| Variable | Purpose |
| --- | --- |
| `API_PORT` | API port assigned by lifecycle |
| `UI_PORT` | UI port assigned by lifecycle |
| `DATABASE_URL` | PostgreSQL connection string |
| `POSTGRES_*` | PostgreSQL components used when `DATABASE_URL` is absent |
| `REDIS_URL` | Optional cache resource |
| `CLAUDE_CODE_CLI` | Optional smart-scan AI CLI |

## Scenario Contracts

- `.vrooli/testing.json` declares strict lint handlers consumed by Quality Health and Test Genie.
- `.vrooli/lighthouse.json` declares UI performance/accessibility targets.
- `Makefile` exposes lifecycle and quality gates.
- `api/.golangci.yml` and `cli/.golangci.yml` define Go lint behavior for this scenario.
- `ui/tsconfig.json` and `ui/eslint.config.js` satisfy Quality Health static-quality contracts.

## Scan Parameters

Scan requests support timeout and target parameters. Smart scans support explicit file lists, campaign ID, and force-rescan. Campaigns support session and file-per-session limits.

## Boundary

Configuration that enforces lint/type/static-quality policy should stay aligned with Quality Health. Configuration for maintainability thresholds, campaigns, and scan behavior belongs to Tidiness Manager.
