# Deployment — Token Economy

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness.

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

## Supported Tiers

**The self-hosted household instance is the product, not a deployment option.**
"Nothing leaves the machine" is half the differentiation (`MONETIZATION.md`), so
tiers that move data off the operator's machine are not merely deferred — they
would require re-deciding what the product is.

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Local Vrooli stack | **active — the target** | Vrooli lifecycle, Go, Node/pnpm, SQLite path, `scenario-authenticator` reachable | No product code exists yet (gate 6). |
| PWA install on a holder's phone | **planned, and load-bearing** | The seeded web app manifest, service worker, maskable icons and safe-area tokens; the household instance reachable on the local network | Holder view is unimplemented. Installability is what makes the holder view something a child actually opens, so this is not a nice-to-have. |
| Desktop/mobile native app | deferred | Cross-platform runtime, packaged UI/API, storage resolver | Run cross-platform readiness before adoption. The PWA path likely covers the real need. |
| Managed cloud/SaaS | **blocked, not deferred** | Hosted runtime, multi-tenant isolation, erasure path, cost model | Would host behavioral records of minors. The undesigned erasure path (`SECURITY.md`, `PROBLEMS.md`) is a hard blocker, and hosting contradicts the product's central privacy claim. Requires an explicit decision to change the product, recorded in `DECISIONS.md`. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Plausible for the P2 recognition-economy use (`TKE-P2-006`); requires operational hardening first. |

## Runtime Requirements

- API port: assigned by lifecycle as `API_PORT`.
- UI port: assigned by lifecycle as `UI_PORT`.
- Storage: `SQLITE_PATH` local file. **Back this up** — it is the only copy of
  the journal, and the journal is the only authority on every balance.
- Resources: **none external**, deliberately. See `../concepts/INTEGRATIONS.md`
  for each candidate resource and why it was rejected.
- Scenario dependencies: `scenario-authenticator` **required and fail-closed**;
  `notification-hub` and `agent-manager` optional and degrading.
- Network: local API/UI communication, plus local-network reach from a holder's
  phone for the PWA surface. No outbound internet requirement at any point —
  the scenario functions fully air-gapped.

## Packaging

| Surface | Packaging Details |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/token-economy/`; generated clients are shared artifacts. |

## Release Checklist

- [ ] `make setup` passes.
- [ ] `make test` passes.
- [ ] PRD operational targets have linked requirements.
- [ ] `vrooli scenario requirements validate token-economy` passes.
- [ ] `experience-manager spec validate token-economy` passes.
- [ ] Template reference domain has been replaced or explicitly retained
      with product justification.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.

Scenario-specific gates — none of these are optional, because each guards a
property the product cannot be trusted without:

- [ ] **Projection equals a full journal replay** on a seeded database of at
      least 10k events (`TKE-P0-004`).
- [ ] **Holder isolation proven at the repository layer**, not only the handler,
      including that a refusal does not disclose whether the other holder exists
      (`TKE-P0-006`).
- [ ] **The holder service descriptor contains no mint, grant, catalog-write or
      rule-write method** (`TKE-P0-005`).
- [ ] **Settlement is idempotent under induced failure** between debit and event
      write, not merely under happy-path retry (`TKE-P0-009`).
- [ ] **No proto field or service method expresses price, payout, or external
      transfer** (`TKE-P0-014`).
- [ ] **The journal repository exposes no update or delete** (`TKE-P0-010`).
- [ ] **Approval works with `notification-hub` unavailable** (`TKE-P0-013`).
- [ ] **A backup of `SQLITE_PATH` exists and has been restored once**, because
      an append-only journal with no repair verb makes backup the only recovery.

## Rollback

Local development rollback is source-control based.

**Data rollback is not symmetric with code rollback, and that asymmetry is
deliberate.** The journal is append-only and has no repair verb by design
(`TKE-P0-010`), so there is no "undo" for a bad write — a mistaken event is
corrected by a compensating event, which is a product behavior rather than an
operational one.

That leaves exactly two recovery paths, and an operator should know which one
they are in:

1. **A wrong value was recorded.** Not a rollback. Issue a correction through
   the product; both the mistake and its fix remain visible, which is the
   intended behavior.
2. **The database is corrupt or a migration went wrong.** Restore
   `SQLITE_PATH` from backup and replay any events recorded since. This is the
   only true rollback, and it is why backup is a release gate rather than a
   maintenance suggestion.

A schema change that alters the meaning of an existing event column has no
rollback at all, because historical rows cannot be reinterpreted — event columns
are additive only (`../concepts/DATA.md`).

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
