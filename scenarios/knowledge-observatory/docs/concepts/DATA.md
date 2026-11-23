# Data ownership

Knowledge Observatory indexes and inspects data owned by other scenarios. Its
own metadata and workflow state use a scenario-scoped SQLite database; vector
records live in Qdrant. The [architecture map](ARCHITECTURE.md#database-schema)
and [API reference](../reference/api-endpoints.md) hold the detailed domain and
route contracts.

| Data | Owner | Boundary |
|---|---|---|
| Source documentation | Each source scenario's repository | Read and indexed; source scenario remains authoritative. |
| Documentation vectors | Qdrant `vrooli-docs` collection | Reconciled from source files; not the original document store. |
| Knowledge vectors | Qdrant collections | Searched and inspected; collection maintenance is bounded and explicit. |
| Metadata, metrics, provenance, access history, graph edges, and KO workflow records | Scenario-scoped SQLite | Separate domain repositories share the scenario database; schemas remain with their owning domains. |
| Agent execution | agent-manager | KO stores job correlation, review, and approval state; agent-manager owns run execution. |

Deletion follows the owning workflow. Collection deletion removes collection
vectors and associated ingest metadata; it does not delete source scenario
documents. Append-log retention is manifest-driven. Job and operational records
must use their domain lifecycle rather than direct ad-hoc database edits.

## Related contracts

- [Integrations](INTEGRATIONS.md) — external service roles and availability.
- [Configuration](../reference/configuration.md) — storage and runtime inputs.
- [Runbook](../operations/RUNBOOK.md) — operational checks.
