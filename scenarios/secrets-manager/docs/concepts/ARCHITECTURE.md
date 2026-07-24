# Architecture — Secrets Manager

## Purpose Of This Document

This document names the stable product boundaries so agents can place changes predictably.

## Scenario Shape

Secrets Manager is a Go API, a React/Vite UI, and a Go CLI. It inventories secret requirements, checks Vault coverage, scans security posture, produces deployment strategies, and records metadata in Postgres or desktop-scoped SQLite.

## System Boundaries

The API owns validation, strategy resolution, persistence, and integration decisions. The UI renders API data. The CLI proxies API operations. Vault is consumed through the declared `resource-vault` boundary; secret values are not returned to clients.

## Contracts And Data Flow

The API registers capability routes in `api/server.go`. UI calls enter through `ui/src/lib/api.ts`; CLI commands use the scenario API client. Deployment consumers receive a generated manifest rather than direct Vault access.

## Shared Infrastructure

`api-core` supplies lifecycle server behavior, database routing, test-mode middleware, and development routing. `resource-vault` supplies secret-storage validation and provisioning. Postgres stores shared metadata; desktop mode uses a private SQLite database.

## UI Deployment Surface

The UI uses `@vrooli/api-base/server` as its only production server. That server owns health, static serving, SPA fallback, and `/api` proxying so direct localhost, tunnel, and app-monitor iframe requests use the same route. The child iframe bridge is initialized only when embedded; spatial navigation is initialized at startup. Tutorial anchors move focus rather than calling cross-frame scrolling APIs.

The Vite base and public PWA assets are relative to the mounted UI. The scoped service worker caches the app shell and same-origin static assets, while API requests remain network-only. This preserves private Vault data and prevents a desktop/offline cache from serving stale secret-status responses.

## Extension Rules

Add product behavior to its owning capability. Keep handler transport concerns thin. Put generic cross-capability mechanics in shared infrastructure only when they contain no secret-management vocabulary.

## Architecture Maturity

The scenario has an explicit API composition root and capability route groups. Test utility and injectable-seam maturity still needs follow-up; see `../internal/PROBLEMS.md`.

## Intentional Deviations

Desktop deployments use private SQLite metadata storage. Shared Vault reuse is an explicit brokered decision; a desktop bundle must not route directly to a remote Vault endpoint.

## Documentation Architecture

`docs/manifest.json` is the documentation contract. Concepts explain stable models, references define lookup material, operations describe runtime work, and internal documents preserve agent context.

## Cross-References

- [Domains](DOMAINS.md)
- [Flows](FLOWS.md)
- [Data](DATA.md)
- [Integrations](INTEGRATIONS.md)
- [Seams](../internal/SEAMS.md)
- [Testing](../internal/TESTING.md)
