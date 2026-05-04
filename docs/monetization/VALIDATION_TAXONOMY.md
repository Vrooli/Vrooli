# Monetization Validation Request Taxonomy

Cross-team-readable canon for how monetization validation requests are partitioned, classified, dispatched, and shaped. This file is the human-readable view of `docs/monetization/validation-taxonomy.json`.

**Owner team:** monetization. **Status:** canon. Operator-curated via monetization decisions.

Cited by:
- `topics.json` for `monetization/market-validator` (`intake[].taxonomy = "monetization-validation"` — both for `validation-inbox/*` and the cross-team `monetization-benchmark-adjacent-record/*` intake).
- The `market-validation-triage` skill (pure judgment).

## Editing rules

Update this markdown when the JSON sidecar changes (or vice versa). Both are paired sources of truth.

## Request types and dispatch

The 5-slot taxonomy:

| Request type        | Definition                                                       | Default method                | Default destination                 |
|---------------------|------------------------------------------------------------------|-------------------------------|-------------------------------------|
| pricing-comp-needed | A specific competitor pricing comp is requested.                 | pricing-comp-capture          | `monetization-benchmark-record/<slug>`     |
| assumption-check    | Validate a financial-model assumption.                           | assumption-grounding          | `monetization-benchmark-record/<slug>`     |
| benchmark-staleness | Auto-populated by benchmark-staleness-sweep; refresh.            | stale-refresh                 | `monetization-benchmark-record/<slug>`     |
| competitor-deep-dive| Broad capture across pricing + packaging + retention.            | competitor-deep-dive-capture  | `monetization-benchmark-record/<slug>`     |
| channel-validation  | Comp CAC / conversion / payback for a channel.                   | channel-validation-capture    | `monetization-benchmark-record/<slug>`     |

Validation-queue entries arrive with a `request_type` set by the producer (opportunity-scout, financial-tracker, vision-walk, or `benchmark-staleness-sweep`). The triage skill classifies (sanity-checks the type), prioritizes by leverage, and recommends a method. The `monetization-benchmark-adjacent-record/*` cross-team intake shares this taxonomy because the validator triages it the same way (the producer's marketing taxonomy owns the front-matter schema for that prefix; see `docs/marketing/SIGNAL_TAXONOMY.md`).

## Evidence rules

- Cite `--source=<url>` on every scan entry.
- Label single-snapshot findings `light-interpretation`.
- Tailwind references must be cited or flagged `tailwind-uncited`.
- Mixed external data stays conflicting; do not average into a fake clean number.
- If source access is blocked, raise `capability-gap`; do not invent.

## Materiality thresholds (decision-raise gates)

| Dimension                         | "Material" threshold                                            |
|-----------------------------------|------------------------------------------------------------------|
| Pricing on a tier-1 competitor    | move >15% on Vrooli's tier-1 target band                         |
| Retention / churn                 | move >5 percentage points on the comparable cohort               |
| Activation / attach-rate          | move >10 percentage points                                       |
| Channel CAC / payback             | move >25% on payback period                                      |
| Tier-1 financial-model assumption | finding contradicts assumption with applicability=high           |

Below threshold: write the scan, no decision. Above threshold: raise `benchmark-update` / `pricing-decision` / `financial-model-assumption-update`.

## Action selection

| Action            | When                                                                                          |
|-------------------|-----------------------------------------------------------------------------------------------|
| drop              | Duplicate / out of scope / low-leverage.                                                      |
| observe           | Single-snapshot fact below threshold. Retag to `monetization-benchmark-record/<slug>`; no decision.  |
| promote-to-canon  | Scan crosses materiality. Retag plus raise the matching owned decision.                       |
| file-decision     | Decision-only outcome (e.g., assumption invalidated, no scan). Raise; delete the queue entry. |
| capability-gap    | Source / tool / scenario missing.                                                             |

## Owned schemas

### market-scan
```yaml
---
type: market-scan
kind: <benchmark-capture|assumption-check|competitive-observation|stale-refresh|channel-assumption-check>
comp: <name or category-wide>
category: <text>
dimension: <pricing|retention|churn|attach-rate|activation|channel-cac|other>
date_observed: <YYYY-MM-DD>
applicability: <high|medium|low>
affects_benchmarks_md: <true|false>
affects_pricing: <true|false>
affects_financial_model: <text or null>
supersedes: <prior-scan-id or null>
original_at: <ISO timestamp>
original_by: market-validator
---
```
Body must include `## Value` and `## Notes`. Source URL goes on the entry's `--source` flag.

## Honesty flags

`light-interpretation`, `tailwind-uncited`, `single-source`, `applicability-low`.

## Pending method skills

`assumption-grounding`, `stale-refresh`, `competitor-deep-dive-capture`, `channel-validation-capture` are not yet registered. Until they ship, the triage skill recommends them as `recommended_method`; the member follows inline guidance from the schemas + evidence rules + materiality thresholds above.
