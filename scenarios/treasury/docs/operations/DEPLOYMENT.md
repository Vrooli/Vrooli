# Deployment — Treasury

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
| Desktop/mobile app | deferred | Cross-platform runtime, packaged UI/API, storage resolver | Run cross-platform readiness before adoption. |
| Managed cloud/SaaS | **deliberately constrained** | Hosted runtime, auth, observability, cost model | The self-hosted position is the product. A hosted tier may offer the *facilitator*, *card issuing* and *approval relay* as convenience, but hosting the policy engine and evidence store for a customer would custody their financial record and contradict the positioning in `../business/MONETIZATION.md`. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Requires operational hardening. This is the tier the product is actually aimed at, so it should mature before the managed one. |

## Runtime Requirements

- API port: assigned by lifecycle as `API_PORT`.
- UI port: assigned by lifecycle as `UI_PORT`.
- Storage: `SQLITE_PATH` local file by default.
- Resources: none external by default. A self-hosted x402 facilitator
  becomes a managed resource at P1; the implementation choice is an open
  decision in `../internal/DECISIONS.md`.
- Network: local API/UI communication for P0. Automated rails require
  outbound network access to payment networks and priced endpoints.
- **Runtime dependency:** `agent-manager` must be reachable for automated
  spend. This is not a soft dependency — spend fails closed without it.
- Storage durability: the evidence journal must be backed up before any
  automated rail is enabled. See `RUNBOOK.md`.

## Packaging

| Surface | Packaging Details |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/treasury/`; generated clients are shared artifacts. |

## Release Checklist

- [ ] `make setup` passes.
- [ ] `make test` passes.
- [ ] PRD operational targets have linked requirements.
- [ ] Template reference domain has been replaced or explicitly retained
      with product justification.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.

Additional gates specific to this scenario. Each exists because the
failure it prevents costs real money:

- [ ] Every P0 requirement's validation is **earned by a passing test**,
      not asserted. The security posture in `../internal/SECURITY.md` is
      entirely `designed` until this holds.
- [ ] `TRS-P0-004` passes — the agent-facing service descriptor declares no
      policy-mutating method.
- [ ] `TRS-P0-005` passes — spend fails closed with the identity authority
      unreachable.
- [ ] `TRS-P0-010` passes — the schema, not just a handler, refuses a
      third-party beneficiary.
- [ ] `TRS-P0-011` passes, including the concurrent case.
- [ ] The kill switch has been exercised against a live authorization, not
      only unit-tested.
- [ ] Evidence backup is configured and a restore has been performed once.
- [ ] Rate limiting exists on the agent-facing authorization surface.

**Before enabling any automated rail**, additionally: run the first real
transaction as a small recurring API or cloud top-up, and confirm the full
loop — mandate, approval, evidence, ledger emission — before raising any
cap.

## Rollback

Local development rollback is source-control based. For deployed
targets, document the deployment-specific rollback path before release.

**Rolling back this scenario is not symmetric with rolling it forward.**
Two constraints:

- **Never roll back to a schema that drops or rewrites
  `idempotency_keys`.** In-flight client retries would become potential
  double charges. A rollback touching that table needs the same explicit
  decision record a migration does.
- **Evidence written by a newer version must remain readable.** Records are
  append-only and are the audit artefact; a rollback that cannot read them
  has lost the property the scenario exists for. Additive schema changes
  only, which is what makes rollback safe by construction.

The safe operational rollback is: **freeze first, then roll back.** A
freeze binds before the next authorization, so it stops new spend while the
version changes underneath.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
