# Integrations

| Dependency | Role | Availability |
|---|---|---|
| Qdrant | Vector search, indexed documentation, and collection maintenance | Required for vector-backed knowledge workflows. |
| Scenario storage | SQLite metadata and workflow persistence through Vrooli's storage resolver | Required; local to the scenario namespace and needs no database credentials. |
| Ollama | Embeddings and optional structured-output fallback | Optional; affected search paths degrade when unavailable. |
| agent-manager | Executes deep-search and documentation-healing agents | Optional for those workflows. |
| prompt-manager | Supplies prompts/skills to agent-backed workflows | Optional for enriched prompts. |

The [data ownership table](DATA.md) describes what each store owns. Runtime
environment variables and docs-search tuning are documented in the
[configuration reference](../reference/configuration.md).

## Failure boundary

An unavailable vector service affects vector-backed search and collection
operations. SQLite holds scenario metadata and job review state. Documentation
inspection reads source repositories and can still report their health without
embedding or agent services.
