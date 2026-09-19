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
