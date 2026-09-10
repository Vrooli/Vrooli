# Bounded recovery and route changes

Record the needed operation, expected result, observed result, evidence and consequences before selecting repair. A quick probe is not an implementation attempt. It still needs a finite transport/probe allowance so a failed readiness check cannot loop indefinitely.

## Intervention accounting

Effort policy declares three cumulative repair limits: per failure fingerprint, per component, and across the whole effort. Component means the stable owning scenario/resource/shared package, not a plan or worker. A fingerprint identifies the failed operation and invariant; renaming the plan or error text does not create a new budget. Record total effort/cost where observed; unknown measurements remain unknown.

Append a `repair_started` event before effects. Include `attempt_id`, `component`, `fingerprint`, `hypothesis`, `new_evidence`, `evidence`, `at`, and owner work/run references when available. Append `repair_finished` with the same identity and its `outcome` and evidence. Interrupted started attempts remain charged and require reconciliation. A repeated report of the same attempt does not count twice. Conflicting identity reuse is invalid.

Count successful and failed interventions toward the component and effort investment. A new hypothesis does not reset limits. Use `probe`, `transport_retry` and `status_read` events with stable attempt, component and fingerprint identities for their separate allowances: `max_probe_attempts`, `max_transport_attempts` and `max_status_reads` (initial defaults: two each). A status fingerprint includes the owner operation and evidence cut, so normal reads after a real transition differ from polling unchanged state. `note` annotates evidence; it cannot substitute for recording an operation. Time passing, another worker, another round, or a successful unrelated test does not reopen a circuit.

If an owner has durable attempt/incident state, reference it and derive the report from that owner. Until an owner supports cross-effort accounting, the single coordinator retains these effort records and must check them before admission. Do not claim runtime enforcement from a local report.

## Decision table

| Condition | Decision |
|---|---|
| Readiness check fails, no effects started | Classify once; use the allowed probe retry only for an evidenced transient problem. |
| Defect is local, consequential and within budget | Fold one falsifiable repair into the relevant owner plan. |
| Several failures share an architectural cause | Allocate a bounded infrastructure child plan; keep unaffected branches moving. |
| Prior repair failed and evidence is unchanged | Stop repetition, even if numerical allowance remains. |
| Any cumulative limit is reached | Open the circuit; do not admit another repair for that component/fingerprint or effort budget. |
| Alternative preserves acceptance and authority | Qualify it, record the lost guarantees and temporary debt, then use it. |
| Alternative cannot satisfy a required outcome | Keep that outcome unmet; continue independent work and surface the actual decision. |
| New external evidence may justify reopening | Record the evidence and obtain the required budget/policy amendment. Never silently reset counters. |
| A forbidden action or grant refusal occurs | Respect it. Another launcher is not a remedy for missing authority. |

A workaround record names affected requirements, route, owner issue, retained evidence, limitations, validation, and a revisit/removal trigger. Operational alternate routes are often valid; a private duplicate of another scenario's repair logic usually creates more debt. Prefer repairing the proper owner until the agreed limit is exhausted.

## Dispatch fallback

Preferred routes are qualified Swarm execution, declared Agent Manager workflow/family, and the control-plane coding-agent launcher. Direct harness execution is a last route only when the effort explicitly authorizes its reduced guarantees and the invocation is known. Scenario servers must still use the scenario lifecycle.

Before changing route, retain original work identity and prove no conflicting live executor, or obtain owner-confirmed termination. Reconcile a timed-out start through the original owner; a network error is not proof of rejection. When reconciliation is unavailable, mark `dispatch-uncertain` and work elsewhere. Reuse supported idempotency keys. The control-plane launcher currently lacks one, so require stronger reconciliation before using it.

Fallback preserves scope, acceptance, budget, path claims, selected agent/member identity, result contract and cancellation obligations. Record lost telemetry, sandboxing, native goal support or continuation guarantees. Do not strip credentials or attribution to convert a denied call into an allowed one. Direct session authority and typed Swarm grants are different evidence; label the one actually held.

There is no universal `--goal` flag. Inspect the exact runner/owner contract. Use Plan Manager completion and durable workflow continuation where native goal mode is unsupported. Do not count a child process exit as accepted plan completion.

Existing owner breakers may have narrower meaning: per-backlog cooldown, same-key park suppression and repeated checkpoint counters do not replace cumulative component investment. Preserve those controls and add the missing cross-plan accounting at the appropriate owner when implementation is authorized.

## Runtime supervision and runner limits

Default to restarting or resuming only the affected child after reconciling its old attempt. Escalate a subtree only when its dependency invariant requires it. A successful finite worker stays finished; do not configure it as a permanently restarted service. Graceful shutdown checkpoints evidence and pending operations before termination. Owner-confirmed cancellation fences later child results before releasing their claims.

Use owner-enforced restart intensity over a time window as well as cumulative effort repair limits. Preserve totals and unknown usage through parent restarts; nested supervisors must not multiply retry allowances. Classify the cause before deciding whether another attempt could succeed.

Apply recovery decisions in this order. First retain claims for any possibly active old dispatch; no replacement may start while its ownership is uncertain. Next apply revoked/denied authority or an exhausted aggregate allowance: fence new effects and retain the required owner remedy. Honor an already accepted finite result without restarting it. For unfinished work, classify observed pool exhaustion before considering transient retry or process failure. Only after those gates pass may the owner admit one compatible resume/restart under the remaining allowance. Use one-child recovery by default; a dependency-group restart requires an explicit owner dependency invariant. If evidence supports conflicting categories, retain the stricter admission restriction and request a bounded owner observation instead of choosing the cheaper-looking route.

| Observation | Required recovery |
|---|---|
| Context window or output capacity reached | Preserve a bounded checkpoint and pending handles; use supported compaction/session continuation or a fresh compatible session. This does not establish subscription quota exhaustion. |
| Session or weekly subscription allowance exhausted | Pause the affected credential/allowance pool, retain the observed reset time and source, and create one durable wake condition. Unknown reset time waits for a new owner/account observation or operator/account change; it does not create a polling timer or another attempt on each heartbeat. A second profile using the same pool is not fresh quota. |
| Transient rate limit or provider overload | Respect an observed retry-after/reset signal with bounded backoff and jitter; reduce that provider's admission pressure. One owner timer replaces polling workers. |
| API credits or spending limit exhausted | Stop paid dispatch for that allowance pool. Retain partial work and reservations. Use another preauthorized funded route only within the original aggregate limit; never buy credits or enable automatic top-up by inference. |
| Authentication/authorization failure or explicit refusal | Report the owner state and required remedy. Route changes must not bypass it. |
| Transport lost after a possible accepted start | Reconcile original work identity; do not assume that a new session or provider is safe. |
| Normal finite completion | Deliver the durable result once and release reconciled reservations. Do not restart to keep the process alive. |

Normalized limit evidence retains the observed category, runner/provider/model, credential-pool reference, affected allowance window, retry-after/reset time when supplied, observation time, resume/session/checkpoint reference and uncertainty. Keep vendor account details and tokens out of logs. Known quota waiting is an operational pause, not a source-code repair attempt; attempts already made still consume usage and applicable transport/restart allowances.

Prefer a compatible native resume when it preserves context and pending effects. A cross-runner/model fallback may need a new session seeded from the durable handoff. It must retain task identity and evidence without pretending to preserve a vendor-specific session. Qualify tool use, structured output, model settings, cancellation and one representative accepted result before admitting that route. Catalog availability and an HTTP success status alone are insufficient; streamed failures and partial work must reach the owner state.
