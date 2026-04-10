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
- Create `scanFileList()` function accepting arbitrary file paths
- Implement `POST /api/v1/security/scan-files` endpoint with synchronous-with-timeout behavior
- Implement `GET /api/v1/security/scan-runs/{id}` polling endpoint
- Create `pii_watchlist` table with AES-256-GCM encrypted values + CRUD endpoints
- Create `scan_allowlist_rules` table with path-glob × finding-type exemptions + CRUD endpoints
- UI pages for watchlist management and allowlist configuration
- Tests for all new functionality

### Out of Scope
- GCT integration client (separate item: `execute/gct-security-review-tab`)
- RBAC/auth for watchlist management (secrets-manager has no auth system currently)
- Encryption key rotation tooling (manual env var swap is sufficient for now)
- Performance benchmarking with large file lists (deferred)
- False-positive tuning beyond initial pattern design

## Current Technical Context

### Key Files
- `scenarios/secrets-manager/api/security_scanner.go` — VulnerabilityPattern definitions (8 patterns), `scanFileForVulnerabilities()`, AST scanner
- `scenarios/secrets-manager/api/security_scan.go` — `walkAndScan()`, `persistSecurityScan()`, `loadPersistedSecurityScan()`, cache layer
- `scenarios/secrets-manager/api/security_handlers.go` — SecurityHandlers struct, HTTP handlers for scan/vulnerabilities
- `scenarios/secrets-manager/api/server.go` — Route registration in `APIServer.routes()`
- `scenarios/secrets-manager/api/schema.sql` — `security_scan_runs` and `security_vulnerabilities` tables
- `scenarios/secrets-manager/api/security_scanner_test.go` — Comprehensive test suite for scanner

### Existing Patterns to Reuse
- `VulnerabilityPattern` struct for PII patterns (Type, Severity, Pattern regex, Description, Title, Recommendation, CanAutoFix)
- `scanFileForVulnerabilities()` for per-file pattern matching
- `persistSecurityScan()` for scan-run persistence with fingerprint dedup
- `SecurityHandlers` struct with `db` and `logger` for new handlers
- Fingerprint format: `component_type|component_name|file_path|line_number|vulnerability_type`

## Target End State
1. Secrets Manager detects PII in any set of files passed via API
2. Users can upload personal values (email, phone, etc.) to a watchlist that flags those exact values
3. Test files/fixtures can be exempted via configurable path-glob × finding-type rules
4. GCT can call `POST /api/v1/security/scan-files` with staged file paths and get findings inline (or poll for large scans)
5. UI provides management pages for watchlist and allowlist

## Implementation Strategy

### Phase 1: Database Schema & Migration
Add two new tables to `schema.sql`:

**`pii_watchlist`**:
- `id` UUID PK DEFAULT gen_random_uuid()
- `label` TEXT NOT NULL (human-readable name, e.g. "My email")
- `encrypted_value` BYTEA NOT NULL (AES-256-GCM encrypted)
- `value_type` TEXT NOT NULL CHECK (value_type IN ('email','phone','path','ssn','custom'))
- `created_at` TIMESTAMPTZ NOT NULL DEFAULT NOW()

**`scan_allowlist_rules`**:
- `id` UUID PK DEFAULT gen_random_uuid()
- `path_pattern` TEXT NOT NULL (glob pattern, e.g. `*_test.go`, `testdata/**`)
- `excluded_types` TEXT[] NOT NULL (vulnerability/PII types to skip, or `{'*'}` for all)
- `description` TEXT
- `enabled` BOOLEAN NOT NULL DEFAULT true
- `created_at` TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Phase 2: PII Pattern Definitions
Add PII regex patterns to the vulnerability patterns slice in `security_scanner.go`:

| Type | Severity | Pattern Description |
|------|----------|-------------------|
| `pii_email` | high | Email address pattern |
| `pii_phone` | high | Phone number patterns (US/international) |
| `pii_ssn` | critical | Social Security Number pattern |
| `pii_credit_card` | critical | Credit card number patterns (Visa, MC, Amex, Discover) |
| `pii_ip_address` | medium | IPv4 address pattern |
| `pii_aws_key` | critical | AWS access key ID pattern |
| `pii_home_dir` | medium | Home directory path references |

Each pattern follows the existing `VulnerabilityPattern` struct — no engine changes needed.

### Phase 3: Watchlist Encryption & CRUD
- Create `watchlist.go` with AES-256-GCM encrypt/decrypt using env var `SECRETS_MANAGER_WATCHLIST_KEY` (32-byte hex key)
- Implement watchlist CRUD handlers:
  - `POST /api/v1/security/watchlist` — create entry (encrypt value before storage)
  - `GET /api/v1/security/watchlist` — list entries (return label + type only, never decrypted value)
  - `DELETE /api/v1/security/watchlist/{id}` — delete entry
- Implement `loadWatchlistValues()` that decrypts all values for scan-time matching

### Phase 4: Allowlist CRUD
- Create `allowlist.go` with CRUD handlers:
  - `POST /api/v1/security/allowlist-rules` — create rule
  - `GET /api/v1/security/allowlist-rules` — list rules
  - `PUT /api/v1/security/allowlist-rules/{id}` — update rule
  - `DELETE /api/v1/security/allowlist-rules/{id}` — delete rule
- Implement `shouldExclude(filePath, findingType string) bool` that checks all enabled rules

### Phase 5: File-List Scan Endpoint
- Create `scanFileList()` in `security_scan.go`:
  1. Accept `[]string` file paths + scan options (pii, secrets, watchlist bools)
  2. For each file: check allowlist → scan with `scanFileForVulnerabilities()` → if watchlist enabled, also scan with literal substring matching
  3. Aggregate results with metrics
  4. Call `persistSecurityScan()` for persistence
- Create `POST /api/v1/security/scan-files` handler:
  - Parse request body: `{files: [...], options: {pii, secrets, watchlist}}`
  - Start scan with 10s context timeout
  - If complete within timeout: return `{scan_id, status: "complete", findings, metrics}`
  - If timeout: return `{scan_id, status: "partial", findings (so far), metrics}`
- Create `GET /api/v1/security/scan-runs/{id}` handler:
  - Load scan run + associated vulnerabilities from DB
  - Return same response shape as scan-files

### Phase 6: UI — Watchlist & Allowlist Management
- Watchlist management page: add/remove personal values with type selection
- Allowlist management page: add/edit/remove path-glob rules with type exclusion checkboxes
- Integration into existing security dashboard navigation

### Final: Cleanup & Verification
- Run `go build ./...` and fix ALL errors, even pre-existing
- Run `golangci-lint run` and fix ALL warnings in modified files
- Run `go test ./... -timeout 300s` and fix any failures
- Format with `gofumpt -w .`
- `vrooli scenario restart secrets-manager`
- Verify health: `curl -s http://localhost:<port>/health`

## Contract Decisions

### POST /api/v1/security/scan-files
**Request:**
```json
{
  "files": ["/path/to/file1.go", "/path/to/file2.ts"],
  "options": {
    "pii": true,
    "secrets": true,
    "watchlist": true
  }
}
```

**Response (complete):**
```json
{
  "scan_id": "uuid",
  "status": "complete",
  "findings": [
    {
      "file": "/path/to/file1.go",
      "line": 42,
      "type": "pii_email",
      "severity": "high",
      "pattern": "email address",
      "snippet": "user@example.com"
    }
  ],
  "metrics": {
    "files_scanned": 2,
    "findings_count": 1,
    "duration_ms": 150
  }
}
```

**Response (partial — timeout):**
Same shape with `"status": "partial"` and findings processed so far.

### GET /api/v1/security/scan-runs/{id}
Returns same response shape as scan-files, with current status.

### Watchlist CRUD
- `POST /api/v1/security/watchlist` — `{label, value, value_type}` → `{id, label, value_type, created_at}`
- `GET /api/v1/security/watchlist` — `[{id, label, value_type, created_at}]` (no decrypted values)
- `DELETE /api/v1/security/watchlist/{id}` — 204

### Allowlist CRUD
- `POST /api/v1/security/allowlist-rules` — `{path_pattern, excluded_types, description, enabled}`
- `GET /api/v1/security/allowlist-rules` — list all
- `PUT /api/v1/security/allowlist-rules/{id}` — update
- `DELETE /api/v1/security/allowlist-rules/{id}` — 204

## Testing Plan
1. **PII pattern tests**: Test each regex pattern against true positives (real PII) and true negatives (similar-looking non-PII like Go import paths, test fixture values)
2. **Watchlist encryption tests**: Round-trip encrypt/decrypt, bad key handling, empty value handling
3. **Allowlist tests**: Glob matching for various patterns, type filtering, enabled/disabled toggle
4. **scanFileList() tests**: Multi-file scan, allowlist filtering, watchlist matching, metrics accuracy
5. **Endpoint integration tests**: scan-files with various options, scan-runs polling, CRUD for watchlist/allowlist
6. **Timeout behavior test**: Verify partial response on scan exceeding timeout

## Risks + Mitigations
| Risk | Mitigation |
|------|-----------|
| PII regex false positives (email pattern matching import paths) | Context-aware filtering: skip patterns inside import blocks, string literals in test files; tune patterns iteratively |
| Watchlist key not set in env | Fail gracefully with clear error on startup; watchlist scan option returns error if key missing |
| Large file list causing timeout | 10s timeout with partial results + polling; per-file timeout to prevent single large file blocking |
| Allowlist glob injection | Validate glob patterns on creation; reject patterns that could match everything |

## Non-goals / Prohibited Patterns
- **This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, or unused re-exports.
- Do not modify the existing 8 vulnerability patterns or change existing scan behavior
- Do not add RBAC or authentication (out of scope for this scenario currently)
- Do not optimize for massive file lists (1000+) — the primary use case is commit-sized batches (1-50 files)

## Definition of Done
- [ ] PII regex patterns detect emails, phones, SSNs, credit cards, IPs, AWS keys, home dirs
- [ ] `POST /api/v1/security/scan-files` accepts file list and returns findings
- [ ] `GET /api/v1/security/scan-runs/{id}` returns scan results for polling
- [ ] Watchlist CRUD works with AES-256-GCM encryption at rest
- [ ] Allowlist CRUD works with path-glob × finding-type filtering
- [ ] Allowlist rules are applied during file-list scans
- [ ] All new code has test coverage
- [ ] `go build`, `golangci-lint run`, `go test` all pass
- [ ] Scenario restarts cleanly and health check passes
- [ ] UI pages for watchlist and allowlist management
