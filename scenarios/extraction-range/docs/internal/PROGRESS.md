# Progress — Extraction Range

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This log records implementation milestones and their evidence.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-11-12 | extraction-range owner | in progress | Added 40 data-defined answer-key levels across all ten families, deterministic scoring, Connect/REST and CLI surfaces, and owned practice-page behaviors. `make setup`, `make endpoints`, lifecycle start/restart, API+CLI Go suites passed. Test Genie run `20261112-202652-397c24ee`: structure/API/business passed; unit failed on a React Router jsdom signal mismatch and duplicate local HTTP/DB test fakes. Follow-up uses a plain MemoryRouter with future flags, aliases the shared API test fakes, and injects synthetic delay; Unit Health UI execution then passed tests and type-check at 94.7% workspace coverage with five existing/boilerplate per-file coverage advisories. A Test Genie rerun after those fixes remains open. Scoreboard is in-memory; operator UI and full Test Genie qualification remain open. |
| 2026-11-12 | extraction-range owner | in progress | Added range-owned SQLite scoreboard storage and an idempotent per-domain schema; best results survive a fresh handler and a live lifecycle restart. Current live scoreboard retains 40 passing level rows across all ten families (260/260). Removed the unused template notes example with template-manager detemplate, and added the operator practice workbench with same-origin API proxying. Targeted docs phase passed; full Test Genie still has contract, proto, security, UI/experience and external TypeScript-provider findings to resolve. The complete Data/FLOWS/Runbook and review page now describe durable scores. |
| 2026-11-12 | extraction-range UI owner | in progress | Replaced the generic home placeholder with a responsive extraction workbench: 40 challenge levels, family filtering, selected-level task and owned target, named record submission, exact score feedback, and durable per-tool scoreboard. Added the same-origin `/levels`, `/score`, `/scoreboard` and `/practice/` API proxy needed by the UI; confirmed `/levels` returns 200 JSON through UI_PORT. UI type-check/build and full configured Vitest run passed (97 tests across 19 files). BAS captures cover desktop, phone portrait and 844×390 landscape; experience-manager spec validation passes with zero findings, and Test Genie run `20261112-222054-ade5f2bf` passed phase `experience` at L3 with no findings. Details and image evidence are in `docs/concepts/EXPERIENCE.md` and `docs/mockups/COMPARISON.md`; human concept/mockup ratification is still unrecorded. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
