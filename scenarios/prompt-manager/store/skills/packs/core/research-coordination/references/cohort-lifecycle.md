# Research cohort lifecycle

Use this small state model for every multi-study commission. The cohort ledger is
the coordination source of truth. A native goal, chat transcript or Agent
Manager run is an execution surface; it does not replace the ledger.

## Cohort state

| State | Meaning | Allowed next state |
|---|---|---|
| `prepared` | Packet is reviewable, but no commission clock, allowance or worker is active. | `active` only after explicit commission and fresh cohort/worker identities; `closed` if the packet is retired without a commission. |
| `active` | Commissioned work may be dispatched within its shared allowance. | `closing` when the stop condition fires, the operator stops, or the allowance reaches closure. |
| `closing` | Stop new admissions; checkpoint every row and reconcile owned effects through their owners. | `closed` when each study has a truthful disposition and remaining effects are recorded. |
| `closed` | The commission is handed back. New work needs a fresh commission and identities. | No transition. |

An explicit operator stop immediately moves the cohort to `closing`; it does not
claim that every worker has stopped. Do not relabel the cohort `closed` until
known effects have been reconciled or clearly recorded as unresolved.
If a row cannot be reconciled, close the commission with that row's last
verified state, `unknown` owner telemetry and exact owner-driven resume route;
never relabel it `ended` to make the cohort look clean.

## Commission series and current pointer

When a portfolio is stopped, retried or recommissioned, keep a small series
index beside `cohorts/` (for example `six-topic-series.md`). Link every cohort
record and mark exactly one as the current commission. Keep execution, worker
and native-goal observations in separate columns. A prepared packet is not
active; update the pointer to `active` only after explicit commission and
verified dispatch. A stopped or superseded cohort remains linked as history.
An operator-confirmed stop and terminal known workers can authorize a fresh
commission while native-goal telemetry remains `unknown`; record that unknown
without listing the predecessor as active or blocking independent rows. Use a
new cohort ID and clock for a fresh commission. Independent portfolio series
may run at the same time.

## Study execution state

Each row uses one of these execution states. Record result disposition and
native-goal telemetry in separate fields.

| State | Meaning | Allowed next state |
|---|---|---|
| `queued` | Not dispatched. If capacity is unavailable, name the constraint and next admission trigger. | `dispatching`, `blocked`, `not-started` |
| `dispatching` | One owner request is in flight. Save the request/idempotency key and do not submit it again until its outcome is known. | `active`, `queued` only after confirmed no admission, `dispatch-uncertain`, `ended` |
| `dispatch-uncertain` | The request may have started but the owner cannot confirm. Hold only this row, reconcile through the owner or an authorized independent source, and continue other rows. | `active` or `ended` when found; `queued` only when no execution is confirmed |
| `active` | A worker identity and current work are confirmed. Review its checkpoint at material events. | `paused`, `blocked`, `ended` |
| `paused` | Work is intentionally suspended; record who/what can resume it and under which commission. Human decisions about later resumption stay in the attention record until resume. | `active`, `ended` |
| `blocked` | A human-owned decision, credential, approval or external action has no safe alternative. Name the exact action and continue independent rows. Technical/infrastructure friction alone is not `blocked`. Record `resume_state` as `queued` or `active`. | `queued` or `active` as recorded when resolved; `ended` |
| `ended` | The execution owner confirms that the worker ended or the operator explicitly stopped it. | Terminal |
| `not-started` | The cohort closed before dispatch. Do not call this `queued` in a closed cohort. | Terminal |

The normal path is `queued → dispatching → active → ended`. A failed admission
returns to `queued` only after the owner confirms no execution was created. If
dispatch is uncertain, use `dispatch-uncertain`; never infer `queued` from a
missing worker record or infer `active` from a goal prompt.

Store result disposition separately as `pending`, `complete`, `partial` or
`no-result`, with evidence and the reason. `ended` says only that execution
stopped; it does not say the research outcome succeeded. `not-started` has
`no-result` disposition.

## Observations are not states

Store each fact with its source and observation time. Use `unknown` for a field
that could not be read; do not turn it into `idle`, `stopped`, `failed`, or
`blocked`. A user-confirmed stop is enough to close the *commissioning decision*
and start a fresh commission, while unavailable native telemetry remains
`unknown`. It does not prove that an unobserved process terminated.

An Agent Manager run record proves that the owner recorded a run; it does not
prove the worker process or session is live. Record execution origin and mode
separately from live owner evidence.

Do not make a portfolio-wide decision from one row's uncertainty. Hold the
affected action, continue independent rows, and escalate to the user only when
the unresolved choice belongs to them and no safe path remains.

## Record layout

A cohort record is three files in the cohort folder (`LONG_EFFORT_DESIGN.md` D15):

| File | Holds | Written | Read |
|---|---|---|---|
| `README.md` | The charter: state line, operator stance verbatim, destination, cards, scheduler procedure, owner brief template, commissioning messages. Stable, under about 25 KB. | When the commission or the operator's direction changes | By the scheduler at every unit start and after every compaction |
| `STATE.md` | One page, overwritten: `Needs you` first; then per card the workstream statuses, the current owner and its start time, the last measured change, what it waits for, and the next review time; then the cohort fields below. | At each review | By the scheduler at every unit start, by a new coordinator, and by the operator in place of a status question |
| `LEDGER.md` | Append-only dated entries, one per unit: what was reviewed, the verdict, the measured change, the dimensions moved. No-change verdicts in a separate list. | At each unit | On demand, by section; never in full |

Owners do not read the cohort folder. Their brief carries their card, and they re-read their
`MISSION.md`. Keep `STANCE-MESSAGE.txt` and `KERNEL.txt` beside the charter; the kernel is set
verbatim.

## Minimum ledger fields

For the cohort: state, commission source, cohort ID, `started_at`, cooperative
allowance/deadline, supervisor/writer, and current checkpoint. `started_at` is
the supervisor's first action after commission, not packet preparation or a
later worker dispatch.

For each study: state, topic/run path, owner and worker identity, dispatch key,
last verified progress time, result disposition, next action, and effect
cleanup. For `blocked` rows, also record `resume_state` so resolution returns
to the correct `queued` or `active` state. Record native-goal status separately
as `{value, source, observed_at}`; missing reads remain `unknown`.

When timing or capacity affects a decision, also record actual dispatch and
observed start/end times, exit reason, successor, shared-resource order and
uncovered intervals. These are timestamped observations, not inferred worker
states. Link the execution owner's records instead of copying their histories.
