# Notes

## Required Fields
- targetScenario: landing-page-business-suite
- problemOrOpportunity: Architecture and docs no longer express the scenario's domain boundaries; entrypoints are monolithic and docs drift from actual layout.
- proposedAction: Restructure API/UI entrypoints into domain-aligned modules, reduce "god" files, align docs and manifest with reality, and remove build artifacts from source tree.
- evidence: evidence/observations.md, evidence/docs-health.json
- riskLevel: medium
- executionModeHint: manual
- createdByTeam: scenario-qa
- sourceRunId: quality-auditor-2026-04-08

## Problem
Landing-page-business-suite has grown into multiple domains (landing, billing, downloads, AI gateway, remote profiles), but the physical layout still reads as a monolith. API wiring and route registration live in single mega files, and the architecture doc no longer matches actual UI/backend structure. This makes the codebase harder to navigate, increases coupling risk, and undermines the documented mental model that downstream agents rely on.

## Documentation Health
| Area | Status | Notes |
| --- | --- | --- |
| docs/manifest.json | Present | docs manifest exists (see evidence/docs-health.json) |
| Mental model documented | Partial | Architecture doc exists but UI tree + initialization paths are outdated |
| Code<->Doc references | Partial (~20% coverage) | `DOC:` and `CODE:` links exist but only for a subset of modules |
| Orphaned docs | 1 file | docs/FAQ.md reported as extra |
| Broken references | 0 reported | Docs health reports 2 misplaced docs + 6 missing docs |

Missing docs per health report: assumptions, coherence-notes, error-semantics, invariants, security-posture, temporal-flows. Misplaced docs: docs/plans/EXPERIENCE_AUDIT.md, docs/reference/api/README.md. (See evidence/docs-health.json.)

## Top Violations
1. `scenarios/landing-page-business-suite/api/main.go:33-67` - Server struct wires 20+ services in one type; initialization happens in a single function, masking domain boundaries.
2. `scenarios/landing-page-business-suite/api/routes.go:9-29` - Route registration registers 18 domains in one file; no domain-aligned module boundaries.
3. `scenarios/landing-page-business-suite/docs/concepts/ARCHITECTURE.md:90-115` - UI tree lists `ui/src/app/App.tsx` and `shared/components`, but actual entrypoint is `ui/src/App.tsx` and shared components live under `shared/ui`.
4. `scenarios/landing-page-business-suite/ui/src/App.tsx:1-11,43` - Entry point imports shared UI and a `surfaces/user-auth` surface that is not documented in the architecture doc.
5. `scenarios/landing-page-business-suite/docs/concepts/ARCHITECTURE.md:138-141` - Backend initialization path documented under `api/initialization/postgres`, but actual path is `initialization/postgres/` at scenario root.

## Impact
- Architectural intent is hard to discover quickly, increasing onboarding time and risk of accidental cross-domain coupling.
- Docs drift makes agent-driven changes risky (agents follow stale layout assumptions).
- Build artifacts in source tree blur the boundary between source and runtime outputs.

## Reproduction
```bash
sed -n '33,120p' scenarios/landing-page-business-suite/api/main.go
sed -n '9,30p' scenarios/landing-page-business-suite/api/routes.go
sed -n '90,140p' scenarios/landing-page-business-suite/docs/concepts/ARCHITECTURE.md
sed -n '1,45p' scenarios/landing-page-business-suite/ui/src/App.tsx
knowledge-observatory docs health -json -scenario landing-page-business-suite
```

## Success Criteria
- API/UI entrypoints are decomposed into domain-aligned modules; top-level files are thin composition layers.
- Architecture docs and manifest reflect actual layout (UI tree, initialization paths, CLI mention).
- Documentation health >=95% (missing + misplaced docs resolved).
- Build/test artifacts removed from repo or moved under a dedicated build output path with .gitignore coverage.

## Proposed Action
1. Create domain modules (e.g., `api/domain/{landing,billing,downloads,metrics,ai,admin,auth,remote}` or equivalent) and move handlers/services accordingly; keep `main.go` and `routes.go` as thin composition layers with explicit `RegisterRoutes` per module.
2. Introduce grouped dependency structs (e.g., `BillingDeps`, `DownloadsDeps`, `AIDeps`) to reduce the `Server` struct surface area and clarify ownership.
3. Split `ui/src/App.tsx` routing into domain route modules (`app/routes/publicRoutes.tsx`, `app/routes/adminRoutes.tsx`, `app/routes/userAuthRoutes.tsx`), keeping App as compositor only.
4. Update `docs/concepts/ARCHITECTURE.md` UI and backend tree to match actual layout; add CLI and AI gateway to architecture overview where appropriate.
5. Fix docs health issues (move misplaced docs, add missing baseline docs, remove/relocate extra or temporary docs). Re-run `knowledge-observatory docs health` to verify >=95%.
6. Remove or relocate build artifacts under `api/` (binaries, coverage outputs, stray files like `api/0`) and add/update `.gitignore` to prevent recurrence.
