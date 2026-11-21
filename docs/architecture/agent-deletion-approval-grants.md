# Agent deletion approval via operator-decision grants

Status: IMPLEMENTED + LIVE-VERIFIED end-to-end, 2026-10-23. Owner: storage-manager + agent-policy.

Live verification (2026-10-23): agent requested a grant for `~/.local/share/Trash/files` →
ask surfaced in the web-console inbox → operator approved → vrooli-events delivered the signed
resolution (HTTP 200, signature verified) → resolver recorded the grant (2h expiry) → snapshot
projected the allow rule → the agent's hook-governed `rm` was **permitted** (zero policy blocks)
and 13 GB reclaimed. One bug found and fixed during the test: the resolver originally looked up
its record by `resolution.AskID` (notification-hub's UUID) and silently no-op'd; it now
reconstructs the correlation id from `resolution.Correlation["grant_ask_id"]` (set in
`Facts.Correlation` at raise time), mirroring deployment-manager.

Republish-on-approval (DONE 2026-10-23): an approval now triggers an immediate snapshot refresh
via `policybridge.Republisher` (a non-blocking, coalescing trigger the publish loop selects on,
with the 30-min ticker as backstop); `deletiongrant.Resolver.WithOnApprove` fires it only on
approve. So a grant takes effect in seconds, no restart. The webhook mounts only when the shared
vrooli-events secret resolves (now via credential-authority, zero-config, like siblings). Minor
residual: an EXPIRED grant's allow rule drops at the next publish (≤30 min over-extension) —
publish-on-expiry is a possible future refinement.

An unattended agent session is governed by the filesystem-removal floor in
`packages/agentharness` (the `resource-claude-code permissions hook-guard` hook). When a
removal lands on a location the floor withholds — a `safe_with_owner` governed root, a home
path outside every declared-safe location, a repository tree/pattern — the hook refuses and,
with no interactive prompt available, the deletion can only be done by a human. This feature
gives the operator a way to approve such a deletion **asynchronously, through the approval
inbox they already use**, after which the agent may perform it unattended.

## Mechanism (maximal reuse, no new approval primitive)

The agent deletion hook already authorises deletions from **provider PathRule snapshots**
(that is how storage-manager tells the hook which roots are safe/ask/deny). So the approval
does not touch the hook at all: an approved request is projected as a time-boxed `allow`
PathRule in the snapshot storage-manager already publishes. The approval itself is a standard
`operatordecision` ask/answer — the same rail deployment-manager uses for release approval —
not a new grant type or signing scheme. An approved, unexpired ask **is** the grant.

Flow:

1. **Request.** The agent runs `storage-manager deletion-grant request --path <abs> [--path ...]
   --reason "<why>" [--expires-in 24h]` (wrapping `POST /api/v1/deletion-grants/request`). It
   does not need to know this command in advance: storage-manager publishes it as the snapshot's
   `removal_approval`, and the hook's refusal for any withheld path ends with
   `to request the operator's approval, run: storage-manager deletion-grant request --path ...`
   (never for hard-floor locations, which no grant can open). The ask flags every path inside
   owner-protected data (`PROTECTED: inside <root>, which <declaration> protects`); a request
   for a protected root itself, or one made when the declarations cannot be read, is refused
   before it reaches the inbox. The handler
   (`deletiongrant.RequestHandler`) raises an `operatordecision` ask via `operatordecision.Raiser`
   and answers `202 pending`. The ask's correlation id is stable per path set (retries dedupe,
   not flood). It never grants authority.
2. **Approve.** The ask appears in the web-console approvals inbox the operator already uses.
   The operator approves, attributed by `Cf-Access-Jwt-Assertion`.
3. **Deliver + record.** vrooli-events delivers `notification_hub.ask.resolved.v1` to
   storage-manager's signed webhook (`POST /api/v1/events/notification-hub`,
   `operatordecision.Webhook` → `deletiongrant.Resolver`). On `approve` the resolver stamps the
   grant's expiry (requested TTL, capped at 30d) on the stored ask; on any other answer it
   records the refusal. Replay-safe and idempotent.
4. **Project.** The policy publish loop (`policybridge.Start` → `applyGrants`) queries approved,
   unexpired asks (`deletiongrant.Store.ActiveGrantRoots`) and adds one `allow` PathRule per
   path, at a priority above declared roots so it overrides an `ask`. The hook (unchanged) then
   permits the deletion until the grant expires.

## Agent tie-back (automatic delivery on resolution)

Raising an ask and getting a grant is only half the loop: historically the
requesting agent had to poll the grant state or be told "it's approved". The
tie-back closes it for **agent-manager-managed runs** (any coding agent the
shared launcher attaches to agent-manager — including web-console and
device-studio agents, not just agent-manager-created runs). It is a general
agent-manager capability, not deletion-specific — any producer opts in by
stamping one correlation key. **Parking is optional**: delivery adapts to the
run's state, so a *running* agent is reached too.

- **Opt-in (producer):** stamp `operatordecision.CorrelationRaisedByRunID`
  (`"raised_by_run_id"`) in the ask's `Facts.Correlation` with the requesting
  agent's run id. The producer gets a *trusted* run id from the shared
  agent-provenance layer — it runs `provenance.Middleware(provenance.CLIUtilVerifier{})`
  (which verifies the agent's identity token against agent-manager's
  `/api/v1/identity/verify`) and reads `provenance.FromContext(ctx)`'s RunID.
  storage-manager does exactly this on the request route. A non-agent (human)
  caller has no run id, so no key is stamped and there is simply no tie-back.
- **Delivery (agent-manager):** the resolution webhook gains a non-owning
  observer effect (`requesterWakeEffect` in `handlers/decision_events.go`) with
  two lanes, tried in order:
  - **Run-id lane** (`raised_by_run_id`): `GetRun` that run and deliver by its
    current state — **parked** on this decision → `WakeRun`; **running
    interactive / terminal / idle** → `ContinueRun` (mid-turn paste into the live
    session, or a continuation turn); **running non-interactive mid-turn** →
    cannot be interrupted (platform limit), logged, agent should have parked.
    An unresolvable run (ended, or token-expired so no id was captured) falls
    through to the next lane.
  - **Stable session-id lane** (`web_console_session_id`): deliver straight to
    the web-console session via `Orchestrator.SendInteractivePrompt` → the
    already-authenticated web-console client (`interactiveSessions.SendPrompt`) →
    `TerminalService/SendInput`. This needs NO agent-manager run, so it reaches a
    long-lived interactive session whose ephemeral identity token has expired —
    the common case for a multi-day web-console session. web-console is unchanged
    (the receiver already exists).
  The injected text is a short answer-agnostic summary (`formatDecisionWakeMessage`).
  When the run lane handles a resolvable run it is NOT also direct-prompted (the
  run lane already reaches the web-console session for an interactive run), so
  there is no double delivery.
- **Optional park (agent):** to burn zero tokens while waiting, the agent may park
  on producer `operator-decision`, key = the decision id
  (`agent-manager run park <run-id> --producer operator-decision --key <id>
  --timeout 12h`; the request response returns this under an `await` block, and
  `operatordecision.CorrelationAgentAwaitKey` confirms the park matches *this*
  decision before `WakeRun`). `operatorDecisionWaiter` (orchestration) backs the
  park with its deadline only — the authoritative wake is the webhook; the
  deadline is a re-check backstop if the operator has not answered.

Properties: the observer never claims the resolution (`handled=false`) and never
errors, so it cannot block or poison the real appliers (run-approval /
authorization / effort) and a delivery failure never triggers a vrooli-events
retry storm — the durable effect is the producer projecting its own grant.
Redelivery is deduped: `WakeRun` is idempotent, `ContinueRun` carries a stable
idempotency key, and a run already woken for this decision (its `LastAwaitKey`)
is not then also continued. The run id is trusted (verified via provenance,
never taken from the wire unverified), and a parked run is only woken when parked
on exactly this decision. Only a session launched OUTSIDE the shared launcher
(no run id AND no web-console session id — e.g. a developer's raw `claude`/`codex`
in a bare terminal) is unreachable and still relies on a manual nudge; the clean
answer for those is to route them through the launcher plus a universal pull
fallback, not per-harness push.

**Live-verified 2026-10-23 (session lane):** a real deletion-grant was raised
from a live, multi-day web-console Claude session whose env identity token had
expired — so only the stable `web_console_session_id` was captured. On operator
approval the tie-back took the session lane and pasted the answer into that exact
pane as its next turn, with no action from the agent. Finding that drove the
design: the env identity token is frozen at process launch and cannot be
refreshed in place, so it ages out on a long session; the web-console session id
does not.

## Safety invariants

- A grant opens only the **exact path** approved. Inside owner-protected data (`deny`) it may open
  one location strictly below the protected root — the ask told the operator it was protected —
  but **never the protected root itself**: `applyGrants` ignores and logs such a grant, because
  changing what an owner protects is a declaration edit. A grant over a tree that contains a
  protected root still cannot delete it (the runtime lets the stricter nested declaration win).
  (Changed 2026-10-28: grants previously could not reach anything under a `deny`, so an
  operator-approved deletion of an orphan inside `~/.vrooli/data` had no route at all.)
- A grant **never** reaches the agent-policy **hard floor** (the filesystem root, the whole
  home or repository tree, `.git`): the floor drops any non-deny PathRule that would open a
  hard-floor location, so a bad or malicious published rule cannot widen it.
- Grants are **time-boxed** (default 7d, cap 30d) and expire automatically — `ActiveGrantRoots`
  stops returning them, so the next snapshot drops the allow rule.
- The grant ledger is written **only** by the signed resolution webhook (operator-authenticated
  via the hub), not by the agent — the agent can request but cannot self-approve.

## Code map

- `scenarios/storage-manager/api/internal/deletiongrant/` — `schema.sql` (the `deletion_asks`
  table, doubling as the `operatordecision.PendingStore`), `store.go` (store + `ActiveGrantRoots`),
  `request.go` (ask building), `resolver.go` (`DomainEffect` recording approvals), `events.go`
  (webhook secret + vrooli-events subscription reconcile), `http.go` (request endpoint). Unit
  tested in `deletiongrant_test.go`.
- `scenarios/storage-manager/api/internal/policybridge/grant.go` — `applyGrants` projection +
  the deny/absolute-path guards. Unit tested in `grant_test.go`. `provider.go` threads a
  `GrantSource` through `PublishOnce`/`Start`.
- `scenarios/storage-manager/api/internal/modules/registry.go` — schema registration.
- `scenarios/storage-manager/api/main.go` — constructs the shared store, passes it to the
  policy bridge, wires the Raiser + request route + signed webhook + subscription reconcile.

## Preconditions / remaining work

- **Webhook secret.** The resolution webhook mounts only when `VROOLI_EVENTS_WEBHOOK_SECRET` is
  set (the shared vrooli-events signing secret). Without it the request path still works but
  approvals are not delivered, so the webhook is not mounted and a log line says so. (The
  credential-authority mint path other scenarios use was deliberately not pulled in, to avoid a
  new module dependency; wiring it is a small follow-up if zero-config mounting is wanted.)
- **Live verification.** The request→approve→project→delete round-trip exercises vrooli-events,
  notification-hub, and the policy-runner snapshot publish, so it can only be verified with the
  control plane healthy. The pure logic (store, resolver, projection, safety guards) is unit
  tested; the wiring is build-verified. Verify end-to-end once the host is not memory-saturated
  (storage-manager CLI admission was PSI-blocked on 2026-10-22).
- **Agent ergonomics.** Done 2026-10-28: `storage-manager deletion-grant request`, named in the
  hook's refusal via the snapshot's `removal_approval` (`packages/agentharness` `approvalClause`).
  The refusal text needs the rebuilt `resource-<agent>` hook CLIs and `vrooli-policy-runner`.

## Sibling: agent-requested evidence prune (agent-manager)

`agent-manager storage evidence-prune --apply` uses the same rail for a different effect. The
agent-manager service, not the agent, performs the deletion, so an approval is not projected
as a hook allow rule. Implemented 2026-10-28 in
`scenarios/agent-manager/api/internal/evidenceretention/approval.go`:

- From an identified agent run, `--apply` without `--local-owner` is admitted to the handler
  (`agentOperationRoute`). The handler answers `202 approval_pending` and never removes.
  It raises `agent_manager.evidence_prune.decision_requested.v1`. The correlation id is
  stable per (efforts, classes). The facts come from the dry-run preview: bytes, files,
  paths and reason. The ask recommends deny, is irreversible, and has no default.
- On an explicit `approve` answer with an `answered_by`, the `Approvals` DomainEffect
  marks the ask decided first. It then runs the prune once, detached from the webhook.
  The prune is limited to the stored efforts, classes and previewed paths, and candidates are
  recomputed at apply time. The receipt records `actor` = approver and an `approval` block.
  A deny, an expiry, a default answer, an unattributed answer, or a replayed resolution
  removes nothing.
- Asks persist as one JSON file per ask in
  `~/.vrooli/state/vrooli/agent-manager/evidence-retention/approvals/`, next to the receipts.
  `agent-manager storage evidence-prune --status <decision-id>` reads one.
- Humans keep the direct path: `--apply --local-owner`.
