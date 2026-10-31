# Decisions

## Purpose Of This Document

Record durable SDA decisions.

## Decision Log

- Actual graph evidence is sourced from `proto-health` and `code-facts`, not direct SDA source scans.
- Drift severity is asymmetric: actual undeclared usage is warning; declared-only dependency is info.
- SQLite is the local persistence substrate for SDA-owned metadata.
- Dependency health owns dependency readiness, runtime dependency status, approved-dependency governance, release-age policy, graph drift, and Security Health dependency-index availability. It does not emit vulnerable-package findings or duplicate the security phase.
- Security Health owns vulnerability scanning, CVE/advisory normalization, and security-phase gating. SDA governance may consume Security Health vulnerability evidence for denied ranges, remediation previews, and approval decisions.

## Superseded Decisions

Port-regex and ad hoc CLI-reference scanners are superseded by upstream fact services where available.

## Cross-References

- `SEAMS.md`
- `../concepts/ARCHITECTURE.md`
