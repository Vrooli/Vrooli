# API Endpoints — Secrets Manager

## System

### `GET /health`

Returns lifecycle and dependency posture. `GET /api/v1/health` is the API-prefixed form for UI consumers.

## Domain Endpoints

| Area | Routes | Purpose |
|---|---|---|
| Credentials | `/api/v1/credentials/secrets/status`, `/validate`, `/provision` | Metadata-only coverage, validation, and stdin-guarded provisioning through the credential authority |
| Security | `/api/v1/security/scan`, `/compliance`, `/vulnerabilities` | Scan and posture reporting |
| Resources | `/api/v1/resources/{resource}` | Resource detail and strategy mutation |
| Deployment | `/api/v1/deployment/secrets`, `/readiness` | Bundle-safe strategy manifests |
| Scenarios | `/api/v1/scenarios`, overrides routes | Inventory and strategy overrides |
| Operations | orientation, campaigns, allowlist, watchlist, receipt signing | Operator workflows and supporting controls |
| Password manager | `/api/v1/vaults`, `/api/v1/grants`, `/api/v1/access-requests`, `/api/v1/assurance`, `/api/v1/audit`, `/api/v1/recovery/status`, `/api/v1/recovery/activate` | Encrypted vault metadata, bounded delegation, one-time assurance, safe activity, recovery evidence, and epoch-bound recovery activation |
| Workspace security | `/api/v1/enrollment/*`, `/api/v1/members`, `/api/v1/machine-principals` | One-time owner bootstrap plus workspace-scoped member roles and revocation; owner/admin manage membership and one-time hashed machine enrollment, viewers remain metadata-only, and security operators read audit metadata |
| Sources | `/api/v1/sources`, `/api/v1/sources/{id}/health`, `/api/v1/vaults/{vault}/items/{id}/source-binding` | Explicit provider registration, capability declarations, and item bindings; unadapted external providers report `unavailable` without fallback reads |
| Native host | `/api/v1/native-host/enroll`, `/api/v1/native-host/unlock`, `/api/v1/native-host/fill`, `/api/v1/native-host/revoke` | Installed-host transport plus owner authentication; exact origin, grant, field, and item-revision checks precede a one-field fill |
| Credential use | `/api/v1/credential-use/browser/*`, `/api/v1/credential-use/ssh/*` | Origin/document-bound browser login through a trusted executor and destination/principal-bound SSH signing; values and private keys stay inside the authority boundary |

Access requests are durable authority records. `POST /api/v1/access-requests` accepts an
optional `Idempotency-Key`; retries from the same workspace and requester return the
original request, while a changed scope returns `revision_conflict`. `GET
/api/v1/access-requests/{id}` resumes a request after a service restart, and `POST
/api/v1/access-requests/{id}/wait` provides a bounded server-side wait for the terminal
state. States are `pending`, `approved`, `denied`, and `expired`; a second concurrent
decision returns `decision_conflict`, and the request digest is required for approval
or denial.

Secret values and authority management credentials are not response fields.

Broker sessions are created only from grants with a pinned `target` origin. The
HTTP adapter resolves and pins destination addresses at session creation,
accepts read-only `GET` or `HEAD` operations with a small header allowlist,
rejects redirects, caps request and response bodies, and replaces the exact
credential plus common encoded forms in the projected response. Set
`one_use: true` for a capability that is consumed by its first operation;
otherwise the session remains bounded by its expiry and grant status.

Credential-use sessions require an `inject` grant for browser login and a
`sign` grant for SSH. Browser sessions bind the exact normalized origin,
selected account, and document identity. Navigation to another origin or a
changed document requires new authorization. The protected BAS policy rejects
arbitrary evaluation, extraction, cookie/storage access, and secret-field
reads; a trusted executor receives the fields only for fill and receives a
clear request after a failed submission. TOTP generation occurs inside that
trusted executor path. The current implementation reports an explicit
unsupported capability when BAS has not been wired to the trusted fill
adapter; it never falls back to raw reveal. SSH signing returns only the
algorithm and encoded signature, with destination, principal, lifetime, grant,
vault-lock, and revocation checks applied before each signature.

Pages can observe fields they receive during their own login flow. This
boundary therefore does not claim model isolation or arbitrary account-action
restriction; it constrains the agent-facing executor surface and closes local
handles on expiry, vault lock, grant revoke, and run completion.

Recovery activation requires an owner/admin, fresh action-bound assurance for
`recovery:activate`, and the current recovery epoch. It advances that epoch,
invalidates old use capabilities, and quarantines restored grants for explicit
reauthorization. Recovery status reports the epoch and separate backup/restore
evidence; it does not claim replacement-host usability from a checksum alone.

## Adding A New Endpoint

Add the handler to its capability route group in `api/server.go`, add handler tests, update `.vrooli/endpoints.json` through `make endpoints`, and document the stable contract here.

## Cross-References

- [CLI Commands](cli-commands.md)
- [Error Handling](../internal/ERROR-HANDLING.md)
