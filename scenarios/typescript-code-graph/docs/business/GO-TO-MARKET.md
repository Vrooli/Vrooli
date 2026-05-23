# Go To Market — TypeScript Code Graph

This document records launch strategy, positioning, channels, and validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

TypeScript Code Graph is **internal infrastructure**. It has no external audience and no positioning surface. The closest thing to an "audience" is *other Vrooli scenario authors*, and they discover it through:

- The architecture-cartographer and react-component-library documentation that names it as a substrate.
- The screaming-architecture audit doctrine: "do not parse source code in your scenario; use the language code-graph scenarios."
- This scenario's own README.

There is no external launch. This document is `not-applicable` rather than stub-filled.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| None | n/a — infrastructure scenario | n/a | (a) react-component-library successfully migrates off regex parsing within 6 months of v1, and (b) a third consumer adopts within 12 months. |

## Launch Motion

There is no public launch. The internal launch motion is:

1. Implement P0 operational targets (sidecar + graph + rewrite domains, fixture determinism gate including the `ts-jsdoc-tags` leading-comment fixture, performance SLA).
2. Update architecture-cartographer's `docs/internal/PROBLEMS.md` to remove the "scenarios do not yet exist" entry.
3. Unstub cartographer's `e2e_lang_graph_test.go` (it covers both Go and TS dependencies) and verify the integration works end-to-end.
4. Coordinate with react-component-library to migrate its regex parser onto this scenario's `Extract` + leading-comment surface (PRD OT-P1-006).
5. Document the decision in cartographer's and rcl's `PROGRESS.md` and link back to this scenario.
6. Update the screaming-architecture audit playbook to point TS-parsing concerns at this scenario.

That sequence is the launch. There is no external announcement.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Don't parse TS in your scenario; use typescript-code-graph." | Other Vrooli scenario authors | Cartographer is the first proof point; rcl migration is the second. | active once both integrations are green |
| "Replace your regex JSDoc scrapers with leading_comments from typescript-code-graph." | scenarios with regex-based TS comment parsing | rcl migration is the proof point. | active once rcl migrates |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Cartographer's MVP ships without writing TS parser code | internal | Cartographer's `graph` domain holds only adapter code (no `ts-morph` imports, no AST walking) | If true, the layered architecture decision was correct |
| rcl successfully migrates off regex JSDoc scraping | internal | rcl's `connect_handler.go` and `indexer.go` no longer contain `regexp.MustCompile` blocks for `@vrooliWidget*` / `@vrooliComponent*`; rcl tests pass against the same widget inventory | If true, the leading-comment contract was the right shape |
| A third consumer adopts typescript-code-graph within 12 months of v1 | internal | At least one non-cartographer non-rcl scenario calls `Extract` in production | If true, justifies the standalone scenario over inlined parsing |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — also not-applicable
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
