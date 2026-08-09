# Deployment — Scenario Authenticator

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness for the fleet's
foundational Identity Provider (IdP) — a Go API (Connect-RPC + a thin
REST edge), a React UI, and a Go CLI sharing one binary.

> **Status: local deployment is implemented and tested.** The Go API,
> Connect/REST edges, CLI, UI shell, SQLite storage, required Redis
> integration, persisted RS256 signing material, sessions, and JWKS are
> wired through the lifecycle. HA/managed-database deployment, true
> multi-realm tenancy, and enterprise packaging remain future tiers.

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

## Two Deployment Shapes, One Binary

The same artifact serves both a household/team server and a multi-tenant
SaaS — the difference is the **realm** primitive (PRD Appendix B) and the
storage seam target, not a code fork.

### (a) Local same-network (default — the P0/P1 shape)

The Tier 1 local Vrooli stack. A single **default realm** issues
`aud`-scoped tokens; persistence is **SQLite via the `api-core/storage`
seam** (never shared Postgres — eliminating the shared-DB blast radius is
*why* this scenario was rewritten); **Redis is a required resource** for
session/revocation hot state and rate limiting. Managed entirely through
the scenario lifecycle (the Makefile / `vrooli scenario`), never by
running the binary directly.

The signing keypair (`private.pem`/`public.pem`) is generated on first
boot (load-or-generate) and **persisted to the storage root**. This file
pair is the single most important thing to back up: **lose or regenerate
it and every live token across every Relying Party is instantly
invalidated.** See the backup/rotation guidance below.

### (b) Hosted / cloud-as-a-dependency (a P2 target)

An adopting scenario embeds scenario-authenticator and **provisions a
realm per customer (B2B) or per product (B2C)**. The `api-core/storage`
seam points at a **managed server DB** instead of a local SQLite file (the
seam is what keeps this a configuration change, not a re-architecture).
The scenario runs **multi-instance behind a load balancer**, with **Redis
as the shared store** for session/revocation and rate-limit state so any
replica can revoke a session another replica issued. This is OT-P2-006
(managed-DB backing + multi-instance HA) and is explicitly **not built
yet**.

In this shape, signing keys still must be shared across replicas and
backed up; per-realm key isolation with automated overlapping-`kid`
rotation is OT-P2-005.

## Supported Tiers

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Local same-network (Tier 1) | supported | Vrooli lifecycle, Go, Node/pnpm, SQLite via storage seam, **required Redis**, persisted signing keypair | Scenario suite and focused API/CLI tests. |
| Desktop / mobile | deferred | Packaged UI/API, storage resolver | Local-network IdP is the design; no standalone packaging planned near-term. |
| Hosted / SaaS (cloud-as-a-dependency) | deferred (P2) | Managed DB via storage seam, load balancer, Redis-shared state, per-realm provisioning | OT-P2-006 (managed-DB + HA) and OT-P2-005 (per-realm key isolation/rotation) not built. |
| Enterprise / self-host | deferred (P2) | Install docs, key backup/rotation runbook, SAML/SCIM | Requires P1+ feature surface and operational hardening. |

Higher tiers are documented in the
[Deployment Hub](../../../../docs/deployment/README.md) but are deferred
until the P0 auth core is green and the device-sync-hub live consumer
migration (OT-P0-012) is proven end-to-end.

## Runtime Requirements

Always-on, declared and started by the lifecycle:

- **Go API server** — Connect-RPC as the primary typed contract plus a
  thin REST edge for non-RPC web standards only (`/.well-known/jwks.json`,
  OAuth/OIDC callbacks, later SAML ACS). Port assigned as `API_PORT`.
- **React + Vite UI** — admin console, end-user self-service, and hosted
  login/consent screens. Port assigned as `UI_PORT`.
- **SQLite via `api-core/storage`** — realms, users, credential hashes,
  refresh-token families, roles/scopes, audit events. At `SQLITE_PATH`
  locally; pointed at a managed DB through the same seam at scale.
  Schema changes are **additive migrations only**, never recreation.
- **Redis (required)** — sessions, token/family revocation, OAuth CSRF
  state, and cross-replica rate-limit coordination. Treated as required,
  not optional: session-revocation correctness depends on it.
- **Persisted signing keypair** — `private.pem`/`public.pem` in the
  storage root, load-or-generate, RS256-locked.

Consumers (Relying Parties) need **no per-request callback** to this
scenario: they resolve it by slug via `api-core/discovery` and verify
tokens locally against JWKS. The hot path never touches SQLite.

## Packaging

| Surface | Packaging Notes |
|---|---|
| API | Go binary built by the scenario lifecycle (`CGO_ENABLED=0`; pure-Go SQLite via `modernc.org/sqlite`). |
| UI | Vite production bundle served by `ui/server.js`; PWA manifest/icons under `ui/public/` remain valid. |
| CLI | Go CLI installed through scenario manifest install hooks; full Connect-RPC surface parity (OT-P0-010). |
| Proto | Schemas under `packages/proto/schemas/scenario-authenticator/`; generated clients are shared artifacts consumed by RPs (e.g. the device-sync-hub Connect client). |
| Signing keypair | **Never packaged.** Generated/persisted per deployment in the storage root and must be backed up out-of-band. |

## Signing-Key Backup & Rotation

The keypair is deployment state, not build state, and is the root of all
token trust:

- **Back it up.** Snapshot the storage-root `private.pem`/`public.pem`
  alongside the SQLite database. The standard scenario backup substrate
  (**data-backup-manager**) is the intended path — back up the SQLite DB
  and the storage namespace (which holds the keypair) together so a
  restore yields a consistent identity store *and* a key that still
  validates the tokens it issued. See [`RUNBOOK.md`](RUNBOOK.md).
- **Rotate deliberately.** Rotation must publish the new key in JWKS with
  a new `kid` while the old key remains published long enough for
  short-lived access tokens minted under it to expire (overlapping
  `kid`s). Automated per-realm rotation is OT-P2-005 and is not built;
  until then, rotation is a manual, deliberate operator action documented
  in [`RUNBOOK.md`](RUNBOOK.md).
- **Never regenerate accidentally.** A missing/unreadable keypair makes
  load-or-generate mint a *fresh* key, silently invalidating every live
  token. Persistence and backup are the guardrails against this.

## Release Checklist

- [ ] `make setup` passes.
- [ ] `make test` passes (all required test-genie phases green).
- [ ] PRD operational targets have linked requirements that `validate`.
- [ ] cli/manifest.json measure blocks present (test-genie measures phase green).
- [ ] **Crypto invariants verified** (PRD Appendix C): RS256 locked
      (`none`/HS confusion rejected), JWKS served, claims
      (`user_id`/`sub`, `email`, `roles`, `iss`, `aud`) match the
      carried-over contract, Argon2id hashing.
- [ ] **Realm isolation verified:** a token minted for realm A is rejected
      by realm B's verifier (`aud`-scoping, OT-P0-008).
- [ ] **Live consumer migration proven:** device-sync-hub verifies tokens
      unchanged and its forwarder runs on the typed Connect client
      (OT-P0-012).
- [ ] Signing-key backup/restore procedure exercised (`RUNBOOK.md`).
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.

## Rollback

Local development rollback is the standard lifecycle restart
(`make restart`) or a code-level revert via the GCT baseline-restore path,
followed by a restart. **Two cautions specific to an IdP:**

1. **Never roll back the signing keypair to a different key** unless you
   intend to invalidate live tokens — code reverts must not regenerate or
   swap the persisted keypair.
2. **SQLite schema migrations are additive only**, so a code rollback
   should never require a destructive down-migration; a newer DB stays
   readable by older code on the additive columns.

For any future hosted tier, document the deployment-specific rollback path
(blue/green with shared keys + Redis) before release.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures (key rotation, revocation, backup)
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies and RP contract
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — domain map (tokens owns the keypair)
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
- [`../../PRD.md`](../../PRD.md) — Appendix B (realm), Appendix C (crypto invariants)
