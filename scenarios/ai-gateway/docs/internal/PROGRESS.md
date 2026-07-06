# Progress — AI Gateway

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-05 | Codex | done | Generated the `ai-gateway` scenario from the React/Vite template and completed the documentation-first charter, requirements modules, experience draft, architecture docs, role/profile reference, and AI conformance maturity ladder. |
| 2026-07-05 | Codex | done | Removed the generated notes example domain from API, CLI, UI, experience specs, docs blocks, proto sources, and generated proto artifacts; refreshed tests and orientation gates around the remaining health/scaffold baseline. |
| 2026-07-05 | Codex | done | Added Phase 2 proto-first gateway, inventory, routing, and conformance API contract skeletons; implemented provider-neutral gateway request validation; mounted Connect handlers; regenerated proto and endpoint metadata. |
| 2026-07-06 | Codex | done | Implemented Phase 3 provider inventory through resource-owned CLI seams: added bounded command adapters for Ollama/OpenRouter policy roles, normalized inventory and smoke API responses, regenerated proto/endpoints, and documented remaining resource-owned role gaps. |
| 2026-07-06 | Codex | done | Implemented Phase 4 routing preview, resource-command execution, and redacted route evidence persistence: added routing RPCs, deterministic profile/locality policy, stdin-backed provider execution, fail-closed SQLite evidence storage, focused routing tests, and regenerated proto/endpoints. |
| 2026-07-06 | Codex | done | Implemented Phase 5 AI conformance provider registration: expanded scanner taxonomy, mounted shared `ScenarioValidationService`, added maturity-backed `.vrooli/test-genie.json`, documented the `ai-conformance` Test Genie phase, regenerated endpoints, and validated provider-contract discovery/checks. |
| 2026-07-06 | Codex | done | Implemented Phase 6 CLI operator surfaces: added manifest-backed gateway, inventory, routing, conformance, and shared validation command groups using generated Connect clients; removed Phase 6 omissions; added proto service coverage tests; and updated API/CLI references. |
| 2026-07-06 | Codex | done | Implemented Phase 7 operator console: replaced the scaffold UI with dashboard, provider inventory, route preview, conformance scan, and policy settings pages; added typed Connect UI clients, selector-grounded experience specs, BAS operator cases, and focused UI tests. |
| 2026-07-06 | Codex | done | Started Phase 8 hardening: reconciled requirements validation refs to concrete API/CLI/UI/BAS tests, removed false complete PRD claims for future P2 work, added `[REQ:...]` test markers, cleared local product-code security findings for command execution/path handling, and reran focused validation. |
| 2026-07-06 | Codex | done | Continued Phase 8 readiness: moved shared gateway request/enums/issues into the AI Gateway shared proto package, regenerated Go/TypeScript/Python proto artifacts and endpoint metadata, fixed governed CLI module drift, added UI coverage tests so `test:coverage` passes thresholds, and documented remaining advisory health findings. |
| 2026-07-06 | Codex | done | Finished the Phase 8 validation pass: security health and docs health are clean, requirements/experience traceability validates with zero findings, tidiness now passes after focused helper/scanner refactors, the remaining route-evidence measures waiver is documented as a follow-up, and swarm record `rec-1c26d70a5324a0ce` captures the slice. |
| 2026-07-06 | Codex | done | Closed the final Phase 8 honesty gap by marking adaptive routing intelligence and fleet enforcement as planned P2 work instead of complete, removed docs-health derived-count warnings, centralized formal-artifact test reads behind a reviewed test-only helper, reran the comprehensive suite (`20260706-142736-cccd13c0`), and recorded `rec-d851bb608b42bed7`. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
