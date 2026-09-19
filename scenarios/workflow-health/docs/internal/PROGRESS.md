# Progress — Workflow Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-02 | Codex | done | Completed Phase 10 rollout hardening slice: workflow-health now self-registers `workflow.flow`, `workflow.test`, and `workflow.fragment` providers with Search Hub; Search Hub federation was verified for flow/test/fragment queries; workflow-health provider-contract, workflow phase, API/CLI tests, and scenario test passed. Also hardened the delegated Test Genie workflow smoke case with a registered selector and executable self-dashboard assertion. |
| 2026-07-02 | Codex | done | Migrated Test Genie to the canonical delegated `workflow` phase backed by workflow-health, kept `playbooks`/`e2e` as deprecated aliases, added workflow finding source mapping, documented the workflow phase, and replaced the stale self BAS case with an executable observer overview smoke workflow. |
| 2026-07-02 | Codex | done | Implemented the first workflow-health operator UI slice: replaced template navigation with Overview, Inventory, Search, Runs, Findings, Fixes, and Settings routes; added workflow-specific selectors, route tests, localized copy, accessible landmark labels, dense operational view models, and full UI validation. |
| 2026-07-02 | Codex | done | Implemented workflow search/action discovery slice: `workflow.flow`, `workflow.test`, and `workflow.fragment` search leaves, deterministic intent ranking, mutating guardrails, `WorkflowSearchService`, CLI `workflows search`, regenerated proto/endpoint metadata, and docs updates. |
| 2026-07-02 | Codex | done | Exposed workflow-health through shared `ScenarioValidationService`: validation handler, fix preview/apply RPCs, native catalog/execution detail, endpoint metadata, CLI `validate scenario`/`fix preview`/`fix apply`, and provider response tests. |
| 2026-07-02 | Codex | done | Implemented the first safe execution and artifact ingestion slice: fakeable BAS client seam, static-only/dry-run paths, observer execution, mutating fail-closed preflight, timeline/latest artifact writer, and focused execution tests. |
| 2026-07-02 | Codex | done | Implemented the static workflow validation, maturity, and deterministic fix slice: `.vrooli/maturity.json`, `api/internal/validation`, rule/spec drift tests, maturity assessment construction, registry rebuild, metadata stubs, execution-mode normalization, and reset normalization. |
| 2026-07-02 | Codex | done | Implemented the initial workflow asset catalog model and scanner for BAS cases, flows, actions, seeds, registry-only entries, requirement links, selector/route refs, safety profiles, and subflow/fixture dependency edges. |
| 2026-07-02 | Codex | done | Generated the workflow-health react-vite scaffold with the `vrooli-default` design kit, replaced starter PRD/requirements with workflow-health contract targets, and documented the Phase 1 ownership model. |
| 2026-07-02 | Codex | partial | Generator post-hook reached proto regeneration but module validation failed until `api/go.mod` gained a local `github.com/vrooli/cli-core` replace; rerun focused validation after contract edits. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
