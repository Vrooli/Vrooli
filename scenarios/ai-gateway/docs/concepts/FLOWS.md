# Flows - AI Gateway

This document names the stateful workflows that matter for AI Gateway.
Implementation should model flows when retries, fallbacks, stale
completion, provider failure, or scan lifecycle can create invalid
states.

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Route preview | routing | Caller submits operation, role, profile, and constraints with `preview=true`. | Candidate providers and selected route reasons are returned without execution. | Stateless computation over live/cached policy, but must preserve rejection reasoning. | Unit matrix tests in `api/internal/routing/service_test.go`. |
| Inference execution | routing/providers | Caller submits an executable request. | Provider resource command runs, result and route evidence are returned. | Stateful request with timeout, fallback, cancellation, and evidence persistence. | Phase 4 service tests cover fallback policy, redacted evidence, fail-closed persistence, and stdin command execution. A formal flow artifact remains a future hardening step if cancellation/fallback states grow. |
| Media receipt (SM12) | routing | Caller submits image/video work with an idempotency key. | One retained request/receipt binds the key; startup resumes only undispatched work and marks interrupted dispatch as outcome unknown. | Durable queued/running/terminal lifecycle, request replay binding, cancellation and stale completion. | `api/internal/routing/flow/flow.json`, generated transition matrix/traces, production adapter replay, service persistence/crash-window tests, handler and CLI tests. |
| Provider circuit breaker | routing | Provider execution succeeds or fails for a `(provider, role, kind)` route. | Persisted provider-health state transitions across closed → open → half_open, and route preview/execution respect it. | Stateful across requests; keyed per provider/role/kind with a deterministic clock and cooldown. | `api/internal/routing/health_test.go` (pure transitions, classification, repository round-trip) and `breaker_service_test.go` (preview skip/probe, fallback, isolation). |
| Provider inventory refresh | inventory | Operator/API/CLI requests inventory or scheduled refresh runs. | Resource roles, models, constraints, and smoke status are normalized. | Stateful refresh with partial provider failure and stale cache handling. | Integration tests with fake resource runners. |
| Role smoke test | inventory/providers | Operator runs smoke test for one role/provider/profile. | Health status and evidence are recorded. | Stateful execution with provider failures and redaction. | Unit and integration tests. |
| Conformance scan | conformance | test-genie or operator scans a scenario. | Findings, maturity score, exceptions, and fix guidance are reported. | Stateful scan over files/config/docs with severity and maturity thresholds. | Fixture-based scanner tests and provider phase tests. |
| Migration report | conformance/routing | Operator requests adoption guidance for a scenario or fleet. | Current AI usage, recommended gateway profiles, and investigation items are reported. | Mostly deterministic report generation over scan findings. | Integration fixture tests. |

## Inference Execution State Model

Implemented Phase 4 states:

| State | Meaning |
|---|---|
| `accepted` | Request contract parsed and policy constraints validated. |
| `planned` | Candidate providers resolved and route selected. |
| `executing` | Resource command is running. |
| `fallback-evaluating` | Primary route failed and policy permits fallback evaluation. |
| `persisting-evidence` | Result/failure evidence is being persisted. |
| `succeeded` | Result returned and evidence persisted. |
| `failed` | No route succeeded or policy blocked fallback. |
| `cancelled` | Caller or timeout cancelled execution and evidence captured cancellation. |

Illegal transitions include executing before planning, succeeding before
evidence persistence, falling back when profile policy forbids it, and
returning a provider result after cancellation without marking it stale.
The current implementation uses context timeouts, resource command
timeouts, and metadata-only route evidence. It does not persist raw
prompts or responses.

## Media Receipt State Model (SM12)

The domain contract is `api/internal/routing/flow/flow.json`; Flow Verifier owns
its generated runtime, matrix, traces and model under `flow/generated/`.
`flow.TransitionReceipt` adapts the public receipt representation to that
contract and supplies the target status for production SQL mutations. SQL
status predicates provide the dispatch and completion compare-and-set guards.
The replay test covers the same representation adapter used by the service.

A receipt starts `QUEUED` before dispatch. The worker claims it as `RUNNING`
before invoking the executor. It can become `SUCCEEDED`, `FAILED`, or
`CANCELLED`; terminal status never changes in place. Explicit Retry creates a
new receipt for eligible terminal results rather than changing the old one.

Startup recovery runs once before route mounting. It first changes interrupted
`RUNNING` receipts to terminal `FAILED` with `error_code=outcome_unknown`, without
reading their retained request. The provider may already have completed and
charged, so these receipts are never automatically dispatched again. Get,
Wait, Cancel, identical Submit and repeated recovery retain that result;
Retry refuses it. Stale success/failure writes cannot replace the unknown
result because they require a still-`RUNNING` row. A quarantine-write failure
returns an error and startup logs incomplete recovery without dispatching that
work. Only `QUEUED` receipts are decoded and resumed through the normal claim;
invalid retained requests fail before dispatch.

Submit compares the complete semantic request on both existing-key lookup
paths, including ordered input references, output reference, intent, budgets,
privacy and request correlation. Only idempotency-key whitespace is normalized.
Changed content conflicts and corrupt retained content fails closed. This binds
references, not mutable file bytes. It does not provide global exactly-once
execution, provider reconciliation, multi-process ownership, or durable caller
artifact recovery. A new key deliberately starts a new operation.

`media_test.go` drives the actual Submit/run path through a controlled result
write failure after the fake executor completes, then runs startup recovery
against the retained database and proves no second dispatch. Matrix/trace and
database proofs serve different boundaries. Ordinary `FAILED` is not evidence
of provider failure or zero cost. The Resource OpenRouter executor requests
its versioned `gateway-v1` image envelope. A provider-reported finite
nonnegative cost is persisted with explicit presence, including zero; migrated
receipts remain unknown. If a usable cost accompanies local payload validation
or output failure, Gateway stores it on the `FAILED` receipt under the same
`RUNNING` status fence. This does not make incomplete output successful or
prove an unknown provider outcome. Failed/cancelled Image Tools job linkage
remains deferred to D33.

## Provider rule admission projection (SM6)

The current owner path reads a resource-owned catalog on each admission and
verifies its exact digest/reference with Notification Hub. It has no durable
proposal publisher, compare-and-promote operation or policy history store.
The six review statuses describe the owner record; they are not evidence that
those missing mutation operations exist.

`path:scenarios/ai-gateway/api/internal/routing/policy_review_workflow.spec.json`
declares the admission projection of every catalog status and the verified,
unverified, review-due and expired observations for a claimed approval.
`policy_review_workflow.go` computes this projection in `ResourcePolicySource`.
An approved catalog declaration requires fresh exact human authority; a changed
clause, agent answer or unverifiable decision projects to reopened, and an
elapsed policy window projects to expired. Nonapproved owner declarations never
become approved through a Hub response. Catalog observations may replace any
prior projected state because each read may observe a new owner revision;
there are no terminal states across revisions. This does not authorize rewriting
catalogs or reopening a particular immutable Hub ask.

Streaming must authorize the same final snapshot that supplies downstream
privacy and pinned-host constraints. The first admission check does not grant
a later catalog read authority: apply canonical eligibility and the complete
pinned route/model/supplier/quote comparison again before using that read for
dispatch. A reopened, expired, changed-route or revoked-agreement snapshot must
refuse before a durable dispatch claim or upstream request.

The consumed matrix and traces cover this read-derived contract. Isolated
catalog/HTTP receipts replay the existing J3 agent and changed-clause behavior;
the J3 journey retains the full picker/execution and zero-charge assertions.
This is Level 4 admission projection evidence, not proof of the unimplemented
proposal publication/promotion lifecycle or a checked formal model.

## Provider Breaker State Model

Provider health is tracked per `(provider, role, request kind)` so one
provider/role's failures never suppress a healthy fallback. Transitions are
deterministic and driven by an injectable clock:

| State | Meaning | Transition |
|---|---|---|
| `closed` | Provider is healthy; requests route normally. | Opens when consecutive failures reach the policy threshold. |
| `open` | Provider is suppressed; preview rejects the candidate and execution skips it. | After the cooldown elapses the effective state becomes `half_open` (computed from the clock, no background writer). |
| `half_open` | Cooldown elapsed; one bounded recovery probe is eligible. | A successful probe closes the breaker; a failed probe reopens it and extends the cooldown. |

Failures are classified into stable provider-neutral classes
(`missing_binary`, `timeout`, `malformed_json`, `policy_error`,
`execution_error`, `cancellation`, `unavailable`) recorded on the health row.
The breaker never inspects prompt/response content. Recording is best-effort:
a health-store write failure does not fail the caller's request, which already
succeeded or failed on its own merits.

## Conformance Scan State Model

Planned states:

| State | Meaning |
|---|---|
| `queued` | Scan request accepted by test-genie or operator surface. |
| `discovering` | Scenario files/config/resources are being inventoried. |
| `classifying` | Findings are categorized by rule and maturity level. |
| `evaluating-exceptions` | Approved exceptions are applied. |
| `reporting` | Findings and recommendations are written. |
| `passed` | No findings exceed the selected maturity threshold. |
| `failed` | Findings exceed the selected threshold. |
| `error` | Scanner could not complete for infrastructure reasons. |

The scan should distinguish product failure from scanner failure. A
malformed target scenario can be a finding; an unreadable repository or
broken scan runner is a provider error.

## Validation Approach

Route preview can begin with pure unit matrix tests. Inference execution
and conformance scans should become modeled workflows before full
implementation because cancellation, fallback, stale provider results,
and threshold handling are easy to get subtly wrong.

When a flow reaches implementation:

1. Add a domain-local `flow/flow.json`.
2. Generate and check the formal artifacts with `flow-verifier`.
3. Add replay tests against production transition functions.
4. Add integration tests around side-effect seams.

## Template Example

The generated `notes` attachment-upload flow is template scaffolding,
not AI Gateway product scope. Keep it only as a reference until
`vrooli scenario detemplate ai-gateway` removes the example domain.
