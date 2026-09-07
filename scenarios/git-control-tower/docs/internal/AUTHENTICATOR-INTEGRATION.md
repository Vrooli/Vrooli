# Scenario-authenticator integration

Git Control Tower is a relying party. It does not mint, refresh, or infer
human credentials.

## Deployment modes and account boundaries

GCT preserves the installation's declared deployment mode. The default
bundled mode is `personal_local`: the local operator uses the app without a
human sign-in, and the desktop supervisor token is only a loopback process
credential. It must never be presented as a human identity or forwarded to a
remote provider.

`local_multi_user`, `remote_vrooli`, and `shared_provider` are explicit
operator choices. They require `scenario-authenticator` identity and policy
before GCT exposes shared or remote capability. An LPBS website session is
not a substitute for this local authority. If paid features are enabled, the
operator separately links the local identity to an LPBS business account;
that link does not broaden GCT's Git policy.

The project-level mode and ownership contract is [Identity and
Authentication](../../../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md).

## Contract

- Issuer: `scenario-authenticator`.
- Compatibility realm: `default`.
- Compatibility audience: `scenario-authenticator:default`. New resource
  profiles should use a resource-specific audience; this compatibility value
  remains only while the migration is in progress.
- Signature: RS256 using the authenticator JWKS endpoint
  `/.well-known/jwks.json`, resolved through api-core scenario discovery when
  `GCT_AUTH_JWKS_URL` is not set.
- JWKS freshness: keys are cached for five minutes by default. Set
  `GCT_AUTH_JWKS_TTL_SECONDS` to a positive deployment-specific value. An
  unknown `kid` forces one immediate refresh, so key rotation is not delayed
  by the normal cache window. A refresh failure fails authentication closed;
  previously cached keys are not used past their freshness window.
- Identity: `user_id` first, then `sub`; the principal also carries email,
  roles, scopes, issuer, and realm.
- Accepted transport: `Authorization: Bearer <token>` or the same-origin
  `gct_access_token` / `access_token` cookie. Credentials are never logged.
- Browser sign-in: the GCT UI exposes a same-origin typed `AuthService.Login`
  facade. That API-to-API call forwards to scenario-authenticator's typed
  `AccountsService`; browser JavaScript never calls a sibling scenario.
- Hosted IdP entry: scenario-authenticator also exposes `/auth/login` with an
  explicit `View read-only version` action. That page does not put bearer
  tokens in redirect URLs.

The middleware records missing, invalid, expired, wrong-audience, wrong-issuer,
unsupported-algorithm, unknown-key, and authenticator-unavailable outcomes in
request context. Read-only requests continue with an unauthenticated standing;
mutation boundaries return an actionable 401 or 403 before a writer runs.

## Revocation

Local JWKS validation is the normal hot path. The authenticator's short-lived
access-token expiry is the default revocation boundary. Deployments that need
logout or password-change revocation to take effect immediately set
`GCT_AUTH_VALIDATE_URL` to the authenticator API base URL. GCT then performs a
typed `AccountsService.Validate` call after local cryptographic checks; a
blacklisted token is treated as revoked and cannot reach a mutation boundary.
The callback is injectable in policygate tests, where a rejected live session
is proven to fail closed without disclosing the token.

## Failure behavior

| Condition | Read-only request | Mutation request |
|---|---|---|
| No credential | continue as signed out | 401: sign in through scenario-authenticator |
| Invalid/expired/revoked credential | continue as signed out | 401: credential invalid or unavailable; sign in again |
| Verified agent attribution | continue and may prepare evidence | 403: agents cannot mutate or issue human intents |
| Verified human, no exact intent | continue reads | 401/409 before the Git writer |

The caller identity headers are deliberately non-authoritative. An agent or
curl process cannot become human by sending `X-Vrooli-Caller: human` or
`X-Vrooli-Authorized: true`.
