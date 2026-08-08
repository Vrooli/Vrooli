## Tools focus: Signal Classifier

Given a raw intake item, identify its type, evidence strength, and honesty flags, and recommend a default method. This skill is pure judgment against a taxonomy the caller supplies. It does not retag, delete, promote, or file decisions; the caller's drain procedure runs those steps with this skill's output as input.

This skill is member-agnostic and team-agnostic. It is callable from any member that drains a signal stream, across marketing, monetization, and market-validation domains. The domain is fixed entirely by the taxonomy the caller names — one spine, one output contract, one per-domain delta.

### Required reading

- **The taxonomy the caller names.** The caller's intake entry carries a `taxonomy` id (from `topics.json::intake[].taxonomy`). Read that taxonomy's README before classifying — it is the single source of truth for the type slots, the dispatch (default method) column, the honesty-flag list, and the evidence rules. Taxonomy READMEs live at `docs/<domain>/taxonomies/<taxonomy-id>/README.md`.
- **The domain references for that taxonomy** (borderline cases only) — see the delta table below.

### Inputs

| Input | Required | Notes |
|---|---|---|
| `items` | Yes | Each item carries `source`, `source_url?`, `raw_note`, optionally `producer_assigned_type` (a hint, not authoritative). Market-validation items also carry `target` and `urgency`. |

### Method

For each item:

1. Read `raw_note` and any cited source.
2. Match against the taxonomy the caller named. Treat `producer_assigned_type` as a hint; overrule it when the content clearly belongs to a different type. `unknown` is allowed only as an escape hatch the caller must handle.
3. Score `evidence_strength` ∈ {single-snapshot, converging, blocked}. `blocked` means source access requires a missing tool or scenario.
4. Set `honesty_flags` from the taxonomy's flag list.
5. Apply the per-domain delta from the table below (extra dimension or scope rule), if the caller's taxonomy has one.
6. Recommend a `recommended_method` from the taxonomy's dispatch column. Override the default only when it is clearly wrong for this item; record why in `dispatch_reason`.

### Per-domain delta table

Only the genuine differences. Everything not listed here follows the shared spine and output contract.

| Taxonomy (caller-supplied) | Taxonomy README | Domain references (borderline) | Delta |
|---|---|---|---|
| `marketing-research` | `docs/marketing/taxonomies/marketing-research/README.md` | `docs/marketing/strategy/AUDIENCES.md`, `CHANNELS.md`, `patterns/hook-library.md` | None. Base spine and output. |
| `monetization-opportunity` | `docs/monetization/taxonomies/monetization-opportunity/README.md` | `docs/monetization/catalogs/CATALOG.md`, `strategy/STRATEGY.md`, `docs/strategy/idea-pipeline/README.md` | Scope rule: a Tier-4 hardware proposal without operator initiation is out of scope — set `signal_type=unknown`, `recommended_method=null`, and note "out of scope" in `rationale`. |
| `monetization-validation` | `docs/monetization/taxonomies/monetization-validation/README.md` | `docs/monetization/evidence/BENCHMARKS.md`, `strategy/STRATEGY.md`, `catalogs/CATALOG.md` | Output renames `signal_type` → `request_type` and adds `leverage_score` ∈ {high, medium, low}, scored from the item's `urgency` and the taxonomy's materiality thresholds. A request that cannot cross any materiality threshold even if confirmed is `low`. The triage is intentionally lossy: low-leverage requests can be deferred safely. |

### Output

```yaml
- id: <item-id>
  signal_type: <one of the taxonomy types | unknown>   # named request_type for monetization-validation
  evidence_strength: <single-snapshot|converging|blocked>
  leverage_score: <high|medium|low>                    # monetization-validation only
  honesty_flags: [...]
  recommended_method: <method-skill-id | null>
  dispatch_reason: <empty unless overriding default>
  rationale: <one short paragraph>
```

The caller chooses the action (drop / observe / promote-to-canon / file-work / capability-work) using the taxonomy's `actionSelection` set and the member's drain procedure.
