# Cross-boundary authority propagation contract

When a request crosses a scenario boundary — a human session reaching Agent
Manager, a service calling another service, a parent run spawning a child — some
authority travels with it and some must not. This document is the contract for
what crosses, what attenuates, and what must never happen. It is the
boundary-crossing companion to [`AUTHORIZATION.md`](AUTHORIZATION.md); the
test-pinned rules are in [`AUTHORIZATION-INVARIANTS.md`](AUTHORIZATION-INVARIANTS.md).

## What authority can cross a boundary

Exactly four proofs carry authority across a scenario boundary. Each is verified
at the receiving boundary against a trusted source; none is trusted because the
caller merely asserts it.

| Proof | What it carries | Verified against |
|---|---|---|
| **Identity token** | Who the principal is (human / agent / service) and its scope list | `packages/api-core/authn/jwt.go` (`JWTVerifier.Verify`) — local JWKS validation, issuer/alg/kid/audience/expiry (INV-2, INV-11) |
| **Human-session delegation** | A verified-human Web Console session authorizing the operations it was launched to do | `agent-manager` verifies the `X-Human-Owner-Delegation` + `X-Wc-Session-Id` headers against `scenario-authenticator` and binds the delegation to the web-console session, minting a session standing grant |
| **Service bearer** | A scenario acting as itself (operator-issued service-principal token) | Shared verifier; a service never satisfies `RequireHuman` / `RequireIndependentHuman` (INV-5) |
| **Standing grant** | Durable, operator-approved authority for a named caller to do a bounded costly write | `packages/api-core/authn/standing_grant.go` (`AuthorizeStandingGrant`), keyed to the caller's verified subject |

A read or cheap write needs only a verified identity token. A costly write needs
one of the latter three on top of identity.

## Human-session delegation: the one way human approval crosses to an agent

A verified-human Web Console session (behind the Cloudflare secure tunnel) mints a
short-lived, session-scoped delegation. It travels as `WC_HUMAN_OWNER_DELEGATION`
in the spawned agent's environment, is presented on attach as the
`X-Human-Owner-Delegation` and `X-Wc-Session-Id` headers, and Agent Manager
verifies it against `scenario-authenticator` and binds it to that web-console
session — minting a **session standing grant** the agent inherits.

Web Console and Agent Manager stay **decoupled**: web-console never calls
agent-manager to launch or attribute a run. The delegation is the entire
contract between them; attribution is env-based on the agent side.

## What attenuates: child ⊆ parent, never broadens

A delegated child run receives the intersection of its parent's authority, never
more:

```text
child authority = owner scopes ∩ agent profile ceiling ∩ requested scopes
```

The child's credential expires no later than its parent and must share the
parent's admitted owner. A child requesting a scope the parent never held is
refused outright — there is no elevation across the delegation boundary.

- **Enforced at** `scenarios/agent-manager/api/internal/orchestration/delegation.go`
  (`MintDelegatedIdentity`, using `identity.Attenuate` / `IntersectScopes`).
- **Invariant** INV-7 in [`AUTHORIZATION-INVARIANTS.md`](AUTHORIZATION-INVARIANTS.md).

Composition through Program Runtime obeys the same rule: the kernel forwards only
the verified agent proof, every target re-checks its own operation/object/expiry/
revocation/independent-human rules, and composition creates no broader grant or
fresh permission lifetime.

## What must never happen

These are the non-negotiable negatives. Each has a test (see the invariant doc).

- **No authority laundering via loopback or env markers.** A harness ID, an
  environment marker (including a bare `WC_SESSION_STATE_ROOT` adoption), a PID,
  localhost/TTY/shared-UID, or Web Console `origin=ui` is **not** human proof. An
  agent cannot grant itself authority by setting a marker or exchanging its shared
  OS-user credential for a human session.
- **Bare attach stays zero-scope.** Attaching to a session establishes agent
  identity only; it confers none of the human's capabilities. A bare attach
  without a delegation is intentionally zero-scope. (A separate, independent-human
  approval is required to add authority — see the interactive path in
  [`AUTHORIZATION-APPROVAL-LIFECYCLE.md`](AUTHORIZATION-APPROVAL-LIFECYCLE.md).)
- **A bare run identity does not authorize arbitrary run creation.** Top-level
  creation requires an exact current `run.create` grant on `agent-manager:*`; the
  exact-parent delegated-child route is separately bound to the verified parent.
  Stop/continue/wake/recover require their object-bound `agent.manage` grant or
  allowed lineage route. Delete, approval, and resume-as-new remain refused to a
  run identity (INV-6, `scenarios/agent-manager/api/internal/handlers/run_identity_guard.go`).
- **A service cannot become a human** (INV-5). Missing or invalid proof cannot
  silently preserve an earlier principal at a nested boundary; retained
  human-decision gates reject agent attribution.
- **No credential forwarding into an agent environment.** Browser/human
  credentials stay on the owner side; only the proof appropriate to the intended
  trusted destination is forwarded.

## The loopback-only continuity contract

Transcript continuity between Web Console and Agent Manager —
`POST /api/v1/runs/import-transcript` and `POST /api/v1/runs/external/tombstone` —
is **LoopbackOnly** (same-host, zero-authority), *not* credential-forwarded. These
routes are wrapped with `lifecycle.LoopbackOnly(...)` in
`scenarios/agent-manager/api/internal/wiring/routes.go`, so they are reachable
only from the same host and carry no principal authority. Continuity moves
transcript bytes for the same host's own sessions; it is explicitly **not** a
channel by which one scenario borrows another's authority. Treating a loopback
reachability as authority would be exactly the laundering the model forbids.

## Related

- [`AUTHORIZATION.md`](AUTHORIZATION.md) — the authorization model.
- [`AUTHORIZATION-INVARIANTS.md`](AUTHORIZATION-INVARIANTS.md) — INV-5, INV-6,
  INV-7 pin the attenuation and impersonation rules.
- [`AUTHORIZATION-APPROVAL-LIFECYCLE.md`](AUTHORIZATION-APPROVAL-LIFECYCLE.md) —
  how a request becomes authority.
- [`IDENTITY-AND-AUTHENTICATION.md`](IDENTITY-AND-AUTHENTICATION.md) — how each
  identity kind is proven.
- [`GATED-UI-AUTHENTICATION.md`](GATED-UI-AUTHENTICATION.md) — the Web Console
  human-ingress contract behind the tunnel.
