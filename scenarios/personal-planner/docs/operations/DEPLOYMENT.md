# Deployment — Personal Planner

## Purpose Of This Document

This document records where Personal Planner can run, the runtime
assumptions each tier requires, how the scenario is packaged, and what
must pass before a release. It is design-stage: only the local
development tier is real today, and only the `/health` endpoints and the
worked-example `notes` domain are running code. Every other tier and job
described here is the **intended** shape from the implementation plan, not
a deployed capability. Where this document describes future runtime
behavior it says so plainly; do not read it as a claim that a cloud,
desktop, or PWA install exists.

Use this document to answer:

- Where can this scenario run today, and where is it planned to run?
- What runtime assumptions must hold for each tier?
- How are the API, UI, and CLI packaged?
- What gates a release, and how is a release rolled back?

## Supported Tiers

| Tier | Status | Requirements | Notes / Blockers |
|---|---|---|---|
| Local Vrooli stack | **active (current tier)** | Vrooli lifecycle, Go, Node 20+/pnpm 9+, writable SQLite path, scenario-authenticator running | The only real deployment surface today. Reference `notes` domain must be replaced before this is a product deployment. |
| scenario-to-cloud (VPS) | planned | Hosted runtime, managed route + auth integration, persistent SQLite volume (or a justified relational resource), job infrastructure, observability, cost model | Adopted through the platform's scenario-to-cloud path. Not yet certified for this scenario. |
| Desktop packaging | planned | Cross-platform runtime, packaged UI/API, storage resolver, signing/update transport | Requires the platform's desktop packaging + cross-machine validation. |
| PWA install | planned | Scenario-specific brand assets (icons/manifest), app-shell service worker, HTTPS origin | Baseline PWA metadata ships from the template; brand assets are placeholders until replaced (see Packaging). |

The three planned tiers are future work. Do not enable them by
hand-editing lifecycle or deployment metadata; they arrive through the
platform's owning deployment path so that ports, routes, auth, and
health wiring stay managed.

## Runtime Requirements

The lifecycle assigns and exposes everything below; do not hardcode
ports, hostnames, tokens, or user identities.

- **Go API** — built by the scenario lifecycle; listens on `API_PORT`
  (range 15000–19999); serves `/health`.
- **Node UI server** — Vite production bundle served by `ui/server.js`;
  listens on `UI_PORT` (range 20000–24999); serves `/health`.
- **SQLite (in-process)** — the default authoritative storage substrate,
  resolved from the scenario id; needs a writable, persistent data path.
  No external storage process in R1. A relational resource (e.g.
  Postgres) is adopted only under the single documented condition (the
  partial-unique one-exclusive-session guard at scale) and only through
  the Scenario Dependency Analyzer — never hand-edited. See
  [`../concepts/DATA.md`](../concepts/DATA.md).
- **scenario-authenticator (required)** — authoritative identity and
  private-workspace isolation, plus the recipient-bound viewer identity
  used by selective sharing. The scenario **refuses to start** without an
  identity provider (`startup_policy: must_start`); a workspace cannot be
  scoped to a subject without it.
- **notification-hub (optional)** — external delivery of reminders,
  latest-safe-stopping-time cues, and material deadline-risk notices with
  quiet hours and receipts (`startup_policy: try_start`). When absent,
  notices remain visible **in-app only**; the product stays fully usable.
- **External-calendar credentials (optional feature)** — read-only
  provider tokens (Google first) held by the credential owner, never in
  logs, bundles, exports, or docs. Absence only disables the optional
  import feature.
- **Timezone/recurrence support** — the runtime must supply current IANA
  timezone data; the `calendar` domain will require an RFC 5545–capable
  recurrence library obtained through the Scenario Dependency Analyzer.
- **Configuration categories** to resolve per environment (plan §23.8):
  public/base URLs, auth integration, database path, job infrastructure,
  provider credential references, timezone runtime, limits, and optional
  model routing.

## Packaging

| Surface | Packaging |
|---|---|
| API | Go binary built by the scenario lifecycle; run via `.vrooli/service.json` (`{{bin.api}}`), never invoked directly. |
| UI | Vite production bundle served by `node server.js` from `ui/`. |
| CLI | Go module (`cli/`) installed as the `personal-planner` command through the scenario manifest. |
| Proto | Domain contracts defined as proto services; generated Connect clients/handlers are shared artifacts. Connect-RPC is the default transport. |
| PWA/brand assets | Web app manifest, standalone mobile tags, proxy-safe relative install URLs, a minimal app-shell service worker, safe-area tokens, and placeholder icons ship from the template. These **must be replaced with valid scenario-specific brand assets and kept valid** before any PWA/desktop tier is released — a broken manifest or missing icon is an install-time failure, and a 404 logo means an unavailable page. |

## Release Checklist

A release gates on the R1 acceptance evidence and the Observatory visual
gate, not just a green lifecycle. Do not treat SKIPPED checks as passing.

- [ ] `make setup` and `make start` succeed on a clean workspace.
- [ ] `make test` passes; the **R1 acceptance suite T01–T36** is green
      (the authoritative acceptance evidence, not a subset).
- [ ] The **Observatory day/night visual gate** passes (design language:
      Auto/Day/Night appearance, tokens, motion, status semantics).
- [ ] scenario-authenticator is reachable in the target environment
      (the scenario cannot start otherwise).
- [ ] The worked-example `notes` domain has been removed, or its retention
      is explicitly justified as product surface.
- [ ] Background jobs (provider sync, occurrence expansion, forecast
      refresh, reminder dispatch, source command retry, learning
      analysis, share maintenance, cleanup) run with leases, durable
      cursors, visible health, and bounded retry; each exposes an
      observable status and no job strands a permanent active lock.
- [ ] PWA/brand assets are valid for the target tier (manifest, icons,
      service worker).
- [ ] Provider credentials are referenced (not embedded); tokens never
      appear in logs, bundles, exports, or docs.
- [ ] `docs/manifest.json` maturity reflects the current docs, and
      [`RUNBOOK.md`](RUNBOOK.md) + [`OBSERVABILITY.md`](OBSERVABILITY.md)
      are active for the target tier.
- [ ] **Pre-cutover verification** (plan §23.8, for any tier beyond local
      dev): backup restoration verified; migration dry-run reconciliation
      counts match; old/new source routing decided; representative
      recurring events expand correctly; rollback accounts for new writes.
- [ ] **Post-cutover verification**: exactly one schedule owner; source
      event delivery works; provider freshness is current; focus sessions
      resume correctly; share authorization holds. An unavailable optional
      provider may leave manual use operational, but its release gate
      stays **explicitly incomplete**, not silently passed.

## Rollback

- **Local development tier:** rollback is source-control based — revert
  the change set and re-run `make setup && make start`. No data migration
  is involved at R0, and there is **no live scheduling data to migrate**
  (the unused legacy `calendar` scenario was removed before this scenario
  was generated).
- **Any future tier with a migration (plan §3.3):** a database snapshot
  alone is **not** a complete rollback. Between cutover and rollback the
  new deployment accepts new writes — accepted allocations, focus
  sessions, actuals, proposals, provider imports, share grants. A correct
  rollback must account for those writes, not just restore the
  pre-cutover snapshot: keep old write-producing jobs stopped only as
  part of the defined cutover, preserve a documented single write
  authority throughout, use stable ID mappings and a recorded migration
  version so a partial roll-forward/roll-back is reconcilable, and decide
  in advance how post-cutover writes are preserved, replayed, or
  discarded. Verify a restore before every cutover so the rollback path
  is known to work rather than assumed.
- Restored data is never assumed live: restored provider connections,
  shares, and running sessions come back inactive/needs-review (see the
  RUNBOOK Backup/Restore section).

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures, incidents, backup/restore
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health, logs, metrics, alerts
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency contract and failure modes
- [`../concepts/DATA.md`](../concepts/DATA.md) — storage, migration, import/export, retention
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — credential and token handling
