## Tools focus: Pricing Comp Capture

Capture a competitor's pricing as a structured `topic[old]:monetization/market-scan/<slug>` knowledge entry. This skill is opinionated — source priority, required fields, honesty flags, and material-change thresholds are baked in so capture is repeatable across heartbeats and across operators.

> **Status:** v1. Use the existing migrated scans (`cursor-pricing-*`, `github-copilot-pricing-*`, `openrouter-pricing-*`, `raycast-pricing-*`, `notion-pricing-*`, `setapp-pricing-*`, `poe-pricing-*` under `topic[old]:monetization/market-scan/`) as exemplars of the target shape.

---

### 1. When To Use

Use this skill when:

- the `signal-classifier` routes a `pricing-comp-needed` request to you
- a `benchmark-staleness` queue entry for a pricing-dimension scan has matured (re-fetch path uses this skill)
- opportunity-scout or vision-walk produced a `competitor-move` signal that converted to a pricing-needed validation request

Do not use this skill for:
- retention / churn / activation / attach-rate captures — those need a different method (TBD)
- channel CAC captures — that's `channel-validation` request type
- writing `docs/monetization/BENCHMARKS.md` — propose via `benchmark-update` decision instead

---

### 2. Source Priority

Try in order; document which source produced the values, and the most-canonical source always wins on conflicts.

1. **Company pricing page** — `<comp>.com/pricing`, `<comp>.com/plans`, or the pricing link from the marketing nav. Canonical source. **Always try this first**, even if you "already know" the prices.
2. **ProductHunt listing** — when the company's own page is gated, geo-restricted, or "temporarily unavailable" (a real condition — Copilot showed this in 2026-04). Capture what's visible there with the `partial-pricing` flag.
3. **G2 listing** — when ProductHunt has nothing useful and the page is fully gated. G2 prices are sometimes user-reported and can drift; flag `g2-user-reported` if used.
4. **Wayback machine snapshot** — `web.archive.org/web/*/<comp-pricing-url>` — when the live page 404s or the company removed pricing transparency. Flag `archived-source-<date>`. Treat as evidence-of-past, not evidence-of-current.
5. **Founder posts / public statements** — Twitter, blog, podcast transcript with explicit price numbers. Flag `founder-post-<date>`. Lower confidence than a published page.

If steps 1-5 all fail, raise a `capability-gap` decision instead of inventing values. Do not estimate from competitor screenshots, sales-call transcripts, or hearsay.

---

### 3. What To Capture

For each tier the comp publishes, capture as a single `## Value` block:

- **Tier name** as the comp uses it (Hobby, Pro, Pro+, Ultra, Teams, Enterprise, etc.)
- **Monthly price** in the comp's primary currency
- **Annual price or annual-discount %** if shown (the toggle is canonical evidence — note if it exists)
- **Included quotas** (seats, requests, GB, agents, etc. — use the comp's units verbatim)
- **Overage pricing** if metered
- **Free tier** existence and limits (if any)
- **Enterprise gate** ("contact sales" vs. published number)
- **Geographic / currency notes** if the page localized you

Do not normalize across comps in this entry. Different comps use different denominators ("seat" vs. "user" vs. "agent"). Normalization is BENCHMARKS.md territory; the scan is raw evidence.

---

### 4. Required Front-Matter

```yaml
---
type: market-scan
kind: <benchmark-capture|stale-refresh|competitive-observation>
comp: <Company or Product Name>
category: <dev-tool SaaS|multi-product bundle|consumer sub|services|other>
dimension: pricing
date_observed: <YYYY-MM-DD>
applicability: <high|medium|low>
affects_benchmarks_md: <true|false>
affects_pricing: <true|false>
affects_financial_model: <text or null>
supersedes: <prior-scan-id or null>
source_tier: <company-pricing|producthunt|g2|wayback|founder-post>
original_at: <ISO timestamp>
original_by: market-validator
---
```

Body shape:

```markdown
## Value

<one paragraph or bullet list per tier — exactly what the page shows, in the comp's terminology>

## Notes

<2-4 sentences of interpretation: how does this anchor or contradict Vrooli's tier-1 target band, what's notable about the structure, what state the page was in (toggle present, "temporarily unavailable", localized prices, etc.)>
```

`--source=<url>` on the CLI invocation is mandatory. If multiple sources contributed (e.g., company page for Pro + ProductHunt for Pro+ when one tier is gated), use the company-page URL as `--source` and document the secondary in `## Notes`.

---

### 5. Honesty Flags

Apply liberally; readers downstream rely on these to know how much weight to give the entry.

| Flag | When to use |
|---|---|
| `light-interpretation` | Only one snapshot; haven't cross-checked. Default for first-capture. |
| `temporarily-unavailable` | The page rendered but said "temporarily unavailable" or similar — capture what's there but flag. |
| `partial-pricing` | Some tiers shown, others gated. |
| `enterprise-gated` | Top tier is "contact sales" with no published number. |
| `localized-pricing` | The page detected your geo and showed local prices; the canonical USD may differ. |
| `g2-user-reported` | Source was G2; prices may be user-submitted not vendor-published. |
| `archived-source-<date>` | Source was wayback; treat as historical, not current. |
| `founder-post-<date>` | Source was a public statement, not a published page. |
| `mixed-evidence` | Different sources gave different numbers; both captured, no average. |

---

### 6. Material-Change Threshold

Raise a `benchmark-update` or `pricing-decision` only when the finding is **material**. Defaults:

- **First-capture for a comp not yet in BENCHMARKS.md**: always propose a `benchmark-update` so it gets indexed.
- **Refresh of an existing scan, price unchanged**: bump `date_observed` on the existing entry (re-write content with updated date). No decision.
- **Refresh, price changed by ≤15% on Vrooli's tier-1 target band ($29-$49 for business bundle)**: write the scan, supersede the old, no decision.
- **Refresh, price changed by >15% on the tier-1 band**: write the scan, supersede the old, raise `benchmark-update` decision with the delta highlighted.
- **New tier added/removed by the comp**: structural change is always material; raise `benchmark-update` regardless of the percentage delta.
- **Comp shipped a multi-product bundle / unbundled / changed denomination**: also structural; raise `benchmark-update` and consider whether `catalog-strategist` needs a heads-up handoff.

The 15% threshold is the default; per-team `taskParameters` may override (current team config: `staleBenchmarkAfterMonths: 12`, no explicit price-delta override).

---

### 7. Worked Example

The migrated entry `topic[old]:monetization/market-scan/cursor-pricing-20260424-9365` is a clean exemplar:

```yaml
type: market-scan
kind: benchmark-capture
comp: Cursor
category: dev-tool SaaS
dimension: pricing
date_observed: 2026-04-24
applicability: high
affects_benchmarks_md: true
affects_pricing: true
affects_financial_model: null
```

Body captured every published tier (Hobby, Pro, Pro+, Ultra, Teams, Enterprise) in the comp's verbatim terminology, with a notes section explaining how Cursor's $20 Pro and $40/seat Teams anchor Vrooli's tier-1 target band. Source: `https://cursor.com/pricing`. That's the shape to match.

---

### 8. CLI Reference

Initial capture (inbox entry exists, retag from `topic[example]:validation-inbox/pricing-comp-needed/<slug>` to scan):

```bash
prompt-manager team knowledge-update monetization "<queue-id>" \
  --topic="monetization/market-scan/<comp>-pricing-<YYYYMMDD>-<id4>" \
  --content="<full content with front-matter and body>" \
  --source="<url>"
```

New capture (no queue entry, e.g., proactive scan):

```bash
prompt-manager team knowledge-add monetization \
  --topic="monetization/market-scan/<comp>-pricing-<YYYYMMDD>-<id4>" \
  --caller-note="market-validator" \
  --content="<full content>" \
  --source="<url>"
```

Slug format: `<comp-kebab>-pricing-<YYYYMMDD>-<id4>` where `id4` is the last 4 digits of the new entry id (or any 4-digit unique suffix). This matches the migration convention.

---

### 9. Output Contract

When called by the router, emit:

```markdown
### Pricing Comp Capture: <Comp Name>

**Source tier used:** <company-pricing|producthunt|g2|wayback|founder-post>
**Source URL:** <url>
**Date observed:** <YYYY-MM-DD>
**Flags:** <light-interpretation, partial-pricing, etc.>

**Tiers captured:** <count>; **Material change vs. prior scan:** <yes/no/n-a>

**Decision raised:** <benchmark-update id, or "none — not material" >

**Knowledge entry:** `topic[old]:monetization/market-scan/<slug>` (id <knw-...>)
```

No known operational edge cases for standard usage.
