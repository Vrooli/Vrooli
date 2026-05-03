## Tools focus: Monetization Signal Classifier

Given raw input describing a monetization signal (competitor move, capability arrival, customer ask, channel observation, bundle hint, retention signal, or benchmark), assign signal_type, evidence_strength, honesty_flags, and a recommended method. Pure judgment.

This skill does not retag, delete, promote, or file decisions. The caller's drain procedure handles those steps with this skill's output as input. The skill is member-agnostic and team-agnostic.

### Required reading

- `docs/monetization/OPPORTUNITY_TAXONOMY.md` — canonical signal types, definitions, dispatch.
- `docs/monetization/CATALOG.md`, `docs/monetization/REVENUE_LINES.md`, `docs/monetization/STRATEGY.md` — domain references for borderline cases.
- `docs/strategy/idea-pipeline/README.md` — when an idea is broader-than-SKU.

### Inputs

| Input | Required | Notes |
|---|---|---|
| `raw_items` | Yes | Each item carries `source`, `source_url?`, `raw_note`, optionally `producer_assigned_type` (a hint). |

### Method

For each item:

1. Read `raw_note` and any source.
2. Match against the taxonomy. If `producer_assigned_type` is present, treat as a hint, not authoritative.
3. Score `evidence_strength` ∈ {single-snapshot, converging, blocked}.
4. Set `honesty_flags` from the taxonomy's flag list (light-interpretation, tailwind-uncited, hardware-tier, legal-surface, single-source).
5. Apply scope rules:
   - Tier-4 hardware proposals without operator initiation: signal_type=unknown, recommended_method=null, rationale notes "out of scope".
6. Recommend a `method_skill` from the taxonomy's dispatch column. Override only when clearly wrong; explain in `dispatch_reason`.

### Output

```yaml
- id: <item-id>
  signal_type: <one of the taxonomy types | unknown>
  evidence_strength: <single-snapshot|converging|blocked>
  honesty_flags: [...]
  recommended_method: <method-skill-id-or-null>
  dispatch_reason: <empty unless overriding default>
  rationale: <one short paragraph>
```

The caller chooses the action (drop / observe / promote-to-canon / file-decision / capability-gap) using the taxonomy's `actionSelection` set and the member's drain procedure.
