# Integrations

## Purpose Of This Document

Document resources, scenario dependencies, and integration contracts.

## Dependency Inventory

SDA depends on SQLite, Ollama, Qdrant, `proto-health`, and `code-facts`.

## Vrooli Resources

SQLite stores local metadata. Ollama and Qdrant support semantic analysis and optimization surfaces.

## Scenario Dependencies

`proto-health` provides batch proto surface facts. `code-facts` provides fleet import facts. `test-genie` consumes SDA drift during dependency validation.

## Third-Party Services

No external SaaS dependency is required for core local operation.

## Failure Modes

If fact services are unavailable, actual graph responses include upstream errors and drift may be incomplete.

## Cross-References

- `../reference/configuration.md`
- `ARCHITECTURE.md`
- `../internal/SEAMS.md`
