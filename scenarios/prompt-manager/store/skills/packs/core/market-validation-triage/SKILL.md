## Tools focus: Market Validation Triage

Given a market-validation request, confirm its request_type, prioritize by leverage, set honesty_flags, and recommend a default method. Pure judgment.

This skill does not modify queues, write scans, or file decisions. The caller's drain procedure handles those steps with this skill's output as input. The skill is member-agnostic and team-agnostic.

### Required reading

- `docs/monetization/VALIDATION_TAXONOMY.md` — canonical request types, dispatch, materiality thresholds.
- `docs/monetization/BENCHMARKS.md` — what is already grounded.
- `docs/monetization/STRATEGY.md`, `docs/monetization/CATALOG.md`, `docs/monetization/REVENUE_LINES.md` — domain references.

### Inputs

| Input | Required | Notes |
|---|---|---|
| `request_items` | Yes | Each item carries `request_type` (producer-assigned hint), `source`, `target`, `urgency`, optionally `raw_note` and `producer_assigned_type`. |

### Method

For each request:

1. Read `raw_note`, `target`, and any cited source.
2. Confirm `request_type` against the taxonomy (5 slots: pricing-comp-needed, assumption-check, benchmark-staleness, competitor-deep-dive, channel-validation). Reclassify if the producer's hint is wrong; "unknown" is allowed only as an escape hatch the caller must handle.
3. Score `evidence_strength` ∈ {single-snapshot, converging, blocked}.
4. Score `leverage_score` ∈ {high, medium, low} from the urgency dimension and the materiality thresholds in the taxonomy. Requests that cannot cross any materiality threshold even if confirmed are `low`.
5. Set `honesty_flags` from the taxonomy's flag list.
6. Recommend a `method_skill` from the dispatch column. Override only when clearly wrong; explain in `dispatch_reason`.

### Output

```yaml
- id: <item-id>
  request_type: <one of the taxonomy types | unknown>
  evidence_strength: <single-snapshot|converging|blocked>
  leverage_score: <high|medium|low>
  honesty_flags: [...]
  recommended_method: <method-skill-id-or-null>
  dispatch_reason: <empty unless overriding default>
  rationale: <one short paragraph>
```

The caller chooses the action (drop / observe / promote-to-canon / file-decision / capability-gap) using the taxonomy's `actionSelection` set and the member's drain procedure. The triage is intentionally lossy: low-leverage requests can be deferred safely.
