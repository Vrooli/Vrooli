# Program Runtime binding adapter: integration contract

## Current boundary

The BAS browser owner is `agentaccess.Broker`; its public transports are the
REST handler registered at `/api/v1/browser` and a typed Connect service
mounted over the same Broker. Program Runtime discovers callable operations
from BAS Connect descriptors and manifest bindings. It does not invoke
arbitrary REST routes.

The typed Connect schema is at
`packages/proto/schemas/browser-automation-studio/v1/agentaccess/service.proto`,
its BAS-owned Go/Connect artifacts are generated, and the adapter is implemented
at `scenarios/browser-automation-studio/api/handlers/agentaccess/module.go`.
The BAS API mounts it over the exact same Broker instance used by its REST
routes. Its CLI manifest declares typed RPC entries for open, observe, act,
attach, task, trace read/promotion, replay and close. Program Runtime binds the
session operations at `browser.session.*`, the typed action-plan runner at
`browser.run_task`, and persisted flow replay at `browser.replay_flow`; each
callable retains its BAS-owned binding ID. Focused owner tests cover
proof-header handling, persistence mapping, aliasing and same-session replay.

## Authority relay

The relay must happen on the Program Runtime server, never in Python source or
caller-supplied program inputs:

1. The binding call handler reads the verified caller token with
   `api-core/provenance.ForwardedIdentityToken(ctx)`. Empty means refuse.
2. It POSTs an empty body to Agent Manager `/api/v1/browser/resolve-context`
   with only `X-Agent-Identity-Token`. It strictly decodes the response and
   requires a non-empty `caller_id`, `run_id`, and opaque `grant_receipt`.
3. It adds the original trusted identity token and returned receipt to the
   outbound BAS Connect request headers:
   `X-Agent-Identity-Token` and `X-Vrooli-Browser-Grant-Receipt`.
4. BAS Connect middleware constructs an `agentaccess.WithBrowserAgentCredentials`
   context from those headers. The existing broker and its Agent Manager
   authorizers re-validate the active run, permission profile, and grant on
   every call. Program Runtime must not cache the receipt beyond one invocation.

No program argument, declared input, result envelope, learning record, or
Search Hub document may contain caller/run IDs, identity tokens, grant receipts,
CDP endpoints, or raw lease IDs. The grant resolver is a server-to-server hop;
it must not be registered as a public program binding.

## Proposed typed service

`browser_automation_studio.v1.agentaccess.BrowserSessionService` is defined in
that schema. Requests use explicit typed fields and cannot supply identity or
grant proof. The service methods map to the broker; they never create or attach
a browser outside it.

The Program Runtime kernel maps the declared BAS session bindings to a
`browser` namespace alias at its host boundary. Only the explicit session/task
binding IDs receive that alias; all other BAS methods retain
`browser_automation_studio.*`. The `browser.run_task` binding is published at
the documented root while retaining its stable `browser-automation-studio/task/run-task`
owner ID. `browser.replay_flow` retains its stable
`browser-automation-studio/task/replay-flow` owner ID.

| Program operation | Request | Broker operation | Effect |
|---|---|---|---|
| `browser.session.open` | backend, start URL, reuse mode, viewport, optional consent/tab refs; permission profile is resolved from trusted Agent Manager run context | `Broker.Open` | write |
| `browser.observe` | session ID, optional page ID | `Broker.Observe` | read |
| `browser.act` | session ID, typed compiled instruction, optional target URL/approval receipt | `Broker.Act` | write |
| `browser.session.attach` | existing session ID, permission profile ref, optional consent/tab/approval refs | `Broker.Attach` | write |
| `browser.run_task` | one existing session ID and 1–25 typed actions | repeated `Broker.Act` calls on that exact session | write |
| `browser.promote_trace` | session ID, existing project ID, name, description, explicit typed assertions | `Broker.PromoteTrace` then BAS `WorkflowService.CreateWorkflow` | write |
| `browser.replay_flow` | session ID, persisted workflow ID/version, required string parameters | BAS `WorkflowService.GetWorkflowAPI` then repeated `Broker.Act` on the exact session | write |
| `browser.session.close` | session ID | `Broker.Close` | write |

The registered `browser.run_task` is a bounded typed action-plan runner, not a
natural-language planner. BAS validates the existing broker session and routes
each instruction back through `Broker.Act`; the broker re-checks caller, lease,
permission profile, URL scope and action policy on every instruction, and each
accepted action is recorded in the same trace. The first denied or failed
instruction stops the plan and returns partial results. This deliberately does
not call `do-task` or `navigate-intent`: workflow execution opens or acquires a
session through its own executor and its low-level navigation path does not
re-enter the Agent Access broker's per-action policy. Those paths cannot
currently be composed over the same qualified lease without changing their
owner boundary.

Trace promotion requires an existing BAS project ID, explicit assertions, and
an injected `ProjectScopeAuthorizer` that verifies an Agent Manager issued
project-write scope receipt from trusted invocation context. The current BAS
WorkflowService checks project existence and persists the generated V2
definition but does not verify caller/project membership. BAS main does not yet
have an owner implementation or receipt to inject, so promotion currently
fails closed before `CreateWorkflow`; an ID supplied by a client is not
authorization. Once an owner-issued scope receipt is available, persistence
continues through the existing WorkflowService; no workflow files or indexes
are written directly. Replay first loads the workflow through the BAS owner,
derives its project ID from that saved record, and requires the same verifier to
authorize project read before any Broker action. With no verifier or a denied
read, replay fails before acting. Authorized replay requires a generated trace
provenance marker and a single linear chain, then routes every typed action,
including assertions, through the exact broker session. Missing parameters,
branching/loops/subflows, or failed actions stop the replay. The replay adapter
is intentionally limited to linear flows; the general workflow executor still
owns its separate session lifecycle.

## Wiring implemented

1. The protobuf schema includes the browser Connect RPCs, and its owner
   generator/verifier passes.
2. BAS main constructs one access Broker and mounts both the REST handler and
   Connect `Module` over that exact instance, with the existing WorkflowService
   as the persistence/catalog owner. The module validates exactly one
   identity/grant header pair and calls only those supplied owners.
3. BAS's manifest exposes descriptor-backed methods with explicit read/write
   governance. `AttachSession` delegates to `Broker.Attach`, preserving the
   existing session owner.
4. Program Runtime uses an invocation-local request preparer. For the exact
   allowlisted BAS browser IDs, it requires verified provenance, resolves a
   fresh grant with an empty-body Agent Manager `resolve-context` request, then
   attaches the proof pair to one downstream Connect call. Missing identity,
   incomplete context, redirect, resolver error, or other BAS method fails
   closed. Proof never enters binding arguments, program source, run args,
   result payloads or telemetry.
5. The one-shot authority relay admits only those exact browser IDs in addition
   to its pre-existing Web Console reads. It treats browser writes as uncertain
   if their reply is lost.
6. The host maps only these declared BAS binding IDs to the `browser` alias.
   Skill publication and a cold-agent Search Hub ranking check remain separate
   discoverability work.

## Required owner evidence

- `go test ./handlers/agentaccess` passes adapter tests for exact field-to-broker
  mapping, refusal with missing or duplicate proof headers, and trusted
  credentials reaching BAS's existing Agent Manager context resolver. Request
  schemas contain no caller-controlled identity/grant fields.
- Program Runtime tests use verified provenance and assert the resolver call
  has no identity body, the grant is fresh per invocation, only exact browser
  IDs receive proof headers, incomplete/unverified context fails before BAS,
  and unrelated BAS calls receive no browser proof.
- End-to-end evidence runs through Program Runtime and the public BAS session
  owner: open an observe-only session, prove `act` is denied, reopen with
  permitted control, perform and verify the task, promote with explicit
  assertions, and rerun the saved revision on the same broker-owned session.

Current limits: project persistence is disabled on the production path until
the Agent Manager owner supplies and BAS injects a verifiable project-scope
receipt contract. `browser.run_task` remains a typed action-plan runner, not the
natural-language `do-task` planner. `browser.replay_flow` is deterministic only
for linear generated trace flows; it is not the general workflow executor and
has no multi-step execution cursor. No other BAS binding inherits browser-grant
headers. Adapter registration does not qualify browser-use, Playwright
MCP/CLI, Stagehand, raw CDP, live extensions, Search Hub cold-agent discovery,
or any real permission-qualified session. Browser Use 1.0 custom actions can
call the existing BAS observe/act surface, but its `Agent` still owns a Browser
object and exposes no broker-native borrowed-session driver; no safe upstream
adapter currently exists. Playwright MCP/CLI likewise operate a Playwright
browser/CDP transport directly, which bypasses per-action Broker authorization.
They remain `unsupported` until an adapter makes every observation/action a
Broker call without exposing a driver endpoint. Browser Use 1.0.3 has no
documented broker-session injection point: `Agent` accepts a `Browser` (or
`browser_session` that must be a `Browser`), starts it in `Agent.run`, and
stops it during shutdown. Supplying `Tools` adds functions to the agent; it
does not replace the Browser lifecycle or its Python REPL. That REPL exposes
browser helpers and raw `cdp()`, so custom broker tools alongside the built-in
Python tool cannot enforce that all actions pass through BAS. A `Browser`
subclass/no-op `start` would depend on undocumented internals and still would
not remove the model's Python/CDP escape path. The same-session REST/Connect
task path alone does not qualify either upstream harness.
