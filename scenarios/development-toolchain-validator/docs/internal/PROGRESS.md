# Progress — Development Toolchain Validator

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
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
