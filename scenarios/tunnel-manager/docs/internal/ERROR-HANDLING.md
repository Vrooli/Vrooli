# Error Handling

The template uses one error path for proto-typed operations and one
documented exception path for multipart REST.

## Proto-Typed Operations

Proto-typed UI, CLI, and inter-scenario calls use Connect-RPC. Errors
move through three layers:

1. Domain/service code returns typed sentinels such as
   `<domain>.ErrInvalid<Entity>` or `<domain>.Err<Entity>NotFound`.
2. The API transport edge maps those sentinels to `connect.Error`
   values in `internal/<domain>/service_error_mapping.go`.
3. The UI receives `ConnectError`, maps `ConnectError.code` to an
   `errors.<code>` i18n key with `ui/src/lib/errorMessage.ts`, and
   renders localized copy.

The CLI uses the same `connect.Error` values through cli-core. Human
output is English for now; future CLI i18n should use the same code
names as the UI catalog instead of string-matching messages.

## Sentinel Mapping

| Domain error | Connect code | UI i18n key |
|---|---|---|
| `ErrInvalid<Entity>` | `invalid_argument` | `errors.invalid_argument` |
| `Err<Entity>NotFound` | `not_found` | `errors.not_found` |
| Unknown service/repository error | `internal` | `errors.internal` |

When you add a domain, keep the mapping file next to that domain's
service layer. The handler should call the mapper instead of switching
on domain error types inline.

## Failure-Classification Model (planned)

> **Status: planned, not built.** Implementation is Phase 2. This is the
> failure taxonomy the `probes` and `recovery` domains
> ([`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)) will use — distinct
> from the per-RPC error mapping above. A *classification* describes
> **where** connectivity broke; the `recovery` engine maps each
> classification to a targeted action (OT-P1-001, OT-P0-011).

Current classification is derived from the *pattern* of the latest
internal-vs-external probe results. Tunnel health (HA connections,
`/ready`), DNS resolver detail, and Cloudflare edge-wide outage signals
are planned inputs, not currently part of `probes.Classify`.

| Classification | Signal pattern | Meaning |
|---|---|---|
| `healthy` | Internal and external probes both pass. | The route is reachable locally and through the tunnel. |
| `tunnel-down` | Internal probe passes; external probe fails. | The scenario is running locally but is not reachable through the tunnel/edge layer. |
| `scenario-down` | Internal and external probes both fail. | The scenario is the most likely culprit. |
| `config-drift` | Internal probe fails; external probe passes. | The public route still serves while the configured local port/path does not, so manifest/config likely drifted. |
| `cloudflare-outage` | Not currently produced. Future signal: broad external failures while internal + `/ready` remain healthy and Cloudflare/API status indicates upstream trouble. | The failure is upstream at Cloudflare, not local. |
| `dns-failure` | Not currently produced. Future signal: resolver-specific hostname failure from the external probe. | DNS/hostname misconfiguration, not a process failure. |

### Classification → recovery action

| Classification | Recovery action |
|---|---|
| `tunnel-down` | Restart cloudflared via the exec/systemd seam, with exponential backoff + circuit breaker. Tunnel Manager is the **single authoritative owner** of this restart ([`DECISIONS.md`](DECISIONS.md)). |
| `scenario-down` | Ensure the scenario is running via the lifecycle seam (delegate, never reimplement); do **not** restart cloudflared. |
| `cloudflare-outage` | Planned: do not actuate — alert/record only; restarting local infra cannot fix an upstream outage. |
| `dns-failure` | Planned: surface a config/DNS error to the operator; do not restart cloudflared. |
| `config-drift` | Re-push ingress / regenerate config from the manifest (`config.Sync`) to reconcile; no restart unless drift persists. |

Every attempt — trigger classification, action, and outcome — is
persisted to the `recovery_events` table for post-incident review
(OT-P1-005). The circuit breaker bounds the blast radius of live
actuation (see [`SECURITY.md`](SECURITY.md) and the timings in
[`PERFORMANCE.md`](PERFORMANCE.md)).

## Connect Error Codes — Product Domains (planned)

> **Status: planned, not built.** The product domains follow the same
> three-layer typed-sentinel → Connect-code → UI-i18n path as the
> template above. Representative mappings:

| Condition | Connect code | Notes |
|---|---|---|
| Scenario/route not ready, or recovery in a circuit-broken state | `failed_precondition` | e.g. expose requested but the scenario can't be ensured running, or an action is blocked while the breaker is open. |
| Unknown route, lease, or scenario | `not_found` | Lookup by subdomain/lease-id/scenario that doesn't exist. |
| Invalid exposure request (bad subdomain, non-fixed port, bad TTL) | `invalid_argument` | Validation at the `routes`/`exposure` service layer. |
| Cloudflare API auth/permission failure | `permission_denied` | Token rejected by Cloudflare v4. |
| Cloudflare outage / upstream unavailable | `unavailable` | Distinguishes transient upstream failure from a local bug. |
| Lease already exists / duplicate subdomain | `already_exists` | One route per subdomain. |
| Unexpected service/repository error | `internal` | Underlying error reaches operator logs (buffer-logger pattern), generic message to caller. |

Keep each domain's mapping in `internal/<domain>/service_error_mapping.go`
beside its service, as the template prescribes.

## Operator-Facing Error Surfacing (planned)

Tunnel Manager is operator infrastructure, so errors must be legible at
a glance:

- **CLI**: `--json` everywhere emits the proto-typed error contract;
  human output renders the Connect code + message via cli-core. Failure
  classifications and recovery outcomes appear in `status`/`recover`
  output so an operator can see *why* a route is down, not just *that*
  it is.
- **UI**: the 5-surface dashboard maps `ConnectError.code` to localized
  copy via `errors.<code>` keys (same catalog as every domain), and the
  Recovery & Events surface renders the classification, action, and
  outcome from the `recovery_events` log. Color-coded health
  (green/yellow/red) per the PRD's "quick glance" UX.

## Multipart REST Exceptions

Opaque file bytes are not proto payloads. Upload endpoints use REST
multipart for bytes and return proto-typed metadata. These endpoints
still use a stable error envelope through `internal/httpx.WriteError`;
the UI maps `ApiError.code` through the same `errorMessage(...)`
utility as Connect errors.

Use this split:

- Connect-RPC for messages that can be described by proto.
- REST multipart for file bytes.
- Proto metadata responses for REST upload results.

Do not introduce a second general JSON transport for internal scenario
calls. If the payload is structured and Vrooli-owned, add a proto
service method.
