# Progress — Proto Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-13 | codex | done | Route implementation proof gap closed at the provider layer: `go-code-graph` now emits static route-registration facts and `code-facts` maps them into endpoint proof evidence. Targeted unit validation passed for `go-code-graph/internal/graph` and `code-facts/internal/facts`; live provider validation should now drop the prior `proto.endpoint_proof_unsupported` warnings for `health` and `notes_attach`, leaving only unrelated template-source warnings when providers are healthy. |
| 2026-06-12 | codex | done | Code Facts Phase 13 closeout: `vrooli scenario test proto-health`, `proto-health validate scenario proto-health --json`, and `test-genie execute proto-health proto --json` pass. Validation currently reports six warnings: endpoint proof is unknown for `health` and `notes_attach`, and four template-source proto files remain annotated until their scaffold contracts are intentionally replaced or adopted. |
| 2026-06-11 | codex | done | Generated `proto-health` from `react-vite`, rewrote the proto style guide around domain organization, and replaced starter PRD/requirements with the proto validation target. |
| 2026-06-11 | codex | done | Added the `ProtoHealthService` proto contract plus shared proto-surface fact messages, generated proto artifacts, and implemented the descriptor-backed `internal/protosurface` reader with source annotation, service/RPC/message/import, transport, and adoption facts. |
| 2026-06-11 | codex | done | Implemented the first `internal/validation` service slice for `ValidateScenario` and `DescribeScenarioProtos`, including cycle/package/version/annotation/import/adoption/transport/stability/domain/unused-message checks, Connect handler/module wiring, generated endpoint metadata, and API tests. |
| 2026-06-11 | codex | done | Implemented the Phase 5 programmatic/direct surfaces: `validate scenario` and `describe scenario` CLI commands backed by `ProtoHealthService`, plus a dashboard proto-health panel that validates a selected scenario, shows finding counts, and renders proto-surface inventory. Also cleared the generated UI router warning, duplicate-landmark test debt, and proto annotation drift that made proto-health fail its own validator. |
| 2026-06-11 | codex | done | Implemented the Phase 6 quality-loop integration slice: `FINDING_SOURCE_PROTO`, test-genie's optional `proto` phase, the maturity `proto-health` R2 dimension, CI proto generated-artifact verification, and the `proto-contract-audit` steer skill. |
| 2026-06-11 | codex | done | Completed the Phase 7 self-validation slice: synchronized proto-health PRD/requirements shape, removed the temporary `notes` measure declaration, moved proto-health UI copy onto generated i18n keys, fixed the theme media-query fallback for jsdom, and brought `vrooli scenario test proto-health` to green. |
| 2026-06-11 | codex | done | Closed the generated-artifact validation gap by adding a fakeable `GenSyncChecker`, production temp-workspace codegen comparison for scenario generated slices, `proto.gen_out_of_sync` service coverage, and sibling smoke checks for `cli-health`, `measures-health`, and `agent-manager`. |
| 2026-06-11 | codex | done | Hardened the `proto-contract-audit` skill with troubleshooting guidance, created late regression baselines for test-genie/ecosystem-manager, revalidated the proto/test-genie/maturity surfaces, and confirmed `proto-health` health, UI, direct validation, test-genie proto phase, and full scenario test remain green. |
| 2026-06-12 | codex | done | Adopted `code-facts` as the implementation-proof provider: validation now consumes proto adoption and REST endpoint proof reports through a fakeable seam, maps missing/contradicted proof to findings, degrades unavailable/unsupported analyzers to warnings, and keeps source parsing out of proto-health. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
