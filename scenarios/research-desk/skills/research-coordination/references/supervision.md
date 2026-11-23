# Commission records and monitoring

Read when you run or take over a commission. The charter carries the unit loop.

## The commission record

`cohorts/<id>/` holds the charter (`README.md`, stable, about 25 KB at most), `STATE.md` (one page, overwritten each review: `Manager:` line, commission and session IDs, `Goal: <status>`, `Needs you` first, then per card status, owner, last measured change, waits, next review), `LEDGER.md` (append-only, one entry per unit, read by section) and `OPERATOR.md` (the operator's steering and the desk's delivery records; written only by them). Owners never read the folder; their brief carries their card. One writer per file.

A window flagged is one that is dead (an hour in, nothing in its folder or run folders written past its first 15 minutes: a manager that only talks), has no `STATE.md` an hour in, or whose `STATE.md` says the goal is active while it is paused or blocked: a `FLAG` line in `research-desk now`, and `flags` (a list of sentences) on that commission in `research-desk now --json` and `commissions --json`. The `Goal:` line is read only from the header above the first `##` heading; the last step of a window sets it to `ended` or `paused`, so a clean wind-down is never flagged, and a window past its allowance is not flagged for a missing STATE.md, but a dead one stays flagged while its goal is open: stop or relaunch it. A `halted` window (its goal stopped because a model turn ended on an error) is resumed by the desk itself when the error class allows; it is flagged only once the desk escalates (class never resumed, cap reached, error stale, allowance ended), and then the steward decides (Research Desk SYSTEM §6).

## States

A commission is `prepared` (packet only), `active`, `closing` (no new admissions; every row checkpointed) or `closed`; a new window gets a fresh ID and clock. A study row is `queued`, `dispatching` (never resubmit until its outcome is known), `dispatch-uncertain` (hold only this row), `active`, `paused`, `blocked` (only a human-owned decision with no safe alternative), `ended` or `not-started`; its result is `pending`, `complete`, `partial` or `no-result`.

Every observation keeps its source and time. A field you could not read is `unknown`, never `idle`, `stopped` or `failed` (R6). One row's uncertainty never stops the others.

## Footprint

Owners stop every resource, service and model they start. A written rule did not prevent a model holding 7.5 GiB for nine hours, so the check is mechanical: `scripts/footprint.py snapshot` once before the first owner, then `diff --baseline B --keep-file K` at every review, before every owner message and before GPU work. It prints what runs now that did not at the baseline and exits 1 while a line remains; it stops nothing. Each owner lists what it starts under `Started` and stops only its own lines; the manager stops lines whose starter ended and writes unclaimed ones into the keep file. `NOT CHECKED` is not clean. Refused admission: run the check first, wait through the owner or use a fleet node, staff other work. Capacity never closes a run.

## Monitoring

Health says a service answers, not that its feature works; friction says agents hit a problem, not that the service caused it. A missing signal is `unknown`, not "no friction".

| Need | Tool | Direct path |
|---|---|---|
| Tools down or `unknown`, who is affected, the fallback | `python3 -I scripts/topic_health.py [--topic T --friction]` | `TOOLS.md`; `vrooli scenario status <s> --json` |
| Plans that stopped steering | `research/plan-health` | `work/README.md` |
| Throughput: landed, stalled, hours | `python3 -I ../research-review/scripts/history.py portfolio` | Ledgers, `MISSION.md` status cells |
| Live owner runs, event gaps | `program-runtime.research-supervision-read`; `effort_anatomy.py drift` | Owner checkpoints |

Each reports a down dependency as `unknown` with its fallback, never a verdict that stops work. A program's home is the research home's `programs/` or a topic's `programs/`; add one only after the same repeated read appears twice (R10).
