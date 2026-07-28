# Integrations

## Purpose Of This Document

List external and Vrooli-managed dependencies at the ownership boundary.

## Dependency Inventory

Dependency changes require Scenario Dependency Analyzer approval. Proto generation is repository infrastructure, not a runtime service.

## Vrooli Resources

The scenario can use managed artifact/storage resources when configured; its default relational state is SQLite.

## Scenario Dependencies

The API supervises the in-repo Playwright driver. UI and CLI consume API contracts rather than connecting directly to browser processes.

## Third-Party Services

AI providers are optional capability integrations. Their availability must not make core workflow execution unsafe or nondeterministic.

## Failure Modes

Unavailable driver, storage, or AI integrations must surface actionable health/diagnostic evidence. A driver failure does not justify accepting untyped instructions.

## Cross-References

- [Configuration](../reference/configuration.md)
- [Architecture](ARCHITECTURE.md)
- [Operations](../operations/RUNBOOK.md)
