# Integrations - Tech Tree Designer

## Purpose Of This Document

Record scenario and resource dependencies for TTD.

## Scenario Dependencies

| Dependency | Required | Purpose | Degraded Behavior |
|---|---:|---|---|
| proto-health | yes | Supplies `DescribeScenariosProtos` surfaces for live graph nodes and proto-import edges. | Planned-only graph mode with a clear unavailable-source finding. |

## Resource Dependencies

| Resource | Required | Purpose |
|---|---:|---|
| SQLite | yes | Embedded scenario data store through lifecycle-provided `SQLITE_PATH`. |

No Ollama, OpenRouter, Postgres, Qdrant, or Redis dependency belongs in this phase.

## Future Integrations

| Integration | Trigger |
|---|---|
| scenario-dependency-analyzer | Add as GraphSource after `DescribeInterfaceGraph` ships. |
| AI model resources | Add only for the deferred strategic-analysis follow-up. |
| scenario generator | Add only after planning materialization is stable and a scaffold-governance design exists. |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md)
