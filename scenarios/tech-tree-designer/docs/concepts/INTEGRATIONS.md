# Integrations - Tech Tree Designer

## Purpose Of This Document

Record scenario and resource dependencies for TTD.

## Scenario Dependencies

| Dependency | Required | Purpose | Degraded Behavior |
|---|---:|---|---|
| scenario-dependency-analyzer | yes | Supplies `DescribeInterfaceGraph` live scenario nodes plus proto and Go import evidence. | Planned-only graph mode with a clear unavailable-source finding. |

## Resource Dependencies

| Resource | Required | Purpose |
|---|---:|---|
| SQLite | yes | Embedded scenario data store resolved by `api-core/storage` from the scenario id. |

TTD does not call Ollama, OpenRouter, Postgres, Qdrant, or Redis directly. SDA may depend on heavier resources internally, but TTD only consumes SDA's graph contract.

## Future Integrations

| Integration | Trigger |
|---|---|
| AI model resources | Add only for the deferred strategic-analysis follow-up. |
| scenario generator | Add only after planning materialization is stable and a scaffold-governance design exists. |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md)
