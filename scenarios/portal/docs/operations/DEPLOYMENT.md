# Deployment — Portal

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
| Local Vrooli stack | active | Vrooli lifecycle, Go, Node/pnpm, SQLite path | Replace template reference domains before product deployment. |
| Desktop companion | candidate (Linux X11 staging only) | Scenario-to-Desktop Electron package, resolved storage, user-session helper | Windows/macOS/Wayland live rows, clean install/update/uninstall, signing, offline bundle, and release approval remain open. |
| Mobile app | deferred | Mobile runtime and owner transport | No mobile package is in this plan. |
| Managed cloud/SaaS | deferred | Hosted runtime, auth, observability, cost model | Requires deployment and monetization review. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Requires operational hardening. |

## Runtime Requirements

- API port: assigned by lifecycle as `API_PORT`.
- UI port: assigned by lifecycle as `UI_PORT`.
- Storage: embedded SQLite, resolved from the scenario id by `api-core/storage`.
- Resources: none external by default.
- Network: local API/UI communication.

## Portal Deployment Profiles

Portal exposes three connection profiles in Settings. The **Client** profile
opens with no control providers and is the default. **Local control** requires
an available device-control provider before control actions can be offered.
**Automation** requires agent-manager before agent actions are enabled. The
endpoint field is optional and accepts only an `http` or `https` URL without
userinfo, query parameters, or fragments; the browser stores that normalized
URL without tokens or other credentials. Provider loss leaves the client
profile usable and reports setup requirements for the other profiles.

Profile selection is local browser configuration, not a secret store. A fresh
install therefore starts in the client profile, can attach an endpoint later,
and never installs or purchases an optional provider implicitly. Required
capabilities are checked before a task is submitted.

## Packaging

| Surface | Packaging Details |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/portal/`; generated clients are shared artifacts. |

The current packaged evidence is the isolated Linux AppImage pipeline with
`PLATFORM_LINUX` / `linux-amd64` identity and a passing owner smoke journey.
That artifact is a staging candidate, not a commercial release. The primary
support rows are `windows-x64`, `macos-arm64`, `linux-x64-x11`,
`linux-x64-gnome-wayland`, and `linux-x64-kde-wayland`; a row stays unclaimed
until its native lifecycle, permissions, and package receipts are recorded.
See the [support inventory](../internal/plans/artifacts/portal-everywhere-20260905/support-matrix.md).

## Release Checklist

- [ ] `make setup` passes.
- [ ] `make test` passes.
- [ ] PRD operational targets have linked requirements.
- [ ] Template reference domain has been replaced or explicitly retained
      with product justification.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] Every advertised platform row has a native producer receipt and tested
      artifact digest.
- [ ] Install, update, uninstall, interrupted-update recovery, signing, and
      offline behavior are validated for the release candidate.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.

## Rollback

Local development rollback is source-control based. For deployed
targets, document the deployment-specific rollback path before release.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
