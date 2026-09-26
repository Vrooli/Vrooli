# Integrations

## Purpose Of This Document

List external and Vrooli-managed dependencies at the ownership boundary.

## Dependency Inventory

Dependency changes require Scenario Dependency Analyzer approval. Proto generation is repository infrastructure, not a runtime service.

## Vrooli Resources

The scenario can use managed artifact/storage resources when configured; its default relational state is SQLite.

## Scenario Dependencies

The API supervises the in-repo Playwright driver. UI and CLI consume API contracts rather than connecting directly to browser processes. Optional `vrooli-memory` provides scoped `learn.*` recall and capture, `workflow-health` provides workflow search, and `search-hub` provides federated candidates. All three use `try_start` and `bundle_policy: either`; BAS falls back to local workflow listing and queued capture when they are unavailable.

## Third-Party Services

AI providers are optional capability integrations. Their availability must not make core workflow execution unsafe or nondeterministic.

## Failure Modes

Unavailable driver, storage, or AI integrations must surface actionable health/diagnostic evidence. A driver failure does not justify accepting untyped instructions.

## Cross-References

- [Configuration](../reference/configuration.md)
- [Architecture](ARCHITECTURE.md)
- [Operations](../operations/RUNBOOK.md)
