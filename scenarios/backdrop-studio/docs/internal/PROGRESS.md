# Progress — Backdrop Studio

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-11 | claude | partial | Scenario generated from `react-vite` 1.6.5 with the `vrooli-default` design kit, then documented ahead of implementation at the operator's request. Landed: `PRD.md` with 18 P0, 8 P1 and 4 P2 operational targets in EARS shape; a seven-module requirements registry carrying 40 requirements (`catalog`, `scaffold`, `compose`, `render`, `legibility`, `release`, `workbench`), replacing the starter `01-foundation` module; `docs/concepts/{ARCHITECTURE,DOMAINS,DATA,FLOWS,INTEGRATIONS,UI-ARCHITECTURE}.md`; `docs/reference/taxonomy.md` defining the five axis enums; `docs/internal/DECISIONS.md` with eight durable decisions; `docs/business/{MONETIZATION,GO-TO-MARKET}.md`; and eight L0 experience page specs replacing the generated dashboard/notes placeholders. Six upstream dependency gaps recorded in `PROBLEMS.md`. **No product code written** — `api/`, `cli/` and `ui/` remain the generated scaffold, so `scaffold-health`, `first-real-vertical-slice` and `example-domain-removed` gates are correctly unmet. Validation: all requirements and experience JSON parse; `make orient` reports the documentation gates. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | status | Summary of what landed, how it was validated, and what remains |
```

Status values: `partial`, `complete`, `blocked`, `reverted`.

## Next

The build order is set in `../concepts/DOMAINS.md`. In dependency order:

1. **Upstream first.** Treatment operations must land in `image-tools`
   (`PROBLEMS.md`, first entry). Nothing here produces a releasable candidate
   until they do, and no local substitute is acceptable — implementing them in
   this scenario would violate `CMP-004` and bury a generic capability inside
   one product.
2. **`catalog` domain.** The taxonomy becomes real and queryable. Nothing
   renders yet, and that is fine.
3. **`scaffold` + `compose` + `render`, procedural lanes only.** This is the
   milestone that matters: a complete, useful product with zero model
   dependency, working offline at no cost. If the procedural lanes work end to
   end, the architecture is proven and everything after is addition rather than
   risk.
4. **`legibility` + `release`**, then the first consumer integration in
   `landing-page-business-suite`.
5. **`guided` and `synthesized` lanes**, which depend on the `asset-studio`
   verdict generalization (`PROBLEMS.md`, second entry).
6. **Workbench depth** — contact sheet, placement preview, remix.

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues and upstream dependencies
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and rationale
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — bounded contexts and build order
