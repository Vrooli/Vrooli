# API Endpoints

## System

### `GET /health`

Lifecycle and operator health probe. The response is proto-shaped JSON matching `vrooli.tech_tree_designer.v1.health.Response`.

```bash
curl "http://localhost:${API_PORT}/health"
```

## Graph Connect RPC

Graph endpoints are generated Connect-RPC procedures:

| Method | Procedure |
|---|---|
| `DescribeTechTree` | `/vrooli.tech_tree_designer.v1.graph.GraphService/DescribeTechTree` |
| `GetNeighborhood` | `/vrooli.tech_tree_designer.v1.graph.GraphService/GetNeighborhood` |
| `FindPath` | `/vrooli.tech_tree_designer.v1.graph.GraphService/FindPath` |
| `ListAncestors` | `/vrooli.tech_tree_designer.v1.graph.GraphService/ListAncestors` |
| `ExportTechTree` | `/vrooli.tech_tree_designer.v1.graph.GraphService/ExportTechTree` |

## Planning Connect RPC

Planning endpoints are generated Connect-RPC procedures:

| Method | Procedure |
|---|---|
| `CreatePlannedScenario` | `/vrooli.tech_tree_designer.v1.planning.PlanningService/CreatePlannedScenario` |
| `ListPlannedScenarios` | `/vrooli.tech_tree_designer.v1.planning.PlanningService/ListPlannedScenarios` |
| `GetPlannedScenario` | `/vrooli.tech_tree_designer.v1.planning.PlanningService/GetPlannedScenario` |
| `PutPlannedProtoFile` | `/vrooli.tech_tree_designer.v1.planning.PlanningService/PutPlannedProtoFile` |
| `DeletePlannedProtoFile` | `/vrooli.tech_tree_designer.v1.planning.PlanningService/DeletePlannedProtoFile` |
| `ValidatePlannedScenario` | `/vrooli.tech_tree_designer.v1.planning.PlanningService/ValidatePlannedScenario` |
| `MaterializePlannedScenario` | `/vrooli.tech_tree_designer.v1.planning.PlanningService/MaterializePlannedScenario` |

## Roadmap Connect RPC

Roadmap endpoints are generated Connect-RPC procedures:

| Method | Procedure |
|---|---|
| `ListSectors` | `/vrooli.tech_tree_designer.v1.roadmap.RoadmapService/ListSectors` |
| `UpsertSector` | `/vrooli.tech_tree_designer.v1.roadmap.RoadmapService/UpsertSector` |
| `ListMilestones` | `/vrooli.tech_tree_designer.v1.roadmap.RoadmapService/ListMilestones` |
| `UpsertMilestone` | `/vrooli.tech_tree_designer.v1.roadmap.RoadmapService/UpsertMilestone` |
| `GetProgress` | `/vrooli.tech_tree_designer.v1.roadmap.RoadmapService/GetProgress` |

## Planned Endpoints

No current proto RPC is intentionally unmounted. Deferred future integrations are tracked in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md), not as placeholder endpoints.

## Cross-references

- [`../../.vrooli/endpoints.json`](../../.vrooli/endpoints.json)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
