# BAS status

## Scoreboard

| surface | target | state | owner |
| --- | --- | --- | --- |
| Workflow codec | RD-WF | accepted | - |
| Recording timeline | RD-REC | accepted | - |
| Export replay | RD-EXP | rough | e8c322ca-72e7-434c-9d37-1d2cf853961f |

## Current: 253452 @ 2026-10-07

## Active

- unit: E26  status: acceptance review (BAS-FB-064 gate amendment)  workers: e8c322ca-72e7-434c-9d37-1d2cf853961f

## Needs operator

(none; the operator answered BAS-FB-063 on 2026-10-06)

## Handoff

- 2026-10-07 operator: act on BAS-FB-064 now, before anything else.
  1. Accept E26 under the FB-064 gate amendment. Generated `ui/dist` identity is
     not a gate.
  2. Rewrite the scoreboard for the quality destination.
  3. Plan and admit the first journey slice (journeys 12 → 24). Spawn its worker.
  4. Keep this section short and replace it each check-in.

## Next

Each item is a redesign slice (`large-effort-orchestration` §2.1).

1. **RD-WF — one workflow model and pipeline (E23 accepted).** Remeasure
   those owners after E23 and admit ordered follow-on tranches only when an
   epoch-scale replacement and exact deletion list are evidenced.
2. **RD-REC — recording UI on one owner.** Return if a wider caller map finds a
   substantial owner overlap.
3. **RD-EXP — export on the replay spec.** E26 moves standalone HTML playback
   to the existing ReplaySpec-driven UI renderer.
4. **RD-WSUI and RD-AI** — plan after RD-WF from the same kind of owner map.
5. **DESK — desktop portability** (D22). Run it when a native machine is
   available.

## Waiting on others

These never hold acceptance or other slices.

- **E22-R1** — intended-player review of capture. Returns when QA bug
  `knw-1791087389154714359` is fixed.
- **DET-SIGNIN-R1** — real sign-ins with persistent profiles. Returns when the
  operator supplies dedicated, authorized test-only sites and accounts.

## Workarounds

- 2026-09-29 · Workspace Sandbox `networkMode: none` cannot reach managed BAS,
  Test Genie or a fixture site · epoch workers run on profiles with
  `SANDBOX_MODE_OFF` in the shared tree (D5) · retry when Workspace Sandbox
  offers loopback-only networking with in-tree writes · open.
- 2026-09-29 · `agent-manager run message` cannot reach codec-pipe runs ·
  steer through the epoch file's Directives section · retry when workers run in
  interactive mode · resolved 2026-09-30: workers run on the interactive substrate.
- 2026-09-29 · `git-control-tower baseline check` reports diff anchor
  `engagement-bas-goal` not found · use `baseline status` for liveness · retry
  after the anchor capture is repaired · open.
- 2026-10-06 · `agent-manager run identity --json` failed in the resumed
  timer-wake shell because no `VROOLI_AGENT_IDENTITY_TOKEN` was set ·
  `agent-manager run list --status running --json` recovered the exact active
  parent · retry only when a resumed shell receives Agent Manager identity
  context · open.
