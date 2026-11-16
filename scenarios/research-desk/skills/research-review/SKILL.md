---
name: "research-review"
description: "Audit and restructure a research topic against the operator's intent: measure what actually landed and what a fresh owner must read, spot farce and clutter, reorganise the folder so it scales, keep plans and risks current, and run the operator's review of selected topics with verdicts recorded verbatim."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["research", "review", "audit", "missions", "planning"]
  icon: "clipboard-check"
  status: "active"
  revision: 4
  createdAt: "2026-11-11T00:00:00Z"
  updatedAt: "2026-11-15T08:00:00Z"
  requires:
    commands: ["python3", "rg"]
  origin:
    kind: "authored"
---
## Research review and topic audit

A research topic drifts: clutter, a mission paraphrased smaller, knowledge as a pile of run logs,
and a mature look while nothing lands.
Your job is to put it back on the operator's intent: smaller to read, clearer to act on, and
measurably moving. "Audit <topic>" is the whole request; you need nothing else to start.

Research home `H` = `~/.vrooli/research`; scripts are in `scripts/`. File grammar:
`scenarios/research-desk/docs/reference/FILE-CONTRACT.md`. When a service fails, read the files
and note it in one line.

### What farce and clutter look like

| Sign | How you see it |
|---|---|
| Busy without landing | `history.py topic T`: window hours and repo lines up, landed workstreams 0, scorecard "none moved" |
| The mission shrank | Operator intent paraphrased; End state hedged ("decision-ready", "foundation"); workstreams under 20 hours; a dimension no workstream moves |
| Entry files a newcomer must wade through | `audit.py measure T`: owner start read large, README a log, MISSION over ~15 KB (aim) or 40 KB (limit), many top-level files |
| Knowledge organised by run | Findings live only in run checkpoints and the ledger; no page says what is true now |
| Structure for show | Template files copied unchanged, empty folders, unbriefed table rows, plans that never drove a commission |
| Stale current claims | Two pages disagree; a "current" page older than the last three runs |
| Lost operator concerns | The operator said it twice and no feedback record shows the recurrence |
| Words put in the operator's mouth | A verdict or "operator stance" an agent wrote (its own return lists the file), then used to narrow the mission |
| Idle behind asks | `waiting-operator` rows the mode already authorizes, or asks the operator already answered |
| Small scope at commission start | A cold owner proposes a module or a report, not a whole API or scenario |

### Audit a topic

1. **Measure before.** `python3 -I scripts/audit.py measure T --out <review>/T.before.json` (read
   cost, clutter, mission check, throughput). Read `history.py topic T` for the full history.
2. **Read the intent.** The operator's verbatim words in `MISSION.md`, the topic's feedback and
   inbox, and verdicts in `H/reviews/*/VERDICTS.md` that name `T`. This is the bar; your taste
   is not.
3. **Diagnose.** Walk the signs above. For a topic that moved nothing, name the cause in the setup
   (mission, card, brief, route, capacity, supervision), not the owners' account
   (`effort-diagnosis` when it is not obvious).
4. **Restructure.** Use the common shape and the knowledge layout for the topic's kind
   (`research-coordination` [topic folder](../research-coordination/references/topic-folder.md)):
   - README becomes the one-screen front page; MISSION keeps operator words, end state, scorecard
     and workstreams, and sends receipts to `LEDGER.md`.
   - Knowledge is rewritten by subject (architecture current vs target, invariants and state
     machines, a claims register, a taste register), linking the runs that prove it.
   - The next commission's work becomes ready-to-run briefs in `work/`, with the risk list; every
     other row becomes a one-line `## Later` entry in the mission, so nothing looks staffed.
   - Superseded pages move to `history/`. Never delete. Nothing inside `runs/` moves. Before
     moving a page, `rg -lF <name> H --glob '**/{README,MISSION,CHECKPOINT}.md'`: update the
     links you can, and leave a one-line pointer at the old path for names inside checkpoints.
   - Add a part only when it saves real rediscovery.
5. **Measure after and compare.** `audit.py measure T --out <review>/T.after.json`, then
   `audit.py compare <before> <after>`. Done means `Smaller overall: yes`: the owner start read
   and the Markdown outside `runs/`, `history/` and `work/` end no larger, so condensing pays for
   new briefs. Never cut scope or operator words to get there.
6. **Record.** `H/reviews/<id>/REVIEW.md` per topic: what changed (every move as `from → to`),
   the before/after table, the diagnosis, what you left for the topic's owners, and decisions.
   One dated line in `LEDGER.md`.

A change to scope, mode, or a restructure that moves most of a topic is a supervisor-level
decision: write it as a needs-you item with a ready-to-run packet (`research-coordination`
[operator attention](../research-coordination/references/operator-attention.md)) and continue the
rest of the audit. Ordinary tidying needs no approval.

### Plans and risks

At each supervision cycle, keep `work/README.md` honest: the next briefs in order, each still
ready to run, and the risks with their last review date. Check it against the evidence: a risk
that fired, a stale starting point, a workstream with no brief, a showcase past its resume date.
`program-runtime research/plan-health --json` reads this across topics; else read the files. A
plan unchanged for three commissions while work landed is not being used.

### Operator review

The operator reviews a few topics at a time: show what actually happened, record what they
decide in their words, realign the missions.

1. **Gather** per topic: `mission_check.py H --topic T --json`; commissions that touched it
   (`rg -l "topics/T/" H/cohorts/*/README.md`, their `STATE.md`, ledger entries naming `T`);
   run checkpoints; `history.py topic T`. Open every claimed landing: it exists, runs or plays,
   and matches the showcase; what you cannot open is "claimed, unverified". A commission is live
   or paused when `research-desk commissions --state live|paused` says so or its `STATE.md`
   changed in the last 70 minutes.
2. **Write** `H/reviews/<id>/REVIEW.md` (`<id>` = topics slug + date): per topic Landed, Claimed
   but unverified, Open, Grew, What failed organisationally (cause and the change to the next
   packet), Organisation changes, Decisions needed; then **Next commission** with topics,
   emphasis and `read_first`. For media or rendered work, add one phone-friendly
   `REVIEW.html` that shows each exact artifact with a feedback form that exports JSON.
3. **Verdicts** are what the operator said about this topic, verbatim, appended only with
   `inbox.py verdict H/reviews/<id> --topic <t> --source "<Hub answer id | OPERATOR.md line |
   chat session and UTC>" --body-file F` (research-coordination scripts; it stamps the time).
   Stance quoted in a brief is never a verdict; no source, no entry. For each verdict on a live
   topic, put under **Direction for live commissions** the commission id and the exact entry
   for the operator to apply with `research-desk say <id> direction`; `OPERATOR.md` is theirs.
4. **Apply** authorised improvements the same day, by the audit steps above. Re-read `MISSION.md`
   before editing it, change only what the verdict touches, never shrink its ambition, and run
   `mission_check.py`.

**Exit:** the review files, the before/after tables, verdicts recorded verbatim, applied changes
with every move listed, and nothing deleted.
