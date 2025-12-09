# Scenario Documentation

Per-scenario deployment notes and orchestration details.

## Orchestration Scenarios

These scenarios work together to enable deployment:

| Scenario | Role | Documentation |
|----------|------|---------------|
| [deployment-manager](deployment-manager.md) | Orchestration hub | This scenario |
| [scenario-dependency-analyzer](scenario-dependency-analyzer.md) | Dependency DAG computation | Upstream data source |
| [scenario-to-desktop](scenario-to-desktop.md) | Desktop packager | Tier 2 packaging |
| [scenario-to-mobile](scenario-to-mobile.md) | Mobile packager | Tier 3 packaging |
| [scenario-to-cloud](scenario-to-cloud.md) | Cloud packager | Tier 4 packaging |
| [secrets-manager](secrets-manager.md) | Secret classification | Secret handling |

## Orchestration Flow

```
                    ┌─────────────────────────────────────┐
                    │         deployment-manager          │
                    │    (orchestrates full workflow)     │
                    └──────────────┬──────────────────────┘
                                   │
         ┌─────────────────────────┼─────────────────────────┐
         │                         │                         │
         ▼                         ▼                         ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ scenario-       │    │ secrets-manager │    │ scenario-to-*   │
│ dependency-     │    │                 │    │                 │
│ analyzer        │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Per-Scenario Notes

Each scenario document covers:

1. **Purpose** - What the scenario does
2. **Integration** - How it connects to deployment-manager
3. **API Endpoints** - Relevant endpoints
4. **Status** - Current implementation state

## For Target Scenarios

When deploying YOUR scenario (not the orchestration scenarios), the relevant documentation is:

- [Workflows](../workflows/README.md) - Step-by-step deployment guides
- [Guides](../guides/README.md) - Technical deep-dives
- [Examples](../examples/README.md) - Real-world case studies

## Related

- [CLI Reference](../cli/README.md) - Command documentation
- [API Reference](../api/README.md) - REST API documentation
