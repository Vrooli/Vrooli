# Authorization approval-request lifecycle

This document describes the state machine an authorization request moves through,
its two origins, the operator-notification path, and the retry-based resume
semantics. It is the operational companion to the model in
[`AUTHORIZATION.md`](AUTHORIZATION.md) and the test-pinned rules in
[`AUTHORIZATION-INVARIANTS.md`](AUTHORIZATION-INVARIANTS.md).

The subject is always one `identity.OperationGrant` shape: a bounded operation +
target + limits + expiry, tied to an accountable owner. A request is the *pending*
form of that grant; approval turns it into an authority a later attempt can use.

## Two origins

An authorization request is raised one of two ways. Both persist a pending
record and publish a distinct decision event, but they resolve through different
paths and that difference is deliberate.

| Origin | Raised by | Event type | Resolved by |
|---|---|---|---|
| **Standing-grant ask ("ask, don't block")** | A costly write with no covering authority returns `authn.ErrNeedsApproval` instead of refusing silently. The orchestrator persists the request and calls `RaiseAuthorizationApproval`. | `agent_manager.authorization.decision_requested.v1` (`AuthorizationDecisionEventType`) | notification-hub → operator approves → `DecisionResolvedWebhook` → `ResolveAuthorizationFromHub` → `CreateStandingGrant` |
| **Interactive (in-session) delegation** | A verified human approves a bounded delegation to an already-running session via `vrooli agent authorize`. `RequestAuthorization` persists the pending grant. | `agent_manager.interactive_authorization.decision_requested.v1` (`InteractiveAuthorizationDecisionEventType`) — **push-only** | the independent-human `ApproveAuthorization` path **only** — never a hub resolution |

The interactive event is push-only: it notifies, but it is **not** approvable by a
hub webhook. An interactive grant is confirmed exclusively through
`ApproveAuthorization` (the browser consent at the configured approval origin),
so an operator-convenience hub resolution can never stand in for the
independent-human confirmation an in-session delegation requires. The two origins
never cross: approving a standing-grant ask mints a standing grant; approving an
interactive request binds a session-scoped delegation.

- **Standing-grant ask** — `scenarios/agent-manager/api/internal/orchestration/authorization_decision_events.go`
  (`RaiseAuthorizationApproval`, `ResolveAuthorizationFromHub`) and
  `scenarios/agent-manager/api/internal/handlers/decision_events.go`
  (`DecisionResolvedWebhook`, wired at `DecisionEventWebhookPath` in
  `scenarios/agent-manager/api/internal/wiring/routes.go`).
- **Interactive** — `scenarios/agent-manager/api/internal/orchestration/authorization.go`
  (`RequestAuthorization`, `ApproveAuthorization`).
- **`ErrNeedsApproval`** — `packages/api-core/authn/standing_grant.go`.

## The state machine

An interactive grant record carries a `State` field. The lifecycle is:

```text
            RequestAuthorization
                   |
                   v
               [pending] --------- review deadline passes ------> [expired]
                 |   |
     Approve-    |   | a new, different request from the same session
     Authorization   |
                 |   v
                 | [superseded]   (the old request; approving it names its replacement)
                 v
             [approved] ---- revoke / changed session process / ended run ----> (lapsed)
                   |
                   v
   authority usable until the approved deadline (default 24h, ≤ 7 days),
   bounded by the retained run ceiling
```

The `State` string transitions are set in
`scenarios/agent-manager/api/internal/orchestration/authorization.go`:

- **pending** — `RequestAuthorization` persists with `Replace(ctx, g, "pending")`.
  A session has at most one pending request.
- **approved** — `ApproveAuthorization` sets `g.State = "approved"` and verifies
  the independent human at the configured approval origin before committing.
- **superseded** — a new, different request from the same session supersedes the
  prior pending request; approving names its replacement. A session uses only its
  newest approved grant.
- **expired** — a pending request past its review deadline reads `expired`
  everywhere. New approval requests remain reviewable for seven days.
- **lapsed** — an approved grant ends early on revocation (`RevokeAuthorization`),
  a changed session process incarnation, or an ended run. When it lapses the
  session returns to the identity it held before the grant.

Neither `pending`, `superseded`, nor `expired` carries any authority. Preparing a
request or printing a review URL grants nothing; only a verified owner confirming
in the browser moves `pending → approved`.

For a standing-grant ask, approval does not mutate a `State` field in place: the
hub resolution calls `CreateStandingGrant`, which writes a durable standing grant
keyed to the caller's verified subject. Denial or expiry leaves no grant, so the
caller's next attempt simply returns `ErrNeedsApproval` again.

## The notify path (standing-grant ask)

```text
costly write, no covering authority
   -> authn.ErrNeedsApproval
   -> RaiseAuthorizationApproval persists a pending request
   -> publish agent_manager.authorization.decision_requested.v1
   -> notification-hub (subscribes **.decision_requested.v1)
   -> operator notified (and given a chat link)
   -> operator approves
   -> DecisionResolvedWebhook (POST, HMAC-verified)
   -> ResolveAuthorizationFromHub
   -> CreateStandingGrant (durable, keyed to the caller's verified subject)
   -> caller's NEXT attempt is admitted by the standing-grant evaluator
```

The `.decision_requested.v1` suffix is the generic contract notification-hub
already subscribes to (`**.decision_requested.v1`), so an authorization request
rides the same operator-notification fabric runs and efforts use — there is no
separate authorization inbox.

## Retry-based resume semantics

There is no callback that "resumes" the blocked caller. Resume is **retry-based**:
the caller fails with `ErrNeedsApproval`, the operator approves out of band, and
the caller's *next* attempt at the same operation finds a covering grant and
proceeds. This is why the authority check is a pure lookup-and-evaluate against
the durable grant store (`AuthorizeStandingGrant` in
`packages/api-core/authn/standing_grant.go`): the same code path refuses before
approval and admits after it, with no in-flight state to reconcile. Expiry,
revocation, over-budget, and over-concurrency are all re-evaluated on every
attempt, so a grant that lapses between attempts stops admitting immediately.

## Related

- [`AUTHORIZATION.md`](AUTHORIZATION.md) — the authorization model.
- [`AUTHORIZATION-INVARIANTS.md`](AUTHORIZATION-INVARIANTS.md) — test-pinned rules
  (INV-3 expiry, INV-4 over-limit, INV-8 one decision path).
- [`AUTHORIZATION-CROSS-BOUNDARY-PROPAGATION.md`](AUTHORIZATION-CROSS-BOUNDARY-PROPAGATION.md)
  — what authority crosses a scenario boundary and how it attenuates.
- [`../guides/working-on-auth-infra.md`](../guides/working-on-auth-infra.md) —
  how to change shared auth safely.
