# API Endpoints

## System

### `GET /health`

Lifecycle and operator health probe. The response is proto-shaped JSON matching `vrooli.tech_tree_designer.v1.health.Response`.

```bash
curl "http://localhost:${API_PORT}/health"
```

## Planned Endpoints

Graph, planning, and roadmap endpoints will be Connect-RPC methods generated from future `graph`, `planning`, and `roadmap` protos. Do not add literal REST paths for product behavior.

## Cross-references

- [`../../.vrooli/endpoints.json`](../../.vrooli/endpoints.json)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
