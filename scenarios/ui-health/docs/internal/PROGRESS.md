# Progress — UI Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-16 | Codex | done | Adopted the shared health maturity assessment contract for UI Health. Added `.vrooli/maturity.json`, emitted `common.v1.MaturityAssessment` from `ValidateScenario`, rendered local maturity in CLI human output, and preserved local maturity in Test Genie ui-health phase summaries. |
| 2026-07-07 | Codex | done | Added component-canon static detectors for unused exported components and raw primitive overuse, wired their maturity mappings, and expanded runtime visual validation to desktop plus mobile viewport profiles. |
| 2026-07-09 | Codex | done | Promoted `standard_raw_primitive_overuse` from advisory to required Project Standards maturity input after fleet shadow measurement. Added `standard_design_token_bypass`, `standard_component_canon_unengaged`, and an EmptyState false-positive regression fixture while keeping non-raw canon findings advisory. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
