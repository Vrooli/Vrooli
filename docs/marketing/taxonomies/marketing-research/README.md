# Marketing Research Signal Taxonomy

Cross-team-readable canon for how marketing research signals are partitioned, classified, dispatched, and shaped. This file is the human-readable view of `docs/marketing/taxonomies/marketing-research/taxonomy.json` (the parseable sidecar consumed by the heartbeat builder and the `unknown_taxonomy` / `missing_destination_schema` validation rules).

**Owner team:** marketing-crew. **Status:** canon. Operator-curated via marketing-crew decisions.

Cited by:
- `topics.json` for marketing-crew members whose `intake[].taxonomy = "marketing-research"` (e.g., the researcher).
- The `signal-classifier` skill (pure judgment; cites this file for definitions and dispatch).

## Editing rules

When the JSON sidecar changes, update this markdown to match — they are paired sources of truth (markdown for human review, JSON for machine consumption). Validation rule `missing_destination_schema` will warn if a member declares an `output[].schema` that no taxonomy declares; promote schemas here before referencing them.

## Signal types and dispatch

| Signal type        | Definition                                                       | Default method skill        | Default destination prefix              |
|--------------------|------------------------------------------------------------------|-----------------------------|-----------------------------------------|
| audience-pain      | Audience frustration, buying trigger, vocabulary, unmet need.    | audience-pain-mining        | `audience-scan/<slug>`                  |
| competitor         | Competitor pricing, packaging, positioning, changelog.           | competitor-positioning-scan | `competitor-record/<slug>`              |
| hook               | Reusable opening, framing, or copy pattern.                      | hook-pattern-mining         | `hook-record/<slug>`                    |
| workflow           | External process, playbook, agent setup worth deconstructing.    | workflow-deconstruction     | `workflow-scan/<slug>`                  |
| skill              | External skill or reusable prompt/process.                       | skill-opportunity-scan      | `skill-scan/<slug>` (capability-gap if blocking) |
| channel            | New acquisition channel observed working in market.              | channel-format-scan         | `channel-scan/<slug>` (channel-strategy-update when material) |
| format             | New marketing post format or channel-native format.              | post-type-discovery         | `format-scan/<slug>` (post-type-proposal when material) |
| benchmark-adjacent | Pricing/market fact relevant to monetization (cross-team).       | benchmark-adjacent-scan     | `monetization-benchmark-adjacent-record/<slug>` (cross-team to monetization) |

## Evidence rules

- One source = observation; three converging independent sources = decision threshold.
- Single-snapshot findings must carry the `light-interpretation` flag.
- Tool classifications are inputs, not proof. (Bookmark-intelligence-hub's tags surface candidates; the analyst still evaluates relevance.)
- Researchers do not edit canon directly. Propose canon changes via owned decisions: `audience-update`, `channel-strategy-update`, `post-type-proposal`, `hook-candidate-promotion`.

## Action selection

The classifier returns a recommendation; the member's drain procedure picks one of:

| Action            | When                                                                                          |
|-------------------|-----------------------------------------------------------------------------------------------|
| drop              | Weak one-off / duplicate / out of scope. Delete the inbox row; mention in handoff if useful.  |
| observe           | Single-snapshot fact with applicability. Retag inbox row to destination prefix with front-matter. |
| promote-to-canon  | Converging evidence meets threshold. Retag plus raise the owned-context decision.             |
| file-decision     | Operator should decide now. Raise the decision; delete the inbox row if the artifact lives elsewhere. |
| capability-gap    | Source / tool / scenario missing. File `capability-gap` decision and leave the inbox row until closed. |

## Owned schemas

The following destination front-matter shapes are owned by this taxonomy. Members on **any** team that write to these prefixes adopt the schema declared here.

### audience-scan
```yaml
---
type: audience-scan
audience_segment: <text>
evidence_strength: <single-snapshot|converging>
honesty_flags: [<...>]
source: <url-or-null>
date_observed: <YYYY-MM-DD>
---
```
Body must include: `## Observation`, `## Implication (light-interpretation)`.

### competitor-observation
```yaml
---
type: competitor-observation
competitor: <name>
dimension: <pricing|packaging|positioning|changelog>
evidence_strength: <single-snapshot|converging>
honesty_flags: [<...>]
source: <url-or-null>
date_observed: <YYYY-MM-DD>
---
```
Body must include: `## Observation`, `## Implication`.

### hook
```yaml
---
type: hook
pattern: <short label>
channel: <text-or-null>
evidence_strength: <single-snapshot|converging>
honesty_flags: [<...>]
source: <url-or-null>
---
```
Body must include: `## Pattern`, `## Example`, `## When to use`.

### workflow-scan
```yaml
---
type: workflow-scan
subject: <text>
evidence_strength: <single-snapshot|converging>
honesty_flags: [<...>]
source: <url-or-null>
---
```
Body must include: `## Steps`, `## Reusable parts`.

### skill-scan
```yaml
---
type: skill-scan
subject: <text>
implies_capability_gap: <true|false>
honesty_flags: [<...>]
source: <url-or-null>
---
```
Body must include: `## Skill`, `## Implication`.

### channel-scan
```yaml
---
type: channel-scan
channel: <text>
evidence_strength: <single-snapshot|converging>
honesty_flags: [<...>]
source: <url-or-null>
date_observed: <YYYY-MM-DD>
---
```
Body must include: `## Observation`, `## Implication`.

### format-scan
```yaml
---
type: format-scan
format: <text>
evidence_strength: <single-snapshot|converging>
honesty_flags: [<...>]
source: <url-or-null>
---
```
Body must include: `## Format`, `## Example`, `## Adoption notes`.

### monetization-benchmark-adjacent (cross-team to monetization)
```yaml
---
type: monetization-benchmark-adjacent
comp: <name or category-wide>
dimension: <pricing|packaging|other>
applicability: <high|medium|low>
honesty_flags: [<...>]
source: <url-or-null>
date_observed: <YYYY-MM-DD>
---
```
Body must include: `## Value`, `## Notes`.

**Cross-team note.** This schema is owned by marketing's taxonomy because marketing's researcher is the producer. The receiving member on monetization adopts the same schema; it does not redefine it under monetization's taxonomy.

## Honesty flags

`light-interpretation`, `tailwind-uncited`, `ai-extracted`, `single-source`, `operator-asserted-only`.

## Pending method skills

Several `defaultMethod` ids above (e.g., `audience-pain-mining`, `competitor-positioning-scan`) are referenced as the canonical method but are not yet registered as standalone skills. Until they ship, the classifier still recommends them as `recommended_method`; members apply inline guidance from the front-matter shape and evidence rules above. See the JSON sidecar's `pendingMethodSkills` field for the live list.
