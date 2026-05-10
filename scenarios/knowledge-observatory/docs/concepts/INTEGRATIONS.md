# Integrations

## Dependency Inventory

| Dependency | Role | Required |
|---|---|---|
| Qdrant | Vector search and knowledge graph source | Yes |
| PostgreSQL | Metadata, metrics, and job persistence | Yes |
| Ollama | Embeddings and structured parsing where enabled | Optional |
| agent-manager | Deep search and documentation-healing runs | Optional for agent workflows |
| prompt-manager | Skill content for agent prompts | Optional for enriched prompts |

## Failure Modes

Resource outages should degrade specific capabilities while preserving basic
health and documentation inspection wherever possible.

## Cross-References

- [DATA.md](DATA.md)
- [../reference/configuration.md](../reference/configuration.md)
- [../operations/OBSERVABILITY.md](../operations/OBSERVABILITY.md)
