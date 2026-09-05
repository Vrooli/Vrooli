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

## Ontology Connect RPC

Ontology endpoints are generated Connect-RPC procedures:

| Method | Procedure |
|---|---|
| `ListCapabilities` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/ListCapabilities` |
| `GetCapability` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/GetCapability` |
| `UpsertCapability` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/UpsertCapability` |
| `DeleteCapability` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/DeleteCapability` |
| `UpsertCapabilityEdge` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/UpsertCapabilityEdge` |
| `DeleteCapabilityEdge` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/DeleteCapabilityEdge` |
| `ImportTopology` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/ImportTopology` |
| `LinkFulfillment` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/LinkFulfillment` |
| `UnlinkFulfillment` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/UnlinkFulfillment` |
| `ListFulfillments` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/ListFulfillments` |
| `GetCoverage` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/GetCoverage` |
| `ListFocus` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/ListFocus` |
| `GetCapabilityScenarios` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/GetCapabilityScenarios` |
| `GetScenarioCapabilities` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/GetScenarioCapabilities` |
| `DescribeOverlayGraph` | `/vrooli.tech_tree_designer.v1.ontology.OntologyService/DescribeOverlayGraph` |

## Planned Endpoints

No current proto RPC is intentionally unmounted. Deferred future integrations are tracked in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md), not as placeholder endpoints.

## Cross-references

- [`../../.vrooli/endpoints.json`](../../.vrooli/endpoints.json)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
