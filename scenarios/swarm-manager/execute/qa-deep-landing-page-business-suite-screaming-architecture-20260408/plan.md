# Implementation Plan

## Purpose
Realign landing-page-business-suite structure so the codebase and documentation clearly express the scenario's domain boundaries, reducing "god" entrypoints and documentation drift without changing behavior.

## Required Reading
```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read knowledge-observatory-tools
knowledge-observatory docs read landing-page-business-suite architecture
knowledge-observatory docs read landing-page-business-suite seams
```

## Problem Statement
The API and UI entrypoints are monolithic (`api/main.go`, `api/routes.go`, `ui/src/App.tsx`) and the architecture documentation no longer matches the current layout (UI tree, initialization paths, user-auth surface). The scenario now spans multiple domains (landing, billing, downloads, AI gateway, remote profiles), but the physical structure does not make these boundaries obvious.

## Scope
In scope:
- Refactor API and UI entrypoints into domain-aligned modules while preserving behavior.
- Update architecture documentation and docs manifest health to reflect actual layout.
- Remove/relocate build artifacts that blur source vs output boundaries.

Out of scope:
- Any changes to business logic, endpoints, or UI behavior.
- Introducing new features or altering data models.
- Large-scale rewrites that require new infrastructure.

## Current Technical Context
Key files and locations:
- `scenarios/landing-page-business-suite/api/main.go`
- `scenarios/landing-page-business-suite/api/routes.go`
- `scenarios/landing-page-business-suite/ui/src/App.tsx`
- `scenarios/landing-page-business-suite/docs/concepts/ARCHITECTURE.md`
- `scenarios/landing-page-business-suite/initialization/postgres/`
- Documentation health output: `knowledge-observatory docs health -json -scenario landing-page-business-suite`

## Target End State
- API routes and service wiring are organized by domain modules; `main.go`/`routes.go` are thin composers.
- UI route assembly is split by surface (public/admin/user-auth) with a minimal App entrypoint.
- Architecture docs accurately reflect actual tree and include all major domains (including CLI + AI gateway).
- Documentation health >=95% with missing/misplaced docs addressed.
- Build artifacts removed from source tree or isolated in a dedicated output path with ignore rules.

## Implementation Strategy
Phase 1: Architecture mapping and boundaries
1. Map current API domains (landing, billing, downloads, metrics, AI, admin, auth, remote) and list handlers/services per domain.
2. Decide on a domain module layout (folder-based within `api/` or subpackages) that keeps public API unchanged.

Phase 2: API structure refactor (behavior-preserving)
1. Introduce domain-level files or packages (e.g., `api/domain/landing`, `api/domain/billing`, etc.).
2. Move handlers/services into domain modules and add `RegisterRoutes` functions per domain.
3. Replace `Server`'s flat fields with grouped dependency structs (e.g., `BillingDeps`, `DownloadsDeps`, `AIDeps`) while keeping existing initialization logic.

Phase 3: UI entrypoint refactor
1. Extract admin/public/user-auth route definitions into dedicated route modules under `ui/src/app/routes/`.
2. Keep `ui/src/App.tsx` as a thin composer that imports route groups and providers.

Phase 4: Documentation alignment
1. Update `docs/concepts/ARCHITECTURE.md` to reflect actual tree (App location, shared UI, user-auth surface, initialization path).
2. Add missing mentions for CLI and AI gateway in architecture overview if they are runtime surfaces.
3. Resolve docs health issues: move misplaced docs, add missing baseline docs, remove/relocate extras.

Phase 5: Artifact cleanup
1. Relocate/remove build/test artifacts under `api/` (binaries, coverage outputs, stray files like `api/0`).
2. Update `.gitignore` to prevent future artifact commits.

## Contract Decisions
- No endpoint paths, request/response shapes, or auth semantics change.
- Domain module boundaries are internal only and must preserve public API behavior.
- Documentation updates reflect actual code; they do not invent new behavior.

## Testing Plan
- Primary: `vrooli scenario test landing-page-business-suite`
- If needed and deps are already installed:
  - `cd scenarios/landing-page-business-suite/api && go test ./...`
  - `cd scenarios/landing-page-business-suite/ui && pnpm test` (or existing UI test runner)

## Rollout / Validation Checklist
- Routes and handlers still registered for all domains (diff check against old `routes.go`).
- No compile errors in API/CLI/UI modules.
- Documentation health report shows >=95% score.
- Manual smoke: `/api/v1/health` and `/api/v1/landing-config` still respond.

## Risks + Mitigations
- Risk: Go package refactor introduces import cycles.
  - Mitigation: start with file-level reorg and register functions before introducing subpackages.
- Risk: Documentation updates fall out of sync again.
  - Mitigation: update docs and re-run `knowledge-observatory docs health` before completion.
- Risk: Artifact removal conflicts with local workflows.
  - Mitigation: document new output path and update ignore rules.

## Non-goals / Prohibited Patterns
- No behavioral changes to billing, downloads, AI gateway, or auth.
- No new configuration flags or schema changes.
- No large-scale rewrites that change service ownership.

## Definition of Done
- API and UI entrypoints are modularized by domain with thin top-level composition files.
- Architecture docs match actual layout and mention all runtime surfaces.
- Documentation health >=95% with missing/misplaced docs resolved.
- Build artifacts no longer live in `api/` and are ignored going forward.
