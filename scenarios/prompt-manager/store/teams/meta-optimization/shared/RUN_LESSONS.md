# Run Lessons

Durable lessons extracted from agent-manager runs by `run-introspector`. One run per heartbeat. Append-only except for status column updates.

## Schema

| Date | Run ID | Agent / Task | Triage Tier | Lesson | Implicated | Handoff | Measurement | Status |

---

## Lessons

### 2026-04-23 · `60116710-f77d-4c33-8058-2bd90c475289` · `agent-manager-investigation` · errored

**Lesson.** The triage ladder's tier-1 ("errored") signal is contaminated by a known false-positive class: `exit_code=429` can be emitted on a **clean** run whose final assistant message merely *discusses* rate limiting as a topic. The substring matcher `detectRateLimit` in `claude_code.go:1518-1559` fires on any assistant text containing "rate limit" regardless of `IsError`, forcing `Success=false, ExitCode=429`. Investigating these as genuine failures wastes a heartbeat and draws the wrong lesson.

**What happened.** Run `60116710` is itself an investigation run that completed successfully, produced a thorough report, and then — because that report contains the phrase "rate limit" dozens of times — got misclassified as FAILED/429. The report itself diagnoses the bug at `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go:1528-1531` and lists `e08357a4` as a sibling false-positive (the assistant discussed a "per-type token-bucket rate limiter" feature and `TestLog_PerTypeRateLimit`). Recent-failed-runs scan confirms the pattern: 2 of 22 failed runs have `exit_code=429` + substantive completion text, not rate-limit banners.

**Implicated.** `scenarios/prompt-manager/store/teams/meta-optimization/members/run-introspector/HEARTBEAT.md` — the triage ladder in "Required Loop" step 3 picks tier-1 errored runs without a verification step for false-positive exit codes. Run-introspector will pick these runs first every time they surface.

**Proposed change.** Add a **tier-1 verification gate** to run-introspector's HEARTBEAT.md: before accepting an errored run for investigation, if `exit_code == 429` AND `error_msg` looks like substantive task output (multi-paragraph, markdown sections, "Summary" / "Classification" / "Report" headings) rather than a terse rate-limit banner, re-classify as tier-5 (random-success-with-misclassification) and note the false-positive. Do **not** modify `detectRateLimit` itself — that's a scenario-qa concern against agent-manager code; our lane is only the introspection workflow.

**Handoff.** `team-agent-optimizer` — edit `HEARTBEAT.md` step 3 to add the verification gate. Separately, the agent-manager code bug (E1 from run `60116710`'s report) is a scenario-qa concern; not a meta-optimization capability-gap since agent-manager already exists and the bug is documented in the run's own investigation artifact.

**Measurement plan.**
- **Baseline:** 2/22 (~9%) of FAILED runs in the current list carry exit_code=429 + substantive completion text (ids `60116710`, `e08357a4`).
- **Post-change:** after HEARTBEAT.md adds the gate, future run-introspector heartbeats should skip these false positives at tier-1. Check by grepping RUN_LESSONS.md — no future lesson should investigate a 429-completion-text run as tier-1; they should be reclassified.
- **Secondary:** when the underlying `detectRateLimit` bug is fixed by scenario-qa, the 9% rate drops to 0 and the gate becomes a no-op (still correct, no removal needed).
- **Revisit:** 7 heartbeats from today (2026-04-30) — check if gate is in place and whether false-positive count has moved.

**Status.** pending (awaits team-agent-optimizer pickup).

---

### 2026-04-24 · `564191ef-e1b4-4328-b21c-c83999676123` · `swarm-manager:initiative:rev-trigger:review:round-001` · slow

**Lesson.** The triage ladder's tier-3 ("slow") signal is contaminated by **approval-blocked** runs. For runs with `resolved_config.requires_approval=true`, `ended_at` is set when the operator approves the run, not when the agent finished work — so wall-clock (`ended_at - started_at`) reflects human latency, not agent latency. The right "slow" signal is **work duration** ≈ `last_heartbeat - started_at`. Without this distinction, run-introspector's tier-3 picks would be dominated by approval-queue artifacts whenever the operator batches approvals.

**What happened.** In the window since 2026-04-23, the top 25 longest runs by wall-clock are all `swarm-manager:initiative:rev-trigger:review:round-001`, each with wall-clock 70k–82k seconds (~19–22h). Inspection of `564191ef` (the longest at 81,648s) shows: `started_at=22:29:09`, `last_heartbeat=22:29:24` (15s of agent work), `ended_at=21:09:57` next day (= `approved_at`), `summary.turns_used=1`, `summary.tokens_used=27523`, `summary.cost_estimate=$0.09`. Cross-check with `86f4378d` (same tag, 13s wall-clock): `started=21:05:01`, `last_heartbeat=21:05:14`, `ended=21:05:14` (operator approved within 2ms), 1 turn / 27,470 tokens. Identical agent work; wall-clock spread is ~6,000× because approval landed at different times. Mass-end pattern at `2026-04-24T21:09:5X` (25 runs ending within ~7 seconds of each other from start times spanning 22h) confirms the operator batch-cleared the approval queue in one sweep.

**Implicated.** `scenarios/prompt-manager/store/teams/meta-optimization/members/run-introspector/HEARTBEAT.md` — "Reasoning Framework" tier-3 says: *"Slow — run exceeded expected tokens or duration by > 50%"*. "Duration" is unqualified; the natural reading (and what triage used this heartbeat) is wall-clock, which is contaminated by approval lag whenever `requires_approval=true`.

**Proposed change.** Add a **tier-3 work-duration clarification** to HEARTBEAT.md: "duration" means `last_heartbeat - started_at` (or equivalent agent-side span), not `ended_at - started_at`. When `resolved_config.requires_approval=true`, wall-clock includes operator latency and is not a valid slow signal — explicitly exclude. Optionally also exclude tier-3 picks where `summary.turns_used <= 1` AND `summary.cost_estimate < $0.20` (a 1-turn $0.09 run is structurally not slow, regardless of wall-clock).

**Handoff.** `team-agent-optimizer` — edit run-introspector's `HEARTBEAT.md` "Reasoning Framework" tier-3 definition (Slow). The underlying `ended_at` semantics in agent-manager are correct (it records when the run ended for the system, including approval); changing them would break audit trails. The fix belongs in run-introspector's interpretation, not in agent-manager.

**Measurement plan.**
- **Baseline.** Of 98 successful runs in this heartbeat's window, the 25 longest by wall-clock all share the `swarm-manager:initiative:rev-trigger:review:round-001` tag, all 1-turn / ~27k tokens / ~$0.09, with `requires_approval=true` and mass-completion at `21:09:5X` UTC. Median wall-clock 64,462s; median work-duration (per spot checks) ~15s — a ~4,300× discrepancy.
- **Post-change.** Future heartbeats compute work-duration directly. The next tier-3 pick should have `last_heartbeat - started_at > 50%` over expected, not `ended_at - started_at`. Approval-blocked clusters fall out of slow-triage entirely.
- **Revisit.** 7 heartbeats (2026-05-01) — grep RUN_LESSONS.md for any tier-3 lesson opened on a `requires_approval=true` run with `turns_used <= 1`. Expected count: 0.
- **Secondary.** This is the second tier-contamination lesson in two heartbeats (yesterday: tier-1 false-positives from `detectRateLimit`; today: tier-3 false-positives from approval lag). If a third surfaces, the contrarian should consider a `framework-update` formalizing tier-signal-contamination as a standing failure mode.

**Status.** pending (awaits team-agent-optimizer pickup).

---

### 2026-04-25 · `cab1c399-9a3e-4b12-99e4-6a21dcb69ddb` · `swarm-manager:backlog:execute:prompt-manager-decision-deferral-primitive:process` · errored

**Lesson.** The triage ladder's tier-1 ("errored") signal is contaminated by a third class: **transient upstream-API failures**. Runs that fail purely because Anthropic returned `5xx Overloaded` carry no meta-layer signal — there is no skill, agent prompt, or team-config edit that would have prevented the failure. Investigating these as tier-1 wastes a heartbeat. This is the third tier-contamination class in three heartbeats (after tier-1 `detectRateLimit` false-positives and tier-3 approval-lag artifacts).

**What happened.** Run `cab1c399` (sandboxed, opus, requires_approval=true, fallback_runner_types=[CLAUDE_CODE, CODEX]) ran for 203,961 ms (~3.4 min) on a 1-turn execution before claude-code terminated with `is_error=true (subtype=success, turns=1)`. The assistant message and error event both contain the literal payload: *"API Error: 529 Overloaded. This is a server-side issue, usually temporary — try again in a moment. If it persists, check status.claude.com."* Run was marked `RUN_STATUS_FAILED`. The declared codex fallback never engaged (the runner's terminate-on-error path does not honor `fallback_runner_types` when claude-code returns `is_error=true` with `subtype=success`). The single FAILED run in this 78-run window; no other tier-1 candidates.

**Implicated.**
- **Meta-layer (in lane):** `scenarios/prompt-manager/store/teams/meta-optimization/members/run-introspector/HEARTBEAT.md` — "Required Loop" step 3 / "Reasoning Framework" tier-1. The current tier-1 verification gate (proposed in `dec-1777069916962818847`) only handles `exit_code=429 + substantive completion text`. It does not handle 5xx upstream-overload failures.
- **Scenario-qa lane (noted, not actioned by this lesson):** agent-manager's claude-code runner terminate-on-error path does not engage `fallback_runner_types` on transient `5xx Overloaded` responses, despite the run config declaring `[CLAUDE_CODE, CODEX]` as fallbacks. Likely callsite: the same `claude_code.go` runner that owns `detectRateLimit` (yesterday's lesson). The `subtype=success` from claude-code masks the upstream API failure so the orchestrator doesn't see it as a retryable error.

**Proposed change.**
- **Meta-layer:** Extend run-introspector's HEARTBEAT.md tier-1 gate (currently scoped to 429 false-positives) to also exclude **transient upstream-API failures**: if the run's terminal error message matches `API Error: 5\d\d (Overloaded|Internal|Bad Gateway|Service Unavailable|Gateway Timeout)` AND `summary.turns_used <= 1`, reclassify as tier-5 (random-success-with-transient-failure), record the transient failure in RUN_LESSONS.md, continue walking the ladder. The two gates can share one prose paragraph or be separate bullets — team-agent-optimizer's call.
- **Scenario-qa (separate lane, separate concern):** Note in RUN_LESSONS.md that claude-code runner's terminate-on-error path should honor `fallback_runner_types` when the underlying error is a transient 5xx from Anthropic. Not raised as a meta-optimization decision (out of lane); surfaced here for visibility.
- **Standing pattern:** With this third tier-contamination lesson, the contrarian should evaluate `framework-update` for "tier-signal-contamination" as a standing failure mode. Three classes now: (1) tier-1 detectRateLimit false-positives, (2) tier-3 approval-lag wall-clock artifacts, (3) tier-1 transient-API-failure runs. All three share the same shape: triage tier fires correctly per its literal definition but the underlying signal is environmental, not agent-behavioral. Not raising the framework update myself — that's contrarian's lane.

**Handoff.** `team-agent-optimizer` — extend the tier-1 gate edit (already pending in `dec-1777069916962818847`) to also cover 5xx transient-API failures, OR add a separate gate paragraph if cleaner. Either works; team-agent-optimizer to choose.

**Measurement plan.**
- **Baseline.** 1/78 (~1.3%) of completed runs in this window are pure-transient-5xx failures (`cab1c399`). Across multiple windows the rate is unknown but expected non-zero given Anthropic 5xx rates; this lesson's value scales with that rate.
- **Post-change.** Future heartbeats encountering a 5xx-only tier-1 run should mark it "tier-1 transient-API-failure (skipped)" rather than open a full lesson. Grep RUN_LESSONS.md 7 heartbeats from now (2026-05-02) — count of tier-1 lessons opened on `API Error: 5xx Overloaded`-only runs should be 0.
- **Standing-pattern watch.** This lesson is the third tier-contamination class. If a fourth surfaces, the contrarian's `framework-update` is overdue.
- **Secondary (out-of-lane).** When/if scenario-qa fixes the runner to engage `fallback_runner_types` on 5xx, 1-turn-5xx-failure rate drops toward 0 and the gate becomes a no-op (still correct, no removal).

**Status.** pending (awaits team-agent-optimizer pickup; coordinates with `dec-1777069916962818847`).

---

### 2026-04-27 · `cdd35a04-a353-499d-85aa-463f864b3c27` · `heartbeat-marketing-crew-brand-manager-2026-04-26T19-00-00Z` · errored

**Lesson.** The triage ladder's tier-1 ("errored") signal is contaminated by a **fourth** class: **silent runner-stall failures**. Runs whose event stream stops abruptly mid-execution, with no terminal `RUN_FAILED`/error event, no `error_message`, and a frozen `last_heartbeat`, then get reaped to `RUN_STATUS_FAILED` minutes later by timeout. The agent did not fail — the runner subsystem (claude-code stream, codex, or the orchestrator's reaper) lost the process without an emitted error. There is no skill, agent prompt, or team-config edit that would have prevented the failure. Investigating these as tier-1 yields the same null finding every time. This is the **fourth** tier-contamination class in four heartbeats.

**What happened.** In the 2026-04-25T22:48Z–2026-04-27T22:48Z window: 6 FAILED runs of 133 total. Three of the 6 cluster within a 1-hour window on 2026-04-26 (18:30 monetization-catalog-strategist, 19:00 marketing-crew-brand-manager, 19:30 marketing-crew-oss-advertiser) and share an identical shape:

| Run ID | Tag | last event elapsed | last_heartbeat elapsed | wall-clock | terminal error |
|---|---|---|---|---|---|
| `057547bf` | monetization-catalog-strategist | 7:08 | 7:00 | 23:45 | none |
| `cdd35a04` | brand-manager | 5:06 | 7:00 | 23:50 | none |
| `ffcfe70b` | oss-advertiser | 0:11 | 0:00 | 17:00 | none |

All three: started with log `runner fallback: codex -> claude-code` (codex unavailable, fell back); flipped to `RUN_PHASE_EXECUTING` normally; emitted ordinary tool calls; then events stopped without any `RUN_EVENT_TYPE_RUN_FAILED`, `STATUS=failed`, or error log; were reaped silently by the orchestrator timeout (~16–18 min later). `summary={}`, `error_message=""`, `progress_message=""`. `last_heartbeat << ended_at` by 16+ min in every case. (Picked `cdd35a04` as the representative for one-run-per-heartbeat discipline; other two corroborate the cluster-not-coincidence reading.)

This is structurally distinct from `cab1c399`'s 5xx-Overloaded lesson (run `cab1c399` had an explicit terminal error payload "API Error: 529 Overloaded"). Today's class has **no** terminal error — the runner died silently.

**Implicated.**
- **Meta-layer (in lane):** `scenarios/prompt-manager/store/teams/meta-optimization/members/run-introspector/HEARTBEAT.md` — "Required Loop" step 3 / "Reasoning Framework" tier-1. The pending tier-1 gates (`dec-1777069916962818847` for 429-substantive-text, `dec-1777157323547139809` for 5xx-API-error) both rely on inspecting the terminal error message. Silent-stall runs have no terminal error message to gate on, so they slip past both gates.
- **Scenario-qa lane (noted, not actioned by this lesson):** agent-manager's runner subsystem (likely `claude_code.go` and/or the orchestrator's run-reaper) terminates a run to `RUN_STATUS_FAILED` without emitting a `RUN_EVENT_TYPE_RUN_FAILED` event or populating `error_message`. From the introspection lane this is opaque — the only signal that the runner stalled is the gap between `last_event_timestamp` and `ended_at`. The codex-fallback log line at start in all three cases also raises an out-of-lane question whether codex unavailability correlates with downstream stalls; not investigated.
- **Standing pattern (out of lane, contrarian's call):** This is the **fourth** tier-contamination class. Per RUN_LESSONS.md 2026-04-25 lesson's standing-pattern watch: *"If a fourth surfaces, the contrarian's `framework-update` is overdue."* Pending decision queue contains no `framework-update`. Flagging here for visibility; not raising myself.

**Proposed change.**
- **Meta-layer:** Extend the run-introspector tier-1 gate (already pending in two prior decisions) to additionally exclude **silent-stall runs**: if the run is `RUN_STATUS_FAILED` AND `error_message` is empty AND no `RUN_EVENT_TYPE_RUN_FAILED` event exists AND (`ended_at - last_heartbeat > 5 min` OR `ended_at - last_event_timestamp > 5 min`), reclassify as tier-5 (silent-stall, no-meta-signal), record the stall in RUN_LESSONS.md as a one-line note with run ID + cluster, continue walking the ladder. Implementation note for team-agent-optimizer: the existing gate paragraph is now covering three sub-classes (429-FP, 5xx-explicit, silent-stall) — at this point a single short bullet list "tier-1 environmental-failure exclusions" is cleaner than three separate prose paragraphs. Implementer's call.
- **Scenario-qa (separate lane, separate concern):** Note that runs reaped by the orchestrator should emit a `RUN_EVENT_TYPE_RUN_FAILED` with `reason="silent_stall"` (or equivalent) and populate `error_message`, so triage downstream can see what happened. Without this, every introspection lane (mine and any future investigator) has to infer stalls from event-stream gaps, which is fragile. Not raised as a meta-optimization decision (out of lane).
- **Standing pattern:** Fourth class observed. Contrarian's `framework-update` for "tier-signal-contamination as a standing failure mode" is now formally overdue per the prior lesson's own watch criterion. Not raising myself — that's contrarian's lane.

**Handoff.** `team-agent-optimizer` — extend the tier-1 gate (already pending in `dec-1777069916962818847` and `dec-1777157323547139809`) to also cover silent-stall runs. With three sub-classes now stacked under tier-1, recommend consolidating into a single "tier-1 environmental-failure exclusions" bullet list rather than three separate prose paragraphs. Implementer's call.

**Measurement plan.**
- **Baseline.** 3/6 (~50%) of FAILED runs in this 2-day window are silent-stalls; 3/133 (~2.3%) of completed runs overall. Cluster timing (within 1 hour) suggests environmental incident — base rate likely lower across longer windows, but non-zero given runner subsystem complexity.
- **Post-change.** Future heartbeats encountering a silent-stall tier-1 candidate should mark it "tier-1 silent-stall (skipped)" and continue walking. Grep RUN_LESSONS.md 7 heartbeats from now (2026-05-04) — count of tier-1 lessons opened on `error_message=""` + frozen-`last_heartbeat` runs should be 0.
- **Standing-pattern watch.** Fourth class now observed. If contrarian does not raise `framework-update` within the next 3 heartbeats, consider this lesson's standing-pattern note itself stale — the prediction held; the systemic action did not follow. (Out-of-lane to escalate further; just noting.)
- **Secondary (out-of-lane).** When/if scenario-qa adds explicit `RUN_FAILED` emission for reaped stalls, the gate's silent-stall predicate becomes inspectable on `reason=` rather than inferred from event-stream gaps, but the gate logic stays the same.

**Status.** pending (awaits team-agent-optimizer pickup; coordinates with `dec-1777069916962818847` and `dec-1777157323547139809`).

---

### 2026-04-27 · `56398acb-2aec-4b2c-bfc4-eafc8dc28c3e` · `heartbeat-meta-optimization-run-introspector-2026-04-26T22-45-00Z` · errored

**Lesson.** Two distinct findings from the prior run-introspector heartbeat's own failure:

1. **Pending 5xx-gate predicate is too narrow.** `dec-1777157323547139809` proposes reclassifying tier-1 environmental failures when terminal error matches `API Error: 5xx ...` AND `summary.turns_used <= 1`. The turn-count predicate is wrong — multi-turn legitimate work that dies on a single transient API 5xx is still environmentally-failed, not agent-failure. Run `13ac79cb` (swarm-manager research, 34 internal turns then `overloaded_error`) is a 34-turn 5xx that the gate would let through. Drop the `turns_used <= 1` predicate; the 5xx-pattern alone is sufficient.

2. **Run-introspector heartbeat is bumping the 50-turn ceiling.** Run `56398acb` ran 51 turns / 7m36s before claude-code terminated with `subtype=error_max_turns`. The agent did legitimate investigation work (read prior decisions, listed runs, fetched events on 4 runs, drafted the 2026-04-26 silent-stall lesson, then began superseding the 5xx gate) and was caught mid-supersession. Last assistant message at sequence 127: *"Picking `13ac79cb` ... confirms my pending 5xx gate's `turns_used <= 1` predicate is too narrow. I'll supersede the prior decision with a broader take."* Sequence 128 was a `decision-update --help / decision-add --help` Bash call (CLI discovery friction — `prompt-manager team decision-update --help` returns generic team-help, not subcommand help). One step later: max_turns. The heartbeat as written investigates ≥1 run + writes ≥1 lesson + writes ≥1 supersession + does CLI navigation; on a heartbeat that produces two interlocking findings (a new lesson AND a supersession of a prior decision) the 50-turn cap is tight.

**What happened.** Tier-1 errored. Run `56398acb` (sandboxed, opus, requires_approval=APPROVAL_STATE_NONE, runner=claude-code, max_turns=50) terminated `is_error=true subtype=error_max_turns turns=51 duration_ms=454664`. Final phase `RUN_PHASE_COLLECTING_RESULTS`, `summary={}`, `error_message=""` (the max-turns error lives in `RUN_EVENT_TYPE_ERROR` event, not on the run record itself). The agent had — within those 51 turns — already drafted the entire 2026-04-26 silent-stall lesson via Edit calls into RUN_LESSONS.md (uncommitted; visible in `git diff HEAD`); the work was preserved by sandbox file-edit semantics, not by run completion. The supersession of `dec-1777157323547139809` did not land — that work is what this heartbeat is now picking up.

**Implicated.**
- **Meta-layer (in lane, primary):** `dec-1777157323547139809` — predicate `summary.turns_used <= 1` is the bug; drop it. The 5xx error pattern alone is the right signal.
- **Meta-layer (in lane, secondary):** `scenarios/prompt-manager/store/teams/meta-optimization/members/run-introspector/HEARTBEAT.md` — the "Required Loop" is dense (10 steps + supersession + multi-decision raises + knowledge snapshot). When a heartbeat produces a new lesson AND must supersede a prior decision, 50 turns is tight. Two tightening options for team-agent-optimizer to weigh: (a) trim the loop's CLI ceremony (e.g., let supersession be deferred to the next heartbeat when same-day work crowds it out — explicit "carry-over" rather than "must complete in this heartbeat"); (b) raise max_turns on this heartbeat's profile from 50 → 75. (a) is in-lane (heartbeat prose); (b) is the agent-manager profile config (out-of-lane).
- **Scenario-qa lane (noted, not actioned):** `prompt-manager team decision-update --help` returns generic `team` subcommand help, not the `decision-update` subcommand's flags. Documented in earlier events of this run as a 1-turn deadweight call. This is a prompt-manager CLI ergonomics issue.
- **Concurrent-heartbeat firing (out of lane, observed):** While drafting this lesson, two run-introspector heartbeats fired simultaneously at 2026-04-27T22:45:00Z (runs `096b1dee` and `937bcb50`, both `RUN_STATUS_RUNNING`, same tag, started 55ms apart). The earlier one wrote the 2026-04-26 silent-stall lesson uncommitted; this lesson is being written by the second. Duplicate-firing is a real coordination hazard (file races on RUN_LESSONS.md, knowledge-entry topic-collision, decision-add races). Out of lane — that's an agent-manager scheduler concern; surfacing here so contrarian / scenario-qa can pick up if pattern recurs.

**Proposed change (this heartbeat is doing it).**
- **Supersede `dec-1777157323547139809`** with a new `run-lesson` decision: drop `turns_used <= 1` from the 5xx-environmental-failure predicate. Keep the rest (terminal-error pattern match, reclassify-as-tier-5, continue walking the ladder).
- Optional secondary handoff to **team-agent-optimizer** for HEARTBEAT.md tightening per (a) above; not raising a separate decision this heartbeat (cap; supersession is the priority).

**Handoff.** This heartbeat's supersession decision **replaces** `dec-1777157323547139809`. team-agent-optimizer should pick up the broader 5xx predicate when implementing the consolidated tier-1 environmental-failure exclusions list (alongside the silent-stall extension from the 2026-04-27 silent-stall lesson above). All three exclusions (429-FP, 5xx-pattern, silent-stall) now collapse cleanly into one bullet list per the implementation note in the silent-stall lesson.

**Measurement plan.**
- **Baseline.** Run `13ac79cb` (34-turn 5xx) is a concrete miss for the original predicate; in this 2-day window the original gate would let it through and the broader gate would catch it (1 saved tier-1 misclassification this window).
- **Post-change.** 7 heartbeats from now (2026-05-04), grep RUN_LESSONS.md for any tier-1 lesson opened on a run whose terminal error matches `5\d\d.*Overloaded|overloaded_error` regardless of turn count — expected 0.
- **Heartbeat turn-budget watch.** Monitor whether subsequent run-introspector heartbeats hit `error_max_turns`. If another max-turns failure appears within 7 heartbeats, escalate the loop-tightening or max-turns-raise option from secondary to primary.

**Status.** pending (awaits team-agent-optimizer pickup; supersedes `dec-1777157323547139809`).
