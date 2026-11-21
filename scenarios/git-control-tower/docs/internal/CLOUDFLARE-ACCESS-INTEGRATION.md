# Cloudflare Access integration

Git Control Tower treats Cloudflare Access as an optional relying-party
provider. The API validates the `Cf-Access-Jwt-Assertion` at the standard
`api-core/server.Run` boundary and maps it to the shared `api-core/identity`
principal. GCT policy code remains responsible for domain authorization and
exact commit or branch approval.

## Runtime binding

The lifecycle derives the provider list from the manifest's `hybrid` profile
and asks tunnel-manager for the route's binding at startup. Tunnel-manager
uses its existing authority-backed Cloudflare credential once to read the
matching primary Access application and account organization. It returns only
non-secret verifier metadata:

```text
VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN=https://<team>.cloudflareaccess.com
VROOLI_CLOUDFLARE_ACCESS_AUDIENCE=<resolved-by-tunnel-manager>
VROOLI_CLOUDFLARE_ACCESS_REQUIRE_USER=true
```

Those values are lifecycle output, not per-scenario operator inputs. The
scenario declares `owner: tunnel-manager` and its route is the source of truth;
there is no audience tag to copy into every scenario. If tunnel-manager is
unavailable, the route is ambiguous, or the primary app metadata is missing,
lifecycle startup fails closed rather than silently starting with local-only
authority.

The team domain and audience are application-specific and are resolved by the
tunnel-manager owner. They are never inferred from an inbound claim. The
provider derives JWKS from the team domain and refreshes it when a previously
unknown `kid` appears. Operators configure Cloudflare credentials and route /
Access application ownership once in tunnel-manager, not once per scenario.

## Authority behavior

- A verified Cloudflare user is a human identity candidate.
- Service-token indicators, missing subject, missing email, invalid claims,
  expired tokens, and JWKS failures do not produce human authority.
- During migration, a scenario-authenticator credential may remain enabled.
  The two providers are accepted together only when an explicit,
  operator-approved identity mapping resolves them to the same canonical
  person. Matching email addresses or equal-looking subject strings are not
  sufficient. If the mapping is missing, ambiguous, expired, or resolves to
  different people, GCT reports a conflict and fails closed.
- Authentication is passive for read-only routes. Mutation handlers require a
  verified human principal and continue to require their existing exact intent
  and approval checks.
- Status responses expose source, state, failure class, and a configured
  recovery URL. They never expose a JWT, cookie, client secret, or raw claim
  blob.

## Operator verification

The remaining external proof is operator-owned because it requires the real
Cloudflare application and a human session:

1. Confirm the route hostname and primary Access application in the
   tunnel-manager account; the lifecycle binding endpoint should resolve the
   team domain and audience without scenario-specific environment variables.
2. Visit the GCT hostname without a session and verify the Access login/recovery
   path is shown.
3. Complete a human login and verify the authority status reports
   `cloudflare_access` and `verified`.
4. Exercise one mutation preview and one exact approved mutation.
5. Re-submit the consumed mutation intent and confirm the replay is refused;
   allow the Access session/assertion to expire (or use the provider's
   supported logout/reauthentication path) and confirm authority returns to a
   non-mutating state.
6. Verify a service token, wrong-audience assertion, stale assertion, forged
   caller header, and agent provenance cannot mutate. Repeat the same refusal
   checks against the direct origin when that origin is reachable for the
   deployment.

Record only typed status codes, authority state/source, operation identifiers,
intent identifiers, and resulting commit/rejection receipts. Do not record
Access cookies, JWTs, login codes, client secrets, or raw claim payloads.

The expected evidence sequence is: Access login → verified
`cloudflare_access` authority → read-only status → exact preview → one intent
issuance → one confirmed mutation → replay refusal → expiry or reauthentication
refusal. A missing receipt is an unproven gate, not a successful result.

No Cloudflare policy or tunnel configuration is changed by the application
code or by this document.

The default bundled GCT mode is `personal_local`, which does not require
Cloudflare Access or a human sign-in. Cloudflare Access is a remote/shared
provider choice and must be declared in the deployment manifest. See the
project [Identity and Authentication contract](../../../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md).
