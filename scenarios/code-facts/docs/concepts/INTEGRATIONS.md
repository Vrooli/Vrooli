# Integrations

## Purpose Of This Document

Track provider, resource, and consumer relationships for Code Facts.

## Dependency Inventory

| Dependency | Kind | Phase | Purpose |
|---|---|---|---|
| go-code-graph | scenario provider | Phase 8 | Go module graph and usage facts. |
| typescript-code-graph | scenario provider | Phase 8 | TypeScript project graph and usage facts. |
| proto-health | consumer | Phase 12 | Consumes proto adoption and endpoint proof facts. |

## Vrooli Resources

No shared resources are required in v1. SQLite may be used locally for cache metadata if Phase 9 needs persistence.

## Scenario Dependencies

Provider dependencies are called through Connect-RPC clients hidden behind analyzer seams. Unit tests use fakes and must not require live providers.

## Third-Party Services

None.

## Failure Modes

- Provider unavailable: return `unsupported` or `unknown` evidence with provider diagnostic.
- Ambiguous target: return target-resolution error and no provider call.
- Stale cache: return stale reason and recompute unless caller disables recomputation.

## Cross-References

- [../reference/configuration.md](../reference/configuration.md)
- [../internal/SEAMS.md](../internal/SEAMS.md)
