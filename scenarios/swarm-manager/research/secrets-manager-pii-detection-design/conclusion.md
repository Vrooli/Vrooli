# Research Conclusion: Design PII detection system for Secrets Manager

## Research Question
How should the PII detection subsystem be designed for Secrets Manager — including custom watchlist support, allowlist exemptions, and a file-list-based scan endpoint — so that Git Control Tower can invoke it during pre-commit security review?

## Summary
Secrets Manager has a mature security scanning infrastructure (8 vulnerability patterns, AST-based Go scanning, progressive scan with caching, PostgreSQL persistence) but **zero PII-specific detection** and **no allowlist/exemption system**. The scanning engine is directory-based (`walkAndScan()`), whereas GCT needs a file-list endpoint for staged-file scanning. The existing `VulnerabilityPattern` struct and fingerprinting system provide a solid foundation — PII detection can be layered on without major architectural changes. The new file-list scan endpoint would be a new integration pattern for GCT (current review services accept scenario paths, not file lists).

## Methodology
- Read all secrets-manager source code: API (Go), schema (PostgreSQL), service configuration, CLI
- Analyzed current scanning pipeline: `walkAndScan()` → pattern regex + AST → fingerprint → persist
- Examined GCT review integration pattern: `ReviewClient` interfaces, capability discovery, `executeReviewRun`
- Reviewed initiative orchestration summary for strategic decisions already made
- Compared with existing GCT review integrations (tidiness-manager, test-genie, scenario-auditor)

## Findings

### Finding 1: Current scanning is directory-based only
The core scanning function `walkAndScan()` (`api/security_scan.go:86-196`) traverses directories with extension filtering and file-size limits. There is no mechanism to scan a specific list of files. The scan endpoint (`GET /api/v1/security/scan`) accepts `?component` and `?severity` filters but not file paths.

### Finding 2: No PII patterns exist
The 8 vulnerability patterns in `api/security_scanner.go:26-100` cover hardcoded secrets, SQL injection, CORS, debug code, etc. — but nothing for PII (emails, phone numbers, SSNs, credit cards, IP addresses, AWS keys). The `VulnerabilityPattern` struct (`api/security_scanner.go:102-110`) has fields for `Type`, `Severity`, `Pattern`, `Description`, `Title`, `Recommendation`, and `CanAutoFix` — easily extensible for PII.

### Finding 3: No allowlist/exemption system
Vulnerabilities can be marked `accepted` via status updates (`POST /api/v1/security/vulnerabilities/{id}/status`), but this is per-finding, not rule-based. There are no path-pattern exemptions, no per-pattern suppressions, and no time-bound exemptions. Test files are scanned identically to production code.

### Finding 4: GCT review integration uses scenario paths, not file lists
Current GCT review clients pass scenario names or paths (`TidinessLightScanRequest.ScenarioPath`, `TestExecutionRequest.ScenarioName`). A file-list endpoint would be a **new contract pattern** for GCT. The capability registry and health-check discovery would work identically — only the request/response contract differs.

### Finding 5: Infrastructure is modernization-ready, not modernization-required
The codebase uses Go modules, structured logging, PostgreSQL with proper schema, gorilla/mux routing, and has a comprehensive test framework (`api/test_patterns.go`). The core architecture is sound. Adding PII detection requires **extending** the existing pattern system, not rewriting it. Key additions: new pattern definitions, new DB tables for watchlists/allowlists, and a new scan endpoint.

### Finding 6: Watchlist requires a two-tier pattern system
The orchestration summary established that PII detection has two layers: (1) generic regex patterns (emails, phones, SSNs, IPs, AWS keys) that are built-in, and (2) a user-uploaded custom watchlist of personal values. The custom watchlist values are literal strings (e.g., "matt@example.com", "/home/matthalloran8"), not regex patterns — they need exact or substring matching, not regex compilation.

## Limitations
- Have not benchmarked pattern matching performance with large file lists (hundreds of staged files)
- Have not evaluated false-positive rates for generic PII patterns (email regex in Go import paths, IP addresses in config, etc.)
- Custom watchlist UX design is deferred to the execute item
- RBAC implications of watchlist management not explored (currently no auth system)

## Actions

<!-- TBD — will be refined as workshop decisions are resolved -->
