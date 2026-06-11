# Progress — Proto Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-11 | codex | done | Generated `proto-health` from `react-vite`, rewrote the proto style guide around domain organization, and replaced starter PRD/requirements with the proto validation target. |
| 2026-06-11 | codex | done | Added the `ProtoHealthService` proto contract plus shared proto-surface fact messages, generated proto artifacts, and implemented the descriptor-backed `internal/protosurface` reader with source annotation, service/RPC/message/import, transport, and adoption facts. |
| 2026-06-11 | codex | done | Implemented the first `internal/validation` service slice for `ValidateScenario` and `DescribeScenarioProtos`, including cycle/package/version/annotation/import/adoption/transport/stability/domain/unused-message checks, Connect handler/module wiring, generated endpoint metadata, and API tests. |
| 2026-06-11 | codex | done | Implemented the Phase 5 programmatic/direct surfaces: `validate scenario` and `describe scenario` CLI commands backed by `ProtoHealthService`, plus a dashboard proto-health panel that validates a selected scenario, shows finding counts, and renders proto-surface inventory. Also cleared the generated UI router warning, duplicate-landmark test debt, and proto annotation drift that made proto-health fail its own validator. |
| 2026-06-11 | codex | done | Implemented the Phase 6 quality-loop integration slice: `FINDING_SOURCE_PROTO`, test-genie's optional `proto` phase, the maturity `proto-health` R2 dimension, CI proto generated-artifact verification, and the `proto-contract-audit` steer skill. |
| 2026-06-11 | codex | done | Completed the Phase 7 self-validation slice: synchronized proto-health PRD/requirements shape, removed the temporary `notes` measure declaration, moved proto-health UI copy onto generated i18n keys, fixed the theme media-query fallback for jsdom, and brought `vrooli scenario test proto-health` to green. |
| 2026-06-11 | codex | done | Closed the generated-artifact validation gap by adding a fakeable `GenSyncChecker`, production temp-workspace codegen comparison for scenario generated slices, `proto.gen_out_of_sync` service coverage, and sibling smoke checks for `cli-health`, `measures-health`, and `agent-manager`. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
