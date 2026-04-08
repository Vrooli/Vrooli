# Research Conclusion: Design PII detection system for Secrets Manager

## Research Question
How should the PII detection subsystem be designed for Secrets Manager — including custom watchlist support, allowlist exemptions, and a file-list-based scan endpoint — so that Git Control Tower can invoke it during pre-commit security review?

## Summary
Secrets Manager has a mature security scanning infrastructure (8 vulnerability patterns, regex + AST-based scanning, progressive scan with caching, PostgreSQL persistence with fingerprint deduplication) but **zero PII-specific detection** and **no allowlist/exemption system**. The existing `VulnerabilityPattern` struct, `scanFileForVulnerabilities()` function, and `security_scan_runs`/`security_vulnerabilities` persistence layer provide a solid foundation that can be extended without architectural changes. The design uses a **scan-run model** (persisted results with scan ID) for GCT integration, **built-in PII patterns in Go source** with a **DB-stored custom watchlist** (AES-256-GCM encrypted at rest), a **two-dimensional allowlist** (path-glob × finding-type exemptions), **synchronous-with-timeout** endpoint behavior, and **tiered PII severity** classification.

## Methodology
- Read all secrets-manager source code: API handlers, scanner, persistence, schema, tests, types
- Analyzed the `walkAndScan()` pipeline end-to-end: directory traversal → pattern matching → fingerprinting → DB persistence
- Examined all GCT review client implementations: tidiness-manager, test-genie, scenario-auditor
- Analyzed GCT capability registry, service discovery, and health-check patterns
- Reviewed initiative orchestration summary for strategic decisions
- Compared scan-run patterns across existing GCT integrations (auditor uses jobID polling)

## Findings

### Finding 1: Current scanning is directory-based only
The core scanning function `walkAndScan()` (`api/security_scan.go:84-196`) traverses directories with extension filtering, file-size limits, and per-component caps. There is no mechanism to scan a specific list of files. The scan endpoint (`GET /api/v1/security/scan`) accepts `?component` and `?severity` filters but not file paths. A new `scanFileList()` function is needed that iterates caller-provided paths, applies allowlist filtering, then delegates to the existing `scanFileForVulnerabilities()` for pattern matching.

### Finding 2: No PII patterns exist but the pattern system is easily extensible
The 8 vulnerability patterns in `api/security_scanner.go:26-100` cover hardcoded secrets, SQL injection, CORS, debug code, etc. — nothing for PII. The `VulnerabilityPattern` struct (lines 102-110) has fields for `Type`, `Severity`, `Pattern` (regex), `Description`, `Title`, `Recommendation`, and `CanAutoFix`. Adding PII regex patterns (email, phone, SSN, credit card, IP address, AWS key) requires only new entries in the patterns slice — no engine changes.

### Finding 3: No allowlist/exemption system exists
Vulnerabilities can be marked `accepted` via status updates (`POST /api/v1/security/vulnerabilities/{id}/status`), but this is per-finding post-detection. There are no path-pattern exemptions, no per-pattern suppressions, and no time-bound exemptions. Test files are scanned identically to production code.

### Finding 4: GCT review integration uses scenario paths, not file lists — but scan-run model aligns with auditor
Current GCT review clients pass scenario names or paths (`TidinessLightScanRequest.ScenarioPath`, `TestExecutionRequest.ScenarioName`). A file-list endpoint is a **new contract pattern** for GCT. However, the scenario-auditor already uses a scan-run ID pattern (`AuditorCheckJobResponse.JobID` with polling via `GET /api/v1/standards/check/jobs/{jobID}`), so the scan-run model chosen for secrets-manager aligns with existing GCT patterns. The capability registry and health-check discovery would work identically — only the request/response contract differs.

### Finding 5: Infrastructure is modernization-ready, not modernization-required
The codebase uses Go modules, structured logging, PostgreSQL with proper schema, gorilla/mux routing, and has a comprehensive test framework. Adding PII detection requires **extending** the existing system, not rewriting it.

### Finding 6: Watchlist requires a two-tier pattern system
PII detection has two layers: (1) generic regex patterns (emails, phones, SSNs, IPs, AWS keys) built into Go source alongside existing vulnerability patterns, and (2) a user-uploaded custom watchlist of personal literal values stored in a DB table with CRUD API. Custom watchlist values are literal strings (e.g., "matt@example.com", "/home/matthalloran8") — they need exact or substring matching, not regex compilation.

### Finding 7: Existing persistence layer is reusable for file-list scans
The `security_scan_runs` and `security_vulnerabilities` tables already support the scan-run model. `persistSecurityScan()` (security_scan.go:608-674) handles INSERT with ON CONFLICT for fingerprint deduplication and status transitions (open → in_progress → resolved/accepted/regressed). The file-list scan endpoint can reuse this persistence layer entirely — no new scan/vulnerability tables needed.

### Finding 8: Watchlist values require application-level encryption
Custom watchlist entries contain the exact values users want to protect (their actual email, phone, home directory path). Storing these in plaintext makes the database a PII target. Since values must be decryptable for literal-string matching (not hashed), application-level AES-256-GCM encryption with a server-managed key (from environment variable) is the chosen approach. This protects data at rest, supports key rotation, requires no PostgreSQL extensions, and uses standard Go `crypto/aes`. The small performance cost of decrypting N watchlist values per scan is acceptable.

### Finding 9: Synchronous-with-timeout endpoint behavior
The `POST /api/v1/security/scan-files` endpoint blocks and returns full results inline for typical commits (1-50 files, expected under 2s). If a scan exceeds a 10-second timeout, it returns partial results along with the scan-run ID so GCT can poll `GET /api/v1/security/scan-runs/{id}` for the remainder. The scan run is persisted regardless (for history/dedup), giving GCT results in a single call for ~95% of cases while gracefully handling large commits.

### Finding 10: Tiered PII severity classification
Built-in PII patterns use tiered severity reflecting regulatory risk:
- **Critical**: SSN, credit card numbers (PCI, GDPR legal consequences)
- **High**: Email addresses, phone numbers (sensitive personal data)
- **Medium**: IP addresses, home directory paths (lower-impact exposure)
- **High**: Custom watchlist values (user explicitly flagged these for detection)

This aligns with the existing vulnerability pattern severity range (critical to low) and gives GCT meaningful severity data for commit gating decisions.

### Finding 11: Proposed database schema for watchlist and allowlist
Two new tables following existing schema conventions:
- **`pii_watchlist`** — id (UUID PK), label (TEXT), encrypted_value (BYTEA), value_type (TEXT: email/phone/path/custom), created_at (TIMESTAMPTZ). Stores user-provided personal values with AES-256-GCM encryption.
- **`scan_allowlist_rules`** — id (UUID PK), path_pattern (TEXT glob), excluded_types (TEXT[] — vulnerability/PII types to skip, or '*' for all), description (TEXT), enabled (BOOLEAN DEFAULT true), created_at (TIMESTAMPTZ). Implements the two-dimensional exemption model (path × finding-type).

## Limitations
- Have not benchmarked pattern matching performance with large file lists (hundreds of staged files)
- Have not evaluated false-positive rates for generic PII patterns (email regex matching Go import paths, IP addresses in config files, etc.)
- Custom watchlist UX design is deferred to the execute item
- RBAC implications of watchlist management not explored (currently no auth system in secrets-manager)
- Encryption key management details (rotation procedure, backup strategy, key loss recovery) not fully designed — deferred to implementation

## Actions

### Action 1: Create backlog item — Implement PII scanner extension for Secrets Manager
- **Kind**: execute (already exists as `execute/secrets-manager-pii-scanner` in initiative)
- **Description**: Add PII detection patterns to `security_scanner.go`, implement `scanFileList()` function, create `POST /api/v1/security/scan-files` endpoint with scan-run persistence, add `pii_watchlist` and `scan_allowlist_rules` tables with migrations, implement CRUD endpoints for watchlist and allowlist management. Follow the existing `VulnerabilityPattern` struct and `persistSecurityScan()` patterns.
- **Key design decisions to carry forward**:
  - Scan-run model with persistence (round 1, d1 → option B)
  - Built-in patterns in Go source + DB-stored watchlist (round 1, d2 → option A)
  - Path-glob × finding-type allowlist (round 1, d3 → option B)
  - Synchronous-with-timeout endpoint: block and return inline, fall back to polling at 10s (round 2, d1 → option A)
  - AES-256-GCM encryption for watchlist values with env-var key (round 2, d2 → option A)
  - Tiered PII severity: SSN/CC=critical, email/phone=high, IP/path=medium, custom watchlist=high (round 2, d3 → option A)

### Action 2: Update document — Add API contract specification to execute item
- **File**: The `execute/secrets-manager-pii-scanner` plan should include the full API contract for the scan-files endpoint so GCT can implement the client:
  - `POST /api/v1/security/scan-files` — request: `{files: [...paths], options: {pii: bool, secrets: bool, watchlist: bool}}`, response: `{scan_id: string, status: "complete"|"partial", findings: [{file, line, type, severity, pattern, snippet}], metrics: {files_scanned, findings_count, duration_ms}}`
  - `GET /api/v1/security/scan-runs/{id}` — polling endpoint for async/partial results
  - CRUD for `/api/v1/security/watchlist` (encrypted storage)
  - CRUD for `/api/v1/security/allowlist-rules` (path-glob × finding-type)
