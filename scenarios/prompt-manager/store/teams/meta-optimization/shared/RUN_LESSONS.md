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
