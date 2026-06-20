# Deployment — Tunnel Manager

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness.

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

> **Status:** Tier 1 local-stack deployment is implemented and validated.
> Tunnel Manager runs where the host `cloudflared` daemon runs; container,
> hosted cloud, SaaS, desktop, and mobile packaging remain deferred.

## Supported Tiers

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Tier 1 local stack | supported | Vrooli lifecycle, Go, Node/pnpm, SQLite path, `cloudflared` (systemd), Cloudflare API token (remote mode) | Operator-attended live Cloudflare credential validation when deploying remote mode. |
| Desktop/mobile app | deferred | Cross-platform runtime, packaged UI/API, storage resolver | Out of scope; see [Deployment Hub](../../../../docs/deployment/README.md) tiers. |
| Managed cloud/SaaS | deferred | Hosted runtime, auth, observability, cost model | Foundational infra runs co-located with the host's cloudflared; no hosted target planned. See Deployment Hub. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Future tier; see Deployment Hub. |

Tunnel Manager is **foundational infrastructure** that runs on the same
host as `cloudflared`. It controls the host's own Cloudflare tunnel, so
it is not a candidate for a separate hosted/cloud tier — it deploys
wherever the local stack and its tunnel daemon run.

## Runtime Requirements

- API port: assigned by lifecycle as `API_PORT` (band `15000-19999`).
- UI port: **fixed at `21240`** (`service.json` `ports.ui`). Tunnel
  Manager enforces the fixed-UI-port contract on others, so it pins its
  own — see [`../internal/DECISIONS.md`](../internal/DECISIONS.md).
- Storage: SQLite only, at `SQLITE_PATH`
  (default `${SCENARIO_DATA_DIR}/tunnel-manager.db`). No external
  database — foundational infra must keep working when other resources
  are down (see DECISIONS: "SQLite only").
- External daemon: `cloudflared` running under systemd (installed by
  `make setup`/Vrooli setup, **not** managed by this scenario).
- Cloudflare API token: required only for **remote mode** (programmatic
  ingress via Cloudflare API v4). **Local mode** — generating
  `~/.cloudflared/config.yml` from the manifest — is the fallback when
  no token is available.
- Resources: none required; `redis` optional (UI pub/sub, falls back to
  HTTP polling).
- Network: local API/UI communication plus outbound to the Cloudflare
  API (remote mode) and reads of cloudflared's local Prometheus endpoint
  (default `127.0.0.1:20241`).

## Lifecycle

Tunnel Manager deploys and runs through the standard Vrooli lifecycle —
never by direct binary execution:

```bash
make setup            # build + install (also ensures cloudflared via setup)
make start            # start API + UI under lifecycle
vrooli scenario start tunnel-manager   # CLI equivalent
vrooli scenario stop  tunnel-manager
make test             # scenario test suite
```

The lifecycle owns process naming, the API port, health checks, and
logs. The UI port is fixed; the API port is lifecycle-assigned.

## Packaging

| Surface | Packaging Details |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/tunnel-manager/`; generated clients are shared artifacts. |

There is **no container image or cloud deployment artifact** today, and
none is planned for the foundational tier. Future packaging (desktop,
mobile, SaaS, enterprise) is deferred to the
[Deployment Hub](../../../../docs/deployment/README.md).

## Release Checklist

- [x] `make setup` passes in recorded implementation slices.
- [x] `make test` / `vrooli scenario test tunnel-manager` passes (latest
      recorded result: 18/18 phases green).
- [x] PRD operational targets have linked requirements.
- [x] Template reference domain has been removed from product code.
- [x] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.
- [ ] Operator-attended live Cloudflare remote-mode validation is complete
      for the target host/account.
- [ ] P2 deployment additions such as alert webhooks, dashboard exports,
      and multi-tunnel support are promoted or explicitly left deferred.

## Rollback

Local development rollback is source-control based. For deployed
targets, document the deployment-specific rollback path before release.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
