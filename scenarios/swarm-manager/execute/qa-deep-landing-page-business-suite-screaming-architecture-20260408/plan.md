# Implementation Plan

## Purpose
Realign the `landing-page-business-suite` scenario so its physical structure and documentation clearly express its domain boundaries (landing, billing, downloads, AI gateway, remote profiles, user-auth, admin), shrink the API and UI god-entrypoints into thin composers, and bring the architecture docs + manifest into sync with reality. Behavior must not change.

**This is a brownfield refactor**: no greenfield rewrite, no new features, no endpoint shape changes. All work is structural + documentation.

## Required Reading
```bash
prompt-manager skill read screaming-architecture-audit
prompt-manager skill read documentation-health
prompt-manager skill read refactor
prompt-manager skill read domain-compression
prompt-manager skill read boundary-of-responsibility-enforcement
prompt-manager skill read knowledge-observatory-tools
knowledge-observatory docs read landing-page-business-suite architecture
knowledge-observatory docs read landing-page-business-suite seams
```

## Problem Statement
The scenario has grown into ~8 domains (landing, billing, downloads, metrics, AI gateway, remote profiles, user-auth, admin) but the codebase still reads as a monolith:

- `scenarios/landing-page-business-suite/api/main.go:33-67` — `Server` struct wires 20+ services flat; single-function initialization masks domain boundaries.
- `scenarios/landing-page-business-suite/api/routes.go:9-29` — 18 domain route groups registered from one `setupRoutes` func; no per-domain module.
- `scenarios/landing-page-business-suite/ui/src/App.tsx:1-11,43` — imports shared UI from `shared/ui` and `surfaces/user-auth`, neither of which is documented.
- `scenarios/landing-page-business-suite/docs/concepts/ARCHITECTURE.md:90-115,138-141` — UI tree claims `ui/src/app/App.tsx` + `shared/components`, and backend tree claims `api/initialization/postgres`. Both are wrong.
- Docs health score currently 0.88 with 2 misplaced docs, 6 missing baseline docs, 1 extra, 2 temporary. Target is >=0.95.

## Scope

In scope:
- Refactor API entrypoints (`api/main.go`, `api/routes.go`) into domain-aligned modules with behavior-preserving register functions and grouped dependency structs.
- Refactor UI `ui/src/App.tsx` into a thin composer over per-surface route modules.
- Update `docs/concepts/ARCHITECTURE.md`, add missing baseline docs under `docs/internal/`, relocate misplaced docs, handle extras.
- Remove/relocate build artifacts under `api/` and extend `.gitignore`.

Out of scope:
- Any endpoint path, request/response shape, auth semantics, or business-logic change.
- New features, new config flags, schema changes, data-model changes.
- Rewriting billing, downloads, AI gateway, or auth implementations.
- Changing the scenario's CLI or test harness.

### Acceptance Patterns
- `acceptance_allow`: `scenarios/landing-page-business-suite/**`
- `acceptance_deny`: (none — scenario is self-contained; no secrets dir to protect)

## Current Technical Context

Target files (primary edit surfaces):
- `scenarios/landing-page-business-suite/api/main.go` — Server struct + wiring
- `scenarios/landing-page-business-suite/api/routes.go` — route assembly
- `scenarios/landing-page-business-suite/api/*.go` — per-domain handlers/services currently colocated at package root
- `scenarios/landing-page-business-suite/ui/src/App.tsx` — UI entrypoint
- `scenarios/landing-page-business-suite/docs/concepts/ARCHITECTURE.md` — architecture doc
- `scenarios/landing-page-business-suite/docs/manifest.json` — docs manifest
- `scenarios/landing-page-business-suite/docs/internal/*` — missing baseline internal docs
- `scenarios/landing-page-business-suite/initialization/postgres/` — actual schema location

Confirmed domains (from `routes.go`): `health`, `landing`, `auth` (admin), `account`, `billing`, `admin-core`, `remote-profile`, `commerce-admin`, `variant`, `content`, `metrics`, `feedback`, `waitlist`, `credits`, `ai`, `docs`, `admin-user`, `update`.

Collapsed grouping used by this plan (8 domain modules):
`landing`, `billing` (billing + credits + commerce-admin), `downloads` (downloads + content + update), `ai` (ai + ai-gateway-deps), `metrics` (metrics + feedback + waitlist), `admin` (admin-core + admin-user + docs-admin), `remote-profile`, `user-auth` (auth + account + user-auth). `health` stays at top-level.

Docs health baseline (from `evidence/docs-health.json`):
- health_score: 0.88
- misplaced: `docs/plans/EXPERIENCE_AUDIT.md` → `docs/internal/EXPERIENCE-AUDIT.md`; `docs/reference/api/README.md` → `README.md`
- missing: `assumptions`, `coherence-notes`, `error-semantics`, `invariants`, `security-posture`, `temporal-flows`
- extra: `docs/FAQ.md`
- temporary: `docs/plans/IMPLEMENTATION_PLAN.md`, `support-agent-docs/06-email-templates.md`

## Target End State
- `api/main.go` contains only `Server` composition, start/stop, and config; per-domain dependency structs live in their domain modules.
- `api/routes.go` is a 5-10 line composer that calls `<domain>.RegisterRoutes(router, deps)` for each domain module.
- Each API domain module has its own file(s) with handlers, services, and a `RegisterRoutes` function (starts as file-level grouping; escalates to subpackages only if it does not introduce import cycles — see **Decision D1**).
- `ui/src/App.tsx` imports route modules from `ui/src/app/routes/{publicRoutes,adminRoutes,userAuthRoutes}.tsx` and is under ~40 lines.
- `docs/concepts/ARCHITECTURE.md` matches the actual tree (App at `ui/src/App.tsx`, shared UI at `shared/ui`, surfaces include `user-auth`, initialization path `initialization/postgres/`) and names CLI + AI gateway as runtime surfaces.
- `docs/manifest.json` registers every documented file and removes references to relocated/deleted ones.
- `knowledge-observatory docs health -scenario landing-page-business-suite` returns `health_score >= 0.95` with 0 misplaced docs, <=1 extra, 0 temporary, and missing-docs count reduced per **Decision D4**.
- `api/` contains no committed binaries/coverage artifacts; `.gitignore` blocks them.

## Implementation Strategy

### Phase 1 — Map + scaffold (read-only, then skeletons)
1. Run the reproduction commands from the spec and record current handler→domain mapping in scratch notes.
2. Create empty per-domain files (e.g., `api/domain_billing.go`, `api/domain_ai.go`, ...) with package-level `RegisterRoutes` stubs and grouped deps structs.
3. Run `go build ./...` in `scenarios/landing-page-business-suite/api/` to confirm a clean baseline.

### Phase 2 — API structure refactor (behavior-preserving)
1. For each of the 8 domains, move handlers + services into the domain file, expose a `RegisterRoutes(router, deps)` function, and replace the flat `Server` field with a grouped deps struct (`BillingDeps`, `AIDeps`, etc.).
2. Rewrite `api/routes.go` to call each domain's `RegisterRoutes`.
3. Shrink `api/main.go` to composition only.
4. Keep every URL path, HTTP method, middleware chain, and response shape identical.
5. If a domain's move triggers import cycles, stay at file-level grouping and document the reason in the domain file header comment.

### Phase 3 — UI entrypoint refactor
1. Extract route blocks from `ui/src/App.tsx` into `ui/src/app/routes/publicRoutes.tsx`, `adminRoutes.tsx`, `userAuthRoutes.tsx`.
2. App becomes providers + `<Routes>{publicRoutes}{adminRoutes}{userAuthRoutes}</Routes>`.
3. Preserve lazy/eager imports exactly as they were.

### Phase 4 — Documentation alignment
1. Update `docs/concepts/ARCHITECTURE.md`:
   - Replace `ui/src/app/App.tsx` with `ui/src/App.tsx`; replace `shared/components` with `shared/ui`; add `surfaces/user-auth/` branch.
   - Fix backend tree: `initialization/postgres/` (not `api/initialization/postgres/`).
   - Add CLI and AI gateway as runtime surfaces.
2. Move misplaced docs:
   - `docs/plans/EXPERIENCE_AUDIT.md` → `docs/internal/EXPERIENCE-AUDIT.md`
   - `docs/reference/api/README.md` → `README.md` (top-level, or the correct per-docs-health expected path)
3. Add missing baseline internal docs per **Decision D4** using `knowledge-observatory docs template <type>` as the starting template.
4. Handle extras/temporaries per **Decision D5** (`docs/FAQ.md`, `docs/plans/IMPLEMENTATION_PLAN.md`, `support-agent-docs/06-email-templates.md`).
5. Update `docs/manifest.json` to match the new layout.

### Phase 5 — Artifact cleanup
1. Identify non-source files under `api/` (binaries like `api/0`, coverage outputs, stray files) and remove.
2. Extend `scenarios/landing-page-business-suite/.gitignore` to cover `*.out`, `*.test`, `cover*.out`, bare binaries, etc. — only patterns that match actually-observed artifacts.

### Final step (mandatory)
```bash
vrooli scenario restart landing-page-business-suite
```
This re-bootstraps the scenario from its source tree so any cached/build state is rebuilt against the refactored layout.

## Contract Decisions
- No endpoint paths, methods, request/response shapes, auth semantics, rate-limit keys, or side-effects change.
- Domain module boundaries are **internal-only** refactors; external contracts are invariant.
- Documentation changes describe actual code; they do not invent behavior or commitments.
- No changes to `initialization/postgres/schema.sql` or `seed.sql` content — only doc path references.

## Testing Plan
Primary (required):
- `vrooli scenario test landing-page-business-suite` — must pass with same pass count as pre-refactor baseline.

Secondary (if env allows):
- `cd scenarios/landing-page-business-suite/api && go build ./... && go vet ./...`
- `cd scenarios/landing-page-business-suite/api && go test ./...` (only if deps already resolved)
- `cd scenarios/landing-page-business-suite/ui && pnpm typecheck` (or project-specific equivalent)

Smoke verification:
- `curl -fsS .../api/v1/health` still 200.
- `curl -fsS .../api/v1/landing-config` still 200 with same shape.
- Admin login flow still reachable through `surfaces/user-auth` route module.

Acceptance verification:
- `knowledge-observatory docs health -json -scenario landing-page-business-suite` reports `health_score >= 0.95`.
- Diff `api/routes.go` new vs old: every `register*Routes` call has a corresponding `<domain>.RegisterRoutes` call in the new file (no dropped domains).

## Rollout / Validation Checklist
- [ ] `go build ./...` clean in API dir.
- [ ] Route registration diff: 18 route groups → 8 `RegisterRoutes` calls, all domains accounted for.
- [ ] `Server` struct no longer holds flat per-service fields; grouped deps structs in place.
- [ ] `ui/src/App.tsx` is under ~40 lines and contains no inline `<Route>` declarations.
- [ ] `docs/concepts/ARCHITECTURE.md` references `ui/src/App.tsx`, `shared/ui`, `surfaces/user-auth`, and `initialization/postgres/`.
- [ ] `knowledge-observatory docs health` score >= 0.95.
- [ ] No build artifacts committed under `api/`; `.gitignore` updated.
- [ ] `vrooli scenario restart landing-page-business-suite` succeeds.
- [ ] `vrooli scenario test landing-page-business-suite` passes.

## Risks + Mitigations
- **Import cycles from Go subpackage split.**
  Mitigation: start at file-level grouping (same package) and only escalate to subpackages where the dep graph is obviously acyclic (see D1).
- **Route registration drop during migration.**
  Mitigation: maintain a checklist of all 18 old `register*Routes` calls; each must map to exactly one new domain `RegisterRoutes`. Diff before/after `routes.go` as part of validation.
- **UI route extraction breaking lazy-loading or providers.**
  Mitigation: move route definitions only; keep provider composition + lazy imports unchanged.
- **Docs update drifting again.**
  Mitigation: re-run `knowledge-observatory docs health` before marking done; add DOC/CODE bidirectional references where they cross the boundaries we are reshaping.
- **Artifact deletion wiping local dev state.**
  Mitigation: only delete files matching build-artifact patterns (binaries, `*.out`, coverage); do not touch source or config.
- **`vrooli scenario restart` failing post-refactor.**
  Mitigation: restart is the last step; any failure here is the signal to revisit and is the canonical regression check.

## Non-goals / Prohibited Patterns
- No behavioral changes to billing, downloads, AI gateway, auth, or admin flows.
- No new configuration flags, env vars, or schema migrations.
- No large-scale rewrites that change ownership of a service or introduce new infrastructure.
- No relaxing existing tests to make refactors pass.
- No changes to endpoint paths, response shapes, or auth semantics.

## Definition of Done
- API `Server` struct + `routes.go` decomposed into 8 domain modules with `RegisterRoutes` + grouped deps; `main.go`/`routes.go` are thin composers.
- UI App is a thin composer over per-surface route modules.
- Architecture doc + manifest match the actual tree; all runtime surfaces named.
- `knowledge-observatory docs health` score >= 0.95 with 0 misplaced, 0 temporary, baseline missing docs addressed per D4.
- No build artifacts under `api/`; `.gitignore` extended.
- `vrooli scenario restart landing-page-business-suite` succeeds and `vrooli scenario test landing-page-business-suite` passes.

## Pending Decisions
- **D1** — API refactor depth (file-level grouping vs Go subpackages).
- **D2** — Domain module grouping (8-module collapse vs 1:1 with current `register*Routes`).
- **D3** — UI route split granularity (3 surface modules vs finer per-feature split).
- **D4** — Missing internal-docs strategy (create all 6 vs create mandatory subset).
- **D5** — Extra/temporary docs handling (move to internal, delete, or leave with suppression).
