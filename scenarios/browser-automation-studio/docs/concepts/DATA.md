# Data

## Purpose Of This Document

Describe durable data ownership and evidence integrity.

## Storage Overview

The API owns routed SQLite schema and artifact references. Driver session state is ephemeral. Replay packages contain immutable timeline and integrity metadata.

## Data Ownership

Workflows and executions belong to the API domain; browser pages and contexts belong to the driver; UI state is disposable client state.

## Schema Map

Core workflow/execution, recording, billing, UX metrics, and triggers are registered schema domains. New tables belong to one owner and must be registered through the schema registry.

## Migrations And Compatibility

Use scenario migrations and preserve V2 proto compatibility. Do not bypass routed connections or add schema setup in feature handlers.

## Import / Export

Recording imports normalize into executions and assets. Replay exports carry a manifest and byte digests for sanitized HAR evidence.

## Retention And Deletion

Retention policy is not yet a deployed product feature; operators must treat recordings and artifacts as potentially sensitive and remove them through supported scenario operations.

## Privacy Notes

Browser evidence can contain page content and interaction data. Secrets must not be written into workflow definitions, artifacts, or logs.

## Cross-References

- [Architecture](ARCHITECTURE.md)
- [Flows](FLOWS.md)
- [Configuration](../reference/configuration.md)
