# Progress — Asset Studio

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative. The
2026-07-28 entry records design and contract authoring only; no implementation
code exists.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |

| 2026-07-28 | claude | done | Scenario generated from `react-vite` 1.6.5 with the `vrooli-default` design kit. PRD authored (15 P0 / 11 P1 / 4 P2 operational targets). Requirements registry rebuilt as three modules — `01-must-ship`, `02-post-launch`, `03-future` — with 30 requirements, every one linked to a PRD target and declaring intended evidence, all statuses `planned` and no fabricated refs; `vrooli scenario requirements validate asset-studio` reports PASSED with no findings. Concept docs written: DOMAINS (5 product domains plus health, with build order and four rejected domain candidates), DATA (ownership, blob fork with content-desk, reproducibility contract, import idempotency), FLOWS (6 flows, 4 state machines, the render job at L5), INTEGRATIONS (dependency and failure contracts, and the one-way direction with content-desk), ARCHITECTURE (production layering and four invariants). DECISIONS records 14 durable decisions. PROBLEMS records 5 open constraints, including that the conformance comparison is unvalidated. **No implementation code written.** |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
