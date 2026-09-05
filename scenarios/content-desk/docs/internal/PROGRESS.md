# Progress — Content Desk

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This is the durable handoff log for landed work; entries distinguish completed
behavior from follow-on work still owned elsewhere.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-28 | codex | done | Added durable coverage dimensions (`campaign`, `lane`, `channel`, `SKU`) to drafts and exposed ledger contamination and coverage reports over Connect and the discoverable CLI. Coverage reports persisted publish cells and marks stale cells from their latest publish time; repository tests prove fresh and stale outcomes. Post types now persist declared craft/policy failure modes, and review recording refuses missing, extra, or duplicate verdict modes before a run is written. |
| 2026-07-28 | codex | done | Verified the no-credential boundary over every Content Desk domain schema and Content Desk proto schema: a case-insensitive scan for credential, secret, token, account-handle, API-key, and password terms returned no matches. Credentials and account operations remain outside this scenario. |
| 2026-07-28 | codex | partial | Completed the operator/agent control-surface slice: the dashboard now reads draft-scoped citations, explains approval blockers from those citations, attaches an existing shared claim to an exact current-body span, and can run a cited claim's verification check. Campaign creation/activation, post-type registration, review verdict recording, draft revision/publish, and claim lifecycle mutations are discoverable through Connect and CLI primitives. Added the paired `content-desk` skill with required post-type-canon reading and an explicit agent no-approval/no-publish boundary. Focused dashboard tests cover attachment and verification actions; claim repository tests cover the draft-scoped read. |
| 2026-07-28 | codex | partial | Added durable production-ledger behavior across the active content-desk domains: campaign slot reservation is transactional and released exactly once on abandonment; draft revisions are immutable and attributed; approved publishes atomically append a publish record and optional series predecessor link. Claims now preserve evidence, require re-runnable checks for quantitative/existence/status assertions, support verification sweeps, and expire dated novelty evidence. The public Connect and CLI surfaces now support campaign creation/activation, draft revision/publish, and claim create/cite/verify/sweep. The generated `notes` reference domain was removed through `template-manager detemplate`; stale UI navigation and tests were removed with it. Local validation passed API and CLI Go suites, UI type-check and 144 UI tests, endpoint inventory, temporal models, and lifecycle setup. Remaining: agent-facing mutation surfaces for post types and reviews, scenario-specific documentation/calibration, requirement evidence, and full maturity remediation. |
| 2026-07-28 | claude | done | Scenario generated from `react-vite` 1.6.5 with the `vrooli-default` design kit, replacing the retired `campaign-content-studio` (moved to `/tmp/campaign-content-studio-retired-2026-07-27`, not deleted). Charter authored through the business-health wizard: 14 P0 / 8 P1 / 4 P2 operational targets. Requirements registry built with one module per tier and all 26 requirements enriched with category, tags, and typed validation entries declaring intended evidence — no fabricated refs, every status `planned`. Starter `01-foundation` module removed. Concepts set written: DOMAINS (six real domains plus health, with an explicit not-a-domain table), DATA (ownership, schema map, idempotent import, retention), FLOWS (six flows, three modelled state machines, draft-path diagram), INTEGRATIONS (dependency inventory, deliberate boundaries, failure modes), ARCHITECTURE (layering shape and three invariants). DECISIONS records 11 durable decisions. Validation: `vrooli scenario requirements validate content-desk` PASSED with no findings; `business-health validate scenario content-desk` PASSED; all four contract dimensions at L3. **No implementation code written** — orientation gates 0 and 6–8 remain open. |
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
