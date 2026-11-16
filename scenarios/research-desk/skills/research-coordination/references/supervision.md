# Supervision records and monitoring

Read when you run or take over a commission. The charter carries the unit loop; this page holds
the records, the footprint check and the monitoring behind it.

## The commission record

Three files in `cohorts/<id>/`, plus the launch messages beside them:

| File | Holds | Written |
|---|---|---|
| `README.md` | The charter: stance verbatim, destination, cards, scheduler procedure. Stable, under 25 KB. | When the commission or the operator's direction changes |
| `STATE.md` | One page, overwritten: `Needs you` first, then per card the statuses, owner, last measured change, what it waits for, next review | Every review |
| `LEDGER.md` | Append-only, one entry per unit: what was reviewed, the verdict, the measured change, dimensions moved | Every unit; read by section, never whole |
| `OPERATOR.md` | The operator's steering, answers and wind-down; the desk's delivery records | By the operator or research-desk only |

Owners never read the commission folder; their brief carries their card. One writer per file.

## States

A commission is `prepared` (packet only, no clock), `active`, `closing` (no new admissions;
every row checkpointed and its effects reconciled) or `closed`. A new window gets a fresh
commission ID and clock. An operator stop moves it to `closing` at once; it does not prove every
worker stopped.

A study row is `queued`, `dispatching` (one request in flight; never resubmit it until its
outcome is known), `dispatch-uncertain` (hold only this row and reconcile through the owner),
`active`, `paused`, `blocked` (only a human-owned decision with no safe alternative; record which
state it resumes to), `ended` or `not-started`. Result is separate: `pending`, `complete`,
`partial` or `no-result`. `ended` says execution stopped, not that it succeeded.

Every observation keeps its source and time. A field you could not read is `unknown`, never
`idle`, `stopped` or `failed`; a run record proves a run was recorded, not that the worker lives.
One row's uncertainty never stops the others.

## Footprint

The host is shared. An owner stops every resource, service and model it starts when the job ends.
A written rule did not prevent leaks (an owner once left a model holding 7.5 GiB for nine hours
while its own GPU showcase waited), so the check is mechanical, with `scripts/footprint.py`:

- **Baseline once**, before the first owner:
  `python3 -I scripts/footprint.py snapshot --out <cohort>/BASELINE.json`. It refuses to overwrite.
- **Check**: `python3 -I scripts/footprint.py diff --baseline <cohort>/BASELINE.json --keep-file
  <cohort>/FOOTPRINT-KEEP.txt` prints what runs now that did not at the baseline and exits 1
  while a line remains. Add `--disk --watch <dir>` for disk growth. It never stops anything.
- **When**: owners before every return message, before GPU work and on refused admission; the
  scheduler at every review and at the allowance end.
- **Whose line**: each owner lists what it starts under `Started` in its checkpoint and stops only
  its own lines. The scheduler stops lines whose starter ended and writes unclaimed lines into
  the keep file with a comment. `NOT CHECKED` is not clean: read that source directly.
- **Refused admission**: run the check first (the holder is usually something this commission
  started), then collect GPU jobs into one window, wait through the owner or move to a fleet
  node, and staff other work meanwhile. Capacity never closes a run.

## Monitoring signals

Scenario health, repeated friction, direct task evidence, open repairs and fallback outcomes are
separate signals: health says a service answers, not that its feature works for this research;
friction says agents hit a problem, not that the scenario caused it. Correlate them to decide what
to investigate, never to assign blame. A missing signal is `unknown`, not "no friction".

## Programs

Use a program when it answers on the first try; on failure take the direct path (the route rule).
The supervisor's programs live in the research home's `programs/` (`research/<name>`); a topic's
in `topics/<t>/programs/` (`research/<t>/<name>`). Run one with
`program-runtime research/<name> --json --input k=v`; a binding-free one runs from its files
when Program Runtime is down.

| Need | Program | Direct path |
|---|---|---|
| Tools down or `unknown`, who is affected, the fallback; friction | `python3 -I scripts/topic_health.py [--topic T --friction]` (a script, so it answers in any outage) | `vrooli scenario status <s> --json`; `TOOLS.md` |
| Plans that stopped steering: workstreams without a brief, stale risks, undrained inboxes | `research/plan-health` | `work/README.md` beside `MISSION.md` |
| Quiet runs and heavy checkpoints | `research/run-freshness` | `ls -t topics/*/runs` |
| Throughput: hours, lines, landed workstreams | `python3 -I ../research-review/scripts/history.py portfolio` | Ledgers, `MISSION.md` statuses |
| Live owner runs, event gaps, attention | `program-runtime.research-supervision-read` | Owner checkpoints; `effort_anatomy.py drift` |
| Workspace size and missing files | `program-runtime.research-workspace-review` | `du`, `find` over the research home |

Each reports a down dependency as `unknown` with its fallback, never a verdict that stops work.
Repeated reads with no fitting program are a reason to add one: explicit inputs, bounded output,
honest `partial`/`unavailable`.
