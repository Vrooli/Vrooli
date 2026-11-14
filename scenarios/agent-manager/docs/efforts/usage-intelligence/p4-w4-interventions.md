# P4-W4 intervention corpus: findings (2026-10-30)

Corpus: `api/internal/runsignal/testdata/interventions/operator-messages.jsonl` (571 rows; schema, label rules and
caveats in the README next to it). Single labeller, blind first pass then a second pass over all 92 low-confidence rows.

## Counts
| category | claude | codex | total | holdout |
|---|---|---|---|---|
| approval_or_info | 104 | 154 | 258 | 77 |
| scope_change | 50 | 41 | 91 | 27 |
| quality_miss | 36 | 44 | 80 | 24 |
| false_inability | 12 | 14 | 26 | 8 |
| early_stop | 8 | 11 | 19 | 6 |
| instruction_ignored | 8 | 9 | 17 | 5 |
| too_dense | 8 | 7 | 15 | 4 |
| wrong_claim | 7 | 6 | 13 | 4 |
| teammate (not operator) | 0 | 50 | 50 | - |
| injected | 1 | 1 | 2 | - |

Prevalence (the 248-row random draw only): about 22% of non-trivial operator messages are friction (54/250), in line
with the SPEC's "about 1 in 6". Quality_miss is 60% of that. wrong_claim is genuinely rare (13 even after cue-mining
roughly 1,000 messages; the 15-example target is missed by 2). Hold-out has only 4-8 rows for the five rare categories:
P4-A4 per-category precision/recall there will be very noisy (one miss moves recall 12-25 points); gate the rare ones
on pooled friction or widen the set before trusting per-category numbers.

## Teammate vs operator discriminators (Codex rollouts)
1. `session_meta.thread_source == "subagent"` (first line, `source.subagent.thread_spawn.parent_thread_id`): all user-role
   text is parent assignment (92 distinct messages). Forked rollouts also replay the parent's earlier
   operator messages with identical timestamps, so dedupe on (timestamp, text) or operator messages double-count.
2. `content_item_metadata[].provenance.type == "user"` on the message (`internal_chat_message_metadata_passthrough`) marks
   operator text, but only from about 2026-10-29 (88 rows); older rollouts have no provenance, so absence proves nothing.
3. `content_item_kinds`: operator text is `["user.text"]` (or with `user.image`); injected is `goal.internal_context`,
   `agents_md.instructions`, `environments.environment_context`, `user.goal`. Roughly 500 injected rows were dropped.
4. Timing: dense "inter-agent" reports arrive 0.01-0.7 s after the previous `task_complete` (task_started follows
   within ms). Not sufficient alone: operator messages queued during a turn also arrive < 0.3 s (25 of ~640).
5. Text shape: glued tokens ("andactualDOMAIN_SEPARATOR", "exactT3response"), `HH:MMZ` prefixes, words like
   "Material proof:", "Handoff received", almost no I/you/please. 48 such messages in one 2026-10-27 burst in
   session 1e5904ed (supervisor-managed research workers). A session-level rule (>= 5 matches) removed the
   2 false positives a per-message rule gave.
6. `inter_agent_communication_metadata` records and `response_item/agent_message` (author `/root/...`) are the
   native-subagent channel; in the 2026-10 rollouts they are separate records, not user messages, so they do not leak
   into the user role. Seen only in one 2026-10-30 forked rollout (not surveyed further).
Claude side: teammate/task-notification/"Another Claude session" messages are already prefix-filterable; the
`<send_user_message_question_reply>` wrapper is the operator answering an agent question (kept operator).

## Recurring friction themes, owner, examples
| theme | n | examples | owner |
|---|---|---|---|
| Agent stops / waits for a word / ends short of agreed window | 19 early_stop | "Keep going I don't know why you keep stopping." / "Are you stuck? ... haven't seen you do anything in like an hour" / "why the heck did you stop? ... much further like I instructed you to" | harness-goal-authoring + large-effort-orchestration (continuation contract, stop conditions, budgets inform never stop); agent-manager park/wake |
| Over-asking for approval/permission it already has | 26 false_inability | "Why do you need me for any of this?" / "I thought storage manager is supposed to handle safe cleanup... Why do you need my approval?" / "You should not need permission to do investigations" | docs/concepts/AUTHORIZATION.md (what is gated), AGENTS.md rule 9; privilege/approval skills; agent-manager approval tie-back |
| Fix reported done but still broken, notifications/approvals not arriving | 80 quality_miss | "The wrapping STILL does not work" / "It's not fixed! And if anything it looks even worse" / "I didn't receive a notification" | scientific-debugging + test-genie/ui-health proof-before-claim; experiential-ui-design; notification-hub owner for delivery |
| Told not to, did anyway (bug reports, plan manager, git, swarm manager, debt) | 17 instruction_ignored | "Don't file any bug reports. I'd much rather you investigate and fix" / "Don't use plan manager" / "DO NOT use git" / "only seen code additions and barely any removal" | report-bug skill (file vs fix default), per-task handoff via harness-goal-authoring, never-use-git memory rule; refactor/debt reminder in implementation-plan-execution |
| Explanations not followable | 15 too_dense | "You need to dumb this down for me. I don't understand a lot of this" / "I don't know what E3 is" / "Give me the tldr" | writing-standards (operator-facing reports: define terms, one decision, plain words); effort-supervision report template |
| Wrong referent / misremembered | 13 wrong_claim | "Those are finite delivery teams. Again not what I'm referring to" / "No, I meant you give me the links now" / "That's not the right video" | recall: search-hub query before answering recall questions; conversation-search; AGENTS.md rule 6 |

Next-order observation: many quality_miss rows are cheap to detect later in the transcript (operator re-reports the same
symptom within 1-3 messages: "still", "STILL", "again"); a "repeat report" feature (same symptom after an agent claimed
a fix) would likely be a stronger signal than the single-message category.

## Regex recall (`correctionPattern`, struggle_detectors.go:129, ported to Python, same pattern text)
Run: `python3` over `operator-messages.jsonl`, source=operator only. Recall = fraction of messages in the category matched.
| category | all (n) | random draw only | hold-out |
|---|---|---|---|
| early_stop | 3/19 = 0.16 | 0/2 | 1/6 |
| false_inability | 4/26 = 0.15 | 2/6 | 1/8 |
| wrong_claim | 6/13 = 0.46 | 1/4 | 3/4 |
| quality_miss | 12/80 = 0.15 | 4/33 | 3/24 |
| too_dense | 3/15 = 0.20 | 1/4 | 2/4 |
| instruction_ignored | 7/17 = 0.41 | 3/5 | 2/5 |
| any friction | 35/170 = 0.21 | 11/54 = 0.20 | 12/51 = 0.24 |
False positives: scope_change 5/91, approval_or_info 10/258. Precision for "friction" = 35/50 = 0.70 (0.58 on the random
draw). Conclusion: the regex fails P4-A4 recall (0.7) by a wide margin for every category (best: wrong_claim 0.46) and
its precision is also under 0.8; it mostly catches "I said / don't / that's not". The SPEC's 10/26 hand-labelled figure
is consistent with ~0.2-0.4 on friction messages that contain explicit correction phrasing.

## Caveats
Single labeller; a second labeller on the 92 low-confidence rows and the hold-out would be worthwhile before gating.
quality_miss vs scope_change is the main blur (bug report on pre-existing product behaviour = scope_change). Only
messages from the last ~3 weeks (Claude: 39 top-level sessions; Codex: 193 rollouts of 1,150) were sampled. Hold-out is
not session-disjoint. The model classifier needs `preceding_agent_text` for early_stop, false_inability and bare
"continue" cases; those are unlabelable from the message alone.

## Classifier benchmark (P4-W4b, 2026-10-30, Winnow-E4B, expanded corpus)

Model: `winnow:e4b` through `decide.accurate` (the package default role), 947/947 messages answered by it, no fallback.
Proof route: option wording written and compared on `split=train` only (659 operator rows); scored once on `split=holdout`
(304) and on the `sampled_by=random` draw (248 rows; most are train rows, so it is not independent of tuning, but it is the
prevalence-faithful view). Wilson 95% intervals. Winnow reads the full corpus state unclipped, so caps were relaxed from
1,200/300 to 2,000/1,000 chars (0 of 659 train states overflowed). State: operator message plus preceding agent text.
Structure kept: one 8-option choice. Tried on train and dropped: a second friction-only choice plus a friction noul
(two-stage) was no better than the single choice's friction mass (2x cost); Laya-era per-category nouls were worse.
Wording now carries the README boundary rules and the operator's literal cues per category (for example
false_inability = "restart it yourself", "you're allowed to", "why do you need my approval"; a bare "continue" is
approval_or_info). Train argmax lifts: false_inability recall 9/64 -> 26/64, quality_miss 45/82 -> 55/82, too_dense 23/42 -> 30/42.
Production rule unchanged: argmax category shown when its probability is at the train-picked threshold (max precision with
train recall >= 0.7). Only too_dense has a reachable threshold (p >= 0.35); the others use argmax.

`AGENT_MANAGER_LIVE_DECIDE=1 go test ./internal/interventions -run TestLiveInterventionBenchmark -v`:

| category | hold-out tp/pred/gold | P [95%] | R [95%] | random tp/pred/gold | P [95%] | R [95%] | shown |
|---|---|---|---|---|---|---|---|
| early_stop | 8/12/18 | 0.67 [0.39,0.86] | 0.44 [0.25,0.66] | 0/2/2 | 0.00 | 0.00 | no |
| false_inability | 16/18/29 | 0.89 [0.67,0.97] | 0.55 [0.38,0.72] | 4/4/6 | 1.00 [0.51,1.00] | 0.67 [0.30,0.90] | no |
| wrong_claim | 10/19/16 | 0.53 [0.32,0.73] | 0.62 [0.39,0.82] | 3/5/4 | 0.60 | 0.75 | no |
| quality_miss | 22/35/36 | 0.63 [0.46,0.77] | 0.61 [0.45,0.75] | 20/26/33 | 0.77 [0.58,0.89] | 0.61 [0.44,0.75] | no |
| too_dense | 13/15/17 | 0.87 [0.62,0.96] | 0.76 [0.53,0.90] | 1/2/4 | 0.50 | 0.25 | no |
| instruction_ignored | 5/5/16 | 1.00 [0.57,1.00] | 0.31 [0.14,0.56] | 1/1/5 | 1.00 | 0.20 | no |
| any friction (per-category rule) | 93/104/132 | 0.89 [0.82,0.94] | 0.70 [0.62,0.78] | 33/40/54 | 0.82 [0.68,0.91] | 0.61 [0.48,0.73] | - |
| pooled friction (mass >= 0.50, train-picked) | 98/108/132 | 0.91 [0.84,0.95] | 0.74 [0.66,0.81] | 34/41/54 | 0.83 [0.69,0.91] | 0.63 [0.50,0.75] | no |

Decision: **`ProvenGate` stays empty; no category and no pooled bucket is shown.** Bars were not lowered; a category must
meet them on both views. Why each fails and what would fix it:
- too_dense: passes hold-out (0.87/0.76) but the random draw has 4 gold rows and 1 hit; nine train too_dense rows are
  read as approval_or_info (information questions about terms). Fix: more random-draw too_dense rows (n=4 cannot prove 0.7).
- false_inability: precision is high (0.89 / 1.00) but recall 0.55: "do X yourself" / "you're allowed" messages that also
  carry a task are read as approval_or_info or scope_change (24 and 8 of 64 on train). Fix: a label rule for mixed
  permission-plus-task messages, or a second labeller; model-side the cue is already in the wording.
- quality_miss: precision 0.63 on hold-out; bug reports are confused with scope_change/wrong_claim, which is the README's
  own blurred boundary (pre-existing bug = scope_change) and the single labeller. Fix: relabel review of the 14 train
  quality_miss rows read as scope_change.
- early_stop (recall 0.44), wrong_claim (precision 0.53), instruction_ignored (recall 0.31; confused with quality_miss, 10 of 31):
  the model cannot separate them from the agent text alone; bare "continue" and "don't ..." instructions are labelled by
  context a 1,000-char agent tail often lacks. Fix: longer agent context or tool activity in `preceding_agent_text`.
- pooled friction: hold-out passes; the random draw misses recall (0.63 at the train-picked mass 0.50). No mass threshold
  satisfies both bars on the train random rows either (0.40: 0.79/0.69; 0.35: 0.76/0.74), so the pooled
  "operator friction (category unproven)" bucket is not implemented.
Speed: 71-134 ms/message depending on GPU contention (Ollama busy slows it; if Winnow is evicted the gateway falls
through, so check the "answering models" line).

### Validation run and final gate (amendment 2, 2026-10-30)

Same classifier (`choice-aa4ecb195d12`, Winnow `winnow:e4b`, `decide.accurate`; 1,262/1,262 rows answered by it, no fallback, 148 ms/row).
Thresholds still train-picked (only too_dense p >= 0.35 reachable; pooled mass >= 0.50). The hold-out was re-scored after the
README boundary rules adjudicated three hold-out labels (quality_miss -1, wrong_claim +1). Validation = `validation-random.jsonl`,
315 operator rows, untouched; "random" rows are retired as a check.

| category | hold-out tp/pred/gold | P [95%] | R [95%] | validation tp/pred/gold | P [95%] | R [95%] | shown |
|---|---|---|---|---|---|---|---|
| early_stop | 8/12/18 | 0.67 [0.39,0.86] | 0.44 [0.25,0.66] | 0/1/3 | 0.00 [0.00,0.79] | 0.00 [0.00,0.56] | no (gold < 10) |
| false_inability | 16/18/29 | 0.89 [0.67,0.97] | 0.55 [0.38,0.72] | 0/0/3 | - | 0.00 [0.00,0.56] | no (gold < 10) |
| wrong_claim | 11/19/17 | 0.58 [0.36,0.77] | 0.65 [0.41,0.83] | 4/4/9 | 1.00 [0.51,1.00] | 0.44 [0.19,0.73] | no (gold < 10) |
| quality_miss | 22/35/35 | 0.63 [0.46,0.77] | 0.63 [0.46,0.77] | 39/54/48 | 0.72 [0.59,0.82] | 0.81 [0.68,0.90] | no (P) |
| too_dense | 13/15/17 | 0.87 [0.62,0.96] | 0.76 [0.53,0.90] | 2/6/2 | 0.33 [0.10,0.70] | 1.00 [0.34,1.00] | no (gold < 10) |
| instruction_ignored | 5/5/16 | 1.00 [0.57,1.00] | 0.31 [0.14,0.56] | 0/1/2 | 0.00 [0.00,0.79] | 0.00 [0.00,0.66] | no (gold < 10) |
| any friction (per-category rule) | 94/104/132 | 0.90 [0.83,0.95] | 0.71 [0.63,0.78] | 48/66/67 | 0.73 [0.61,0.82] | 0.72 [0.60,0.81] | - |
| pooled friction (mass >= 0.50) | 99/108/132 | 0.92 [0.85,0.96] | 0.75 [0.67,0.82] | 48/66/67 | 0.73 [0.61,0.82] | 0.72 [0.60,0.81] | **no** |

**Final decision: `ProvenGate` stays empty.** No category is shown (only quality_miss has >= 10 validation gold rows, and it fails
precision on the hold-out 0.63 and validation 0.72). The pooled "operator friction (category unproven)" bucket passes the
hold-out but fails precision on validation (0.73, bar 0.8; the Wilson interval [0.61,0.82] does not rule the bar out, it does not
prove it). Bars were not lowered. The validation misses are 18 false alarms: quality-miss-like reads of scope_change bug reports and
information questions, the boundary the second labeller also found unstable (kappa 0.87 on the hold-out). What would change the
verdict: a larger validation draw (about 800 rows per ~8 positives for rare categories) or a model/wording change followed by a
fresh validation set; the validation set is spent for tuning purposes once it is used to choose wording.
The gate API supports the pooled bucket when proven: `Gate.ShownPooled(model, answer.FrictionMass())`, key `PooledFriction`.

## Corpus expansion (2026-10-30, P4-A4 statistical power)

Added 428 operator rows (`sampled_by=targeted-expansion`; corpus now 999 rows) mined from all history, not just the last
30 days: 38,228 Codex user messages across 5,452 rollouts (2025-09 .. 2026-10) and 687 Claude messages; after dropping
duplicates, forks, templates and sessions already in the corpus about 11,900 candidates remained. Claude yielded no new
sessions (its stored transcripts are all October and already sampled); all new rows are Codex (2025: 210, 2026: 218).

| category | before (hold-out) | new train | new hold-out | now total (hold-out) |
|---|---|---|---|---|
| early_stop | 19 (6) | 26 | 12 | 57 (18) |
| false_inability | 26 (8) | 47 | 21 | 93 (29) |
| wrong_claim | 13 (4) | 29 | 12 | 54 (16) |
| quality_miss | 80 (24) | 26 | 12 | 118 (36) |
| too_dense | 15 (4) | 31 | 13 | 59 (17) |
| instruction_ignored | 17 (5) | 19 | 11 | 47 (16) |
| scope_change | 91 (27) | 26 | 12 | 129 (39) |
| approval_or_info | 258 (77) | 92 | 40 | 390 (117) |

Every friction category now has >= 15 hold-out rows (target >= 50 total met for early_stop, false_inability,
wrong_claim, too_dense; instruction_ignored 47). New rows are session-disjoint across the split (272 new sessions, 42 in
hold-out, 0 shared); old rows are not. Wilson 95% half-width at n=16 is still about +/-0.2 around p=0.8, so per-category
gates remain loose; pooled friction is the sturdier gate.

### Cue yield (how many candidates read, how many true positives) - detector design input
Lexical cues are weak and category-specific. Rates are share of cue matches whose reading-time label was that category
(cue lists excluded rows labelled in earlier passes, so later lists are slightly understated).
| cue family | read | true positive | rate |
|---|---|---|---|
| early_stop: "keep going / don't stop / why did you stop / are you stuck / finish" | 52 | 21 | 0.40 |
| early_stop: any short "continue / keep going / finish / proceed" (<=160 chars) | 130 | 5 | 0.04 |
| early_stop: broad "stop / finish / done / wrap up" (<=260 chars) | 135 | 8 | 0.06 |
| false_inability: "yourself / you're allowed / permission / shouldn't need / can't you" | 159 | 67 | 0.42 |
| wrong_claim: "that's not / I meant / wrong / not what I / referring to" | 114 | 32 | 0.28 |
| too_dense: "confused / don't understand / dumb it down / simplify / explain exactly" | 146 | 40 | 0.27 |
| instruction_ignored: "I told you / I said / don't / never / again / you keep" | 96 | 17 | 0.18 |
Observations: (1) bare "Continue please" is overwhelmingly approval (harness crash/limit resumes, post-status nudges): 1
of ~11 such rows is early_stop, and only visible from the preceding agent text (offer to stop, unfinished list, "You
got stuck"). (2) The best early_stop signal is an explicit complaint about stopping ("keep stopping", "you stopped",
"are you stuck", "I didn't tell you to stop"). (3) false_inability is the most lexically detectable: "...yourself",
"you're allowed/you have permission", "you shouldn't need my approval", "can't you do that" (0.42 precision). (4)
too_dense and wrong_claim cues also fire on genuine information questions ("What is X exactly?", "I meant ...?") - needs
the model. (5) instruction_ignored cues mostly catch "don't ..." instructions (commit-scoping requests, "Do not commit
anything outside this scenario") that are approvals, not violations; the real signal is the operator citing an earlier
instruction ("I told you", "as I said", "you keep", "NEVER", "you weren't supposed to"). (6) Several complaints combine
categories ("You keep stopping" + "don't use X"): label followed the README tie rule. (7) The old regex
`correctionPattern` misses the highest-yield phrasings: "yourself", "keep going", "dumb it down", "you stopped".

### Caveats
Single labeller, 125 of 428 new rows low-confidence; cue-selected rows overstate prevalence (only the 80 random
negatives are unbiased, of which 9 were friction); older (2025) Codex sessions are mostly agent-manager-era, short
interactive turns and may differ in style from 2026; preceding_agent_text is the last assistant message only. Three
"Keep going please" rows come from one session (019ab4, same split). The classifier benchmark above was re-run on the expanded corpus.

## Validation sample (2026-10-30)
Files: `api/internal/runsignal/testdata/interventions/validation-random.jsonl` (316 rows), `agreement.jsonl` (60 rows). Procedure,
seeds and exclusions are in that directory's README ("Validation set"). Labels written before any hold-out label was read.

### Part A: fresh uniform random sample
Pool 9,008 operator messages in 2,900 sessions not in the corpus (Codex 9,002; Claude 6 because every Claude transcript is
already a corpus session). Seed 20261030, 320 drawn, 4 fork replays removed, 1 teammate found -> 315 operator rows.

| category | rows | low-conf |
|---|---|---|
| approval_or_info | 158 | 36 |
| scope_change | 90 | 23 |
| quality_miss | 48 | 13 |
| wrong_claim | 9 | 7 |
| early_stop | 3 | 3 |
| false_inability | 3 | 3 |
| instruction_ignored | 2 | 1 |
| too_dense | 2 | 0 |

Friction prevalence 67/315 = 21.3% (quality_miss alone 15.2%). Four friction categories have < 8 gold rows (early_stop,
false_inability, instruction_ignored, too_dense), so P4-A4 recall cannot be evidenced for them from a random draw this size:
at ~1% prevalence about 800 rows are needed per ~8 positives. Only quality_miss supports a tight interval; wrong_claim (9, 7 of
them low confidence) is borderline. Note the template filter (60-char prefix >= 4x) removes repeated boilerplate approvals, so
approval_or_info is somewhat under-represented relative to raw traffic.

### Part B: inter-annotator agreement (60 hold-out rows, blind)
Observed agreement 54/60 = 0.900; Cohen kappa 0.870 (8 categories). Friction-vs-not: agreement 0.933, kappa 0.864 (27 original
friction vs 25 mine). All six disagreements are distinct pairs (original -> second labeller): scope_change -> wrong_claim,
false_inability -> approval_or_info, approval_or_info -> scope_change, quality_miss -> scope_change, quality_miss ->
approval_or_info, instruction_ignored -> quality_miss. Four of six involve the scope_change / approval_or_info / quality_miss
triangle, as expected; the sample is too small to rank pairs.

### Boundary rules the disagreements show to be ambiguous (proposed wording)
1. Correction plus request ("X was bad, now find better X"). One disagreement was my own breach of the existing tie rule.
   Proposed: "If a message criticises prior agent output and also asks for related work, label quality_miss, even when the
   criticism is a past-tense remark ('that was my biggest complaint'). scope_change only when the earlier output is not criticised."
2. Restated instruction vs still-broken. "I said X ... you are adding complications. That didn't fix it." Proposed: "If the
   message cites an earlier operator statement ('I said', 'as I told you', 'again') that the agent is now contradicting, label
   instruction_ignored; plain 'did not work / still broken' without such a citation is quality_miss."
3. Answers to agent verification questions. `<send_user_message_question_reply>` or a plain reply that reports a negative test
   result ('No, nothing appeared') was labelled quality_miss by the original and approval_or_info by me. Proposed: "A reply
   that only supplies requested information is approval_or_info; if the agent had asserted success and the reply reports the
   deliverable failing, it is quality_miss."
4. Operator challenges an agent-stated limitation ('the API rejects X'; 'I promise the API is running'). Original: false_inability
   / scope_change; mine: approval_or_info / wrong_claim. Proposed: "If the agent said it cannot do something and the operator
   says it should be able to, false_inability; if the operator disputes a factual diagnosis ('it IS running'), wrong_claim; a
   neutral question about why the limitation exists ('what exactly is blocking?') is approval_or_info."
5. Question that embeds a design proposal ('what do you mean by X? ... better to do Y'). Original approval_or_info, mine
   scope_change. Proposed: "A question that also states a different design or new requirement is scope_change; a bare
   clarifying question is approval_or_info."
6. First-message investigation requests ('explain how X works', 'investigate and report'). Both labellers lean scope_change
   for 'investigate/report/plan' and approval_or_info for pure 'explain/tell me' questions; the README should say so explicitly,
   because first messages have no preceding agent text and the current text ('new task' vs 'information questions') does not decide it.
Caveat: the second labeller is also the author of the Part A validation labels; kappa measures reproducibility of the README
rules, not correctness against an external truth.
