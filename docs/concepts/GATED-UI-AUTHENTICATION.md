# Gated UI authentication

This document defines provider ingress for configured UI surfaces. It is not
the full identity model. Read
[`IDENTITY-AND-AUTHENTICATION.md`](IDENTITY-AND-AUTHENTICATION.md) first for
the distinction between person identity, local desktop identity, business
accounts, entitlements, and scenario authorization.

Scenario manifests may declare an `authentication` profile in
`.vrooli/service.json`. The profile is non-secret metadata:

```json
{
  "authentication": {
    "profile": "hybrid",
    "hostname": "gct.example.test",
    "team_domain": "https://team.cloudflareaccess.com",
    "audience": "one-application-audience",
    "policy_mode": "operator_managed",
    "public_asset_path": "/public",
    "owner": "tunnel-manager"
  }
}
```

`cloudflare_access` and `hybrid` require `team_domain`, `audience`, and
`owner`. The API receives these non-secret identifiers through the lifecycle
binding. For managed routes, lifecycle asks tunnel-manager to resolve them
from the primary Access application and account organization, so operators do
not repeat an audience tag in every scenario. The resulting binding includes
`VROOLI_AUTH_PROVIDERS=cloudflare_access` and
`VROOLI_CLOUDFLARE_ACCESS_*`; it never receives Cloudflare management or
connector credentials. `packages/api-core/cloudflareaccess` validates the
signed `Cf-Access-Jwt-Assertion` at the origin, and `packages/api-core/authn`
composes it with any provider-owned Vrooli verifier.

The shared middleware is passive for profiles without a provider. It records
signed-out, expired, service, agent, and conflict states for same-origin UI
status while leaving read routes available. Domain handlers decide whether a
verified human may perform an operation and retain any exact, single-use
approval requirements.

## Human approval crosses the scenario boundary

A verified `Cf-Access` token proves a human at browser ingress, but on its own it
does not reach a scenario the operator spawns work in. When a verified-human UI
session launches a run, the session mints a short-lived, session-scoped
`OperationGrant` bound to the human owner, and that grant is carried into the
spawned agent's request — so the created run is attributed to the operator
(non-empty creator) and the agent inherits the operator's authority, bounded to the
session and to what it was launched to do. The grant expires with the session; a
caller without a verified human session cannot obtain it. This is one of the three
ways an `OperationGrant` comes to exist; see
[`AUTHORIZATION.md`](AUTHORIZATION.md) for the full model.

Tunnel-manager's existing `/public` capability is only an anonymous public
asset exception. It is not a human-authentication provider, does not provision
the primary identity application, and does not verify runtime user JWTs.

## Profile examples

- `cloudflare_access`: one Access application is the browser authority.
- `scenario_authenticator`: the Vrooli IdP remains the only provider.
- `hybrid`: both providers may be accepted only after an explicit trusted
  identity mapping proves that they represent the same local principal; a
  provider-subject string comparison is not sufficient. A conflict fails
  closed.
- `local_read_only`: no human mutation authority is implied.

Per-application audiences are preferred. An Access application must use an
identity-based Allow policy for human authority; Service Auth and Everyone
bypass are never human defaults.

This profile does not define local desktop single-user mode. A bundled app may
use `personal_local` with no human sign-in, while its Electron shell still
authenticates to the desktop supervisor with the app-private loopback token.
It also does not define LPBS subscription or download entitlements; those
remain owned by the commerce authority.
