# Tunnel Manager

Vrooli's external-access control plane: an **exposure broker** and
**self-healing tunnel manager**. Tunnel Manager programmatically controls
which scenarios are reachable from the public internet through the
Cloudflare tunnel, maintains a route/exposure manifest as the single
source of truth, enforces fixed-port contracts, and auto-recovers the
tunnel from failure. It replaces the operator's current manual step of
adding public hostnames in the Cloudflare dashboard.

> **Status: documentation-first.** Only the `react-vite` scaffold and the
> fenced `notes` worked example exist today. The CLI, API, and UI
> described below are the **planned contract surface** — implementation is
> Phase 2. See [`docs/internal/DECISIONS.md`](docs/internal/DECISIONS.md)
> and [`PRD.md`](PRD.md) for the authoritative scope.

This scenario was **regenerated from `react-vite` 1.1** (rather than
migrated in place) and ports the prior tunnel-manager logic onto a clean
Connect-RPC + screaming-architecture foundation. See the regeneration &
adoption plan at the repo root:
[`docs/plans/tunnel-manager-regen-adoption-plan.md`](../../docs/plans/tunnel-manager-regen-adoption-plan.md).

## Why It Exists

- **Remote access is mission-critical.** Without the tunnel, Vrooli is
  unreachable outside the local network.
- **Exposure is manual today.** Exposing a scenario means hand-adding a
  public hostname in the Cloudflare dashboard, pointed at the scenario's
  fixed UI port. This must be programmatic and native.
- **The hostname budget is finite.** Exposure is **tiered** and
  budget-aware so essential scenarios are never crowded out.
- **The tunnel must self-heal.** Distinguishing "tunnel down" from
  "scenario down" from "Cloudflare outage" enables targeted, automatic
  recovery instead of blind restarts.

## Exposure Tiering Model

Exposure is split into two tiers, reconciled against the manifest
(SSOT):

| Tier | Source | Lifetime | Behavior |
|---|---|---|---|
| **CORE** | `packages/api-core/coreset` | Always-on | Every coreset member is guaranteed exposed and never auto-expired. |
| **LEASED** | On-demand request | Time-bounded (default TTL ≈ 1 week) | Requested by the operator or another scenario ("expose me, I'll be used soon"); extendable, revocable, and auto-reaped on expiry unless the scenario is also CORE. |

## Domains

Seven product domains plus the scaffold `health` domain (see
[`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) for the canonical
map):

| Domain | Owns | Purpose |
|---|---|---|
| `routes` | `routes` table | Exposure manifest (SSOT): subdomain, scenario, domain, local port, tier, lease, enabled. |
| `exposure` | `leases` table | Tiered broker: CORE reconciliation + LEASED request/extend/revoke/reap; ensure-running delegation; exposure-query for app-monitor. |
| `config` | `tunnel_config` | Cloudflare API ingress (remote), local `config.yml` generation (fallback), mode switching, sync. |
| `audit` | (computed) | Port-compliance auditor: exposed scenarios must declare a fixed UI port in `service.json` matching the manifest. |
| `tunnel` | `metrics` table | Tunnel health (systemd + `/ready`), Prometheus scraping, degraded-mode detection. |
| `probes` | `probes` table | Internal + external liveness probing, scheduling, failure classification. |
| `recovery` | `recovery_events` table | Auto-recovery engine (backoff + circuit breaker, live); single cloudflared-restart owner. |
| `health` | none | Runtime readiness + dependency reachability (scaffold). |

## Key Planned CLI Commands

All commands are **planned (Phase 2)** and emit proto-typed `--json`:

```bash
tunnel-manager status          # tunnel + exposure health overview
tunnel-manager routes          # list the exposure manifest
tunnel-manager expose <scenario>      # request exposure (leased)
tunnel-manager lease extend|revoke|list
tunnel-manager probe           # run internal + external liveness probes
tunnel-manager audit           # port-compliance findings
tunnel-manager recover         # inspect / trigger auto-recovery
tunnel-manager config sync|mode       # reconcile ingress / switch remote↔local
```

## Architecture

- **API** — Go on **Connect-RPC** (proto contracts under
  `packages/proto/schemas/tunnel-manager`), one service per domain.
- **CLI** — Go via `cli-core` / `cliapp` manifest, a thin wrapper that
  mirrors the API one command per endpoint.
- **UI** — React + Vite + Tailwind, **5 operator surfaces** (Overview,
  Exposure, Recovery & Events, Metrics, Audit), one feature folder per
  domain.
- **Storage** — **SQLite only** (manifest, leases, metrics history,
  probe history, recovery log). No external database — foundational infra
  must keep working when other resources are down.

## Dependencies

- **Required resources:** none (SQLite is in-process).
- **Optional resources:** `redis` (UI pub/sub for real-time updates;
  falls back to HTTP polling).
- **External:** the `cloudflared` daemon (managed by systemd; setup
  installs it — Tunnel Manager does not). A **Cloudflare API token** is
  required for remote mode only; without it, Tunnel Manager falls back to
  **local mode** (`~/.cloudflared/config.yml`).

## Relationship To Other Scenarios

- **vrooli-autoheal** — Tunnel Manager is the **single authoritative
  owner** of cloudflared restarts. autoheal's cloudflared check downgrades
  to alert-only (defense-in-depth) to avoid dueling restarts.
- **app-monitor** — its reverse proxy stays in `packages/api-base`,
  unchanged. Tunnel Manager exposes `IsExposed` / `ExposeAndGetURL` to
  back app-monitor's "open in new tab" feature; the app-monitor-side
  change is a separate later task.

---

The scaffold also ships the standard full-stack Vrooli scenario shape:

- Go API (`api/`)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry + progress log (`requirements/`, `docs/internal/PROGRESS.md`)

> **Start here:** open [`docs/START-HERE.md`](docs/START-HERE.md). It
> owns the first-session initialization protocol — charter, requirements,
> domain map, design language, placeholder replacement, and first real
> vertical slice. Run `make orient` for a machine-readable gate status.

## What You Get

- Go API (`api/`), Go CLI (`cli/`), and React/Vite UI (`ui/`)
  coordinated through generated proto contracts.
- Lifecycle metadata, Makefile entrypoints, health checks, endpoint
  metadata, testing config, and CLI install wiring.
- Domain-first API shape with per-domain service, repository, schema,
  handler module, mocks, and tests.
- SQLite by default. Add external resources to `.vrooli/service.json`
  only when this scenario actually needs them.
- UI/CLI guardrails for i18n, accessibility, API base resolution,
  declarative command args, generated Connect clients, and report-shaped
  output.
- Baseline PWA branding metadata: web app manifest, standalone-mode
  mobile tags, and generic placeholder icons ready for scenario-specific
  replacement.
- Root-level `DESIGN.md` plus generated UI token assets from the
  selected design kit.
- A documentation contract in `docs/manifest.json`, with stubs for
  domains, flows, data, integrations, monetization, deployment,
  runbooks, observability, security, performance, and durable
  decisions.

## Customize Safely

> Placeholders vs. durable scaffolding — what to replace and what to keep.

The generated scaffold is intentionally not the product. When you build
the real UX, treat these as **placeholders** to replace:

- The `notes` domain (proto, API, CLI, UI feature) — a worked vertical
  slice meant to be copied once and then deleted.
- The `AppShell` and the centered single-panel home page in `ui/src/`.
- The bare-minimum settings surface (currently just locale switching).

Treat these as **durable seams** to preserve, even as you rewrite the
visual layout:

- i18n wiring (`SUPPORTED_LOCALES`, `useTranslation`, `setLocale`).
- Accessibility primitives (`role`, `aria-*`, `data-testid` selectors).
- Design tokens (`bg-app-background`, `rounded-panel`, etc.).
- The feature-folder pattern under `ui/src/features/<name>/`.
- The proto → API → CLI → UI vertical-slice shape.

**Connect-RPC is the default transport.** Every domain endpoint goes
through a proto service and generated Connect handlers/clients. If
you find yourself writing `Path: "/api/v1/..."` as a literal string in
an `EndpointDescriptor`, stop — use a proto service method instead.
Codegen rejects literal Paths that lack an explicit `RESTException`
tag; the four allowed REST reasons (multipart upload, webhook
receiver, third-party shape, ops probe) are enumerated in
`api/internal/module/module.go`. The notes attachments endpoint is
the worked REST example.

[`docs/START-HERE.md`](docs/START-HERE.md) describes the replacement
workflow in full.

## Running The Scenario

```bash
# Build API + UI, install pnpm deps, install scenario CLI
make setup   # wraps `vrooli scenario setup`

# Start API + UI in the background
make start   # wraps `vrooli scenario start`
```

See [`docs/QUICKSTART.md`](docs/QUICKSTART.md) for the full clone-to-running flow.

Run tests with `make test` (which runs `vrooli scenario test`) or invoke
`test-genie execute tunnel-manager --preset comprehensive` directly for
finer-grained presets.

## Documentation Map

| Need | Start Here |
|---|---|
| Initialize after generation | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Establish UI design language | `DESIGN.md` at this scenario's root |
| Run the scenario | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |
| Understand the architecture | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Map product domains | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Track workflows, data, and integrations | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md), [`docs/concepts/DATA.md`](docs/concepts/DATA.md), [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| Capture monetization and launch strategy | [`docs/business/MONETIZATION.md`](docs/business/MONETIZATION.md), [`docs/business/GO-TO-MARKET.md`](docs/business/GO-TO-MARKET.md) |
| Prepare deployment and operations | [`docs/operations/DEPLOYMENT.md`](docs/operations/DEPLOYMENT.md), [`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md), [`docs/operations/OBSERVABILITY.md`](docs/operations/OBSERVABILITY.md) |
| Write tests | [`docs/internal/TESTING.md`](docs/internal/TESTING.md) |
| Add or update seams/fakes | [`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) |
| Configure env vars, ports, CLI config | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| Add API endpoints | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md) |
| Add CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |

## Working Rules

1. **Read [`docs/START-HERE.md`](docs/START-HERE.md) first.** It owns the first implementation workflow.
2. **Run `make orient`** as a progress check — it reports initialization gates from `.vrooli/orientation.json`.
3. **Update `PRD.md` and `requirements/`** before feature work. Operational targets drive code + tests.
4. **Read root `DESIGN.md` before UI work.** Tokens, motion, and status semantics are binding; specific component lists in the design are illustrative — implement everything your scenario actually needs.
5. **Update `docs/concepts/DOMAINS.md`** before adding product code.
6. **Keep `docs/manifest.json` accurate.** Durable docs should be registered there with a truthful maturity value.
7. **Append progress entries** to `docs/internal/PROGRESS.md` whenever you land work.
8. **Add resources** to `.vrooli/service.json` only when needed; this scenario ships with no resource dependencies (SQLite is in-process).
9. **Keep boundaries**: only edit within this scenario's directory.

## pnpm Everywhere

This scenario assumes pnpm. If you run another package manager, convert
lockfiles yourself before committing. Scripts use `pnpm` directly (no
`npm` fallbacks) to reduce drift.

## Need Inspiration?

Open `scenarios/browser-automation-studio/` to see the same template
shape taken to completion.
