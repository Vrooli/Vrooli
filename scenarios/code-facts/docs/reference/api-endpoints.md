# API Endpoints

`CodeFactsService` exposes the target resolver, fact query, proof, and cache diagnostics contract. Target/surface discovery, analyzer brokering, generic fact normalization, cache reuse, proto adoption proof, and REST exception endpoint proof are active.

| Operation | Purpose | Primary Requirement |
|---|---|---|
| `DescribeCodeFacts` | Resolve a target and return selected fact families. | CF-P0-004 |
| `ListSurfaces` | Return target context and surface inventory. | CF-P0-002 |
| `CheckProtoAdoption` | Return proto adoption evidence. | CF-P0-006 |
| `CheckEndpointProof` | Return endpoint implementation evidence. | CF-P0-007 |
| `GetCacheStatus` | Inspect cache keys and entries. | CF-P0-008 |
| `InspectCache` | Inspect matching cache entries with key/hash evidence. | CF-P0-008 |
| `ClearCache` | Clear cache entries by target/options. | CF-P0-008 |

The only REST exception currently exposed by Code Facts is the operational `/health` probe.

`CheckProtoAdoption` uses normalized generated-proto import facts. `CheckEndpointProof` is static-only: it reads endpoint declarations, then proves response/error payload usage only when graph facts expose recognizable helper calls or typed arguments. It returns `unknown`, `missing`, or `contradicted` rather than treating incomplete static evidence as success.
