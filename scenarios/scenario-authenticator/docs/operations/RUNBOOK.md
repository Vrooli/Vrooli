# Runbook — Scenario Authenticator

This document records operator procedures for running, diagnosing,
recovering, and maintaining the fleet's Identity Provider (IdP).

> **Status: implemented foundation.** Lifecycle, account registration and
> login, RS256/JWKS publication, refresh-token rotation, session revocation,
> audit, rate limiting, and password change are live. The procedures below
> identify shipped paths and explicitly label future capabilities such as
> true multi-realm administration and automated key rotation.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- How do I create the default realm and the first account?
- How do I preserve signing keys and revoke sessions during an incident?
- How do I back up and restore identity state?

## Start / Stop / Status

Use lifecycle-managed commands from the scenario directory:

```bash
make setup     # build API/CLI/UI, install scenario CLI (run once / on dep changes)
make start     # start API + UI; lifecycle starts Redis when selected
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

`/health` should report SQLite (storage seam) and the selected hot-state
implementation. A single-replica deployment may use durable local hot state;
when shared Redis is selected, its reachability must be healthy. JWKS must
return the active public key — if it is empty or 404s, RPs cannot verify
tokens.

## Create The Default Realm + First Account

On a fresh install the **default realm** is created at first boot and
issues `aud`-scoped tokens (OT-P0-008). Registering the first local account
is a one-time, first-run step (the device-sync-hub live first-run owner
bootstrap is the reference flow, OT-P0-012):

```bash
# The default realm is created by boot; registration uses a masked prompt.
scenario-authenticator auth register --realm default \
  --email admin@example.com                               # first account
```

The same surface exists as Connect RPCs. The complete admin-console UI remains
future work; use the CLI or typed API during bootstrap.
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

Overlapping-key rotation and retirement are OT-P2-005 and are not built.
There is no supported key-rotation CLI command yet. The safe current
procedure is to preserve and back up the existing pair; do not delete or
regenerate it as an attempted rotation, because that invalidates every live
token across every Relying Party. A future rotation implementation must add
an API/CLI contract and evidence before operators use it.

## Revoke A Session / "Revoke All" (incident response)

Sessions are server-tracked with the configured hot-state store (OT-P0-005).
Cross-scenario consumers use the generated `SessionsService` client.

```bash
# Session listing and revocation are implemented:
scenario-authenticator sessions list --user <user-id>    # enumerate active sessions
scenario-authenticator sessions revoke --id <session-id> # revoke one session
scenario-authenticator sessions revoke-all --user <user-id>   # "log out everywhere"
```

For a credential-compromise incident: `revoke-all` for the affected
principal, then force a password reset (identity domain). Presenting a
**rotated/reused refresh token** automatically revokes the entire token
family (reuse detection, OT-P0-003) and is recorded in the audit log.
Every revoke is an audited security event.

## Shared hot-state outage — behavior and recovery

When shared Redis is configured and unavailable, the service must not silently
switch to a local store. During an outage:

- Relying parties can continue verifying already-issued tokens locally while
  their JWKS cache is valid; that verification does not touch the provider's
  hot-state store.
- Authenticator operations that need hot state (login, refresh, session
  revocation, and rate limiting) fail closed or remain unavailable. The system
  must not silently accept stale sessions or switch to an uncoordinated local
  store.

When no shared Redis configuration exists, the single-replica deployment uses
the durable local hot-state store. That mode must not be used as a substitute
for shared state in a multi-replica deployment.

Recovery:

```bash
make status                 # inspect the selected hot-state resource
make restart                # restart the selected dependencies and API
curl -s "http://localhost:${API_PORT}/health"   # confirm hot state is healthy
```

After recovery, treat any sessions that should have been revoked during
the outage as suspect and re-revoke. See
[`../guides/troubleshooting.md`](../guides/troubleshooting.md).

## Backup / Restore

Persistence is SQLite via the `api-core/storage` seam (no shared
Postgres). Default path:

```bash
echo "${SCENARIO_DATA_DIR}/scenario-authenticator.db"
```

| Data | Backup | Restore |
|---|---|---|
| SQLite identity and durable hot-state store (realms, users, credential hashes, roles/scopes, audit events, and local hot state) | Snapshot the scenario database via the **data-backup-manager** scenario backup (storage namespace). | Restore the snapshot, then `make restart`. |
| **Signing keypair** (`private.pem`/`public.pem`) | Backed up as part of the storage namespace — **back it up with the DB, not separately**. | Restore the *same* keypair so issued tokens still verify. A different key invalidates all live tokens. |
| Shared Redis hot state (sessions, CSRF, rate-limit counters) | Not backed up — reconstructable/ephemeral. | None needed; sessions re-establish on next sign-in. |

Back the SQLite DB and the keypair up **together and restore them
together** — a DB restored against a different signing key yields users
whose tokens no longer verify. Only hashes and signed material are at
rest; there are no plaintext secrets to protect in transit (OT-P0-004).

## Reading The Audit Log

Security-relevant events (sign-in, sign-out, token-family revoke, MFA
changes, admin actions) are recorded to a queryable audit log
(OT-P0-007). It is the primary security event stream — see
[`OBSERVABILITY.md`](OBSERVABILITY.md).

Audit inspection is implemented through the audit repository/API, but the
current CLI and complete admin-console query surface do not expose it yet.
Use the typed API only from an authorized operator integration until the
dedicated audit query contract is shipped.

The audit log is append-only and queryable per realm; a complete admin-console
audit surface remains future work.

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
