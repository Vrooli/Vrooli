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
- Refactor API entrypoints (`api/main.go`, `api/routes.go`) to extract a chosen subset of domains into real Go subpackages under `api/domain/<name>/` with `RegisterRoutes` + grouped deps structs (per **Decisions D1 + D2.1 + D2.2**).
- Refactor UI `ui/src/App.tsx` into a thin composer over three surface-aligned route modules (`publicRoutes`, `adminRoutes`, `userAuthRoutes`).
- Update `docs/concepts/ARCHITECTURE.md`, add the 6 missing baseline docs under `docs/internal/`, relocate misplaced docs, handle FAQ + temporary docs.
- Remove/relocate build artifacts under `api/` and extend `.gitignore`.
- File a single follow-up `execute` backlog item covering the deferred domains (per **Decision D2.4**).

Out of scope:
- Any endpoint path, request/response shape, auth semantics, or business-logic change.
- New features, new config flags, schema changes, data-model changes.
- Rewriting billing, downloads, AI gateway, or auth implementations.
- Changing the scenario's CLI or test harness.
- Extracting any new shared package (e.g., `api/shared/`, `api/platform/`) mid-execution — explicitly forbidden by **Decision D2.3**.

### Acceptance Patterns
- `acceptance_allow`: `scenarios/landing-page-business-suite/**`
- `acceptance_deny`: (none — scenario is self-contained; no secrets dir to protect)

## Settled Decisions

### Round 1
- **D1 — API refactor depth: Full Go subpackages** (user override; recommendation was file-level grouping). User note: *"If it's too much to do in one execution, pick a few domains and do those properly. If going that route, create a new backlog item or items to follow-up with the remaining refactoring later."* — drives the phased subset approach below.
- **D2 — Domain grouping: 8-module collapse** (`landing`, `billing` [+credits +commerce-admin], `downloads` [+content +update], `ai`, `metrics` [+feedback +waitlist], `admin` [+admin-user +docs-admin], `remote-profile`, `user-auth` [+auth +account]; `health` stays top-level).
- **D3 — UI route split: 3 surface-aligned modules** (`publicRoutes`, `adminRoutes`, `userAuthRoutes`).
- **D4 — Missing baseline docs: create all 6** using `knowledge-observatory docs template` starters (`assumptions`, `coherence-notes`, `error-semantics`, `invariants`, `security-posture`, `temporal-flows`).
- **D5 — Extra/temporary docs: move FAQ → `docs/guides/faq.md` + register in manifest; delete both temporary docs** (`docs/plans/IMPLEMENTATION_PLAN.md`, `support-agent-docs/06-email-templates.md`).

### Round 2
- **D2.1 — Subpackage path convention: `api/domain/<name>/`** (e.g., `api/domain/ai/`, `api/domain/remote-profile/`, `api/domain/user-auth/`, `api/domain/metrics/`). The `domain/` directory itself screams architecture and is trivially greppable for "all domains."
- **D2.2 — Execution subset: subpackage the 4 most isolated domains now** — `ai`, `remote-profile`, `user-auth`, `metrics`. Defer `billing`, `downloads`, `admin`, `landing` to the follow-up item. These 4 touch the fewest shared helpers (AI has its own `AIGatewayDeps`; remote-profile is a single service; user-auth owns its rate limiter + session; metrics is read-mostly), so cycle risk is lowest.
- **D2.3 — Cycle-resolution policy: revert + defer.** If moving a domain into `api/domain/<name>/` triggers an import cycle that would require extracting a new shared package, revert that domain's move and add it to the deferred set covered by the follow-up item. **Do not extract `api/shared/`, `api/platform/`, or any other new shared package in this execution.** Consumer-defined interfaces are also not to be introduced as a workaround in this run.
- **D2.4 — Follow-up backlog shape: single `execute` item** covering all deferred domains, inheriting `acceptance_allow = scenarios/landing-page-business-suite/**`. The follow-up's plan can sequence domains itself and is the right place to deliberately design `api/shared/` if cycle work is needed.

## Current Technical Context

Target files (primary edit surfaces):
- `scenarios/landing-page-business-suite/api/main.go` — Server struct + wiring (becomes thin composer)
- `scenarios/landing-page-business-suite/api/routes.go` — route assembly (becomes thin composer)
- `scenarios/landing-page-business-suite/api/*.go` — per-domain handlers/services currently colocated at package root
- `scenarios/landing-page-business-suite/api/domain/ai/*.go` — new subpackage destination for `ai` domain
- `scenarios/landing-page-business-suite/api/domain/remote-profile/*.go` — new subpackage destination for `remote-profile` domain
- `scenarios/landing-page-business-suite/api/domain/user-auth/*.go` — new subpackage destination for `user-auth` domain (groups current `auth` + `account`)
- `scenarios/landing-page-business-suite/api/domain/metrics/*.go` — new subpackage destination for `metrics` domain (groups current `metrics` + `feedback` + `waitlist`)
- `scenarios/landing-page-business-suite/ui/src/App.tsx` — UI entrypoint
- `scenarios/landing-page-business-suite/ui/src/app/routes/{publicRoutes,adminRoutes,userAuthRoutes}.tsx` — new route modules
- `scenarios/landing-page-business-suite/docs/concepts/ARCHITECTURE.md` — architecture doc
- `scenarios/landing-page-business-suite/docs/manifest.json` — docs manifest
- `scenarios/landing-page-business-suite/docs/internal/*` — missing baseline internal docs
- `scenarios/landing-page-business-suite/docs/guides/faq.md` — relocated FAQ
- `scenarios/landing-page-business-suite/initialization/postgres/` — actual schema location

Confirmed domains (from `routes.go`): `health`, `landing`, `auth` (admin), `account`, `billing`, `admin-core`, `remote-profile`, `commerce-admin`, `variant`, `content`, `metrics`, `feedback`, `waitlist`, `credits`, `ai`, `docs`, `admin-user`, `update`.

Collapsed grouping (D2 = A): `landing`, `billing` (billing + credits + commerce-admin), `downloads` (downloads + content + update), `ai` (ai + ai-gateway-deps), `metrics` (metrics + feedback + waitlist), `admin` (admin-core + admin-user + docs-admin), `remote-profile`, `user-auth` (auth + account + user-auth). `health` stays at top-level.

Execution subset (D2.2 = A):
- **Subpackaged this run** under `api/domain/<name>/`: `ai`, `remote-profile`, `user-auth`, `metrics`.
- **Deferred to follow-up**: `landing`, `billing`, `downloads`, `admin` (these stay in the flat `api` package and continue to be registered via the existing `register*Routes(s)` calls).

Docs health baseline (from `evidence/docs-health.json`):
- health_score: 0.88
- misplaced: `docs/plans/EXPERIENCE_AUDIT.md` → `docs/internal/EXPERIENCE-AUDIT.md`; `docs/reference/api/README.md` → `README.md`
- missing: `assumptions`, `coherence-notes`, `error-semantics`, `invariants`, `security-posture`, `temporal-flows`
- extra: `docs/FAQ.md` → relocate to `docs/guides/faq.md`
- temporary: `docs/plans/IMPLEMENTATION_PLAN.md`, `support-agent-docs/06-email-templates.md` → delete

## Target End State
- `api/main.go` contains only `Server` composition, start/stop, and config; per-domain dependency structs for the 4 subpackaged domains live in their domain subpackages under `api/domain/<name>/`.
- `api/routes.go` is a thin composer that calls `<domain>.RegisterRoutes(router, deps)` for each subpackaged domain (`ai`, `remote-profile`, `user-auth`, `metrics`) and the legacy `register*Routes(s)` for the four deferred domains (`landing`, `billing`, `downloads`, `admin`) plus `health`.
- Each subpackaged domain lives at `api/domain/<name>/` with handlers, services, and an exported `RegisterRoutes` function.
- `ui/src/App.tsx` imports route modules from `ui/src/app/routes/{publicRoutes,adminRoutes,userAuthRoutes}.tsx` and is under ~40 lines.
- `docs/concepts/ARCHITECTURE.md` matches the actual tree (App at `ui/src/App.tsx`, shared UI at `shared/ui`, surfaces include `user-auth`, initialization path `initialization/postgres/`) and names CLI + AI gateway as runtime surfaces.
- `docs/concepts/ARCHITECTURE.md` includes a transitional callout that lists the 4 deferred domains and points at the follow-up backlog item, so mixed subpackaged + flat state is not silently confusing.
- `docs/manifest.json` registers every documented file (including the 6 new internal docs and relocated FAQ) and removes references to relocated/deleted ones.
- `knowledge-observatory docs health -scenario landing-page-business-suite` returns `health_score >= 0.95` with 0 misplaced docs, <=1 extra, 0 temporary, and all 6 baseline docs present.
- `api/` contains no committed binaries/coverage artifacts; `.gitignore` blocks them.
- A single follow-up `execute` backlog item is filed (per **Decision D2.4**) covering `landing`, `billing`, `downloads`, `admin` plus any domains additionally deferred at runtime under **D2.3**.

## Implementation Strategy

### Phase 1 — Map + scaffold (read-only, then skeletons)
1. Run the reproduction commands from the spec and record the current handler→domain mapping in scratch notes.
2. Build a per-domain dependency graph for the 4 execution-subset domains (`ai`, `remote-profile`, `user-auth`, `metrics`): list which top-level helpers each touches (rate limiters, session manager, shared DTOs, logging middleware). Confirm acyclic before any move.
3. Create empty subpackage skeletons under `api/domain/<name>/` for each of the 4 with `RegisterRoutes` stubs and grouped deps structs.
4. Run `go build ./...` in `scenarios/landing-page-business-suite/api/` to confirm a clean baseline.

### Phase 2 — API subpackage refactor (behavior-preserving, phased)
For each of the 4 execution-subset domains, in order of lowest dep risk first (recommended order: `ai` → `remote-profile` → `metrics` → `user-auth`):

1. Move handlers + services into `api/domain/<name>/`; keep file/symbol names where possible (rename only to satisfy export rules).
2. Define the domain's grouped deps struct (e.g., `ai.Deps`) with only the helpers that domain actually consumes.
3. Export a `RegisterRoutes(router *mux.Router, deps Deps)` function with the same path/method/middleware as the prior `register*Routes(s)`.
4. Update `api/routes.go` to call `<domain>.RegisterRoutes(s.router, <domain>Deps(s))` instead of the old `register*Routes(s)`.
5. Run `go build ./...` and any available `go test ./...` after each domain.
6. **Cycle handling (D2.3):** if a move surfaces an import cycle that would require extracting a new shared package, revert that domain's move (`git restore` the moved files), leave it in the flat `api` package, and add it to the deferred list for the follow-up item. Do not extract `api/shared/`, `api/platform/`, or any other new package to break the cycle in this execution.
7. Shrink `api/main.go` so removed services no longer appear on the `Server` struct (they live inside the domain subpackage instead).
8. The 4 deferred domains (`landing`, `billing`, `downloads`, `admin`) plus any additional reverts from step 6 stay at file-level grouping in the existing `api` package and are listed in the follow-up backlog item.

Behavior invariants (must hold for every moved domain):
- Every URL path, HTTP method, middleware chain, status code, and response body shape identical to pre-refactor.
- Logging tags, metrics labels, and rate-limit keys unchanged.
- Auth middleware applied at the same boundary it was before.

### Phase 3 — UI entrypoint refactor
1. Extract route blocks from `ui/src/App.tsx` into `ui/src/app/routes/publicRoutes.tsx`, `adminRoutes.tsx`, `userAuthRoutes.tsx`.
2. App becomes providers + `<Routes>{publicRoutes}{adminRoutes}{userAuthRoutes}</Routes>`.
3. Preserve lazy/eager imports exactly as they were.

### Phase 4 — Documentation alignment
1. Update `docs/concepts/ARCHITECTURE.md`:
   - Replace `ui/src/app/App.tsx` with `ui/src/App.tsx`; replace `shared/components` with `shared/ui`; add `surfaces/user-auth/` branch.
   - Fix backend tree: `initialization/postgres/` (not `api/initialization/postgres/`).
   - Add CLI and AI gateway as runtime surfaces.
   - Reflect the new `api/domain/<name>/` layout for the 4 subpackaged domains; add a "Transitional layout" callout naming the 4 deferred domains (`landing`, `billing`, `downloads`, `admin`) plus a pointer to the follow-up backlog item.
2. Move misplaced docs:
   - `docs/plans/EXPERIENCE_AUDIT.md` → `docs/internal/EXPERIENCE-AUDIT.md`
   - `docs/reference/api/README.md` → `README.md`
3. Add all 6 missing baseline internal docs (D4 = A) using `knowledge-observatory docs template <type>` as the starting template: `assumptions`, `coherence-notes`, `error-semantics`, `invariants`, `security-posture`, `temporal-flows`.
4. Relocate `docs/FAQ.md` → `docs/guides/faq.md` and register in manifest. Delete `docs/plans/IMPLEMENTATION_PLAN.md` and `support-agent-docs/06-email-templates.md` (D5 = A).
5. Update `docs/manifest.json` to match the new layout.

### Phase 5 — Artifact cleanup
1. Identify non-source files under `api/` (binaries like `api/0`, coverage outputs, stray files) and remove.
2. Extend `scenarios/landing-page-business-suite/.gitignore` to cover `*.out`, `*.test`, `cover*.out`, bare binaries, etc. — only patterns that match actually-observed artifacts.

### Phase 6 — Follow-up backlog item
File one new `execute` backlog item per **Decision D2.4** covering the deferred domains. The item must:
- Reference this plan and the round-002 + round-003 decisions.
- List the 4 deferred domains by name (`landing`, `billing`, `downloads`, `admin`) plus any additional reverts that happened under D2.3 during execution, with the cycle/dep reason recorded for each.
- Inherit `acceptance_allow = scenarios/landing-page-business-suite/**`.
- Note that the follow-up plan is the right place to deliberately design any new `api/shared/` (or equivalent) package needed to break cycles for the deferred domains.

### Final step (mandatory)
```bash
vrooli scenario restart landing-page-business-suite
```
This re-bootstraps the scenario from its source tree so any cached/build state is rebuilt against the refactored layout.

## Contract Decisions
- No endpoint paths, methods, request/response shapes, auth semantics, rate-limit keys, or side-effects change.
- Domain subpackage boundaries are **internal-only** refactors; external contracts are invariant.
- Documentation changes describe actual code; they do not invent behavior or commitments.
- No changes to `initialization/postgres/schema.sql` or `seed.sql` content — only doc path references.
- Mixing `api/domain/<name>/` subpackaged domains + legacy flat-package domains in `routes.go` for the duration of the rollout is explicitly accepted, in service of the phased approach driven by D1 + D2.2 + D2.3.
- No new shared packages (`api/shared/`, `api/platform/`, etc.) are introduced in this execution; cycle handling is revert-and-defer only.

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
- One smoke per subpackaged domain (`ai`, `remote-profile`, `user-auth`, `metrics`) hitting a representative route.

Acceptance verification:
- `knowledge-observatory docs health -json -scenario landing-page-business-suite` reports `health_score >= 0.95`.
- Diff `api/routes.go` new vs old: every `register*Routes` call has an equivalent `<domain>.RegisterRoutes` (subpackaged) or surviving `register*Routes` (deferred) — zero domains dropped.
- `go list ./...` in the API dir shows the 4 new `api/domain/<name>/` subpackages and no import-cycle errors.

## Rollout / Validation Checklist
- [ ] Per-domain dep graph captured for `ai`, `remote-profile`, `user-auth`, `metrics` before any move.
- [ ] `go build ./...` clean in API dir after each domain move.
- [ ] Route registration diff: every old `register*Routes` accounted for (subpackaged or deferred).
- [ ] `Server` struct no longer holds fields owned by subpackaged domains.
- [ ] Any domain that hit a cycle was reverted and added to the deferred list (no new shared packages introduced).
- [ ] `ui/src/App.tsx` is under ~40 lines and contains no inline `<Route>` declarations.
- [ ] `docs/concepts/ARCHITECTURE.md` references `ui/src/App.tsx`, `shared/ui`, `surfaces/user-auth`, `initialization/postgres/`, and the new `api/domain/<name>/` layout, and includes the transitional callout listing deferred domains.
- [ ] All 6 missing baseline internal docs created.
- [ ] FAQ relocated and registered; both temporary docs deleted.
- [ ] `knowledge-observatory docs health` score >= 0.95.
- [ ] No build artifacts committed under `api/`; `.gitignore` updated.
- [ ] Single follow-up `execute` backlog item filed listing the deferred domains (`landing`, `billing`, `downloads`, `admin`, plus any runtime reverts).
- [ ] `vrooli scenario restart landing-page-business-suite` succeeds.
- [ ] `vrooli scenario test landing-page-business-suite` passes.

## Risks + Mitigations
- **Import cycles from Go subpackage split.**
  Mitigation: build dep graph in Phase 1 for the 4 execution-subset domains; on cycle discovery during Phase 2, revert that domain's move and add it to the deferred list per **D2.3**. Do not extract a new shared package mid-execution.
- **Route registration drop during migration.**
  Mitigation: maintain a checklist of all 18 old `register*Routes` calls; each must map to a subpackaged `RegisterRoutes` or remain in the legacy composer. Diff before/after `routes.go` as part of validation.
- **Mixed `api/domain/<name>/` + legacy flat state confusing future readers.**
  Mitigation: ARCHITECTURE.md transitional callout names the deferred domains explicitly + points at the follow-up backlog item.
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
- No new shared packages (`api/shared/`, `api/platform/`, consumer-defined-interface shims) introduced in this execution to break import cycles.

## Definition of Done
- API `Server` struct + `routes.go` decomposed: the 4 subpackaged domains (`ai`, `remote-profile`, `user-auth`, `metrics`) expose `RegisterRoutes` + grouped deps under `api/domain/<name>/`; legacy composer paths remain only for the 4 deferred domains (`landing`, `billing`, `downloads`, `admin`) and any runtime reverts, all documented in the follow-up item.
- UI App is a thin composer over per-surface route modules.
- Architecture doc + manifest match the actual tree; all runtime surfaces named; subpackaged + transitional sections both reflected.
- All 6 baseline internal docs present; FAQ relocated; both temporary docs deleted.
- `knowledge-observatory docs health` score >= 0.95 with 0 misplaced, 0 temporary.
- No build artifacts under `api/`; `.gitignore` extended.
- Single follow-up `execute` backlog item exists listing every deferred domain.
- `vrooli scenario restart landing-page-business-suite` succeeds and `vrooli scenario test landing-page-business-suite` passes.
