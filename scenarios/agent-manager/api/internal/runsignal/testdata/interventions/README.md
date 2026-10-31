# Operator-intervention corpus (pass 4, P4-W4)

Labelled messages for the pass-4 intervention classifier (SPEC section 12, P4-F3 / P4-W4 / acceptance P4-A4:
precision >= 0.8 and recall >= 0.7 per shown category on the held-out split). One JSON object per line in
`operator-messages.jsonl` (999 records). Built 2026-10-30; the first 571 from transcripts dated 2026-10-06 .. 2026-10-30, 428 more
(`sampled_by=targeted-expansion`) from all older history (2025-09 .. 2026-10, Codex only; see Corpus expansion).

## Schema
`id` (sha1 of runner+session+timestamp+text, 12 hex) | `runner` claude|codex | `session` | `timestamp` |
`source` operator|teammate|injected | `category` (null unless source=operator) | `confidence` high|low |
`rationale` (one line) | `split` train|holdout | `sampled_by` | `text` (<=2000 chars, redacted) |
`preceding_agent_text` (last assistant text before the message, last <=1500 chars; empty when none).

## Categories (single label, source=operator only)
- early_stop: agent stopped, ended, parked or declared done before the work/time budget was exhausted. "Keep going",
  "why do you keep stopping", "are you stuck", agreed 8-hour window not honoured. Bare "keep going" counts only when
  the agent had just offered to stop or wait; "we got interrupted, continue" after a harness crash/session limit is
  approval_or_info.
- false_inability: agent claimed it cannot, needs permission, or asked the operator when it was allowed and able
  ("you're allowed to...", "request the approval yourself", "why do you need me", "you should be able to do that").
  Over-asking and delegating agent-doable steps to the operator belong here.
- wrong_claim: false statement, wrong referent or recall miss ("that's not the right video", "I meant X", "you said
  Y, that's not true"). Boundary with quality_miss: wrong_claim is about what the agent asserted or understood; quality_miss
  is about an artifact or outcome.
- quality_miss: output below expectation: still broken, "not fixed", did not work, unprofessional, notification never
  arrived. Operator bug reports about a prior agent change count; a bug report about a pre-existing product issue is
  scope_change.
- too_dense: operator cannot follow ("dumb this down", "I'm confused", "give me the tldr", "I don't know what E3 is").
- instruction_ignored: agent did what it was told not to, or ignored a stated standing preference ("don't file bug
  reports", "don't use plan manager", "do not use git", reminders about debt reduction/no hardcoding).
- scope_change: new direction, new task or new design topic; not a correction.
- approval_or_info: approvals, answers to agent questions, thanks, status pings, information questions, slash commands.
Friction categories = the first six. Ties: pick the category that names the agent's failure; if the message is a
correction plus a new task, label the correction.

## Boundary rules (adopted 2026-10-30 from the inter-annotator check; apply to every label, wording is binding)
Proposed in `p4-w4-interventions.md` "Validation sample" after 6 of 60 blind second labels disagreed. The tie rule above still
applies; these refine it. The classifier wording is frozen and does not mention them.
1. Correction plus request. If a message criticises prior agent output and also asks for related work, label quality_miss, even
   when the criticism is a past-tense remark ("that was my biggest complaint"). scope_change only when the earlier output is not
   criticised.
2. Restated instruction vs still-broken. If the message cites an earlier operator statement ("I said", "as I told you", "again")
   that the agent is now contradicting, label instruction_ignored. Plain "did not work / still broken" without such a citation is
   quality_miss.
3. Replies to agent questions. A reply that only supplies requested information (including `<send_user_message_question_reply>`
   answers, also a negative test result) is approval_or_info. It is quality_miss only if the agent had asserted that the work
   succeeded and the reply reports the deliverable failing.
4. Challenged limitation. If the agent said it cannot do something and the operator says it should be able to: false_inability.
   If the operator disputes a factual diagnosis or assumption ("it IS running", "the services already have X"): wrong_claim. A
   neutral question about why a limitation exists ("what exactly is blocking?") is approval_or_info.
5. Question carrying a design. A question that also states a different design or new requirement is scope_change. A bare
   clarifying question is approval_or_info.
6. First-message investigations. With no preceding agent text, "investigate", "report back", "plan", "research" requests are
   scope_change; pure "explain how X works" / "tell me" questions are approval_or_info.

## Changelog
- 2026-10-30 adjudication of the 6 hold-out disagreements (original -> second labeller) under the rules above; only rows a rule
  decides were edited, no other relabelling:
  - `d412d717488f` scope_change -> **wrong_claim** (rule 4: disputes the agent's assumption about health endpoints; the correction
    is labelled by the tie rule, the "investigate and report" request does not outweigh it).
  - `e94c44ae5d0a` approval_or_info -> **scope_change** (rule 5: asks what the 120 s cap means but proposes a different design).
  - `d6c748154874` quality_miss -> **approval_or_info** (rule 3: a reply to the agent's device-confirmation question; the agent had
    only said the server accepted the send and that it still needed confirmation, not that delivery succeeded).
  - `860ad178eccc` false_inability kept (rule 4: the agent stated the limitation, the operator says the team should be updatable).
  - `1d095d1717fd` quality_miss kept (rule 1: criticises the other devices as amateur and asks for better models).
  - `87f344dfb011` instruction_ignored kept (rule 2: cites "I said ... overcomplicated" and the agent is adding complications).
  Net hold-out effect: quality_miss -1, wrong_claim +1; scope_change and approval_or_info unchanged. `validation-random.jsonl` was
  labelled before the rules and is not relabelled.

## Source rules
- operator: typed by the human. Claude: `type=user`, not isMeta/isCompactSummary/sidechain/tool_result, system-reminders
  stripped, injected prefixes dropped. Codex: `response_item` message role=user with `content_item_kinds` starting
  `user.text` (incl. `<send_user_message_question_reply>` answers, kept as operator).
- teammate (50 sampled of 140 found): Codex user-role messages that are not the operator: (a) every user.text in rollouts
  whose first session_meta has `thread_source=subagent` (parent task/assignment); (b) dense telegraphic reports
  (glued tokens, `HH:MMZ` prefixes, almost no first/second-person words) that arrive < 1 s after the preceding
  `task_complete` with no `provenance.type=user`, in sessions with >= 5 such messages (burst). No label category.
- injected: programmatic prompts typed into a session by tooling (supervisor/recovery agent templates); 2 records.
  Codex `goal.internal_context`, `agents_md.instructions`, `environments.environment_context` blocks (~500) were dropped.

## Sampling and split
`sampled_by`: random (248 unbiased draws from the 1,000 non-trivial operator messages, seed 7), targeted (190, drawn
from the remainder by broad lexical cues, so over-represent friction), targeted_rare (61, cue-mined to fill rare
categories), trivial (20 bare approvals), teammate_all. Report recall on `random` rows only; targeted rows inflate
friction prevalence. Split: ~30% hold-out stratified per category, grouped by identical 80-char text prefix so
fork-duplicated messages never straddle the split (hash order, deterministic). Hold-out: 171 rows.

## Counts (operator)
approval_or_info 258, scope_change 91, quality_miss 80, false_inability 26, early_stop 19, instruction_ignored 17,
too_dense 15, wrong_claim 13; teammate 50; injected 2. Low-confidence 92 (second pass was done on all of them).

## Caveats
Single labeller (the building agent), no inter-annotator check; labels are from text + previous agent turn only.
Claude context is the preceding assistant text, not tool activity. Redaction: key/token/JWT/bearer/40+ hex/0x-address
patterns replaced; paths and screenshot upload paths are left intact. Sessions are dominated by a handful of long Codex
and Claude conversations; hold-out is not session-disjoint.

## Corpus expansion (2026-10-30, `sampled_by=targeted-expansion`, 428 rows)
Mined from ALL Codex rollouts (`~/.vrooli/state/vrooli/web-console/sessions/codex/**`, `~/.codex/sessions/**`, 2025-09 ..
2026-10) and every `~/.claude/projects/*` dir; sessions already in the corpus were excluded, so new rows come from
272 new sessions (Claude contributed none: all its transcripts are the October sessions already sampled).
Sub-agent threads, injected/template prompts (same 60-char prefix >= 4 times) and replayed forks (same text) were dropped.
Candidates came from per-category lexical cues, then every row was labelled by reading the message and the preceding
agent text; cue matches that were not friction were kept as hard negatives. 80 further messages were drawn at random
from the same new sessions as unbiased negatives (rationale prefixed `[random-negative]`; 9 of them turned out to be
friction and are labelled as such). `confidence=low` on 125 of the new rows (no second labeller).
Split for new rows is SESSION-disjoint (greedy by session until every category reached >= 30% and hold-out >= 15);
no new session appears in both splits. Existing rows keep their original (non session-disjoint) split.
Do not compute recall/prevalence on the new rows: they are cue-selected (except `[random-negative]`).

Counts after expansion (operator; hold-out in brackets): early_stop 57 (18), false_inability 93 (29), wrong_claim 55 (17),
quality_miss 117 (35), too_dense 59 (17), instruction_ignored 47 (16), scope_change 129 (39), approval_or_info 390 (117);
teammate 50; injected 2. Hold-out total 304. Bare "continue/keep going" rows are labelled early_stop only when the
agent had just offered to stop, summarised remaining work or been told it stopped; otherwise approval_or_info.

## Validation set (`validation-random.jsonl`) and agreement (`agreement.jsonl`), 2026-10-30
Independent, untuned set: do NOT use for training, cue mining or prompt tuning; report precision/recall on it only.
- Pool: every operator user-text message in sessions that contain no row of `operator-messages.jsonl` (by session id, and by
  first-200-char text). Codex rollouts from both roots (5,452 files, realpath-deduped), Claude `~/.claude/projects/*`.
  Dropped: sub-agent threads (313), injected/non-`user.text` kinds, session bursts (>=5 messages < 1 s after `task_complete`
  without provenance; 410 sessions), 60-char-prefix templates (>= 4 occurrences; 4,903 messages), duplicate timestamp+text.
  Result 9,008 messages / 2,900 sessions (Claude: only 6 messages, none drawn).
- Draw: pool sorted by (runner, session, timestamp, text), `random.Random(20261030).shuffle`, first 320, then 4 fork replays
  (same text AND same preceding agent text as an earlier draw) removed -> 316 rows (315 operator, 1 teammate that slipped
  through the burst rule: row with rationale "teammate", category null). No cue filtering, no stratification.
- Labels: single labeller (second labeller of the corpus, working from README rules only), `confidence=low` on 86 rows.
  Rows whose text matches an earlier row in another session were kept when the preceding agent text differed.
- Counts (operator): approval_or_info 158, scope_change 90, quality_miss 48, wrong_claim 9, early_stop 3, false_inability 3,
  instruction_ignored 2, too_dense 2. Friction prevalence 67/315 = 21.3%. early_stop, false_inability, instruction_ignored and
  too_dense have < 8 gold rows and were deliberately NOT topped up; per-category recall for them needs a larger random draw.
- Agreement: `agreement.jsonl` = 60 hold-out operator rows chosen by id only (`random.Random(4242).shuffle` of the sorted ids,
  first 60), labelled blind. Observed agreement 54/60 = 0.90, Cohen kappa 0.870; friction-vs-not (first six categories vs
  scope_change/approval_or_info) agreement 0.933, kappa 0.864. Hold-out contains cue-selected rows (45% friction here), so
  kappa is not a prevalence-matched estimate. Details and boundary proposals in `p4-w4-interventions.md` "Validation sample".

## Operator-signal corpus (`signal-turns.jsonl`), 2026-10-30
Separate from the intervention categories above: five broad labels on operator messages, judged with the agent's preceding
text (`reported_failure`, `correction`, `frustration`, `satisfaction`, `neutral`; priority in that order when several apply;
definitions in `docs/reference/operational-stats.md` "Operator signals"). Used by `signal_benchmark_test.go`.
- Pool: Codex rollouts of both roots (realpath-deduped) whose session id is in neither `operator-messages.jsonl` nor
  `validation-random.jsonl` (fresh sessions); user.text only, sub-agent/guardian threads, injected/`<...>` blocks and provenance-tagged
  messages dropped; 60-char-prefix templates (>= 4 occurrences) and fork replays removed; a preceding agent text is required.
  5,847 messages / 2,066 sessions. Draw: `random.Random(20261031)` shuffle, at most 2 messages per session, first 270 (244 sessions).
- Split: by session (sha1 of the session id, `% 100 < 50` -> holdout), so no session is in both: 133 train / 137 holdout.
- Fields: `label` (first labeller, the building agent, from message + preceding agent text), `label_b` (independent second
  labeller: a fresh read-only `claude -p --model sonnet` run given only the label definitions and the blind messages; it saw no
  other label). `text` <= 2000 chars, `preceding_agent_text` <= 1500 chars. Messages are not redacted beyond what the rollouts
  hold: do not publish this file.
- Agreement on all 270: 254/270 = 0.941, Cohen kappa 0.894 (explicit-negative vs rest, i.e. reported_failure+correction: kappa 0.903).
  Disagreements are not adjudicated; `label` is the gold used for scoring.
- Counts (gold): neutral 172, reported_failure 45, correction 32, satisfaction 18, frustration 3. Frustration is too rare
  in this operator's messages for any accuracy claim.
- Tune on train only. The hold-out is scored once per model. Do not add rows after scoring and re-score.
