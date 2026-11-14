# Go To Market — Portal

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

- Audience: Vrooli operators and builders who move between conversations and
  owned browser or desktop capabilities.
- Positioning: a local-first front door that keeps target identity,
  permissions, and provider readiness visible.
- Main claim: hypothesis only; no public performance or platform claim is made.
- Proof needed: receipt-backed native support rows, authenticated journeys, and
  repeat operator usage.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Internal operator cohort | Validate repeat context-to-task workflows. | Linux staging artifact, Portal journey scripts, support inventory. | Five interviews and repeat use on a supported row. |
| Public launch | Deferred. | Signed cross-platform artifacts, privacy/recovery guides, cost model. | Release review passes with no unsupported claims. |

## Launch Motion

1. Validate the internal operator workflow on a named supported row.
2. Reconcile platform, security, privacy, and recovery evidence.
3. Measure inference, relay, and support costs from real usage records.
4. Interview operators before selecting personal or team packaging.
5. Publish externally only after release approval and claims-to-receipt review.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| Local-first operator front door | Internal operators | Linux staging smoke and focused Portal tests | internal hypothesis |
| Cross-platform native companion | Public buyers | No complete support matrix yet | deferred |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Repeat workflow study | Internal operator cohort | Five interviews; two repeat sessions per participant | Decide whether team packaging merits investment |
| Cost and support measurement | Internal operator cohort | Token, relay, host, signing, and support inputs present | Set a price only if cost model is complete |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
