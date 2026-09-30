# Workarounds

Broken or unsuitable shared infrastructure and how this goal routes around it
(`large-effort-orchestration` §5). The supervisor owns repair. Format: date ·
what failed · fallback in use · retry condition · status.

- 2026-09-29 · Workspace Sandbox `networkMode: none` cannot reach managed BAS,
  Test Genie or a fixture site; `localhost` mode is unrestricted network ·
  epoch workers and the orchestrator run on profiles with `SANDBOX_MODE_OFF` in
  the shared tree (D5) · retry when Workspace Sandbox offers loopback-only
  networking with in-tree writes · open.
- 2026-09-29 · `agent-manager run message` cannot reach codec-pipe runs (no
  mid-turn input channel) · steer through the epoch file's Directives section ·
  retry when workers run in interactive mode · resolved 2026-09-30: workers run
  on the interactive substrate and `run continue --message` reaches them.
- 2026-09-29 · BAS `programs` Test Genie phase failed once (run
  20260929-161405-4916f34d, fleet codes) and passed on the next run
  (20260929-161433-b9a758c3) · none needed · watch for recurrence · resolved.
- 2026-09-29 · Shadow engagement `bas-goal` started (live BAS kept serving on
  its port; shadow healthy on its own ports), but shadow data population was
  skipped: `data-backup-manager safety register-targets --scenario
  browser-automation-studio` exited 1. The shadow database starts empty ·
  journeys seed their own fixture data; do not rely on live user data in the
  shadow · retry when data-backup-manager registers BAS targets · open.
- 2026-09-29 · `git-control-tower baseline check --scenario
  browser-automation-studio --name bas-goal` reports diff anchor
  `engagement-bas-goal` not found · use `baseline status` for liveness and the
  inventory script for net lines; skip the anchor diff · retry after the anchor
  capture is repaired · open. The anchor capture is a full Test Genie suite
  (run 20260929-162345-179f1949) that holds the BAS test lock while it runs;
  scoped phases wait for it.
- 2026-09-29 · `prompt-manager team message-send effort-supervision
  effort-supervisor --from=browser-automation-studio-orchestrator` rejected the
  sender as not a member of the destination team · retain the complete step-back
  evidence and return condition in E0.md and QUEUE.md for the supervisor's
  workspace read · retry when Prompt Manager provides a supported cross-team
  sender route · open. Route in use since 2026-09-30: record the event in this
  goal home, then `prompt-manager team heartbeat-trigger effort-supervision
  effort-supervisor`; the supervisor reads the goal home on wake.
