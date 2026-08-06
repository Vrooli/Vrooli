# Integrations — Secrets Manager

## Purpose Of This Document

This document describes the external contracts Secrets Manager relies on.

## Dependency Inventory

Postgres is required for shared metadata. Ordinary credential storage and
resolution use the canonical credential authority backed by the host key
service or encrypted authority storage. Claude Code is optional for remediation
workflows. Vault is not an implicit scenario dependency.

## Vrooli Resources

The credential authority validates and provisions secret storage through
stdin-only control-plane commands. The managed-service contract requires
metadata-safe status, recovery-bundle support, and no plaintext API response.
Postgres stores shared metadata.

## Scenario Dependencies

Deployment Manager and scenario-to-desktop consume deployment strategy outputs. Scenario Dependency Analyzer supplies deployment metadata used during manifest generation.

## Third-Party Services

No remote secret service is a runtime dependency. Vault may be selected only by
an explicitly governed Vault-specific capability such as Transit signing.

## Failure Modes

Missing native key-service/authority support fails credential operations with
actionable remediation. Recovery-bundle import/export remains operator
controlled. Database failures produce degraded metadata posture rather than
secret disclosure.

## Cross-References

- [Configuration](../reference/configuration.md)
- [Deployment](../operations/DEPLOYMENT.md)
- [Architecture](ARCHITECTURE.md)
