# Implementation Plan: PII Scanner, Custom Watchlist, and File-List Scan Endpoint

## Purpose
Extend Secrets Manager with PII detection patterns, encrypted custom watchlist, path-based allowlist exemptions, and a file-list scan endpoint for Git Control Tower integration.

## Required Reading
```bash
prompt-manager skill read api-steer seam-discovery-and-enforcement cli-steer test
```

## Problem Statement
Secrets Manager currently has 8 vulnerability patterns (hardcoded secrets, SQL injection, CORS, etc.) but **zero PII detection** and **no allowlist/exemption system**. The scanning pipeline is directory-based only — there's no way to scan a specific list of files. Git Control Tower needs a file-list scan endpoint to check staged files during pre-commit review, regardless of where they live in the repo.

## Scope

### In Scope
- Add PII regex patterns (email, phone, SSN, credit card, IP, AWS keys, home directory paths) to `VulnerabilityPattern` slice
- Context-aware false-positive filtering (Go imports, build tags, go.mod/go.sum, lockfiles, URLs in comments, version strings)
- Create `scanFileList()` function accepting arbitrary file paths, with a **2s per-file timeout** and oversized-file skip
- Implement `POST /api/v1/security/scan-files` endpoint with synchronous-with-timeout behavior (10s overall budget)
- Implement `GET /api/v1/security/scan-runs/{id}` polling endpoint
- Create `pii_watchlist` table with AES-256-GCM encrypted values + CRUD endpoints
- Create `scan_allowlist_rules` table (with UNIQUE `path_pattern`) and seed rules + CRUD endpoints
- Migration `003_pii_watchlist_allowlist.sql` mirrored in `schema.sql`
- UI: `PIIWatchlistManager.tsx` and `AllowlistRulesManager.tsx`, reachable via tab switcher on the existing `SecurityTables` section
- Tests for all new functionality

### Out of Scope
- GCT integration client (separate item: `execute/gct-security-review-tab`)
- RBAC/auth for watchlist management (secrets-manager has no auth system currently)
- Encryption key rotation tooling (manual env var swap is sufficient for now)
- Performance benchmarking with large file lists (deferred)
- False-positive tuning beyond initial pattern + filter design

## Current Technical Context

### Key Files
- `scenarios/secrets-manager/api/security_scanner.go` — VulnerabilityPattern definitions (8 patterns), `scanFileForVulnerabilities()`, AST scanner
- `scenarios/secrets-manager/api/security_scan.go` — `walkAndScan()`, `persistSecurityScan()`, `loadPersistedSecurityScan()`, cache layer
- `scenarios/secrets-manager/api/security_handlers.go` — SecurityHandlers struct, HTTP handlers for scan/vulnerabilities
- `scenarios/secrets-manager/api/server.go` — Route registration in `APIServer.routes()`
- `scenarios/secrets-manager/api/file_limits.go` — Existing file-size cap reused for the oversized skip
- `scenarios/secrets-manager/api/schema.sql` — `security_scan_runs` and `security_vulnerabilities` tables
- `scenarios/secrets-manager/initialization/storage/postgres/migrations/002_scenario_overrides.sql` — Existing migration template (BEGIN;/COMMIT; wrapper, `CREATE TABLE IF NOT EXISTS`)
- `scenarios/secrets-manager/api/security_scanner_test.go` — Comprehensive test suite for scanner
- `scenarios/secrets-manager/ui/src/sections/SecurityTables.tsx` — Existing security dashboard section (host for the new tab switcher)

### Acceptance-Allow Expansion Required
The current `acceptance_allow` is `scenarios/secrets-manager/api/**` and `scenarios/secrets-manager/ui/**`, which does **not** cover the migrations folder this plan writes to (`scenarios/secrets-manager/initialization/storage/postgres/**`). The executor MUST widen `acceptance_allow` to include `scenarios/secrets-manager/initialization/storage/postgres/**` before running, otherwise the "no changes outside acceptance_allow" verification will fail. This is the first item on the Rollout/Validation Checklist.

### Existing Patterns to Reuse
- `VulnerabilityPattern` struct for PII patterns (Type, Severity, Pattern regex, Description, Title, Recommendation, CanAutoFix)
- `scanFileForVulnerabilities()` for per-file pattern matching
- `persistSecurityScan()` for scan-run persistence with fingerprint dedup
- `SecurityHandlers` struct with `db` and `logger` for new handlers
- Fingerprint format: `component_type|component_name|file_path|line_number|vulnerability_type`
- Migration 002 style: BEGIN;/COMMIT; wrapper with `CREATE TABLE IF NOT EXISTS`

## Target End State
1. Secrets Manager detects PII in any set of files passed via API
2. Users can upload personal values (email, phone, etc.) to a watchlist that flags those exact values
3. Test files/fixtures can be exempted via configurable path-glob × finding-type rules
4. GCT can call `POST /api/v1/security/scan-files` with staged file paths and get findings inline (or poll for large scans)
5. UI provides management pages for watchlist and allowlist via the SecurityTables tab switcher

## Implementation Strategy

### Phase 1: Database Schema & Migration
Create `scenarios/secrets-manager/initialization/storage/postgres/migrations/003_pii_watchlist_allowlist.sql` (BEGIN;/COMMIT; wrapper, matching the 002 style) AND mirror the same DDL in `schema.sql` with `CREATE TABLE IF NOT EXISTS`.

**`pii_watchlist`**:
- `id` UUID PK DEFAULT gen_random_uuid()
- `label` TEXT NOT NULL
- `encrypted_value` BYTEA NOT NULL (AES-256-GCM)
- `value_type` TEXT NOT NULL CHECK (value_type IN ('email','phone','path','ssn','custom'))
- `created_at` TIMESTAMPTZ NOT NULL DEFAULT NOW()

**`scan_allowlist_rules`**:
- `id` UUID PK DEFAULT gen_random_uuid()
- `path_pattern` TEXT NOT NULL **UNIQUE** (glob pattern; UNIQUE makes the seed `ON CONFLICT (path_pattern) DO NOTHING` well-formed)
- `excluded_types` TEXT[] NOT NULL (vulnerability/PII types to skip, or `{'*'}` for all)
- `description` TEXT
- `enabled` BOOLEAN NOT NULL DEFAULT true
- `created_at` TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Seed allowlist rules** (idempotent insert, `ON CONFLICT (path_pattern) DO NOTHING`):
`*_test.go`, `testdata/**`, `**/fixtures/**`, `**/vendor/**`, `go.sum`, `package-lock.json`.

### Phase 2: PII Pattern Definitions + Context-Aware Filter
Add PII regex patterns to the vulnerability patterns slice in `security_scanner.go`:

| Type | Severity | Pattern Description |
|------|----------|--------------------|
| `pii_email` | high | Email address pattern |
| `pii_phone` | high | Phone number patterns (US/international) |
| `pii_ssn` | critical | Social Security Number pattern |
| `pii_credit_card` | critical | Credit card number patterns (Visa, MC, Amex, Discover) |
| `pii_ip_address` | medium | IPv4 address pattern |
| `pii_aws_key` | critical | AWS access key ID pattern |
| `pii_home_dir` | medium | Home directory path references |

Add a context-aware filter that runs **after regex match, before emit**, suppressing matches whose line is inside any of: Go import blocks, build tags, `go.mod`/`go.sum` lines, `package-lock.json`/`yarn.lock` lines, URLs in comments, version strings (e.g. `N.N.N.N` in recognized version contexts). Each context is a named predicate so it can be unit-tested in isolation.

### Phase 3: Watchlist Encryption & CRUD
- Create `watchlist.go` with AES-256-GCM encrypt/decrypt using env var `SECRETS_MANAGER_WATCHLIST_KEY` (32-byte hex key)
- Optional-key behavior: if the key is unset, watchlist CRUD endpoints return **503**, and `scan-files` ignores the `watchlist` option and surfaces `watchlist_available: false` in the response metrics
- Watchlist CRUD handlers:
  - `POST /api/v1/security/watchlist` — create entry (encrypt value before storage)
  - `GET /api/v1/security/watchlist` — list entries (label + type only, never the decrypted value)
  - `DELETE /api/v1/security/watchlist/{id}` — delete entry
- Implement `loadWatchlistValues()` that decrypts all values for scan-time matching (case-insensitive substring via lowercased compare)

### Phase 4: Allowlist CRUD
- Create `allowlist.go` with CRUD handlers:
  - `POST /api/v1/security/allowlist-rules`
  - `GET /api/v1/security/allowlist-rules`
  - `PUT /api/v1/security/allowlist-rules/{id}`
  - `DELETE /api/v1/security/allowlist-rules/{id}`
- Implement `shouldExclude(filePath, findingType string) bool` checking all enabled rules

### Phase 5: File-List Scan Endpoint
1. Create `scanFileList()` in `security_scan.go`. For each file: enforce a **2s per-file timeout** via `context.WithTimeout`, scoped inside the 10s overall budget.
2. **Oversized-file handling**: if a file exceeds the existing `file_limits.go` cap, do NOT scan; emit a `scan_skip` finding with `snippet: "skipped: too_large"`.
3. **Per-file timeout handling**: if the per-file 2s context expires, emit a `scan_skip` finding with `snippet: "skipped: timeout"`.
4. Otherwise: check allowlist → scan with `scanFileForVulnerabilities()` → if watchlist enabled and key present, also scan with literal substring matching (case-insensitive).
5. Aggregate results with metrics; call `persistSecurityScan()`. **`scan_skip` markers are NOT persisted to `security_vulnerabilities`** — they appear in the in-memory response only.
6. `POST /api/v1/security/scan-files` handler:
   - Parse request body: `{files: [...], options: {pii, secrets, watchlist}}`
   - Start scan with 10s context timeout
   - If complete within timeout: return `{scan_id, status: "complete", findings, metrics}`
   - If timeout: return `{scan_id, status: "partial", findings (so far), metrics}`
7. `GET /api/v1/security/scan-runs/{id}` handler:
   - Load scan run + associated vulnerabilities from DB
   - Return same response shape as scan-files (without `scan_skip` markers, since those are not persisted)

### Phase 6: UI — Watchlist & Allowlist Management
Add two new sections under `ui/src/sections/`:
- `PIIWatchlistManager.tsx` — add/remove personal values with type selection
- `AllowlistRulesManager.tsx` — add/edit/remove path-glob rules with type-exclusion checkboxes

Reach them via a tab switcher added to `SecurityTables.tsx` (Vulnerabilities | Watchlist | Allowlist). No new journey scaffolding.

## Contract Decisions

### POST /api/v1/security/scan-files
**Request:**
```json
{
  "files": ["/path/to/file1.go", "/path/to/file2.ts"],
  "options": {"pii": true, "secrets": true, "watchlist": true}
}
```

**Response (complete):**
```json
{
  "scan_id": "uuid",
  "status": "complete",
  "findings": [
    {"file": "/path/to/file1.go", "line": 42, "type": "pii_email", "severity": "high", "pattern": "email address", "snippet": "user@example.com"}
  ],
  "metrics": {"files_scanned": 2, "findings_count": 1, "duration_ms": 150, "watchlist_available": true}
}
```

**Response (partial — overall timeout):** Same shape with `"status": "partial"` and findings processed so far.

### Finding Shape
**Normal finding** — persisted to `security_vulnerabilities`:
```json
{"file": "...", "line": <int>, "type": "<vuln_or_pii_type>", "severity": "critical|high|medium|low", "pattern": "<human-readable>", "snippet": "<matched text or redacted>"}
```

**`scan_skip` marker** — in-memory response only, NOT persisted:
```json
{"file": "...", "line": 0, "type": "scan_skip", "severity": "info", "pattern": "scan_skip", "snippet": "skipped: too_large" | "skipped: timeout"}
```

GCT and other consumers must treat `type=scan_skip` as a coverage gap signal, not a vulnerability. The shape (including `watchlist_available`) is the integration contract for downstream initiative items (`gct-security-review-tab`, `gct-commit-level-gating`, `gct-commit-level-agent`); preserve it across post-plan tweaks.

### GET /api/v1/security/scan-runs/{id}
Returns same response shape as scan-files (persisted findings only — no `scan_skip` markers).

### Watchlist CRUD
- `POST /api/v1/security/watchlist` — `{label, value, value_type}` → `{id, label, value_type, created_at}`
- `GET /api/v1/security/watchlist` — `[{id, label, value_type, created_at}]`
- `DELETE /api/v1/security/watchlist/{id}` — 204
- All return **503** if `SECRETS_MANAGER_WATCHLIST_KEY` is unset.

### Allowlist CRUD
- `POST /api/v1/security/allowlist-rules` — `{path_pattern, excluded_types, description, enabled}`
- `GET /api/v1/security/allowlist-rules` — list all
- `PUT /api/v1/security/allowlist-rules/{id}` — update
- `DELETE /api/v1/security/allowlist-rules/{id}` — 204

## Testing Plan
1. **PII pattern tests**: each regex against true positives and true negatives (Go import paths, version strings, lockfile lines)
2. **Context-aware filter tests**: each named predicate (imports, build tags, go.mod/go.sum, lockfiles, URLs in comments, version strings) in isolation
3. **Watchlist encryption tests**: round-trip encrypt/decrypt, missing-key 503 path, `watchlist_available:false` propagation, empty value handling
4. **Allowlist tests**: glob matching, type filtering, enabled/disabled toggle, seed-rule idempotence (ON CONFLICT)
5. **scanFileList() tests**: multi-file scan, allowlist filtering, watchlist matching, metrics accuracy, oversized-file `scan_skip`, per-file timeout `scan_skip`, non-persistence of `scan_skip` markers
6. **Endpoint integration tests**: scan-files happy path, scan-runs polling, CRUD for watchlist/allowlist
7. **Timeout behavior tests**: per-file 2s timeout emits skip and continues; overall 10s timeout returns `status: partial`

## Risks + Mitigations
| Risk | Mitigation |
|------|-----------|
| PII regex false positives (email pattern matching import paths) | Context-aware filter with named predicates per context; iterate based on real findings |
| Watchlist key not set in env | CRUD returns 503; scan-files surfaces `watchlist_available:false` rather than failing |
| Large file list causing timeout | 10s overall + 2s per-file timeout; oversized-file skip via `file_limits.go` cap |
| Allowlist glob injection | Validate glob patterns on creation; reject patterns matching everything |
| Regex backtracking on hostile input | Per-file 2s timeout bounds worst-case latency regardless of pattern complexity |
| Encryption key loss | Document that lost key = unrecoverable watchlist; recreate entries from scratch |
| acceptance_allow does not cover migrations folder | Executor MUST widen `acceptance_allow` to `scenarios/secrets-manager/initialization/storage/postgres/**` before running; first item on the Rollout/Validation Checklist |

## Non-goals/Prohibited Patterns
- **This is greenfield work.** No compatibility shims, legacy wrappers, dead code, or unused re-exports.
- Do not modify the existing 8 vulnerability patterns or change existing scan behavior
- Do not add RBAC or authentication
- Do not optimize for massive file lists (1000+) — primary use case is commit-sized batches (1-50 files)

## Rollout/Validation Checklist
- [ ] Widen `acceptance_allow` in `spec.json` to include `scenarios/secrets-manager/initialization/storage/postgres/**`
- [ ] `go build ./...` passes with no errors
- [ ] `golangci-lint run` passes on modified files
- [ ] `go test ./... -timeout 300s` passes (PII patterns, context filter, watchlist crypto, allowlist, scanFileList, endpoints, timeouts)
- [ ] `gofumpt -w .` applied
- [ ] Migration `003_pii_watchlist_allowlist.sql` applies cleanly on existing installs; `schema.sql` DDL idempotent on fresh installs
- [ ] Automated curl/HTTP test against `POST /api/v1/security/scan-files` returns valid finding + scan_skip shapes (captured in `review/captures/`)
- [ ] Automated curl/HTTP test against `GET /api/v1/security/scan-runs/{id}` returns persisted findings only
- [ ] Automated curl/HTTP test against allowlist CRUD returns 200/204 as expected
- [ ] Watchlist CRUD returns 503 when `SECRETS_MANAGER_WATCHLIST_KEY` unset; scan response carries `watchlist_available:false`
- [ ] `vrooli scenario restart secrets-manager` and `/health` endpoint returns 200
- [ ] UI tab switcher renders Watchlist and Allowlist sections without console errors

## Definition of Done
- [ ] PII regex patterns detect emails, phones, SSNs, credit cards, IPs, AWS keys, home dirs
- [ ] Context-aware filter suppresses the documented false-positive contexts
- [ ] `POST /api/v1/security/scan-files` accepts file list and returns findings (with `scan_skip` markers where applicable)
- [ ] `GET /api/v1/security/scan-runs/{id}` returns persisted scan results for polling
- [ ] Watchlist CRUD works with AES-256-GCM at rest; gracefully degrades when key absent
- [ ] Allowlist CRUD works with path-glob × finding-type filtering; seed rules present
- [ ] Allowlist rules are applied during file-list scans
- [ ] All new code has test coverage
- [ ] `go build`, `golangci-lint run`, `go test` all pass
- [ ] Scenario restarts cleanly and health check passes
- [ ] UI tab switcher with `PIIWatchlistManager.tsx` and `AllowlistRulesManager.tsx`
