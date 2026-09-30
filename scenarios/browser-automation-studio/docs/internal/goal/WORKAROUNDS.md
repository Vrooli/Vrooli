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
  sender route · resolved 2026-09-30: a delivery member may now trigger a
  supervision member's heartbeat (`large-effort-orchestration` §2).
- 2026-09-30 · Orchestrator runs 37ff0201 and 0fccc932 failed with Codex
  "thread already has an active writer": `vrooli scenario test` auto-parked the
  orchestrator, the Test Genie result was already available, and Agent Manager
  resumed the conversation before the parked turn's process was stopped · the
  liveness heartbeat relaunched the orchestrator · resolved 2026-09-30: WakeRun
  now waits for the parked turn to end (`TestWakeRun_WaitsForParkedTurnToEnd`);
  Agent Manager restarted on the fix.
- 2026-09-30 · Gremlins is approved for BAS but not installed; `command -v gremlins` found no binary and SDA's `tools/gremlins` install route has no matching scenario/package surface · E2 proceeds with Go coverage baselines and source consolidation while its worker records the exact mutation-tool request; orchestrator will use an available governed install route before E2 acceptance · retry when SDA exposes an owned standalone-tool target or the approved binary is provisioned · open.
- 2026-09-30 · `prompt-manager team heartbeat-trigger effort-supervision effort-supervisor` failed with attribution `team_mismatch` (header `browser-automation-studio-delivery`, URL `effort-supervision`), including the E6 `goal_blocked` scope-mapping event · do not retry the rejected cross-team route; the supervisor can read the step-back from this goal home on its regular heartbeat · retry when Prompt Manager supports cross-team heartbeat attribution · open.
- 2026-09-30 · `agent-manager run stop eb50fb2c-0c96-4ca1-9b7c-c98b723a6acf` rejected the orchestrator identity because lifecycle stop requires operator context; the child had already written its D1 acknowledgement and step-back handoff · leave the exact child identity and handoff in E2.md/QUEUE.md, do not retry the stop command, and park this coordinator for 12h while the external Gremlins route is repaired · retry when an operator lifecycle route is available · open.
- 2026-09-30 · BAS workflow Test Genie run `20260930-142521-9d49fe44` failed after 901 seconds: its workflow-health child `c35be7b1-061b-4a0e-a01d-36e8e3ff545a` stayed queued until deadline; workflow-health itself is healthy, while Test Genie reported host swap pressure and serial fallback · rely on focused Go/UI checks, the passing Linux shadow journeys, and direct V2 validation for this pass; no live BAS restart · retry the workflow phase when the provider queue and memory pressure are clear · open.
- 2026-09-30 · BAS flow-finder returned a partial result because its normalization step was unavailable during E6 target discovery · use a scoped recursive source/asset scan and record exact candidates without guessing · retry when the normalization step is healthy · open.
- 2026-09-30 · Resolved both 2026-09-30 route failures above: `heartbeat-trigger
  effort-supervision effort-supervisor` from this team's orchestrator is admitted
  (Prompt Manager allows delivery → supervision wakes), and `agent-manager run
  stop|wake|continue` on the orchestrator's own children is no longer blocked by
  the CLI (the API admits only the caller's lineage).
