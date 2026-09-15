# Progress Log

This log keeps recent dated milestones. It does not certify current readiness.
Read [PROBLEMS.md](PROBLEMS.md) for tracked gaps and the owning Plan Manager
record for live execution status.

## Earlier history

The complete pre-cleanup log, including intermediate results and unresolved
handoffs, is preserved byte-for-byte at the protected runtime-home location:

```text
<runtime-home>/plan-artifacts/docs-history-20260907/scenarios/swarm-manager/docs/internal/PROGRESS.md
```

SHA-256: `a58601221f15880de33f83aaa26007f5ae8973fe01c204b53b671cf6df8d4b54`. Restore or read it before relying on an
older completion claim; archival does not close any outstanding work. The
[project preservation record](../../../../docs/internal/PROGRESS.md#documentation-cleanup--2026-09-07)
records ownership and recovery.

## Recent milestones

| Date       | Author            | Status Snapshot | Notes |
|------------|-------------------|-----------------|-------|
| 2026-07-16 | Codex | Generic agent-workflow runtime and phased-plan pilot | Added durable provider-aware final results, structured-result abstention, fresh/continue/child/wait/branch workflow semantics, bounded journal bindings and budgets in Agent Manager; added authored phased-plan slice-review/drain workflows plus a narrow Swarm adapter; hard-cut phased plan execution to pinned workflow definitions with durable exactly-once apply, approval, cancellation, and restart recovery. Focused Go, race, graph simulation, architecture-boundary, requirements, and dependency-governance gates pass. Agent comprehensive evidence remains red on inherited debt after two changed-surface findings were fixed; Swarm terminal evidence is unavailable because of the tracked one-shot wait defect. |
| 2026-06-25 | Claude Opus 4.8 | Standards-alignment refactor cleanup pass | Audited the in-progress standards-alignment refactor (uncommitted working tree, net ~-9.3k LOC: god-file decomposition of `agentsessions`/`feedback`/`initiatives`/`initiativereview`/`execution` `service.go` into focused `service_*.go` files, testutil subpackages consolidated into a flat `testutil` package, +178 test functions). Refactor judged disciplined (clean delete-as-you-go, no duplicated old/new logic; legacy on-disk-state migration code in `execution` is live and correct). Cleaned up the one incomplete migration found: removed the dead settings-overlay profile cluster left behind by the manifest-driven-profile migration in `internal/agentmanager` (`ProfileConfig`, `ProfileConfigFromSettings`, `DefaultProfileConfig`, `buildProfile`, `defaultProfileRef`, `DefaultAgentMaxTurns`, `SettingsReader`/`SetSettingsReader`, `DefaultServiceConfig`, the unused `cfg *ProfileConfig` Initialize param, and the write-only `settings.NewAgentAdapter` wiring at main.go); kept the test-only `NewHTTPClient*` seams (deadcode false-positives — used by `client_test.go`). Fixed two stale prose docs invalidated by the testutil consolidation (SEAMS.md testutil growth-path, UTILS_UNIFICATION_NOTES.md `assertx.Eventually`). Removed the untracked `PRD.md.backup.20260512-213236` cruft. `go build`/`go vet`/affected tests green. |
| 2026-03-28 | Codex | Execution telemetry split and graph runtime lineage | Added first-class `AgentActivity` proto/domain/API contracts and durable `.vrooli/agent-activities.json` storage; introduced a tracked AgentManager seam that records spawn/continue activity across backlog research/workshop/finalize, execution, spec-sync, fixup, follow-up, and capture classification flows; kept `ExecutionRecord` as the governed implementation-control object; updated graph flow/operations projections and UI stores/components to render agent activity nodes plus run lineage; removed the legacy local-only `agent-runs-store`; verification passed with `make generate`, `go test ./...`, `pnpm type-check`, and `pnpm test` |

## New entries

Append a dated milestone with its outcome, remaining constraint, and durable
owner reference. Store detailed command output with the producing owner.
