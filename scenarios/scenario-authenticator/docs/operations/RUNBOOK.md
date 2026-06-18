# Runbook — Scenario Authenticator

This document records operator procedures for running, diagnosing,
recovering, and maintaining the fleet's Identity Provider (IdP).

> **Status: documentation-first orientation.** The procedures below are
> the **target** operator workflow derived from [`../../PRD.md`](../../PRD.md);
> the auth domains they drive are not implemented yet. Lifecycle
> commands (start/stop/logs/test) are real today; the auth-specific
> procedures (realm/admin creation, key rotation, revocation, audit) are
> the intended shape, not working commands.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- How do I create the default realm and the first admin?
- How do I rotate signing keys and revoke sessions during an incident?
- How do I back up and restore identity state?

## Start / Stop / Status

Use lifecycle-managed commands from the scenario directory:

```bash
make setup     # build API/CLI/UI, install scenario CLI (run once / on dep changes)
make start     # start API + UI + Redis
make status    # running surfaces and their ports
make logs      # tail API + UI logs
make stop      # clean shutdown
make restart   # stop then start
make test      # run the scenario test lifecycle
```

Equivalently `vrooli scenario start scenario-authenticator` (and
`stop`/`status`/`logs`/`test`). **Never** start the API/UI binary
directly — the lifecycle owns process naming, ports, health checks, logs,
and the persisted storage root that holds the signing keypair.

## Health Check

```bash
API_PORT=$(vrooli scenario port scenario-authenticator API_PORT)
curl -s "http://localhost:${API_PORT}/health"          # reachability + dependency status
curl -s "http://localhost:${API_PORT}/.well-known/jwks.json"   # public key RPs verify against
```

`/health` should report SQLite (storage seam) and Redis reachable. JWKS
must return the active public key — if it is empty or 404s, RPs cannot
verify tokens. (Target endpoints; not wired yet.)

## Create The Default Realm + First Admin

On a fresh install the **default realm** is created at first boot and
issues `aud`-scoped tokens (OT-P0-008). Bootstrapping the first admin is a
one-time, first-run step (the device-sync-hub live first-run owner
bootstrap is the reference flow, OT-P0-012):

```bash
# Target shape — depends on the realms + identity + authorization domains (P0, unbuilt):
scenario-authenticator realms list                       # confirm the default realm exists
scenario-authenticator realms ensure-default             # idempotent: create if missing
scenario-authenticator users create --realm default \
  --email admin@example.com --role admin                 # first admin (admin role, OT-P0-009)
```

The same surface exists as Connect RPCs and in the admin-console UI.
Passwords are hashed with Argon2id; only hashes are stored (OT-P0-004).

## Rotate Signing Keys (overlapping `kid`s)

The signing keypair (`private.pem`/`public.pem`) lives in the storage
root and is the root of all token trust. Rotation must be deliberate so
live tokens are not invalidated mid-flight:

1. **Back up the current keypair first** (see Backup / Restore).
2. **Add the new key alongside the old**, published in JWKS under a *new*
   `kid`. New tokens sign with the new key; the old public key stays in
   JWKS so already-issued (short-lived) access tokens still verify.
3. **Wait out the access-token TTL** so all tokens minted under the old
   `kid` have expired.
4. **Retire the old key** from JWKS.

```bash
# Target shape — depends on the tokens domain (P0) / per-realm rotation (OT-P2-005, P2):
scenario-authenticator tokens keys list                  # show kids and active key
scenario-authenticator tokens keys rotate                # add new kid, keep old published
scenario-authenticator tokens keys retire --kid <old>    # after access-token TTL elapses
```

Automated per-realm rotation with overlapping `kid`s is OT-P2-005 and is
not built; until then rotation is a manual operator action. **Never delete
or regenerate the keypair without rotation — doing so invalidates every
live token across every Relying Party.**

## Revoke A Session / "Revoke All" (incident response)

Sessions are server-tracked with Redis hot state (OT-P0-005). The live
`/api/v1/sessions/{id}` revoke contract device-sync-hub calls is preserved
(or delivered as the Connect equivalent in lockstep).

```bash
# Target shape — depends on the sessions domain (P0, unbuilt):
scenario-authenticator sessions list --user <user-id>    # enumerate active sessions
scenario-authenticator sessions revoke --id <session-id> # revoke one session
scenario-authenticator sessions revoke-all --user <user-id>   # "log out everywhere"
```

For a credential-compromise incident: `revoke-all` for the affected
principal, then force a password reset (identity domain). Presenting a
**rotated/reused refresh token** automatically revokes the entire token
family (reuse detection, OT-P0-003) and is recorded in the audit log.
Every revoke is an audited security event.

## Redis Outage — Behavior & Recovery

Redis is **required** (PRD operational risks). During an outage:

- **Token issuance/verification is unaffected** — verification is stateless
  against JWKS and does not touch Redis.
- **Session revocation and "revoke all" cannot be honored**, and
  distributed (cross-replica) rate-limit accuracy degrades. The system
  must **fail safe, not fail open**: surface unhealthy via `/health` and
  do not silently accept stale sessions.

Recovery:

```bash
make status                 # confirm the Redis resource state
make restart                # bring Redis + API back under the lifecycle
curl -s "http://localhost:${API_PORT}/health"   # confirm Redis reachable again
```

After recovery, treat any sessions that should have been revoked during
the outage as suspect and re-revoke. See
[`../guides/troubleshooting.md`](../guides/troubleshooting.md).

## SQLite Location + Backup / Restore

Persistence is SQLite via the `api-core/storage` seam (no shared
Postgres). Default path:

```bash
echo "${SQLITE_PATH:-${SCENARIO_DATA_DIR}/scenario-authenticator.db}"
```

| Data | Backup | Restore |
|---|---|---|
| SQLite identity store (realms, users, credential hashes, refresh-token families, roles/scopes, audit events) | Snapshot `SQLITE_PATH` via the **data-backup-manager** scenario backup (storage namespace). | Restore the snapshot, then `make restart`. |
| **Signing keypair** (`private.pem`/`public.pem`) | Backed up as part of the storage namespace — **back it up with the DB, not separately**. | Restore the *same* keypair so issued tokens still verify. A different key invalidates all live tokens. |
| Redis hot state (sessions, CSRF, rate-limit counters) | Not backed up — reconstructable/ephemeral. | None needed; sessions re-establish on next sign-in. |

Back the SQLite DB and the keypair up **together and restore them
together** — a DB restored against a different signing key yields users
whose tokens no longer verify. Only hashes and signed material are at
rest; there are no plaintext secrets to protect in transit (OT-P0-004).

## Reading The Audit Log

Security-relevant events (sign-in, sign-out, token-family revoke, MFA
changes, admin actions) are recorded to a queryable audit log
(OT-P0-007). It is the primary security event stream — see
[`OBSERVABILITY.md`](OBSERVABILITY.md).

```bash
# Target shape — depends on the audit domain (P0, unbuilt):
scenario-authenticator audit list --realm default --since 1h
scenario-authenticator audit list --user <user-id> --event token_family_revoked
```

The audit log is append-only and queryable per realm; the admin console
surfaces the same data.

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Back up identity store + keypair | per backup policy | data-backup-manager snapshot of the storage namespace |
| Verify JWKS is serving the active key | after deploy / key rotation | `curl /.well-known/jwks.json` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Escalation

If you spot a defect outside your current scope, file it via the
`report-bug` workflow to scenario-qa (`prompt-manager skill read
report-bug` → knowledge-add). Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) and append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment shapes, key backup/rotation
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, audit as event stream
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../../PRD.md`](../../PRD.md) — operational risks, Appendix C (crypto invariants)
