# Deployment — Nutrition Planner

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness.

The working product name is **Nooch** (formerly Daily; decision D-042). Scenario id is `nutrition-planner`.
The **local single-user tier is the primary and only active deployment**.
Every other tier is a future capability, not a commitment.

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

## Supported Tiers

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| **Local single-user** | **active / primary** | Vrooli lifecycle, Go toolchain, Node/pnpm for the UI build, writable SQLite path, a declared `authentication` profile (`hybrid`, default mode `personal_local`) | The scenario starts and reports healthy, but **every workspace RPC returns `401 unauthenticated`** because `.vrooli/service.json` declares no authentication profile (blocker B1, decision D-032). No product flow works until that is fixed and the redesign's D0 foundation repair lands (`docs/internal/REDESIGN_PLAN.md` §5). |
| Desktop / mobile packaging | future capability (deferred) | Packaged API+UI runtime, storage resolver, install assets, offline read policy | Cross-platform readiness and a storage story that does not write inside the app bundle. |
| Managed cloud / SaaS | deferred | Hosted runtime, auth, per-tenant isolation, observability, cost model | Contradicts the self-hosted, no-custody posture unless deliberately re-scoped. Requires R3 and commercial review. |
| Enterprise / self-host | deferred | Install docs, backup/restore, support model, optional auth integration | Operational hardening; this is the tier the self-hosted product eventually grows into. |

The local single-user tier is not a stepping stone that gets retired. The
product's privacy and ownership guarantees (`SYS-03`, `SYS-04`, `BIZ-01`)
are simplest and strongest when the data stays on the user's own machine.
Moving the runtime is a product decision with security consequences, not a
hosting detail.

## Runtime Requirements

| Requirement | Detail |
|---|---|
| API | Go binary. Listens on `API_PORT` (lifecycle band `15000-19999`), health at `/health`. |
| UI | Built Vite/React bundle served by `ui/server.js` on `UI_PORT` (`20000-24999`), health at `/health`; proxies `/api/*` and the Connect RPC namespace to the API. |
| Storage | SQLite file, resolved from the scenario id by `api-core/storage`. No environment variable may point one scenario at another's database. |
| Authentication | The lifecycle must supply authentication configuration from an `authentication` block in `.vrooli/service.json` (see repository `docs/concepts/IDENTITY-AND-AUTHENTICATION.md`; `scenarios/git-control-tower/.vrooli/service.json` is the `hybrid`/`personal_local` exemplar). Without it `api/main.go` installs no `authn` middleware and no principal exists. **Not yet declared (planned, D-032).** |
| External services | **None required at runtime.** SQLite is in-process and curated artwork ships as bundled assets. Optional: `image-tools` (routing models through `ai-gateway`) for in-app generation, `personal-planner` for calendar links, `ai-gateway` for assisted import. Producing the curated artwork during development uses `image-tools` (D-029). |
| Media assets | Curated scene, editorial, equipment, and ingredient assets with a versioned manifest under the UI's public assets root; only the current appearance and composition load on first paint (R17.4, R19, R25.4). **Planned** — no food or scene media exists yet. |
| Fonts | Self-hosted editorial serif and UI sans subsets with recorded licences; the PDF renderer embeds its fonts instead of reading `/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf` from the host (blocker B14). |
| Provider/AI credentials | Held in the platform secret store and read by reference at use time; never stored in scenario config, the database, or an export (`SYS-03`, `SYS-04`). Optional; an unconfigured provider is a supported state. |
| Network | Local API/UI communication at R0. R2 provider and URL-fetch adapters require outbound network access and are opt-in. |
| Clock/timezone | UTC instants plus IANA timezones and local dates for planning (`SYS-06`). The host clock must be correct for scheduled drafts. |

Preserve the guardrail: a host that is missing a provider, a model, or
network access must still serve recipes, editing, existing plans, and owned
data export. Deployment must never make a core workflow depend on an
optional adapter.

## Packaging

| Surface | Packaging Details |
|---|---|
| API | Go binary built by the scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`; PWA install assets (`site.webmanifest`, icons, service worker) ship with the bundle. |
| CLI | Go CLI installed through scenario manifest install hooks; a typed mirror of the API, not a second planner. |
| Proto | Schemas live under `packages/proto/schemas/nutrition-planner/`; generated clients are shared artifacts and are not hand-edited. |
| Native data | `daily.recipes` and `daily.workspace` JSON envelopes at `schemaVersion: 2` (`UX-DAT-05`, spec §20.9). Export is a release artifact, not a side feature. |
| Printable output | Vector recipe, week, and grocery PDFs in the light print theme with embedded fonts, plus shopping CSV (R25.2, `UX-DAT-04`, `UX-REC-07`). PDFs are generated server-side with the real page size and content. Today's PDFs still use the old direction and a host font. |
| Curated artwork | Optimized WebP/AVIF renditions plus fallbacks and the asset manifest (provenance, rights, approval, hashes). Originals and private source photos stay out of the client bundle (R19.3). |
| Demo seed | A repeat-safe command that loads the vegan fixture catalog into a separate demo workspace. **Planned**; never part of the normal boot path (R27.1). |

## Release Checklist

- [ ] `make setup` passes and `make test` passes for the release's phases.
- [ ] PRD operational targets have linked requirements, and the release's
      acceptance tests (`ACT-*`) pass or are listed as scoped deviations.
- [ ] No scaffold placeholder or unresolved template marker remains in release-facing docs.
- [ ] The scaffold `notes` example domain is removed or explicitly
      retained with justification (see [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md)).
- [ ] The local runtime authenticates: a fresh workspace can be created and
      read through the UI and CLI with no `401` (blocker B1).
- [ ] Provider-optional release: the app starts and all core flows work
      with no provider, no model, no image generation, and no network.
- [ ] Every primary surface, Explore, recipe detail, cooking, and equipment
      captured at the R27.5 viewports in Light and Evening and reviewed against
      `docs/reference/mockups/` with no open fidelity findings in
      `docs/internal/REDESIGN_LEDGER.md` (`OT-P0-021`).
- [ ] Asset manifest validates; measured media bytes meet the R25.4 budgets or
      carry a recorded exception; no demo data is reachable from a real
      workspace.
- [ ] Schema migrations have been tested on a **populated** prior database,
      not only an empty one (`SYS-05`).
- [ ] **Export/backup round-trip verified as part of the release**: export,
      restore into a clean workspace, and compare semantic content —
      including unknown/null values (`ACT-029`).
- [ ] Real documents inspected, not just returned with HTTP 200: recipe PDF,
      weekly PDF with open slots, and CSV with commas/quotes/Unicode
      (`ACT-031`, `ACT-032`, `ACT-033`).
- [ ] Secrets audit: no token, provider key, or billing identifier appears
      in the client bundle, an export, a log, or a generated PDF.
- [ ] `docs/manifest.json` maturity reflects the real state of these documents.
- [ ] Schema migration version is recorded and an upgrade from the previous
      release's populated database is tested (D-034, AT-051).
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, and `SECURITY.md` are active; the
      business docs are honestly marked as hypothesis/deferred.

## Rollback

Local development rollback is source-control based. For any future
packaged or managed target, document the tier-specific rollback path before
release. Two constraints are not optional:

- **Never roll back to a schema that drops immutable recipe revisions,
  historical plan/occurrence references, intake events, batches, or
  idempotency records.** Doing so would rewrite history or make a retry
  double-apply a purchase, consumption, import, or replan (`DOM-07` #12,
  `SYS-02`). A rollback touching those tables needs the same explicit
  decision record a migration does.
- **Restore from a recoverable checkpoint.** A full restore stages and
  validates first and creates a checkpoint before replacing domain state;
  if restoration fails, the previous usable state must remain available
  (`SYS-05`). Prefer additive schema changes so rollback is safe by
  construction.

The safe operational sequence is: **stop writes, back up, checkpoint, then
roll back** — never roll a database back underneath a live process.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures and recovery
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health checks and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency contracts
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — what must never leave the host
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — reference dataset and budgets
- [`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) — current blockers and the redesign build order
- [`../reference/product-specification.md`](../reference/product-specification.md) — R19, R25, R28, R30, and Appendix A sections 11, 18, 19, and 22
