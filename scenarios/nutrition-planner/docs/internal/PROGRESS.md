# Progress — Nutrition Planner

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Entries are appended when work lands, not while it is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-09-18 | opencode | documented | Filled PRD, requirements, concept/business/operations/internal docs, and experience contract from the canonical product specification; implementation not started |
| 2026-09-18 | opencode | documented | Closed remaining internal-doc stubs: filled ERROR-HANDLING with stable error codes and semantics, added the planned product seams to SEAMS, and recorded decisions D-018–D-020 |
| 2026-09-18 | codex | implemented | Added exact decimal, money, unit, and nutrient registries; mounted an authenticated, SQLite-backed workspace domain with server-derived ownership and idempotent create retries; generated workspace Connect artifacts; removed the template notes domain; regenerated endpoint metadata. |
| 2026-09-18 | codex | implemented | Added review-only receipt purchase proposals with source/transaction/normalized-line deduplication, explicit idempotent purchase application, authenticated inventory RPCs, and provider-disabled manual fallback. |
| 2026-09-18 | codex | implemented | Added durable server-side workspace entitlements and idempotent optional-compute reservations, enforced job budgets, a disabled billing adapter boundary, and scoped diagnostics/data-health reporting for failed jobs, stale prices, unknown targets, unresolved recipe data, incomplete plans, schema version, and provider availability. |
| 2026-09-18 | codex | implemented | Made Today’s recipe-map action load the authenticated planned revision into the shared recipe viewer, with explicit Error and unknown-error handling covered by UI regressions. |
| 2026-09-18 | codex | implemented | Added a source-aware USDA FoodData Central adapter behind the canonical HTTP seam, including search/detail identity, observed dates, nutrient evidence, API-key fallback, and classified HTTP failures. |
| 2026-09-18 | codex | implemented | Expanded onboarding apply results to show fits, needs-review, and excluded meal counts from the shared eligibility evaluator, with explicit no-match messaging and generated proto/client coverage. |
| 2026-09-18 | codex | implemented | Added canonical recipe yield, serving unit, and ingredient quantities to immutable recipe revisions; the shared viewer now performs display-only exact-decimal scaling and flags fractional discrete ingredients without changing stored data or method time. |
| 2026-09-18 | codex | implemented | Added week lock/skip persistence and regeneration rehydration, lossless native recipe portability, staged transfer-center exports/restores, A4/US Letter weekly and recipe PDFs with page-count footers, shared swap-dialog focus containment and Escape restoration, grocery CSV artifacts, and focused UI/backend validation. |
| 2026-09-18 | codex | implemented | Wired the deterministic nutrition scope evaluator into the Nutrition page: configured targets now show recorded-so-far subtotal, pass/fail/unknown status, and completeness; new intake refreshes the evaluation without treating unavailable evidence as zero. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
