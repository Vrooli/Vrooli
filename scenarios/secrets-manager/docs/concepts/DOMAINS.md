# Domains — Secrets Manager

## Purpose Of This Document

This document maps product vocabulary to owning source paths.

## Domain Inventory

| Domain | Responsibility | Owns Data | Primary Archetype | Secondary Traits | Source Paths |
|---|---|---|---|---|---|
| Credential coverage | Validate required credentials and provision declared values through the credential authority | validation metadata | validation | service, provider | `api/credential_*.go`, `cli/domains/credentials/` |
| Security intelligence | Scan files, manage findings, allowlists, and watchlists | scan records and findings | reporting | validation, mutation | `api/security_*.go`, `api/allowlist.go`, `api/watchlist.go` |
| Resource intelligence | Show resource secret details and update strategies | resource and secret metadata | query | mutation | `api/resource_*.go`, `ui/src/features/resource-panel/` |
| Deployment readiness | Build tier-specific secret manifests and campaigns | deployment manifests and campaigns | orchestration | reporting | `api/deployment_*.go`, `api/campaign_*.go` |
| Scenario overrides | Merge tier and scenario-specific strategies | override records | crud | mutation | `api/scenario_override_*.go`, `cli/domains/overrides/` |

## Domain Details

Credential coverage never exposes secret values to response consumers. Security
intelligence may start remediation workflows but does not silently modify
source. Deployment readiness provides manifest data to deployment consumers and
keeps bundle resource choice explicit.

## Shared Concepts

Secret metadata, tier strategy, validation results, vulnerability status, and lifecycle-owned resource dependencies are shared concepts.

## Deferred Domains

Automated remediation, forecasting, and external policy gates remain future product capabilities.

## Non-Domains

HTTP transport, database routing, lifecycle management, and generic test utilities are infrastructure, not product domains.

## Cross-References

- [Architecture](ARCHITECTURE.md)
- [Flows](FLOWS.md)
- [Data](DATA.md)
- [Seams](../internal/SEAMS.md)
