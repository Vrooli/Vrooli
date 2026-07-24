# Data — Secrets Manager

## Purpose Of This Document

This document defines metadata ownership and secret-value boundaries.

## Storage Overview

Postgres stores shared secret requirements, validation history, scans, strategies, and overrides. Desktop mode uses a scenario-private SQLite database for metadata. Vault stores secret values.

## Data Ownership

Secrets Manager owns the metadata it creates. `resource-vault` owns Vault secret values and access control. Clients receive status, identifiers, and remediation guidance, never values.

## Schema Map

Postgres schema setup is in `api/postgres_schema.go`. Desktop schema and database-path handling are in `api/desktop_storage.go`.

## Migrations And Compatibility

Schema initialization is idempotent. Routed database selection keeps shared and desktop-private stores separate.

## Import / Export

Deployment exports contain strategies and required secret identifiers. They must not contain secret values or Vault management credentials.

## Retention And Deletion

Validation and scan records are operational metadata. Deletion and cleanup flows must preserve audit semantics and must not delete Vault values implicitly.

## Privacy Notes

Treat secret names, paths, and scan evidence as sensitive operational metadata. Redact values in logs, responses, test artifacts, and handoffs.

## Cross-References

- [Architecture](ARCHITECTURE.md)
- [Integrations](INTEGRATIONS.md)
- [Security](../internal/SECURITY.md)
