### Runs in window
- Errored: 22 (total FAILED in list; this is a first-heartbeat full-history pass, no prior last-heartbeat marker)
- Retried: not separately counted (retry-loops visible in `lpbs-payment-anomaly-log-and-alerts` tag group — 4+ consecutive failures, noted for future heartbeat)
- Slow: not triaged this heartbeat (tier-1 non-empty)
- User-flagged: none observed
- Successful: 92

### Run picked this heartbeat
- Run ID: `60116710-f77d-4c33-8058-2bd90c475289`
- Agent: `agent-manager-investigation` (profile `bc6b95c5-9f22-4e28-9300-53caf15b26f5`, run_mode sandboxed)
- Triage tier: errored (tier-1), but with a twist — the "error" is itself a misclassification

### What happened
An investigation run completed successfully, produced a thorough report, and got marked FAILED/429 because its own report text contained the phrase "rate limit" as a topic of discussion. `detectRateLimit` (agent-manager runner, `claude_code.go:1518-1559`) uses an unanchored substring match, so any successful run whose final message discusses rate limiting as a subject gets force-failed with ExitCode=429. The run's own `error_msg` field contains the full self-diagnosis — it's a completed investigation report misfiled as an error.

### Implicated
- **Tier-1 triage ladder** in `scenarios/prompt-manager/store/teams/meta-optimization/members/run-introspector/HEARTBEAT.md` (Required Loop step 3) — no gate filters misclassified-success from real errored runs, so run-introspector would keep picking these false positives first.
- **Underlying bug** (out of my lane, scenario-qa): `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go:1518-1559` `detectRateLimit` substring matcher; the run's own report already prescribes the fix.

### Proposed lesson
- Add a tier-1 verification gate: if `exit_code=429` AND `error_msg` looks like substantive completion text (markdown headings like "Summary"/"Classification"/"Report", multi-paragraph), reclassify as tier-5 (misclassified success) and continue the ladder.
- Handoff to: **team-agent-optimizer** (HEARTBEAT.md edit).
- Underlying code bug deliberately not flagged as capability-gap — agent-manager exists, the bug is code quality, scenario-qa owns that.

### Measurement plan
- Baseline: 2 of 22 FAILED runs (~9%) carry exit_code=429 + substantive completion text (`60116710`, `e08357a4`).
- Post-edit: no future RUN_LESSONS.md entry should pick an exit_code=429-with-completion-text run as tier-1 errored.
- Revisit: 2026-04-30 (7 heartbeats).
- Secondary: when scenario-qa fixes `detectRateLimit`, the false-positive rate drops to 0 and the gate becomes a silent no-op (keep it — defense in depth).

### Decisions raised this heartbeat
- `dec-1776984436121140045` · `run-lesson` · tier-1 verification gate for exit_code=429 false positives → team-agent-optimizer
- (1 of ≤2 cap used; no second decision — depth over breadth, retry-loop pattern on `lpbs-payment-anomaly-log-and-alerts` logged for future heartbeat)

### Knowledge entries written
- `knw-1776984422361345123` · topic `run-lessons-2026-04-23` (first entry — no prior to supersede)

### Supersession check
- No prior pending `run-lesson` or `capability-gap` decisions raised by run-introspector. No supersessions needed.