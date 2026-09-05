# Domains — Tunnel Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`health` is the scaffold domain. Tunnel Manager adds seven product
domains plus one presentation boundary beside it (below). The old template example domain has been
removed; if `notes` reappears, treat it as regression unless a new PRD
explicitly introduces that product scope.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| routes | Exposure manifest (SSOT): which scenario is exposed at which subdomain/port, tier, lease, enabled. | CRUD / entity | `routes` table | API, CLI, UI | 01-exposure-manifest (OT-P0-001) | `api/internal/routes/`, `api/handlers/routes/`, `cli/domains/routes/`, `ui/src/features/routes/`, `packages/proto/schemas/tunnel-manager/v1/routes/` |
| exposure | Tiered exposure broker: CORE always-on (from `api-core/coreset`) + LEASED on-demand (TTL request/extend/revoke/reap); ensure-running delegation; exposure-query for app-monitor. | Policy / orchestration | `leases` table | API, CLI, UI | 03-exposure-tiers (OT-P0-003/004/005/006), 10-app-monitor-integration (OT-P1-007) | `api/internal/exposure/`, `api/handlers/exposure/`, `cli/domains/exposure/`, `ui/src/features/exposure/`, `packages/proto/schemas/tunnel-manager/v1/exposure/` |
| config | Cloudflare API ingress management (remote), local `config.yml` generation (fallback), credential-authority status/write flow, mode switching, config sync. | Adapter / integration | `tunnel_config` plus credential-authority references | API, CLI, UI settings | 02-cloudflare-ingress (OT-P0-002, OT-P1-002) | `api/internal/config/`, `api/handlers/config/`, `cli/domains/config/`, `ui/src/pages/SettingsPage.tsx`, `packages/proto/schemas/tunnel-manager/v1/config/` |
| audit | Port-compliance auditor: verify exposed scenarios declare fixed UI ports in `service.json` matching the manifest. | Reporting / query | None (computed) | API, CLI, UI | 04-port-compliance (OT-P0-007) | `api/internal/audit/`, `api/handlers/audit/`, `cli/domains/audit/`, `ui/src/features/audit/`, `packages/proto/schemas/tunnel-manager/v1/audit/` |
| tunnel | Tunnel health (managed resource + `/ready`), Prometheus metrics scraping + time-series, degraded-mode detection. | Monitoring | `metrics` table | API, CLI, UI | 05-tunnel-health (OT-P0-008, OT-P1-003/006) | `api/internal/tunnel/`, `api/handlers/tunnel/`, `cli/domains/tunnel/`, `ui/src/features/metrics/`, `packages/proto/schemas/tunnel-manager/v1/tunnel/` |
| probes | Internal + external liveness probing, scheduler, failure classification, probe history. | Monitoring | `probes` table | API, CLI, UI | 06-liveness-probes (OT-P0-009/010, OT-P1-001) | `api/internal/probes/`, `api/handlers/probes/`, `cli/domains/probes/`, `ui/src/features/probes/`, `packages/proto/schemas/tunnel-manager/v1/probes/` |
| recovery | Auto-recovery engine (backoff + circuit breaker, live), recovery event log; single cloudflared-restart owner. | Control / actuation | `recovery_events` table | API, CLI, UI | 07-auto-recovery (OT-P0-011, OT-P1-005) | `api/internal/recovery/`, `api/handlers/recovery/`, `cli/domains/recovery/`, `ui/src/features/recovery/`, `packages/proto/schemas/tunnel-manager/v1/recovery/` |
| presentation | Aggregate operator surfaces that compose several product domains: CLI command registry, overview dashboard, settings, app shell, and cross-domain UI state. | Adapter / interface | Browser/local UI preferences only. | CLI, UI | 08-cli-interface (OT-P0-012), 09-web-dashboard (OT-P1-004) | `cli/domains/domains_test.go`, `ui/src/features/overview/`, `ui/src/pages/`, `ui/src/App.tsx` |
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/tunnel-manager/v1/health/` |

## Domain Details

### `routes` — exposure manifest (SSOT)
- Owns the `routes` table: `subdomain`, `scenario`, `domain` (a field,
  not a constant; default `itsagitime.com`), `local_port`, `tier`
  (`core`|`leased`), `lease_id` (nullable), `enabled`, `health_path`,
  `source` (`scenario`|`external`), `service_target` (external only).
  Derives `public_url` = `https://<subdomain>.<domain>`.
- Does not own ingress creation (`config`) or tier policy (`exposure`);
  it is the record of truth other domains reconcile against.
- Validation: subdomain is a valid DNS label; one route per subdomain.
  **Scenario routes** require `scenario` + `local_port` (the fixed UI
  port). **External routes** (`source=external`) skip that rule and
  instead require an absolute `http(s)` `service_target` (e.g.
  `http://127.0.0.1:9000`) — first-class CRUD for exposing non-scenario
  targets. `source` is orthogonal to `tier`.
- Why: every other domain reads the manifest, so it is foundational.

### `exposure` — tiered exposure broker
- Owns the `leases` table: `scenario`, `requested_by`, `created_at`,
  `expires_at`, `extended_count`, `status`. Owns the reconciliation
  policy: every `api-core/coreset` scenario is a CORE route; leased
  scenarios are LEASED routes that auto-expire.
- Behavior: `Expose(scenario, ttl)` → ensure a route exists (LEASED) →
  ensure the scenario is running (delegate to `internal/lifecycle`) →
  request ingress (delegate to `config`); `Extend`/`Revoke`; a reaper
  removes expired leases + ingress unless the scenario is also CORE.
  `IsExposed` + `ExposeAndGetURL` back the app-monitor new-tab feature.
- Does not own process lifecycle (delegates), the manifest schema
  (`routes`), or Cloudflare calls (`config`).
- Why: the conceptual heart — turns "should this be reachable, for how
  long?" into manifest + ingress + run state.

### `config` — Cloudflare ingress, mode & drift management
- Owns `tunnel_config`: mode (`remote`|`local`), tunnel id, account id,
  credential reference, Prometheus endpoint. Also owns the
  **`ingress_ownership` ledger** (keyed on full hostname; owner ∈
  `MANAGED`|`EXTERNAL`|`IGNORED`) — the authoritative record of who owns
  each live ingress entry. Owns the credential status and write-only
  setup workflow; secret values live outside SQLite in the shared
  Vrooli credential authority.
- Behavior:
  - `Sync` is **additive by default**: it publishes the desired manifest
    merged onto current live ingress (union), preserving unmanaged/ignored
    entries. `prune` removes only orphaned entries (ledger-managed, route
    gone). Remote mode pushes via Cloudflare API v4 (hot-reload); local
    mode merges into `~/.cloudflared/config.yml` (restart on change).
  - `SwitchMode` is **pure** — it persists the mode and performs zero
    ingress writes (switching to remote does a read-only credential check).
  - `GetDrift` reconciles desired ∪ live ∪ ledger into a classified read
    model: `MANAGED` / `MISSING` / `EXTERNAL_OK` / `ORPHANED` / `IGNORED` /
    `UNMANAGED`. `AdoptIngress` / `IgnoreIngress` / `PruneIngress` are the
    per-entry decisions an operator applies to drift.
- Non-destructive by construction: unmanaged ingress TM did not author is
  never auto-removed under any code path. No DNS-record writes (ingress
  configurations only).
- Why: the adapter that makes exposure programmatic and legible —
  replacing the operator's manual dashboard step while never clobbering
  what it doesn't own.

### `audit` — port-compliance auditor
- Owns nothing persistent; computes findings from scenario
  `service.json` files vs the manifest.
- Behavior: confirm each manifested route's scenario declares a fixed UI
  port matching the route; report mismatches, missing ports, and ranged
  (non-fixed) ports.
- Why: drifted ports silently break ingress; auditing catches it.

### `tunnel` — tunnel health & metrics
- Owns the `metrics` table: time-series of HA connections, request
  errors, RTT, active streams.
- Behavior: read managed cloudflared resource status + `/ready`; scrape the
  Prometheus endpoint; detect degraded mode (HA < 4 or RTT spikes).
- Why: recovery and operators need a truthful health signal distinct
  from per-route probes.

### `probes` — liveness probing & classification
- Owns the `probes` table: probe history (route, kind
  internal|external, status, latency, error).
- Behavior: probe each exposed route's local port (internal) and public
  URL (external) on a schedule; classify current probe pairs as healthy,
  tunnel-down, scenario-down, or config-drift. DNS-failure and
  Cloudflare-outage isolation require future resolver/upstream signals.
- Why: knowing where a failure is lets recovery act precisely.

### `recovery` — auto-recovery engine (live)
- Owns the `recovery_events` table: attempts, trigger, action, outcome,
  timestamps.
- Behavior: on `/ready` failure or HA=0, restart cloudflared / re-push
  config with exponential backoff + circuit breaker; acts live from day
  one. Single authoritative owner of cloudflared restart
  (vrooli-autoheal downgrades to alert-only).
- Why: hands-off recovery is the core value promise; live action is an
  explicit operator decision (see
  [`../internal/DECISIONS.md`](../internal/DECISIONS.md)).

### presentation

- Purpose: own the aggregate operator-facing shell that composes multiple
  domains without inventing separate business state.
- Owns: CLI registry coverage, overview dashboard composition, settings
  page composition, app shell/routing, and cross-domain UI tests.
- Does not own: route manifests, ingress credentials, leases, probes,
  recovery events, audit findings, or tunnel metrics.
- Why: CLI and dashboard requirements intentionally validate the complete
  operator surface. Assigning those tests to one product domain would make
  the domain map misleading.

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| DNS/upstream diagnostics | Current probe data cannot isolate resolver failure or Cloudflare-wide outage. | Add resolver and upstream-status/API signals for OT-P1-001. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `api/internal/authz/` — shared authorization guard used by privileged
  mutation handlers; policy ownership remains with the called domain.
- `api/internal/scheduler/` — shared cancellable loop helper; concrete
  schedules belong to exposure, probes, and recovery.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
