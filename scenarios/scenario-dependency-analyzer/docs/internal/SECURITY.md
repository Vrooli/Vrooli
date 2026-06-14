# Security

## Purpose Of This Document

Capture SDA security posture and gaps.

## Data Sensitivity

SDA processes repository metadata, manifests, imports, and generated graph evidence.

## Auth And Authorization

Local lifecycle deployment currently relies on local operator trust boundaries.

## Secrets

Do not persist source contents or secrets in graph evidence. Secret-like test fixtures must be clearly marked as examples.

## Threat Model

Primary risks are stale dependency evidence, poisoned local manifests, and accidental secret exposure in generated artifacts.

## Security Gaps

Dependency vulnerability findings are reported by the scenario test security phase.
As of 2026-06-14, the blocking dependency issues are resolved:

- API and CLI Go modules pin `go 1.25.11`, with local `GOTOOLCHAIN=auto` so the patched toolchain is selected for SDA builds.
- API dependencies use patched `github.com/rs/cors`, `golang.org/x/net`, `golang.org/x/crypto`, and `golang.org/x/sys` versions.
- UI dev tooling uses patched Vite-era dependencies; `pnpm audit --audit-level moderate` reports no known vulnerabilities.
- `security-health validate scenario scenario-dependency-analyzer` passes with no error or warning findings.

The remaining security-health entries are info-level only. Current notable classes are local gitignored fingerprint placeholders, test-helper ignored-error observations, and OSV matches inside nested package-manager fixture lockfiles under `node_modules`.

## Hardening Notes

The API server sets explicit read-header, read, write, and idle timeouts. Generated/exported local artifacts use private file modes by default. Scanner paths skip symlink entries before reading repository files, and reviewed gosec suppressions document root-constrained file access and fixed-executable subprocess calls.

## Cross-References

- `../operations/RUNBOOK.md`
- `ERROR-HANDLING.md`
