# Go To Market — Go Code Graph

This document records launch strategy, positioning, channels, and validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

Go Code Graph is **internal infrastructure**. It has no external audience and no positioning surface. The closest thing to an "audience" is *other Vrooli scenario authors*, and they discover it through:

- The architecture-cartographer documentation that names it as a dependency.
- The screaming-architecture audit doctrine, which says "do not parse source code in your scenario; use the language code-graph scenarios."
- This scenario's own README.

There is no external launch, no website, no demo, no positioning narrative. This document is `not-applicable` rather than stub-filled.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| None | n/a — infrastructure scenario | n/a | A second consumer (beyond architecture-cartographer) adopts the scenario within 6 months of v1, validating that the layered architecture decision was correct. |

## Launch Motion

There is no public launch. The internal launch motion is:

1. Implement P0 operational targets (graph + rewrite domains, fixture determinism gate, performance SLA).
2. Update architecture-cartographer's `docs/internal/PROBLEMS.md` to remove the "scenarios do not yet exist" entry.
3. Unstub cartographer's `e2e_lang_graph_test.go` and verify the integration works end-to-end.
4. Document the decision in cartographer's `PROGRESS.md` and link back to this scenario.
5. Update the screaming-architecture audit playbook to point Go-parsing concerns at this scenario.

That sequence is the launch. There is no external announcement.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Don't parse Go in your scenario; use go-code-graph." | Other Vrooli scenario authors | Cartographer is the first proof point. | active once cartographer integration is green |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Cartographer's MVP ships without writing Go parser code | internal | Cartographer's `graph` domain holds only adapter code (no AST walking, no `go/parser` imports) | If true, the layered architecture decision was correct |
| A second consumer adopts go-code-graph within 6 months | internal | At least one non-cartographer scenario calls `Extract` in production | If true, justifies the standalone scenario over inlined parsing |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — also not-applicable
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
