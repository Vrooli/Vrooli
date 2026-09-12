# API Endpoints

`CodeFactsService` exposes the target resolver, fact query, proof, and cache diagnostics contract. Target/surface discovery, analyzer brokering, generic fact normalization, cache reuse, proto adoption proof, and REST exception endpoint proof are active.

| Operation | Purpose | Primary Requirement |
|---|---|---|
| `DescribeCodeFacts` | Resolve a target and return selected fact families. | CF-P0-004 |
| `Search` | Return bounded lexical hits with evidence and source provenance. | Search Hub code leaves |
| `ListSurfaces` | Return target context and surface inventory. | CF-P0-002 |
| `CheckProtoAdoption` | Return proto adoption evidence. | CF-P0-006 |
| `CheckEndpointProof` | Return endpoint implementation evidence. | CF-P0-007 |
| `GetCacheStatus` | Inspect cache keys and entries. | CF-P0-008 |
| `InspectCache` | Inspect matching cache entries with key/hash evidence. | CF-P0-008 |
| `ClearCache` | Clear cache entries by target/options. | CF-P0-008 |

`DescribeCodeFactsRequest.page_size` enables bounded report pages. The
response's `next_page_token` is an offset token for the next page and
`total_facts` describes the unpaged report. A zero page size retains the
single-response contract.

The only REST exception currently exposed by Code Facts is the operational `/health` probe.

`CheckProtoAdoption` uses normalized generated-proto import facts. `CheckEndpointProof` is static-only: it reads endpoint declarations, asks Code Facts framework adapters to normalize implementation evidence from provider graph facts, then proves route and response/error payload usage only when the selected adapter recognizes helper calls or typed arguments. It returns `unknown`, `missing`, `unsupported`, or `contradicted` rather than treating incomplete static evidence as success.
