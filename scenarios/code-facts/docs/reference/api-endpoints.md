# API Endpoints

Phase 5 defines the planned Connect-RPC contract. Phase 6 will add proto schemas and generated handlers.

| Operation | Purpose | Primary Requirement |
|---|---|---|
| `DescribeCodeFacts` | Resolve a target and return selected fact families. | CF-P0-004 |
| `ListSurfaces` | Return target context, surfaces, and parse units. | CF-P0-002 |
| `DescribeProtoAdoption` | Return proto adoption evidence. | CF-P0-006 |
| `DescribeEndpointProof` | Return endpoint implementation evidence. | CF-P0-007 |
| `GetCacheStatus` | Inspect cache keys and entries. | CF-P0-008 |
| `ClearCache` | Clear cache entries by target/options. | CF-P0-008 |

REST exceptions are limited to template health and multipart example endpoints until implementation replaces the generated notes domain.
