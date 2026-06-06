# Architecture

## Purpose Of This Document

This document explains the scenario shape, runtime boundaries, and durable contracts future agents should preserve.

## Scenario Shape

Fall Foliage Explorer is a three-surface scenario: Go API, browser UI, and Go CLI. It is started and tested through Vrooli lifecycle commands.

## System Boundaries

The scenario owns foliage planning behavior inside `scenarios/fall-foliage-explorer/`. Shared resources such as PostgreSQL, Redis, Ollama, Qdrant, and app-monitor remain external dependencies.

## Runtime Surfaces

Fall Foliage Explorer has three primary surfaces:

- Go API: [CODE: api/main.go] serves health, regions, foliage status, predictions, weather, reports, and trips.
- Browser UI: [CODE: ui/src/app.js] renders the map-centered experience and calls the API through a proxy-aware base URL resolver.
- Go CLI: [CODE: cli/app.go] and [CODE: cli/domains/domains.go] provide command-line access to the same API domains.

These surfaces implement the PRD operational targets [REQ: REQ-P0-001], [REQ: REQ-P0-003], [REQ: REQ-P0-006], and [REQ: REQ-P0-007].

## Contracts And Data Flow

The API owns persistence, validation, and fallback responses. The UI owns presentation state, map interactions, filters, form state, and client-side exports. The CLI owns command parsing and transport formatting, but should not duplicate business rules from the API.

1. Lifecycle starts the API and UI using [CODE: .vrooli/service.json].
2. The UI resolves an API base from app-monitor proxy metadata, current location, or loopback defaults in [CODE: ui/src/app.js].
3. API handlers read PostgreSQL through the shared database helper and return packaged sample data when the database is unavailable for read-only discovery paths.
4. Prediction requests call Ollama through `OLLAMA_URL` when available, then fall back to typical peak-week logic if AI prediction fails.

## Shared Infrastructure

The scenario depends on Vrooli lifecycle, PostgreSQL, Redis, Ollama, Qdrant, Browserless as an optional fallback, and app-monitor iframe proxy metadata.

## Extension Rules

Do not replace lifecycle-managed commands with direct process execution. Do not move domain logic into the CLI or UI. Keep API responses compatible with the CLI support types in [CODE: cli/internal/support/types.go].

## Architecture Maturity

The core runtime surfaces are implemented, but requirements modules, BAS registry, and UI type-safety gates remain incomplete.

## Intentional Deviations

The UI is plain JavaScript rather than React/TypeScript despite the source template. Treat that as scaffold drift to resolve in a dedicated UI standards pass.

## Documentation Architecture

Documentation is registered in [DOC: docs/manifest.json]. Internal agent memory lives under `docs/internal/`.

## Cross-References

- [DOC: docs/concepts/DOMAINS.md]
- [DOC: docs/concepts/FLOWS.md]
- [DOC: docs/concepts/DATA.md]
- [DOC: docs/concepts/INTEGRATIONS.md]
- [DOC: docs/internal/SEAMS.md]
- [DOC: docs/internal/TESTING.md]
