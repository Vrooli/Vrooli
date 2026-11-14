# Progress — Development Toolchain Validator

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-05-29 | agent | done | Tooling-baseline finished: replaced the exit-code-only `dev_tools` stub with a generic typed tool **registry** (`integrations/dev_tools/registry.go`) seeded with `test-genie` (runs `execute --preset comprehensive --json`; expects every catalog phase to pass) and `scenario-completeness-scoring` (runs `score calculate` then `score get --json`; expects `score >= floor`, default 96); `scenario-auditor` dropped (covered inside test-genie). Per-tool expectations load from `data/tool-expectations/<tool>.json` (`expectations.go`). Runner captures stdout/stderr separately → richer `vrun.ToolResult{Ran, ExpectationMet, Detail, RawOutput}`; evaluator now maps the two-layer signal (couldn't-run → `run_failure`, ran-but-missed → `tool_failure`, ran+met → `pass`). Raw output + detail persisted on the validation record (new `tool_detail`/`tool_raw_output` columns + idempotent `EnsureColumns` migration + proto fields 17/18) and surfaced through the report read path; the golden×tool VerdictGrid already renders tool verdicts. Resolves PROBLEMS.md 2026-05-18 "tool expectations parsing is structural-only". Prereq landed in test-genie: `comprehensive` preset is now catalog-derived (was a stale hand-list omitting `ui-health`+`architecture`) with an anti-drift guard test, and the advisory `architecture` phase degrades to a skip when the cartographer is unreachable so it can never gate `comprehensive`. Plan: `tooling-baseline-domain-drift-proof-test-genie-comprehensive-preset`. |
| 2026-05-29 | agent | done | P0 finalize: proved the core loop live end-to-end against a real agent-manager + the committed `reference-react-vite` golden. Fixed two substrate bugs found by the first live run — (1) golden was never seeded into the sandbox (`agent_manager/client.go` now passes an absolute `ProjectRoot`=golden tree + `ScopePath="."` via `resolveGoldenRoot`); (2) the skill's instructions were never delivered to the agent (new `SkillContentSource` seam + `prompt_manager.SkillContentRESTAdapter` fetching `/skills/{id}`; worker injects it via `buildSkillPrompt`). Also fixed `diff_hash` to be `sha256(RunDiff.content)` instead of the run id. Verdicts proven: `run_failure` (no manifest), `pass` (empty-diff manifest + converged skill), `stale` (skill_drift) live; `unexpected_mutation` unit-proven. CLI clean-build fix (`cli/go.mod` go directive). New flow-verified `runs` UI surface (RunsIndex + RunDetail) + nav + a "Run validation" affordance on TupleDetail; coverage dashboard renders report data. Requirements 01–10 reconciled to `done` with test/E2E evidence (module 30 routed-DB e2e left `planned` pending its BAS run). Plan: `dtv-p0-finalize-prove-the-loop-fix-hashing-build-run-ui-reconcile-requirements`. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
