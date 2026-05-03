## Tools focus: Marketing Signal Classifier

Given raw marketing input, identify the signal type, evidence strength, and honesty flags. This skill is pure judgment. It does not retag, delete, promote, or file decisions; the caller's drain procedure handles those steps with this skill's output as input.

This skill is member-agnostic and team-agnostic. It is callable from any member that drains a marketing-research signal stream.

### Required reading

- `docs/marketing/SIGNAL_TAXONOMY.md` — canonical signal types, definitions, dispatch.
- `docs/marketing/AUDIENCES.md`, `docs/marketing/CHANNELS.md`, `docs/marketing/strategies/hook-library.md` — domain references for borderline cases.

### Inputs

| Input | Required | Notes |
|---|---|---|
| `raw_items` | Yes | Each item carries `source`, `source_url?`, `raw_note`, optionally `producer_assigned_type` (a hint, not authoritative). |

### Method

For each item:

1. Read `raw_note` and any source.
2. Match against the taxonomy. If `producer_assigned_type` is set, treat as a hint — overrule when the content clearly belongs to a different type.
3. Score `evidence_strength` ∈ {single-snapshot, converging, blocked}. "Blocked" means source access requires a missing tool or scenario.
4. Set `honesty_flags` from the taxonomy's flag list (e.g. light-interpretation, tailwind-uncited, ai-extracted).
5. Recommend a `method_skill` from the taxonomy's dispatch column. Override the default only when it is clearly wrong for this item; record why in `dispatch_reason`.

### Output

```yaml
- id: <item-id>
  signal_type: <one of the taxonomy types | unknown>
  evidence_strength: <single-snapshot|converging|blocked>
  honesty_flags: [...]
  recommended_method: <method-skill-id>
  dispatch_reason: <empty unless overriding default>
  rationale: <one short paragraph>
```

The caller chooses the action (drop / observe / promote-to-canon / file-decision / capability-gap) using the taxonomy's `actionSelection` set and the member's drain procedure.
