# Validation targets

Tiers per `large-effort-orchestration` §5 (D22). The orchestrator changes a tier
on new evidence and never lets an unavailable target block the goal.

| Target | Tier | How |
|---|---|---|
| Linux x64 (this host) | required | Journeys against the local fixture site, focused package checks, scoped Test Genie phases |
| macOS arm64 (Mac mini "minimouse") | occasional | Through the Vrooli bridge, only when the bridge works; log friction in WORKAROUNDS.md |
| macOS x64 | unavailable | No machine connected |
| Windows x64 | unavailable | No machine connected; Wine and simulation do not count |

Latest target evidence (2026-09-30): Linux x64 passes the 5 journeys
(J01, J02, J07, J08, J23) against the `bas-goal` shadow.

## Running BAS during epochs (D18)

About 10 fleet scenarios call the live BAS (workflow-health behind every Test
Genie playbook run, ui-health, chart-generator, git-control-tower, and others).
Use the goal's one shadow engagement (`bas-goal`, started 2026-09-29) to rebuild
and run changed BAS code (`vrooli scenario ... --instance shadow`):
`git-control-tower baseline start --scenario browser-automation-studio --mode shadow`
(`status`, `promote` at a stable point, `abandon`). `promote` ends the
engagement: right after a promote, run the same `baseline start` command again so
the goal keeps one shadow. If the shadow is broken, fall back to restarting the
live BAS in a quiet window: first confirm `agent-manager run list --status running`
and `test-genie runs list` show no in-flight fleet validation. Record the fallback
in WORKAROUNDS.md.

## Daily qualification

Once a day the orchestrator runs the journey command from
[TESTING.md](../TESTING.md) against the `bas-goal` shadow, keeping the JSON output
at `~/.vrooli/evidence/bas-goal/journeys-<date>.json` (outside the repo) and recording the pass count in the
active epoch's log. A failure becomes a Qualification findings item in QUEUE.md,
never a reopened epoch. Targets that need soaks, native machines or external
accounts stay unverified and never block an epoch.

## Quality bands

Product targets the journeys and daily program check where a sensor exists;
the rest stay unverified until one does (the retired contract is archived in
`~/.vrooli/plan-artifacts/epoch-based-delivery-orchestration-and-bas-rehabilitation/evidence/archive/bas-rehabilitation/retired-docs-2026-09-29.tar.gz`).

- Preservation: all 24 journeys (`requirements/08-rehabilitation`) pass their positive, negative and recovery cases on applicable targets.
- Interactive feedback: local input-to-paint p50 ≤50 ms, p95 ≤100 ms, p99 ≤200 ms; remote p95 ≤200 ms.
- Motion: a 30 FPS fixture renders ≥30 FPS for 5 minutes with p95 frame age ≤100 ms; slow readers stay bounded.
- Readiness: warm usable tab p95 ≤1 s; cold usable browser p95 ≤5 s.
- Capture: warm screenshot plus computed snapshot p95 ≤2 s at exact viewport and DPR.
- Passive fidelity: zero lost or duplicated acknowledged observations, including crash/reconnect.
- Profile durability: cookies/localStorage/IndexedDB survive close/reopen and restart; no cross-profile state; unacknowledged-checkpoint window ≤5 s.
- Cancellation: new input stops ≤1 s, cleanup ≤5 s, session recovery ≤10 s; uncertain effects are reconciled before replay.
- Resources: idle API+driver ≤300 MiB PSS and <2% of one core; one fixture browser plus shell ≤1 GiB PSS.
- Evidence: every required artifact is attributable and hash-verifiable, or the execution is explicitly failed/degraded.
- Unverified until a sensor exists: known-flow reliability (≥99% first-attempt success over ≥1000 runs), 8-hour soak (≤50 MiB/hour growth), desktop portability, agent usefulness.
