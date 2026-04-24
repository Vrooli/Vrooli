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
