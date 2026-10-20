# Commissioning — 2026-10-06

The operator approved the plan on 2026-10-06 (FEEDBACK DPF-FB-001, DPF-FB-002)
and asked for full setup, with the team started later on "go". This receipt
follows `large-effort-orchestration` §6.1.

## 1. Target integrated

- Design of record: `docs/architecture/packaging-strategies.md`, registered in
  `docs/manifest.json` under architecture.
- Goal home: this folder, containing:
  - `GOAL.md` (2,035 bytes);
  - `QUEUE.md` with E1–E6, the Waiting-on-others item M1 (mobile), and the
    Forecast;
  - `TARGETS.md`, `FEEDBACK.md`, `WORKAROUNDS.md`, `INSTALLS.md`;
  - `epochs/E1.md`, admitted.
- Canonical sources linked, not restated: the scenario-to-desktop
  ARCHITECTURE, live-desktop-api, PROBLEMS and `performance-baseline.json`,
  plus the performance-health PRD and code-facts discovery.

## 2. Materials

There are no images or external reference files. The research inputs are cited
in the design §10 (web) and by `path:line` (repo). Raw research transcripts are
not required to execute.

## 3. Independent verification

A fresh reader (general-purpose subagent, no authoring conversation, read-only)
read the goal home, the design, the skill, the worker card and the team files.

- Verdict: **READY-WITH-FIXES**. No blockers; 9 should-fix items and 5 nits,
  all applied the same day:
  - `main.ts` line range corrected to 1340-1353;
  - design §3/§5 now say `packaged-launch.v1` supersedes `launch-trace.v1`;
  - `app_interactive` template mark moved to E6, with the fallback stated;
  - comparability columns `display`, `profiler_mode`, `power_state` added;
  - INSTALLS cold helper owner aligned with E1;
  - concrete mobile-ready contract rules added (availability per mark,
    `oneof` `platform_detail`, generic role taxonomy);
  - shared-tree proto verifier gate scoped;
  - admitted write paths listed in E1;
  - effort enrolled;
  - WebDriver adapter routing aligned (scenario-to-desktop first, BAS through
    FEEDBACK);
  - TARGETS command path and flags;
  - `test-genie runs list --scenario`;
  - budget key `performance.budgets.packaged.desktop`;
  - baseline sum corrected to ~7.0 s;
  - PSS fields named in the gate.
- Shared-contract case: the reader confirmed that a worker finds the governed
  proto route (`packages/proto/README.md`,
  `make generate SCENARIO=<s>`, `make verify-committed-gen`) and would not read
  E1's non-goals as a ban on the required contract work.
- Spot-checked references:
  - `api/generation/types.go:39`, `api/generation/service.go:633`,
    `api/build/handler.go:96`;
  - `runtime/health/checker.go:115-140`, `templates/vanilla/main.ts:1340-1353`;
  - code-facts `api/internal/facts/discovery.go:82-169`;
  - baseline p50 values 635 ms and 6,178 ms;
  - 124/127 Go APIs.
- Limit: the reader ran in a Claude subagent, not through the Agent Manager
  worker route. The route itself is qualified by the live BAS, Nooch and
  Planner delivery teams on the same profiles.

## 4. Provisioned, disabled

| Item | Identity | State |
|---|---|---|
| Effort enrollment | `effort:desktop-packaging-performance`, revision 1, target `goal:desktop-packaging-performance@2026-10-06`, authority `operator-instruction:desktop-packaging-2026-10-06` (local owner) | enrolled |
| Team | `desktop-packaging-delivery` (finite, delivery, leader-led, serialized ×1); operating contract in `scenarios/prompt-manager/store/teams/desktop-packaging-delivery/team.json` | **disabled** |
| Leader | `desktop-packaging-orchestrator` (agent + member, AGENT/HEARTBEAT/RESPONSIBILITIES) | active member of a disabled team |
| Heartbeat binding | `*/30 * * * *`, profile `prompt-manager/delivery-orchestrator`, finite leader `effort:desktop-packaging-performance`, keepAlive true, coordinator `skill:large-effort-orchestration` | **disabled** |
| Worker profile | `prompt-manager/delivery-epoch-worker` (`4c19aef1-36a2-4cc5-8961-42648692883b`), orchestrator profile `01ed3efc-aff6-44c7-b77c-0c548d56d1c9` | existing |

## 5. Start ("go")

Run these in order. They are also in the operator summary.

```bash
prompt-manager team heartbeat-enable desktop-packaging-delivery desktop-packaging-orchestrator \
  --schedule='*/30 * * * *' --profile=prompt-manager/delivery-orchestrator
prompt-manager team update desktop-packaging-delivery --enabled=true
prompt-manager team heartbeat-trigger desktop-packaging-delivery desktop-packaging-orchestrator
prompt-manager team heartbeat desktop-packaging-delivery desktop-packaging-orchestrator   # expect: scheduled, leader runId set
```

- Pass `--profile` explicitly. Without it, `heartbeat-enable` falls back to
  the judgment profile.
- Enabling a finite delivery team also arms the `effort-supervision` team.

## 6. Known limits at commissioning

- `scenario-to-desktop` API unreachable (WORKAROUNDS DPF-WA-001). E1 handles it
  first.
- Live BAS `start-failed` (DPF-WA-002). Journey parity falls back to the
  xdotool platform journey.
- E4 needs the operator's `sudo vrooli setup` for the WebKitGTK headers and
  driver.
