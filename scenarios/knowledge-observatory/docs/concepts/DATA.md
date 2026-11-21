# Data

## Storage Overview

Knowledge Observatory reads and writes operational data through explicit stores:
Qdrant for vector records and PostgreSQL for metadata, metrics, access stats,
ingest jobs, deep-search jobs, and documentation-healing jobs.

## Data Ownership

| Data | Owner | Notes |
|---|---|---|
| Vector knowledge | Qdrant | Searched and inspected, not structurally owned by this scenario |
| Metrics and metadata | PostgreSQL | Scenario-owned operational tables |
| Scenario documentation | Filesystem | Validated through scenario-local manifests |
| Agent job state | PostgreSQL + agent-manager | KO stores job metadata and asks agent-manager for run state/diffs |

## Retention And Deletion

Append-log retention is manifest-driven. Vector and metadata retention should
stay explicit through maintenance commands rather than implicit background
deletion.

## Cross-References

- [INTEGRATIONS.md](INTEGRATIONS.md)
- [../reference/configuration.md](../reference/configuration.md)
- [../operations/RUNBOOK.md](../operations/RUNBOOK.md)
