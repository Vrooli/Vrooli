## Tools focus: Market Validation Router

Triage validation requests in the monetization team's `validation-queue` and route each to the right method: a market-scan capture, a decision proposal, or a drop. This skill is the router, not the place where capture methods live.

> **Status:** v1 router. Plan-of-record lives in `docs/monetization/BENCHMARKS.md`, `docs/monetization/STRATEGY.md`, and `docs/monetization/REVENUE_LINES.md`. The market-scan canon is `monetization` team knowledge under `monetization/market-scan/<slug>`.

---

### 1. When To Use

Use this skill when market-validator needs to process:

- inbound validation requests in `validation-queue/*` (from operator vision walks, opportunity-scout conversions, catalog-strategist requests, financial-tracker assumption checks, or auto-populated staleness sweeps)
- a heartbeat where the queue is non-empty
- a handoff that says validation is blocked by a missing source

Do not use this skill for:
- the actual fetch-and-capture work — that's `pricing-comp-capture` (and future per-method skills as they emerge)
- staleness detection / queue auto-population — that's `benchmark-staleness-sweep`
- writing `docs/monetization/BENCHMARKS.md` — that's operator-curated; validator proposes via `benchmark-update` decisions

---

### 2. Required Reading

Read first:

- queue view: `prompt-manager team knowledge-list monetization --topic-prefix=validation-queue/ --json`
- existing scans: `prompt-manager team knowledge-list monetization --topic-prefix=monetization/market-scan/ --json`
- `docs/monetization/BENCHMARKS.md` — what's already grounded
- `scenarios/prompt-manager/store/teams/monetization/members/market-validator/last-handoff.md`
- pending decisions for market-validator's owned contexts (`benchmark-update`, `pricing-decision`, `financial-model-assumption-update`)

Read as needed:

- `docs/monetization/STRATEGY.md`, `docs/monetization/CATALOG.md`
- `docs/monetization/REVENUE_LINES.md`
- `scenarios/prompt-manager/store/teams/monetization/shared/operator-inputs.json` (for assumptions to validate)

---

### 3. Inputs

| Input | Required | Notes |
|---|---|---|
| `queue_items` | No | Specific entry IDs to triage. Default: triage everything in `validation-queue/*`. |
| `request_intent` | No | Hint at request type (see taxonomy). `unknown` allowed. |
| `proactive_topic` | No | If queue empty, optional topic for a proactive scan (a single benchmark gap). |

If the queue is empty AND no proactive topic supplied, write a brief no-validation-needed handoff and stop. Do not fabricate work.

---

### 4. Request-Type Taxonomy

Six slots:

| Request type | Source | Default method |
|---|---|---|
| `pricing-comp-needed` | opportunity-scout / catalog-strategist / vision-walk | `pricing-comp-capture` skill |
| `assumption-check` | financial-tracker / vision-walk | grounding pass — find 2-3 comps, write scans, raise `financial-model-assumption-update` if material |
| `benchmark-staleness` | auto-populated by `benchmark-staleness-sweep` | re-fetch source, compare, supersede via `benchmark-update` if changed |
| `competitor-deep-dive` | catalog-strategist / vision-walk | broad capture across pricing + packaging + retention; multiple scans |
| `channel-validation` | catalog-strategist / vision-walk | comp CAC / conversion / payback for the channel |
| `unknown` | escape hatch | classify before routing |

---

### 5. Routing Process

1. **Normalize each queue entry.** Confirm it has:
   - `request_type` (one of the six)
   - `source` (who asked, what they want)
   - `target` (which assumption / SKU / channel / comp this is about)
   - `urgency` (blocks-decision / nice-to-have / staleness-sweep)
   
   If any are missing, edit the entry to add them rather than guess silently.

2. **Apply leverage triage.** Per `HEARTBEAT.md`'s priority stack, pick the highest-leverage 1-2 items per heartbeat. Skip the rest with a note. Do not attempt the whole queue in one heartbeat.

3. **Pick the method.** Use the table in §4. For `pricing-comp-needed`, invoke `pricing-comp-capture`. For others, follow the inline guidance until a dedicated skill exists.

4. **Apply collection discipline.**
   - Cite `--source=<url>` on every scan entry.
   - Label single-snapshot findings `light-interpretation`.
   - Tailwind references must be cited or flagged `tailwind-uncited`.
   - Mixed external data stays conflicting; do not average into a fake clean number.
   - If source access is blocked, raise a `capability-gap` decision; do not invent.

5. **Raise decisions when material.** Default thresholds (override per `taskParameters` if team config differs):

   | Dimension | "Material" threshold |
   |---|---|
   | Pricing on a direct competitor | move >15% on Vrooli's tier-1 target band |
   | Retention / churn | move >5 percentage points on the comparable cohort |
   | Activation / attach-rate | move >10 percentage points |
   | Channel CAC / payback | move >25% on payback period |
   | Tier-1 financial-model assumption | finding contradicts assumption with applicability=high |

   Below threshold: write the scan, no decision.

6. **Resolve the queue entry.** Every routed item must leave `validation-queue/*` in one of three ways:

   - **Converted** to a market-scan — retag the entry:
     ```bash
     prompt-manager team knowledge-update monetization <id> --topic="monetization/market-scan/<slug>" --content="<content with required front-matter>"
     ```
     If the request produces multiple scans, retag the queue entry to the *primary* scan and create the additional scans as new knowledge entries; then delete the queue entry.

   - **Decision-only** (e.g., `capability-gap`, `assumption invalidated`, no new scan written) — raise the decision, then delete the queue entry:
     ```bash
     prompt-manager team knowledge-delete monetization <id>
     ```

   - **Dropped** — request is a duplicate, out of scope, or low-leverage:
     ```bash
     prompt-manager team knowledge-delete monetization <id>
     ```

   Do not leave entries under any `validation-queue/*` topic after triage. The queue view is the unrouted set.

7. **Emit output.** Concise routing summary for the heartbeat handoff.

---

### 6. Required Front-Matter Shape (Market-Scan)

Every promoted scan must carry this front-matter (matches `monetization-opportunity-router` for cross-skill consistency):

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

---

### 7. Output Contract

```markdown
### Market Validation Routing Summary

**Queue depth:** <count>; **triaged this heartbeat:** <count>; **deferred:** <count>

**Routed items:**
- `<queue-id>` (`<request_type>`) -> `<converted|decision-only|dropped>`; method=`<>`; flags=`<>`

**Scans written:**
- `<slug>` (comp=`<>`, dim=`<>`, applicability=`<>`) — one-line takeaway

**Decisions raised:**
- <context, rationale, threshold met>

**Capability gaps:**
- <missing source/tool/scenario, if any>

**Deferred queue items:**
- `<id>` — reason (lower leverage / blocked / pending input)
```

No known operational edge cases for standard usage.
