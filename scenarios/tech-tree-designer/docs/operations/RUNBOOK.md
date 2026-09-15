# Runbook — Tech Tree Designer

## Purpose Of This Document

Operate the existing scenario safely and define recovery obligations for the proposed bundle system. Target procedures below are not invented CLI commands.

## Start / Stop / Status

Use make start, make status, make logs and make stop from the scenario directory, or the Vrooli scenario lifecycle. Never run binaries directly. Select validation scope with docs/TESTING.md rather than running all suites for every documentation edit.

## Common Incidents

| Symptom | Safe response |
|---|---|
| API/UI unavailable | Inspect lifecycle status/logs and configured ports; route host remediation to the control plane |
| Observed graph unavailable/stale | Inspect SDA availability, source revision and reported coverage; preserve labeled cached/proposal context |
| Proposed operation unsupported | Discover qualified owner capability; leave explicit held work, not a direct-write fallback |
| Relevant base changed | Preserve current content, expose exact conflict and prepare a new reviewed revision as required |
| Apply interrupted or acknowledgement lost | Resolve durable operation and per-entry owner receipts before retrying; never blindly repeat writes |
| Partial artifact/ontology application | Show completed, held and failed entries with dependencies; obtain authority for recovery or compensation |
| Draft compute lost | Rehydrate from retained immutable manifest/bytes through the workspace owner; never reconstruct reviewed content from plan prose |
| Resource pressure | Narrow scope and apply documented limits; do not silently truncate into a complete result |

## Backup / Restore

Current metadata uses SQLite. A qualified production backup/restore procedure is still required. The target covers metadata, referenced immutable bytes, graph revisions, authority references and apply receipts consistently. Restore must validate identities and external owner effects before resuming pending operations. Restoring a database is not permission to reverse canonical repository changes.

Recovery point/time objectives and retention windows remain explicit decisions. Do not delete referenced drafts, databases or receipts to troubleshoot startup.

The [development review proposal](../internal/DECISIONS.md#retention-recovery-and-deployment-proposal)
contains initial retention and backup numbers for review. They are not an
implemented backup job, accepted recovery guarantee or deletion authorization.

## Maintenance Tasks

Inspect freshness, cache/storage growth, retention pins and unresolved apply operations. Use owner validation and generators for changed artifacts. Garbage collection must be bounded, auditable and reference-safe. Capture recurring issues and durable work evidence.

## Escalation

Use the shared reporting/workflow owner for defects. Record product qualification gaps in [PROBLEMS](../internal/PROBLEMS.md). Ask for direction when recovery requires effects outside the existing mandate.

## Cross-References

- [Observability](OBSERVABILITY.md)
- [Deployment](DEPLOYMENT.md)
- [Flows](../concepts/FLOWS.md)
- [Security](../internal/SECURITY.md)
