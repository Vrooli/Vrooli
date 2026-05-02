## Tools focus: Monetization Opportunity Router

Classify monetization intake and route each signal to the right next action: promote to the opportunity pool, convert to a market-scan canon entry, escalate as a decision, or drop. This skill is the router, not the permanent home for every analysis method.

> **Status:** v1 router. The plan-of-record lives in `docs/monetization/CATALOG.md`, `docs/monetization/REVENUE_LINES.md`, and `docs/monetization/BENCHMARKS.md`. The opportunity pool and market-scan canon both live as `monetization` team knowledge entries — the JSONL files are gone.

---

### 1. When To Use

Use this skill when the monetization team's opportunity-scout (or a peer member with read access) needs to process:

- operator-fed alpha from a morning vision walk
- bookmark/source/social-post links pointing at competitor moves, pricing changes, or capability arrivals
- proactive baseline scan findings (own scenarios inventory, capability-arrival sweeps)
- a handoff that says monetization analysis is blocked by a missing source/CLI/scenario
- a periodic triage of the opportunity-inbox queue

Do not use this skill for:
- pool hygiene / retirement / revisit-trigger evaluation — that's `opportunity-pool-hygiene`
- writing CATALOG.md candidate files — that's `catalog-strategist` after a `catalog-promotion` decision
- editing financial model assumptions — that's `financial-tracker`

---

### 2. Required Reading

Read first:

- `docs/monetization/CATALOG.md` — current SKU lifecycle and bundle structure
- inbox view: `prompt-manager team knowledge-list monetization --topic-prefix=opportunity-inbox/ --json` (filter to a single signal type with `--topic-prefix=opportunity-inbox/<signal-type>/`)
- pool view: `prompt-manager team knowledge-list monetization --topic-prefix=monetization/opportunity/ --json`
- `scenarios/prompt-manager/store/teams/monetization/members/opportunity-scout/last-handoff.md`
- recent `monetization/market-scan/*` knowledge entries
- pending decisions for `opportunity-scout`

Read as needed:

- `docs/monetization/STRATEGY.md`
- `docs/monetization/REVENUE_LINES.md`
- `docs/monetization/BENCHMARKS.md`
- `docs/monetization/scenario-sku-map.json`
- `docs/strategy/idea-pipeline/README.md` — when an idea is broader-than-SKU and may belong to operator-curated staging instead

---

### 3. Inputs

Inputs can be supplied by the caller or discovered from the required reading:

| Input | Required | Notes |
|---|---|---|
| `source_items` | No | Links, bookmark export rows, operator notes, vision-walk alpha, or inbox entries. |
| `scan_mode` | No | `inbox-first`, `proactive-baseline`, or `specific-source`. Default: `inbox-first`. |
| `signal_intent` | No | Hint at signal type (see taxonomy below). `unknown` is allowed. |
| `time_window` | No | Relevant for proactive scans. |

If source items are present, route those first. If no source items are present and the inbox is empty or stale, run a small proactive baseline scan (own scenarios inventory + 1-2 cited external comps) with cited sources.

---

### 4. Routing Process

1. **Normalize intake.** For each item, capture:
   - `source` and `source_url` if available
   - `raw_note`
   - `initial_signal_type`
   - `evidence_strength` (single-snapshot vs. converging)
   - `honesty_flags` (tailwind-uncited, light-interpretation, hardware-tier, legal-surface, etc.)

2. **Classify signal type.** The 8-slot taxonomy (matches morning-vision-walk Phase 8 routing):

| Signal type | Typical promotion target |
|---|---|
| `competitor-move` — competitor pricing/packaging/positioning/changelog change | market-scan canon (often) OR opportunity if it implies a new SKU/bundle |
| `capability-arrival` — Vrooli gained a scenario/resource that unlocks a new SKU/bundle | opportunity (almost always) |
| `customer-ask` — operator-fed signal that someone asked for X | opportunity (if SKU-shaped) OR notebook-debt (if vague) |
| `channel` — new acquisition channel observed working in market | market-scan canon if observation-only; opportunity if Vrooli can ship into it |
| `bundle-hint` — signal that two existing things should be packaged together | opportunity (almost always) |
| `retention-signal` — observed retention lever (own or competitor) | market-scan canon if observation; opportunity if it implies an add-on |
| `benchmark` — comparable pricing / market fact | market-scan canon |
| `unknown` — escape hatch; classify before routing |  N/A — must reclassify or drop |

3. **Choose the smallest useful action.**

| Condition | Action |
|---|---|
| Weak one-off signal, no Vrooli capability fit | **Drop** (`knowledge-delete`) |
| Single-snapshot market fact (pricing, packaging, comp benchmark) | **Convert to market-scan** — retag entry to `monetization/market-scan/<slug>`. Required front-matter: `comp`, `dimension`, `date_observed`, `applicability`, `source`. |
| Plausible SKU-shaped idea with classifiable fit | **Promote** — retag entry to `monetization/opportunity/<slug>`. Required front-matter: `kind`, `catalog.proposed_sku`, `catalog.parent_bundle`, `revisit_trigger`, `acquisition_hypothesis`, `retention_hypothesis`, `capability_reuse`, `tam`, `effort`, `status`. |
| Idea is broader-than-SKU OR not-yet-ready-for-active-tracking | Promote AND raise `catalog-promotion`-class decision proposing idea-pipeline as the staging target. |
| Strong signal, clear fit, evidence threshold met | Promote to opportunity pool AND raise an owned-context decision. |
| Repeatable analysis method has no skill | Propose a skill via the normal meta-optimization path. |
| Collection requires missing source/tool/scenario | Raise a `capability-gap` decision; leave the inbox entry until the gap is closed. |
| Tier-4 hardware proposal with no operator initiation | Drop. Out of scope for opportunity-scout per safety-critical rules. |

4. **Apply collection discipline.**
   - Cite sources for every external claim (`--source=<url>`).
   - Label single-snapshot findings `light-interpretation`.
   - Do not silently invent benchmarks; raise a `capability-gap` if source access is blocked.
   - Tailwind references (regulatory, market, demographic) must be cited or flagged `tailwind-uncited`.

5. **Resolve the inbox entry.** Every routed item must leave the inbox view (`team knowledge-list monetization --topic-prefix=opportunity-inbox/`) in one of three ways:

   - **Promoted** to the opportunity pool — retag:
     ```bash
     prompt-manager team knowledge-update monetization <id> --topic="monetization/opportunity/<slug>" --content="<content with required front-matter>"
     ```

   - **Converted** to market-scan canon — retag:
     ```bash
     prompt-manager team knowledge-update monetization <id> --topic="monetization/market-scan/<slug>" --content="<content with required front-matter>"
     ```

   - **Dropped** as weak/duplicate/out-of-scope:
     ```bash
     prompt-manager team knowledge-delete monetization <id>
     ```

   Do not leave entries under any `opportunity-inbox/*` topic after routing — that breaks the unrouted-set invariant. The inbox view is, by definition, the unrouted set.

6. **Emit output.** Provide a concise routing summary for the scout heartbeat.

---

### 5. Required Front-Matter Shapes

#### Opportunity (`monetization/opportunity/<slug>`)

```yaml
---
type: opportunity
kind: <sku-candidate|addon-candidate|services-line-candidate|channel-candidate>
catalog:
  proposed_sku: <new-base-bundle|addon|services-line|null>
  parent_bundle: <business|lifestyle|null>
capability_reuse: <high|medium|low>
tam: <S|M|L>
effort: <S|M|L>
status: <idea|candidate|trigger-met|active|shipped|retired>
original_at: <ISO timestamp>
original_by: <agent id or operator>
---
```

Body must include: `# <Name>`, description, `## Revisit trigger`, `## Acquisition hypothesis`, `## Retention hypothesis`, `## Signal`.

#### Market-scan (`monetization/market-scan/<slug>`)

```yaml
---
type: market-scan
kind: benchmark-capture
comp: <name or category-wide>
category: <text>
dimension: <pricing|other>
date_observed: <YYYY-MM-DD>
applicability: <high|medium|low>
affects_benchmarks_md: <true|false>
affects_pricing: <true|false>
affects_financial_model: <text or null>
original_at: <ISO timestamp>
original_by: <agent id>
---
```

Body must include: `## Value`, `## Notes`. Source URL goes on the entry's `--source` flag.

---

### 6. Output Contract

```markdown
### Opportunity Routing Summary

**Inputs reviewed:** <count and source modes>

**Routed items:**
- `<inbox-id>` -> `<signal_type>` -> `<promoted|converted|dropped>`; evidence=`<strength>`; flags=`<flags>`

**Promoted to opportunity pool:**
- `<slug>` (kind=<>, sku=<>, parent=<>) — one-line rationale

**Converted to market-scan:**
- `<slug>` (comp=<>, dimension=<>) — one-line rationale

**Dropped:**
- `<id>` — one-line reason

**Decisions raised:**
- <context, rationale, evidence threshold status>

**Skill or capability gaps:**
- <proposed skill/scenario/capability-gap, if any>

**Proactive baseline:**
- <run/not run and why>
```

No known operational edge cases for standard usage.
