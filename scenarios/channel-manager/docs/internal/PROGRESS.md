# Progress — Channel Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |
| 2026-07-28 | claude | done | Scenario generated from `react-vite` 1.6.5 as the replacement for `social-media-scheduler`, which was retired to `/tmp` the same day — it had not compiled since 2025-09-08, had no consumers, and nothing was migrated (D-001). PRD authored with 19 P0 / 7 P1 / 4 P2 operational targets; requirements registry built with 30 requirements, one per target, every entry `planned` and declaring intended evidence with no fabricated refs. Concepts set written: DOMAINS (5 real domains plus health), ARCHITECTURE (action layering and the five invariants), DATA (ownership, rebuild contract, no-credential-storage rule), FLOWS (queued-action lifecycle at L5, four state machines, scheduling diagram), INTEGRATIONS (dependency and failure contracts, marketing canon as read-only input). DECISIONS records D-000 through D-009. Seeded descriptors: two TikTok warming programs and one platform descriptor, all marked `speculative` with provenance and revisit triggers. No implementation code written; `api/internal` still holds only the template scaffold. |
| 2026-07-28 | claude | done | Business and operations docs completed, and four targets added from a competitive scan. MONETIZATION corrected: an earlier revision concluded there was no revenue line by checking for an ai-gateway dependency, but the executor is metered — `browser-automation-studio` charges AI credits per AI-assisted operation through LPBS, so warming generates indirect revenue under a Tier 1 subscription (D-010 context; sizing and kill signals recorded). Deployment tiers resolved: Tier 1 and Tier 2 viable, **Tier 3 hosted ruled out on technical grounds** since datacenter egress defeats the per-identity residential-proxy precondition. SECURITY, OBSERVABILITY, RUNBOOK, and DEPLOYMENT filled with scenario-specific content. Targets added: `OT-P1-008` first-comment atomicity, `OT-P1-009` per-platform post preview, `OT-P1-010` environment liveness, `OT-P2-005` deliberate re-release — registry now 34 requirements against 34 targets, still one-to-one. D-010 records three permanently refused capabilities (content spinning, bulk account creation, engagement pods) with per-capability reasoning. Still no implementation code. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
