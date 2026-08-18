# Deployment — Notification Hub

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness.

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

## Supported Tiers

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Local Vrooli stack (Linux) | active | Vrooli lifecycle, Go, Node/pnpm, SQLite path | Replace template reference domains before product deployment. |
| Fleet node (macOS) | required at P1 | Same build, no Docker, `vrooli-bridge` enrolment | This is not an optional tier. OT-P1-001 and OT-P1-002 exist only if the scenario runs on `minimouse`, so macOS is a first-class target rather than a future nicety. |
| Fleet node (Windows) | possible, unvalidated | Same build, no Docker | No Windows node is enrolled. The zero-resource decision keeps the door open; nothing has been proven. |
| Desktop/mobile app | deferred | Cross-platform runtime, packaged UI/API, storage resolver | Run cross-platform readiness before adoption. |
| Managed cloud/SaaS | not applicable | — | Deliberately out of scope. A hosted multi-tenant notification service is the product this scenario was regenerated *away* from; see [`../internal/DECISIONS.md`](../internal/DECISIONS.md). |
| Enterprise/self-host | not applicable | — | Same reason. |

**The macOS tier is why the resource set is empty.** `resource-postgres`
and `resource-redis` are OCI-acquired and recorded `unsupported` on
macOS and Windows, so any dependency on them would silently reduce this
table to one row. Anything added to `.vrooli/service.json` must be
checked against
[`docs/reference/platform-support.md`](../../../../docs/reference/platform-support.md)
before it is added.

## Runtime Requirements

- API port: assigned by lifecycle as `API_PORT`.
- UI port: assigned by lifecycle as `UI_PORT`.
- Storage: `SQLITE_PATH` local file. No external database.
- Resources: none required. Optional `cloud-api` resources supply
  channel credentials.
- Network: local API/UI communication, plus outbound HTTPS to whichever
  delivery providers are enabled. A host with no outbound network can
  still accept notifications; they fail with a stated reason rather than
  disappearing.
- Fleet: `vrooli-bridge` enrolment on any node expected to serve a
  relayed channel. A node that is not enrolled is simply never selected.
- macOS nodes serving iMessage additionally need Full Disk Access for
  the agent, an unlocked session, and a signed-in Messages account.
  These are node provisioning facts, not scenario configuration, and
  they are the reason OT-P1-002 is best-effort.

## Packaging

| Surface | Packaging Details |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/notification-hub/`; generated clients are shared artifacts. |

## Release Checklist

- [ ] `make setup` passes.
- [ ] `make test` passes.
- [ ] PRD operational targets have linked requirements.
- [ ] Template reference domain has been replaced or explicitly retained
      with product justification.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.

Scenario-specific gates, because a green test suite does not prove this
scenario works:

- [ ] **A real notification arrived on a real device.** Not a simulated
      send, not a passing adapter test. The predecessor scenario passed
      its suites for ten months while delivering nothing, and this gate
      exists specifically to make that impossible to repeat.
- [ ] `.vrooli/service.json` declares no resource that is `unsupported`
      on macOS in the platform-support matrix.
- [ ] A `secret`-sensitivity notification was observed leaving no body
      text on any channel.
- [ ] A quiet-hour hold released correctly across a midnight boundary.
- [ ] A failed delivery reported a reason an operator could act on,
      naming the device or node rather than saying "delivery error".

## Rollback

Local development rollback is source-control based. For deployed
targets, document the deployment-specific rollback path before release.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
