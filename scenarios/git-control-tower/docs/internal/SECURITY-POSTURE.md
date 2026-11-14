# Security Posture

## Last Updated
2026-09-06

## Hardening Status by Category

### Secrets Management
- [x] No hardcoded secrets in the policy gate
- [x] Environment variable validation remains owner-specific
- [ ] Secret rotation support for external host connectors
Status: partial — host connectors are owned by Integration Hub.

### Authentication & Authorization
- [x] Mutating Connect procedures deny requests without an authenticated principal context
- [x] Test-mode database and configuration-file writes route through the leased Test Genie pools/roots
- [x] Router-level baseline security headers are stamped before first-party handlers
- [x] Caller-supplied `X-Vrooli-Caller` and `X-Vrooli-Authorized` headers cannot grant authority
- [x] Human intent shape binds subject, repository, operation, expected revision, expiry, and consumption
- [x] Scenario-authenticator RS256/JWKS relying-party verification checks issuer, realm audience, expiry, algorithm, key id, and subject
- [x] Optional authenticator Validate RPC hook rejects live revoked sessions after local verification
- [x] Typed Connect-RPC commit and branch writers require verified human authority and exact request-bound intents
Status: partial — the remaining HTTP boundary, exact commit/branch intents, consumed-intent domain
writer guard and JWKS freshness policy,
core Git writer services, grouped gitignore moves, credentials, SSH key writers,
and repository-registry service are hardened. Legacy mutation adapters without a
typed preview/intent client fail closed, and the live human actuation receipt
remains separate.

### Input Validation
- [x] Policy decisions fail closed for missing or unknown authority
- [x] Safety admission classifies unknown subprocess commands as not admitted
- [ ] Full legacy REST input inventory
Status: partial.

### Error Handling
- [x] Refused writer requests use typed permission-denied responses
- [x] Audit events record decision and authority standing without credentials
- [ ] Uniform refusal semantics across every legacy route
Status: partial.

## Known Vulnerabilities

- Existing tests still invoke real Git mutations in temporary repositories;
  see `SAFETY-ADMISSION-REPORT.md`. Comprehensive validation remains
  unadmitted until those fixtures are replaced.
- Some non-Git state/filesystem writers still need the same domain-level
  principal seam: remaining review/audit-adjacent REST writers. Repository
  registry, credential, remote URL, precommit configuration, and SSH key
  generation/deletion are now typed and covered by the seam; read resolution
  no longer creates or activates registry records.

## Priority Hardening Areas
1. Extend the verified-principal seam to remaining non-typed writer services,
   then complete the transport inventory.
2. Replace mutating test bootstrap with immutable records and recording fakes.
