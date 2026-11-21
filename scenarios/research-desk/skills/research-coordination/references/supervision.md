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

## Reports and feedback

The supervisor owns this loop; it does not ask owners to assemble a separate report. Topic
files remain authoritative and Research Desk is not required. For each topic:

1. **Before staffing.** Verify the builder, Hub template/schema and delivery path; repair gaps.
   Run `python3 -I scripts/report_feedback.py <topic-folder>` before packet build and kickoff.
   It migrates `reports/*.json` receipts (preserving files and recording old paths), reads all
   history through Hub, saves answers in `feedback/` and flags reviews in `work/README.md`.
   Reconcile any read error. Inspect saved answers and packet feedback; retrieve and inspect
   referenced images, or leave unavailable content pending. Preserve words and weights; mark-read
   is not feedback. For each item, update the plan and affected briefs with its workstream or a
   reason it does not apply. Read all steering since the last commission, including `FEEDBACK.md`;
   copying an item alone is not application.
2. **At allowance end, pause, stop, or on request.** Gather the mission, plan, ledger, inbox,
   feedback, commission state, runs and checkpoints. Review what landed, changed, waits, remains
   unverified and comes next. Judge every scorecard dimension from evidence; missing evidence is
   unavailable, never a fabricated measure. Collect sources without `--review`, then write review
   JSON with `meta.title` and one exact mission `dimension`, `level` and evidence-backed `reason`
   per scorecard row. Add other reviewed findings. Run `python3 -I scripts/research_report.py <topic-folder>
   --commission <id> --review <review.json> --output <report.json> --validate-schema
   scenarios/notification-hub/skills/notification-hub/kit/schemas/research-report.schema.json`.
   The builder reports JSON paths for schema violations. Preview and check the page at phone
   width in light and dark as directed by the notification-hub skill.
3. **Deliver and record.** Send with the status template, stable topic subject key, plain cover,
   and `--attach <topic>/runs/<commission>/evidence=evidence`. Read back the conversation; record
   ID, key, report path, evidence time and outcome in `STATE.md` or `LEDGER.md`. Save review,
   report, receipt (`ask_id`, key, attachment name) and answer in
   `reporting/history/<UTC yyyy-mm-ddTHHMMSSZ>-<commission>/`; update `reporting/latest.json`.
   Never overwrite receipts or claim failed sends. Keep the report and exact error pending for a
   Hub retry; a held-back push is not a failed send.
4. **Reconcile steering.** Run `report_feedback.py`; set each item in
   `feedback/hub-*-items.json` to applied, planned or declined with a reason via
   `python3 -I scripts/report_feedback.py <topic> --reconcile <item-id> <status> <reason>`.
   The next report shows these outcomes. Update affected live briefs and owners; keep each source
   answer and distinguish applied, pending and not applicable items.

If Hub is unavailable, keep the report and feedback state in files and record the route failure;
do not substitute a needs-you decision for this status update. Repair the path before the next
staffing. Do not let a paused or closed goal erase the pending delivery.

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
