# Security

## Security Posture

Tidiness Manager scans repository files and can invoke subprocess-based analyzer tools. The security posture depends on bounded paths, allowlisted commands, timeouts, sanitized errors, and scoped dependency updates.

## Current Controls

- JSON decoding rejects oversized and unknown-field requests.
- Scan paths are constrained to scenario/repository roots.
- Subprocesses use context timeouts.
- SQL access should remain parameterized.
- Client responses should not expose database credentials or internal resource details.

## Known Hardening Work

The comprehensive hardening plan includes a security phase for gosec command-execution findings and UI dependency vulnerabilities. Track remaining work in `PROBLEMS.md`.

## Static-Quality Boundary

Static-quality policy is enforced by Quality Health. Security fixes here should reduce real risk without moving lint/type policy back into Tidiness Manager.
