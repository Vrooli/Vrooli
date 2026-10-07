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
- 2026-09-30 · The active `bas-goal` `@shadow` instance predates E12: J23's recording start logged `frameStreaming:false`, so the observer timed out before receiving its first frame; the live BAS instance is separate and must stay running · refresh only the isolated shadow through `vrooli scenario start browser-automation-studio --instance shadow`, then rerun `pnpm test:journeys --json --outputFile=<path>` against the same `bas-goal` engagement; focused current-source API/driver/UI tests remain the fallback while refreshing · retry when the shadow lifecycle reports healthy on its own ports · resolved 2026-09-30: restarted only `browser-automation-studio@shadow` through Vrooli lifecycle; the unchanged `bas-goal` engagement then passed all 11 journeys, including J23; live BAS remained untouched.
- 2026-10-01 · Agent Manager was temporarily unavailable during E18's token/receipt query; after recovery, `run tokens` returned 716,600 weighted non-cache tokens (usage not final) and `run receipts` returned `unobserved` with no observations · preserve those exact statuses in E18 and the queue; do not infer final usage or receipt success · retry only if Agent Manager exposes new receipt observations · open.
- 2026-10-01 · `agent-manager effort board --json` returned no board data during the pre-WF1 planning check-in · use the accepted goal home and verified `agent-manager run list --status running --json` as the fallback for destination and run liveness · retry when the effort board returns a structured observation for the active effort · open.
- 2026-10-02 · The referenced BAS-OBSERVABILITY-REDACTION-01 Google Doc could not be opened with the available web reader · use the approved BAS-FB-061 redaction scope and local admission checkpoint for DIAG-REDACT; keep unavailable full-acceptance details explicit · resolved 2026-10-03: native Google Drive `get_document` returned the linked Open work briefs tab, including the historical redaction brief and a current-authority note that the owner’s 23:46 correction supersedes its 120-minute cap/parent-release wording; current local goal, queue and epoch govern · resolved.
- 2026-10-02 · `agent-manager run create` has no supported `--timeout` flag; E20 admission resolves to the profile's 43,200-second timeout · parent will stop its direct child at the recorded 15-minute window through `agent-manager run stop`, and park/wake for reconciliation · retry bounded timeout creation only if Agent Manager adds a supported option · open.
- 2026-10-03 · REC-FIX shadow refresh via `vrooli scenario start browser-automation-studio --instance shadow --timeout 1200 --json` failed while preparing replacement artifacts: `component ui: build component ui: exit status 1`; retained log `/home/matthalloran8/.vrooli/logs/browser-automation-studio@shadow.log` identifies Vite ENOTDIR resolving `ui/node_modules/@bufbuild/protobuf/dist/esm/index.js/wire` after 1879 modules transformed. The supported setup also emitted a lockfile-up-to-date pnpm install and `added 6`; no raw install was invoked and no dependency repair is authorized. `packages/proto` `make verify-committed-gen` passed, the old isolated shadow processes remain running/healthy at localhost:15372 on prior build identity (start operation status failed), and zero active BAS executions or Test Genie runs were observed · continue with current-source focused driver tests/typecheck and preserve old shadow/raw baseline evidence; do not treat old-shadow output as post-change qualification · retry supported shadow lifecycle only after the UI build failure has an identified resolution and a quiet window is confirmed · open.
- 2026-10-04 · `agent-manager run identity --json` in the resumed tool shell failed because no `VROOLI_AGENT_IDENTITY_TOKEN` was set, although the coordinator run remained active · recover the exact parent identity from `agent-manager run list --status running --json` and match the known run id, label and effort reference · retry `run identity` when the resumed shell receives Agent Manager identity context · open.
- 2026-10-06 · `agent-manager run identity --json` again failed in the resumed timer-wake shell because no `VROOLI_AGENT_IDENTITY_TOKEN` was set · `agent-manager run list --status running --json` recovered the exact active parent by matching run ID `70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb`, label and verified effort reference; stop retrying `run identity` in this shell · retry only when a resumed shell receives Agent Manager identity context · open.
