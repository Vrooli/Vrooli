# Workarounds

Broken or unsuitable shared infrastructure and how this goal routes around it
(`large-effort-orchestration` §5). The supervisor owns repair. Format: date ·
what failed · fallback in use · retry condition · status.

- 2026-09-29 · Workspace Sandbox `networkMode: none` cannot reach managed BAS,
  Test Genie or a fixture site · epoch workers run on profiles with
  `SANDBOX_MODE_OFF` in the shared tree (D5) · retry when Workspace Sandbox
  offers loopback-only networking with in-tree writes · open.
- 2026-09-29 · `agent-manager run message` cannot reach codec-pipe runs ·
  steer through the epoch file's Directives section · retry when workers run in
  interactive mode · resolved 2026-09-30: workers run on the interactive substrate.
- 2026-09-29 · `git-control-tower baseline check` reports diff anchor
  `engagement-bas-goal` not found · use `baseline status` for liveness · retry
  after the anchor capture is repaired · open. The anchor capture is a full Test
  Genie suite that holds the BAS test lock while it runs.
- 2026-09-30 · Resolved both 2026-09-30 route failures above: the orchestrator's
  heartbeat-trigger is admitted.
- 2026-10-06 · `agent-manager run identity --json` failed in the resumed timer-wake shell because no `VROOLI_AGENT_IDENTITY_TOKEN` was set · `agent-manager run list --status running --json` recovered the exact active parent · retry only when a resumed shell receives Agent Manager identity context · open.
