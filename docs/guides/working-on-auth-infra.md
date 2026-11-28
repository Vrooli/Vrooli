# Working on auth-infra safely

Changing the shared auth stack is the highest-blast-radius edit in the fleet. The
shared packages `packages/api-core/{authn,identity,scopecatalog}` recompile into
roughly nineteen scenarios at once, and three services are fleet **kill-switches**:
`scenario-authenticator` (token issuance + JWKS), `tunnel-manager` (Cloudflare
Access bindings), and the hot-state store. A bad shared change reaches every
scenario simultaneously, so a regression that slips past review becomes a fleet
outage, not a one-scenario bug.

This runbook is the safety procedure. The model is in
[`../concepts/AUTHORIZATION.md`](../concepts/AUTHORIZATION.md); the normative,
test-pinned rules are in
[`../concepts/AUTHORIZATION-INVARIANTS.md`](../concepts/AUTHORIZATION-INVARIANTS.md).

## Keep operator authority distinct from agent grants

Require a standing or interactive grant only when the operation is explicitly
governed by the authorization model. Ordinary operator-authorized host maintenance
does not become an agent-grant operation because an agent identity is present; that
identity is neither approval nor a blanket refusal. When a permission granted by an
owner-launched Web Console action disappears at a control-plane boundary, trace and
repair authority propagation instead of inventing a new permission request or
expanding a grant.

Before declaring a maintenance operation blocked, trace its actual supported
command path. Standalone setup or dependency rebuild may have a different owner gate
from normal restart, which can stage replacement artifacts before stopping the
serving instance. A lifecycle fence, complete inventory, or drain establishes a
safety condition, not authorization. Keep explicitly human-only approval decisions
on their existing human path.

Evaluate approved operation/target rules through `OperationGrant.Allows`. A fixed
request target can be covered by an approved wildcard in the same rule; exact
request binding does not require a literal target entry in the grant. Preserve
expiry, actor and process checks without inventing replacement approval.
For a governed operation, reuse `authn.RequireOperation` rather than copying its
policy into a route. A manual `principal.Grant != nil` guard rejects the separate
operation proof supported by A75; an agent-only guard also rejects capable human
operators. Human-only approval writes retain `RequireIndependentHuman`.

## Before you rely on a change: run the gates

Run both before trusting any edit to `packages/api-core/**` auth or a relying
scenario's authorization seam. They are self-contained (in-process JWKS), so they
need no running services and can gate a PR or pre-merge hook.

1. **The cross-scenario auth smoke check** — one end-to-end *issue → verify →
   decide* path over the real shared provider and the real standing-grant decision,
   no fakes on the hot path:

   ```bash
   cd packages/api-core && go test ./authn/ -run TestCrossScenarioAuthSmoke -v
   ```

   It fails if a shared change breaks token issuance, cross-scenario verification,
   or the authority decision — the exact blast-radius class a bad shared change
   ships to every scenario at once.
   (`packages/api-core/authn/smoke_test.go::TestCrossScenarioAuthSmoke`.)

2. **The full invariant suite** — all twelve invariants plus the smoke gate and the
   Cf-Access binding self-check, in one command:

   ```bash
   cd packages/api-core && go test ./authn/ ./identity/ \
     -run 'TestInvariant|TestCrossScenarioAuthSmoke|TestManagedCloudflareBinding'
   ```

If a refactor cannot keep these green, the refactor is wrong — or an invariant
genuinely changed, in which case update
[`../concepts/AUTHORIZATION-INVARIANTS.md`](../concepts/AUTHORIZATION-INVARIANTS.md)
**first** and justify it.

## The Cf-Access binding startup self-check

Human operators authenticate a session-delegation mint through the web-console
Cloudflare Access binding, which `scenario-authenticator` resolves from
tunnel-manager. A misconfigured binding or an unreachable tunnel-manager would
otherwise surface only on the first real operator login — a silent fleet
human-login outage. The authenticator probes the binding **once at startup** and
surfaces the result loudly:

- **Broken / unreachable binding:** returns `authn.ErrBindingUnreachable` and logs
  a loud `WARNING`.
- **Good binding:** passes, logs a confirmation, and primes the resolution cache so
  the first real login does not pay for resolution.

It is deliberately **best-effort, not boot-fatal**: a healthy authenticator must
still boot to serve native logins and JWKS when Cloudflare is simply not
configured in a deployment. Contrast with the signing key and hot-state store,
which **are** boot-fatal (local, security-critical, always required). If you touch
Cloudflare-binding code, watch the startup log for the WARNING and confirm
`packages/api-core/authn/managed_cloudflare.go`
(`ManagedCloudflareProvider.CheckBindingReachable`, wired into
`scenarios/scenario-authenticator/api/main.go`) still passes
`TestManagedCloudflareBindingReachabilitySelfCheck`.

## Blast-radius facts

- **`packages/api-core/{authn,identity,scopecatalog}`** — shared; a change here
  recompiles ~19 scenarios. Treat any edit as a fleet change.
- **`scenario-authenticator`** — issues every token and publishes JWKS. Down or
  mis-keyed ⇒ nothing verifies. Kill-switch.
- **`tunnel-manager`** — owns the Cloudflare Access bindings. Down ⇒ no operator
  human-login through the tunnel. Kill-switch.
- **hot-state store** — down ⇒ sessions/grants cannot be read. Kill-switch.

## Testing live grants

- **Standing grant (ask-don't-block):** a covered attempt is admitted by
  `AuthorizeStandingGrant`; an uncovered one returns `authn.ErrNeedsApproval` and
  raises `agent_manager.authorization.decision_requested.v1`. Approve via the
  notification-hub decision → `DecisionResolvedWebhook` → `ResolveAuthorizationFromHub`
  → `CreateStandingGrant`; the caller's **next** attempt is admitted (resume is
  retry-based — see
  [`../concepts/AUTHORIZATION-APPROVAL-LIFECYCLE.md`](../concepts/AUTHORIZATION-APPROVAL-LIFECYCLE.md)).
- **Interactive (in-session) grant:** issue, inspect, and revoke with the operator
  CLI:

  ```bash
  vrooli agent authorize "$VROOLI_AGENT_SESSION" \
    --policy run-creation --target 'agent-manager:*' --expires-in 24h \
    --approval-base https://agent-manager.itsagitime.com
  vrooli agent authorization "$VROOLI_AGENT_SESSION"
  vrooli agent revoke-authorization "$VROOLI_AGENT_SESSION" \
    --approval-base https://agent-manager.itsagitime.com
  ```

  Approval requires a verified human confirming at the approval origin; a prepared
  request or a printed URL grants nothing.
- **Never** test a refusal path by revoking a live node's privileges — use a
  reversible lever (an expired/scoped-out grant), not privilege removal.

## Recovering a bad binding or key

- **Bad owner binding:** owner capabilities come only from an explicit binding —
  `VROOLI_AUTH_OWNER_BINDINGS` (inline JSON), `VROOLI_AUTH_OWNER_BINDINGS_FILE`, or
  `$HOME/.config/vrooli/owner-bindings.json`. A missing optional binding grants no
  scopes; a malformed or unreadable **present** binding **fails configuration** —
  so a bad binding refuses to start rather than fail open. Fix or remove the file
  and restart; inline policy takes precedence over a file.
- **Bad / rotated signing key:** the signing key is boot-fatal for
  `scenario-authenticator`; it must be restored from the existing credential
  authority before the authenticator will boot. Secret custody is separate from
  identity (INV-9) — the key never lives in the identity packages.
- **Unreachable Cf-Access binding:** non-fatal; the startup self-check logs the
  WARNING. Repair the binding in tunnel-manager and restart the authenticator to
  re-prime the cache. Native logins and JWKS keep working meanwhile.

## The invariant checklist

Before merging a shared-auth change, confirm each invariant still holds — each is
pinned by a named test in
[`../concepts/AUTHORIZATION-INVARIANTS.md`](../concepts/AUTHORIZATION-INVARIANTS.md):

- INV-1 anonymous refused · INV-2 forged/bad-signature refused · INV-3 expired
  token **and** expired grant refused · INV-4 over-limit refused (concurrency
  **and** budget) · INV-5 service cannot impersonate a human · INV-6 run identity
  cannot create runs · INV-7 child never broadens parent · INV-8 one decision path
  per operation · INV-9 secret custody separate from identity · INV-10 third-party
  proof not trusted as human without the human claim · INV-11 audience mismatch
  rejected · INV-12 scope-catalog wildcard matrix exact.

## Related

- [`../concepts/AUTHORIZATION.md`](../concepts/AUTHORIZATION.md) — the model.
- [`../concepts/AUTHORIZATION-INVARIANTS.md`](../concepts/AUTHORIZATION-INVARIANTS.md)
  — the twelve invariants, the smoke gate, and the self-check, each with its test.
- [`../concepts/AUTHORIZATION-APPROVAL-LIFECYCLE.md`](../concepts/AUTHORIZATION-APPROVAL-LIFECYCLE.md)
  — the request state machine and resume semantics.
- [`../concepts/AUTHORIZATION-CROSS-BOUNDARY-PROPAGATION.md`](../concepts/AUTHORIZATION-CROSS-BOUNDARY-PROPAGATION.md)
  — what authority crosses a boundary.
