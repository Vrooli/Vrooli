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

## Planned Endpoints

Planning and roadmap endpoints are already declared as proto contracts, but their handlers are intentionally deferred to later phases. Do not add literal REST paths for product behavior.

## Cross-references

- [`../../.vrooli/endpoints.json`](../../.vrooli/endpoints.json)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
