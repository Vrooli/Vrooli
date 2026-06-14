# Decisions

## Purpose Of This Document

Record durable SDA decisions.

## Decision Log

- Actual graph evidence is sourced from `proto-health` and `code-facts`, not direct SDA source scans.
- Drift severity is asymmetric: actual undeclared usage is warning; declared-only dependency is info.
- SQLite is the local persistence substrate for SDA-owned metadata.

## Superseded Decisions

Port-regex and ad hoc CLI-reference scanners are superseded by upstream fact services where available.

## Cross-References

- `SEAMS.md`
- `../concepts/ARCHITECTURE.md`
