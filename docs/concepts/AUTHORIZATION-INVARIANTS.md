# Authorization invariants

These statements are **normative**: every change to the shared auth stack
(`packages/api-core/{authn,identity,scopecatalog}` and the relying-scenario
authorization seams) is expected to preserve them. If a refactor cannot preserve
an invariant, the refactor is wrong — or the invariant has changed and this
document must change first.

The shared auth packages recompile roughly nineteen scenarios at once, and
`scenario-authenticator` (token issuance + JWKS), `tunnel-manager` (Cloudflare
Access bindings), and the hot-state store are fleet kill-switches. A bad
shared-package change therefore reaches every scenario simultaneously. Each
invariant below is paired with the test that pins it, so a regression that
violates one fails a test rather than leaking past review into a fleet outage.

This document mirrors the one-invariant-one-test discipline of
[`../../scenarios/agent-manager/docs/internal/INVARIANTS.md`](../../scenarios/agent-manager/docs/internal/INVARIANTS.md).
It is the rigor net described in section 5 of
[`../internal/auth-consolidation/MATURITY-RESEARCH-AND-MANIFEST.md`](../internal/auth-consolidation/MATURITY-RESEARCH-AND-MANIFEST.md)
and the executable companion to the model in
[`AUTHORIZATION.md`](AUTHORIZATION.md).

Format: **INV-n — statement — enforced at `<path>` — tested by `<test>`.**

## Shared identity & authority invariants

### INV-1 — Anonymous is refused for any protected operation

An unauthenticated request never satisfies a capability or operation gate. The
passive middleware labels it signed-out; it never produces a verified principal.

- **Enforced at** `packages/api-core/authn/operation.go` (`RequireOperation`,
  `RequireCapability`) and `packages/api-core/authn/authn.go` (`Middleware`).
- **Tested by** `packages/api-core/authn/invariants_test.go::TestInvariantAnonymousRefused`.

### INV-2 — A forged / bad-signature token is refused

A token signed by a key the JWKS does not publish, or whose payload was tampered
after signing, fails closed. The verifier never sends a presented token to the
issuer; it validates locally against the acquired JWKS.

- **Enforced at** `packages/api-core/authn/jwt.go` (`JWTVerifier.Verify`).
- **Tested by** `packages/api-core/authn/invariants_test.go::TestInvariantForgedTokenRefused`
  (and, for issuer/algorithm/expiry classes,
  `packages/api-core/authn/jwt_test.go::TestJWTVerifierFailsClosedForIssuerAudienceAlgorithmAndExpiry`).

### INV-3 — An expired token or an expired grant is refused

A principal past its token expiry, and an agent whose `OperationGrant` has
expired, are both refused — expiry is enforced on every lookup, not only at mint.

- **Enforced at** `packages/api-core/authn/operation.go` (`RequireOperation`
  expiry check) and `packages/api-core/identity/identity.go`
  (`OperationGrant.Allows`).
- **Tested by** `packages/api-core/authn/invariants_test.go::TestInvariantExpiredRefused`
  (standing-grant expiry also in
  `packages/api-core/authn/standing_grant_test.go::TestAuthorizeStandingGrantRefusalPaths`).

### INV-4 — Over-limit is refused — concurrency AND budget

A standing grant that covers the operation/target is still refused when a
declared bound is at or beyond its limit. Both the concurrency bound and the
budget bound are enforced (closing gap G1 — budget was previously advisory). The
over-limit refusal (`ErrStandingGrantLimit`) is distinct from "no authority"
(`ErrOperation`) and from "ask the operator" (`ErrNeedsApproval`).

- **Enforced at** `packages/api-core/authn/standing_grant.go`
  (`AuthorizeStandingGrant`).
- **Tested by** `packages/api-core/authn/invariants_test.go::TestInvariantOverLimitRefused`
  (fuller refusal matrix in
  `packages/api-core/authn/standing_grant_test.go::TestAuthorizeStandingGrantRefusalPaths`
  and `TestRequireOperationRefusesServiceOverLimit`).

### INV-5 — A service cannot impersonate a human

A verified service principal never satisfies `RequireHuman` /
`RequireIndependentHuman`, and the middleware never promotes a service to a human
state. The standing-grant ingress seam also refuses a human routed through the
service path (its own authorization stays separate).

- **Enforced at** `packages/api-core/authn/authn.go` (`RequireHuman`,
  `Middleware` state assignment) and `packages/api-core/authn/operation.go`
  (`RequireIndependentHuman`).
- **Tested by** `packages/api-core/authn/invariants_test.go::TestInvariantServiceCannotImpersonateHuman`
  (and the seam refusal in
  `packages/api-core/authn/standing_grant_test.go::TestRequireOperationStillRefusesUnknownActorThroughSeam`).

### INV-6 — Run identity alone does not authorize arbitrary run creation

A verified agent-run identity may inspect the service. Identity alone does not
authorize arbitrary top-level run creation: that requires an exact current
`run.create` grant on `agent-manager:*`. The exact-parent delegated-child route
is separately bound to the verified parent. Stop/stop-by-tag/continue/wake/recover require
their exact object-bound `agent.manage` grant or an allowed lineage route;
stop-all requires an `agent.manage` grant covering every run; delete requires an
exact `run.delete` grant (`run-deletion` policy). Approval, apply-investigation,
sandbox-sync and resume-as-new remain unavailable to run identities. An agent
principal that arrives without the run identity header is held to the same
grants.
Operator requests carry no run-identity token and are unaffected.

- **Enforced at** `scenarios/agent-manager/api/internal/handlers/run_identity_guard.go`
  (`denyRunInitiatedLifecycleOperation`) and the exact-parent check in
  `scenarios/agent-manager/api/internal/handlers/runs_create.go`.
- **Tested by** `scenarios/agent-manager/api/internal/handlers/run_identity_guard_test.go::TestRunIdentityGuardRefusesRunAuthorityAndUnverifiedClaims`
  (ungranted create refusal), `scenarios/agent-manager/api/internal/handlers/run_create_grant_guard_test.go::TestRunCreateGrantGuardUsesExactCanonicalGrantRule`,
  `scenarios/agent-manager/api/internal/handlers/task_create_authority_test.go::TestTaskCreateUsesOneRunCreateAuthority`, and
  `scenarios/agent-manager/api/internal/handlers/runs_create_test.go::TestCreateRunAllowsOnlyVerifiedParentLinkedChildren`.

### INV-7 — A child run never broadens its parent

A delegated child credential is the intersection of owner scopes ∩ profile
ceiling ∩ requested scopes, expires no later than its parent, and must share the
parent's admitted owner. A child requesting a scope the parent never held is
refused outright — no elevation across the delegation boundary.

- **Enforced at** `scenarios/agent-manager/api/internal/orchestration/delegation.go`
  (`MintDelegatedIdentity` + `identity.Attenuate` / `IntersectScopes`).
- **Tested by** `scenarios/agent-manager/api/internal/orchestration/delegation_test.go::TestInvariantChildRunNeverBroadensParent`
  (full ceiling/expiry/owner matrix in `TestMintDelegatedIdentityRespectsChildAuthority`).

### INV-8 — One decision path per operation (middleware and handler agree)

There is exactly one evaluator for whether a costly operation is authorized. A
forwarded human credential cannot widen an agent's grant at the middleware, and
`RequireOperation` reaches the same verdict the middleware set up. No second,
always-on gate silently overrides the decision.

- **Enforced at** `packages/api-core/authn/operation.go` (`RequireOperation`)
  and `packages/api-core/authn/authn.go` (`Middleware` grant intersection).
- **Tested by** `packages/api-core/authn/invariants_test.go::TestInvariantOneDecisionPathPerOperation`
  (forwarded-owner matrix in
  `packages/api-core/authn/operation_test.go::TestForwardedOwnerProofCannotExpandAnInteractiveGrant`).

### INV-9 — Secret custody is separate from identity

Grant and run identity live in Agent Manager's authoritative run domain; secret
custody uses the existing credential authority, and diagnostics never print
plaintext run/owner/grant credentials. A lease/process/cgroup check establishes
attribution, not isolation from owner secrets — so the configured human-consent
boundary is required rather than relying on file modes or an environment marker.
This is an architectural invariant (prose), enforced by the separation of the
credential-authority package from the identity packages.

- **Enforced at** `packages/credential-authority-go/**` (custody) kept distinct
  from `packages/api-core/{authn,identity}` (identity); grant storage contract
  in [`AUTHORIZATION.md`](AUTHORIZATION.md) §"Grant storage and resolution
  invariants".
- **Tested by** the credential-authority package tests and the grant-storage
  contract; no identity package imports a plaintext-secret custodian.

### INV-10 — Third-party proof is never trusted as a human without the human claim

A Cloudflare Access assertion (or any third-party proof) is accepted as human
only when it verifies to a non-empty human user claim. An absent/empty/invalid
assertion never yields a human principal, and capabilities are never inferred
from tunnel reachability, email equality, or an unsigned owner field — only from
an explicit owner binding.

- **Enforced at** `packages/api-core/authn/managed_cloudflare.go`
  (`ManagedCloudflareProvider.VerifyRequest`) and
  `packages/api-core/authn/authn.go` (owner-binding application).
- **Tested by** `packages/api-core/authn/invariants_test.go::TestInvariantThirdPartyProofNotTrustedAsHuman`
  (owner-binding exactness also in
  `packages/api-core/authn/operation_test.go::TestOwnerBindingRequiresExactVerifiedProviderSubject`).

### INV-11 — Audience mismatch is rejected

A token minted for one application audience (e.g. `scenario-authenticator:default`)
is refused where a different resource audience is required. A token for one app
cannot be replayed at another. The only exception is an explicitly declared,
bounded `AcceptedAudiences` migration set.

- **Enforced at** `packages/api-core/authn/jwt.go` (audience check in `Verify`).
- **Tested by** `packages/api-core/authn/invariants_test.go::TestInvariantAudienceMismatchRejected`
  (bounded migration audience in
  `packages/api-core/authn/jwt_test.go::TestJWTVerifierAcceptsBoundedMigrationAudience`).

### INV-12 — Scope-catalog wildcard matrix is exact

The coarse scope grammar accepts exactly: an exact `scenario:effect`, a scenario
wildcard `scenario:*`, an effect wildcard `*:effect`, and the full wildcard `*`.
Whitespace is not normalized and malformed scopes never grant access, so a
narrower held scope never covers a wider required one. `RequireCapability` and
the catalog evaluator must agree.

- **Enforced at** `packages/api-core/scopecatalog/catalog.go` (`Resolve`,
  `MatchCapability`) and `packages/api-core/authn/authn.go`
  (`RequireCapability`).
- **Tested by** `packages/api-core/authn/invariants_test.go::TestInvariantScopeCatalogWildcardMatrix`.

## The cross-scenario auth smoke gate

The single highest-ROI blast-radius net is an end-to-end **issue → verify →
decide** test that uses the real shared provider and the real standing-grant
decision — no fakes on the hot path. It models scenario-authenticator issuance
(RS256 sign + JWKS publish), a relying scenario verifying through the shared
`JWTVerifier`, and the one authority decision via the standing-grant ingress. It
is self-contained (an in-process JWKS server), so it needs no running services
and can gate a PR or pre-merge hook.

**Run it as a single command:**

```bash
cd packages/api-core && go test ./authn/ -run TestCrossScenarioAuthSmoke -v
```

It fails if a shared `packages/api-core/**` auth change breaks token issuance,
cross-scenario verification, or the authority decision — the blast-radius class
that a bad shared change would otherwise ship to every scenario at once.

- **Test:** `packages/api-core/authn/smoke_test.go::TestCrossScenarioAuthSmoke`.

To run the full shared-auth invariant suite (all of the above plus the smoke
gate) in one command:

```bash
cd packages/api-core && go test ./authn/ ./identity/ -run 'TestInvariant|TestCrossScenarioAuthSmoke|TestManagedCloudflareBinding'
```

## The Cf-Access binding reachability startup self-check

Human operators authenticate a session-delegation mint through the web-console
Cloudflare Access binding, which `scenario-authenticator` resolves from
tunnel-manager. If that binding is misconfigured or tunnel-manager is
unreachable, the failure would otherwise surface only on the first real operator
login — a silent fleet human-login outage. The authenticator probes the binding
once at startup and surfaces the result loudly.

Behavior:

- **Broken / unreachable binding** (resolution error, or a resolvable binding
  that cannot build a verifier): the self-check returns
  `authn.ErrBindingUnreachable` and the authenticator logs a loud `WARNING`.
- **Good binding:** the self-check passes, logs a confirmation, and primes the
  provider's resolution cache so the first real operator login does not pay the
  resolution.

It is deliberately **best-effort, not boot-fatal**: resolution depends on a live
tunnel-manager, and a healthy authenticator must still boot (to serve native
logins and JWKS) when Cloudflare is simply not configured in a deployment. This
matches the authenticator's broader fatal-vs-degrade convention — the signing
key and hot-state store are boot-fatal (they are local, security-critical, and
always required), whereas an external tunnel dependency degrades loudly.

- **Enforced at** `packages/api-core/authn/managed_cloudflare.go`
  (`ManagedCloudflareProvider.CheckBindingReachable`), wired into
  `scenarios/scenario-authenticator/api/main.go` alongside the other startup
  validation.
- **Tested by** `packages/api-core/authn/managed_cloudflare_test.go::TestManagedCloudflareBindingReachabilitySelfCheck`
  (fails on a broken/unusable binding, passes on a good one, and proves the
  cache is primed).

## How to add an invariant here

1. The statement must be checkable. "Don't do X" without a test that fails when
   someone does X is not an invariant.
2. Pair the statement with the test that pins it, in the package where the
   invariant actually lives. If a test already exists, reference it rather than
   duplicating.
3. Name the enforcement path so a future agent can find the one place that
   decides.

## Related

- [`AUTHORIZATION.md`](AUTHORIZATION.md) — the authorization model these
  invariants protect.
- [`IDENTITY-AND-AUTHENTICATION.md`](IDENTITY-AND-AUTHENTICATION.md) — how each
  identity kind is proven.
- [`../internal/auth-consolidation/MATURITY-RESEARCH-AND-MANIFEST.md`](../internal/auth-consolidation/MATURITY-RESEARCH-AND-MANIFEST.md)
  — the research and the rigor-mitigation plan (section 5).
- [`../../scenarios/agent-manager/docs/internal/INVARIANTS.md`](../../scenarios/agent-manager/docs/internal/INVARIANTS.md)
  — the agent-manager invariant pattern this mirrors.
