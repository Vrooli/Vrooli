# Authorization

Scenario Authenticator owns principal scope assignment and token claim
emission. It does not own the permission decision for another scenario and it
never imports or interprets that scenario’s CLI manifest.

## Scope grammar

Scopes are additive and use two axes:

```text
<scenario>:<effect>
```

`effect` is `read`, `write`, or `destructive`. `*` may replace either axis:
the universal wildcard, a global read wildcard, and a bridge-scenario
wildcard are valid grants. There are no deny
rules. An absent scope claim is never permission; issued tokens carry an
explicit list, including an empty list.

The shared catalog is derived from `governance.effect` in existing scenario
`cli/manifest.json` files. The authenticator stores strings opaquely and does
not validate them against that catalog. This keeps the identity provider
independent from every relying-party release cycle.

## Assignment and enforcement

### Trust posture defaults

The shared typed posture reader consumes `.vrooli/operator-state.json` and
defaults a missing `trust_posture` to `personal`. The posture never disables
verification. Its defaults are:

| Posture | Access-token TTL | Break-glass | Default node execution scopes | JWKS cache grace |
|---|---:|---|---|---:|
| `personal` | 60 minutes | available, 15-minute credential | `vrooli-bridge:read`, `vrooli-bridge:write` | 24 hours |
| `shared` | 15 minutes | available, 10-minute credential | none | 4 hours |
| `hosted` | 10 minutes | unavailable | none | 1 hour |

An operator transition is separate from agent-readable state and is recorded
as a typed `trust_posture.transition` event. No posture is a fail-open mode.

The break-glass credential is provisioned with machine linking, not through a
separate ceremony. It is an Ed25519-signed, time-boxed capability verified
against a pinned public key without a live authenticator call. Its requested
scopes are intersected with the account's scopes before signing, so it cannot
exceed the owner's grant. The relying party records accepted use with the
typed `ActionBreakGlass` audit action and refuses if that audit write fails.

Linking writes the private key and provisioning metadata under the owner-only
break-glass directory and pins the public key for the local bridge. The CLI's
`auth issue-break-glass` command writes a short-lived credential to a required
owner-only token file; it never prints the credential. The local Unix-socket
exchange is preferred for normal CLI login, with `VROOLI_AUTH_TOKEN_FILE` as
the platform-agnostic fallback. Unlinking a binding does not revoke JWTs that
were already issued; they remain valid until expiry or explicit revocation.

The authorization domain exposes these `AccountsService` RPCs:

- `GrantScope(access_token, principal_id, scope)` assigns an opaque string.
- `RevokeScope(access_token, principal_id, scope)` removes it.
- `ListScopes(access_token, principal_id)` returns the current list.

Each grant and revoke writes a typed `scope.granted` or `scope.revoked` audit
event. The authenticated principal may manage its own assignments; the
principal id defaults to the token subject and cross-principal mutation is
rejected until an explicit administrative policy exists.

Relying parties read the verified `scope` claim and enforce their own required
scope. The pure resolver in the derived catalog handles exact and supported
wildcard matches; the authenticator does not perform that decision.

Every issued JWT contains `scope`, including `"scope": []` when the principal
has no assignments. `ValidateResponse` and the account response expose the
same list.

Agent delegation is one-way: a derived run token contains an explicit,
materialized scope list that is the intersection of the account, profile, and
request grants. A child token can only be narrower or equal in scope and
expiry, and parent token material is scrubbed before the child starts.

## Current boundary

The default realm is still the only realm. Windows peer credentials, API keys,
interactive bridge sessions, true multi-realm support, and bridge gap G8 are
deferred with revisit triggers in the plan and the requirements registry.
