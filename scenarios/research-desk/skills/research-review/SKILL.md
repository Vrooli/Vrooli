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
  revision: 5
  createdAt: "2026-11-11T00:00:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    commands: ["python3", "rg"]
  origin:
    kind: "authored"
---
## Research review and topic audit

Rules: [RULES.md](../../../../docs/agent-system/RULES.md). A research topic drifts: clutter, a mission paraphrased smaller, knowledge as a pile of run logs, a mature look while nothing lands. Put it back on the operator's intent: smaller to read, clearer to act on, measurably moving (R2, R7). "Audit <topic>" is the whole request.

Research home `H` = `~/.vrooli/research`; scripts are in `scripts/`; file grammar in `scenarios/research-desk/docs/reference/FILE-CONTRACT.md`. If a service fails, read the files.

### Who decides

A review settles every question inside a topic's scope itself (a split, a merge, scope, order, what to cut): the mission and the operator's words already decide it (R5, R9). A question that needs the topic's manager goes to that topic's `INBOX.md` with `scripts/inbox.py` (`--from review`), not to the operator. Only a bright line (spending, outside contact, secrets, unrecoverable deletion) or a direction (what the end state is) becomes a needs-you item, with `If silent: <default> on <date>` ([operator attention](../research-coordination/references/operator-attention.md)). Never open an ask about a topic's internal scope.

### What farce and clutter look like

| Sign | How you see it |
|---|---|
| Busy without landing | `history.py topic T`: hours and repo lines up, landed 0, scorecard "none moved" |
| The mission shrank | Intent paraphrased; End state hedged; workstreams under 20 hours; a dimension no workstream moves; a cold owner's first scope smaller than before |
| Entry files a newcomer must wade through | `audit.py measure T`: MISSION over 20 KB (the launch limit), README a log, many top-level files |
| Knowledge organised by run | Findings only in checkpoints and the ledger; no page says what is true now |
| Stale current claims | Two pages disagree; `active` with nobody on it; a `waiting-time` row past its `resume` date |
| Lost operator concerns | Said twice, no feedback record shows the recurrence |
| Words put in the operator's mouth | An agent-written verdict or "operator stance" used to narrow the mission |
| Idle behind asks | `waiting-operator` rows the mode already authorizes, or asks already answered |

### Audit a topic

1. **Measure before.** `python3 -I scripts/audit.py measure T --out <review>/T.before.json`; read `history.py topic T`.
2. **Read the intent.** The operator's verbatim words in `MISSION.md`, the topic's feedback and inbox, verdicts in `H/reviews/*/VERDICTS.md` naming `T`.
3. **Diagnose.** For a topic that moved nothing, name the cause in the setup (mission, card, brief, route, capacity, management), not the owners' account (`effort-diagnosis` when unclear). No runs: "no history: nothing to diagnose".
4. **Restructure** to the common shape ([topic folder](../research-coordination/references/topic-folder.md)): README one screen; MISSION keeps operator words, end state, scorecard and workstreams, history moves to `history/` and receipts to `LEDGER.md`; the next commission's work becomes ready-to-run briefs in `work/`; knowledge rewritten by subject. Superseded pages move to `history/`; nothing inside `runs/` moves. Before a move, `rg -lF <name> H`, fix links, leave a pointer.
5. **Measure after.** `audit.py measure T --out <review>/T.after.json`, then `audit.py compare <before> <after>`. A smaller folder is never a merit by itself, and never trim operator words, End state, showcases or the scorecard to get there. Then replay the owner start: have a cold owner read the restructured folder and state the first checkpoint it would propose, and compare that scope with the one before. Smaller proposed work is a regression: undo the move that caused it.
6. **Record** `H/reviews/<id>/REVIEW.md`: every move as `from → to`, the before/after table, the diagnosis, what you left for the topic's manager.

### Plans and risks

Keep `work/README.md` honest: next briefs in order and ready, risks with their last review date. A risk that fired, a stale starting point, a workstream with no brief or a showcase past its resume date is the manager's next task. `program-runtime research/plan-health --json` reads this across topics; else read the files.

### Operator review

When the operator asks to review selected topics: gather per topic `mission_check.py H --topic T --json`, the commissions that touched it and `history.py topic T`; open every claimed landing (it exists, runs, matches the showcase; otherwise "claimed, unverified"). Write `H/reviews/<id>/REVIEW.md` (Landed, Claimed but unverified, Open, What failed organisationally, Decisions needed, Next commission). For work the operator judges by looking (media, screens, comparisons), also write `REVIEW.html`, one self-contained page beside it whose media links stay relative inside `H`; the desk opens it sandboxed, without scripts. Record the operator's verdicts verbatim, appended only with `inbox.py verdict H/reviews/<id> --topic <t> --source "<Hub answer id | OPERATOR.md line | chat session and UTC>" --body-file F`; stance quoted in a brief is never a verdict. For a live topic, put under **Direction for live commissions** the exact entry to apply with `research-desk say <id> direction`. Apply authorised changes the same day, never shrink ambition, and re-run `mission_check.py`.
