# Integrations — Secrets Manager

## Purpose Of This Document

This document describes the external contracts Secrets Manager relies on.

## Dependency Inventory

Required resources are Vault and Postgres. Claude Code is optional for remediation workflows.

## Vrooli Resources

`resource-vault` validates and provisions secret storage. The managed-service contract requires verified artifacts, brokered shared use, and no direct remote endpoint bypass. Postgres stores shared metadata.

## Scenario Dependencies

Deployment Manager and scenario-to-desktop consume deployment strategy outputs. Scenario Dependency Analyzer supplies deployment metadata used during manifest generation.

## Third-Party Services

HashiCorp Vault is distributed as a provenance-verified artifact. No direct public-cloud secret service is a runtime dependency.

## Failure Modes

Missing Vault or Secret Service tooling fails resource preflight with remediation. Broker denial or expired shared permission falls back to an authorized private bundle resource where policy allows. Database failures produce degraded metadata posture rather than secret disclosure.

## Cross-References

- [Configuration](../reference/configuration.md)
- [Deployment](../operations/DEPLOYMENT.md)
- [Architecture](ARCHITECTURE.md)
