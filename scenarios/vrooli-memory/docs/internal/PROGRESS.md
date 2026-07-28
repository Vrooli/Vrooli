# Progress — Vrooli Memory

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |
| 2026-07-27 | codex | partial | Hardened Claude Code import into a durable, observable, single-flight job. `RunImport` returns/joins a run ID; `GetImportStatus` reports durable counters and checkpoints; unchanged sources are short-circuited by import key before inference. Focused API and CLI tests passed; full Test Genie evidence remains pending on its separate stuck run. |
| 2026-07-28 | codex | done | Completed the harness capability measurement gate. Inspected each installed runtime and its resource-owned integration: documented exact stores, native write surface, hook availability, capture channel, and projection behavior. Corrected Codex from a falsely claimed `PreToolUse` hook to store-diff capture; its resource adapter confirms no native hook. Confirmed `~/.codex/memories/` exists, Codex clips injected `AGENTS.md` context rather than rewriting the file, and Claude rejects over-limit memory-index writes explicitly. No harness memory source was written. |
| 2026-07-28 | codex | done | Completed facets-domain invariants: the seeded, table-owned vocabulary remains exactly six rows after repeated live restarts; journal writes reject an explicit unknown facet with the typed error; only episode entries are compaction candidates and pins exempt them; re-facet retains assignment history; supersession preserves the original journal entry. Focused domain, handler, journal, formatter, and lint validation passed. |
| 2026-07-28 | codex | done | Completed cross-depth recall and wake-budget contract hardening. The persistent source unions leaves and derived summary vectors while retaining absorbed leaves; collapse returns an antichain; wake reports pinned-content overflow explicitly; response hits identify the zoomable node plus summary/depth/span metadata. Added replay-safe summary-vector migration, independent validated `VROOLI_MEMORY_FRONTIER_TARGET` and `VROOLI_MEMORY_WAKE_BUDGET` controls, focused API/handler/migration coverage, regenerated contracts, and a healthy managed restart. The formal Test Genie run passed API and proto phases but remains broadly red on pre-existing maturity debt. |

## Entry Template

| 2026-07-27 | design workshop | done | Scenario generated from `react-vite`. PRD authored via the business-health wizard (10 P0 / 7 P1 / 4 P2 targets); requirements registry built with every target covered and every validation entry declaring intended evidence (no fabricated refs, all statuses `planned`). Concepts set written: DOMAINS (6 real domains), DATA (ownership + rebuild contract), FLOWS (write/compaction/recall state machines), INTEGRATIONS (dependency + failure contracts), ARCHITECTURE (memory layering invariants), REPLACEMENT (absorption scope + transition sequence). DECISIONS records 13 durable decisions including 5 supersessions. No implementation code written. |

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
