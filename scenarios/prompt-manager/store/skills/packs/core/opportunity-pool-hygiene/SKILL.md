## Tools focus: Opportunity Pool Hygiene

Periodically sweep the monetization opportunity pool — knowledge entries under `monetization/opportunity/<slug>` — to keep it honest. Evaluate each entry's revisit trigger; promote, retire, or leave as-is. Complement to `monetization-signal-classifier`, which handles intake; this skill handles outflow and decay.

> **Status:** v1. The pool lives as `monetization` team knowledge entries under `monetization/opportunity/<slug>` topics. There is no separate JSONL file.

---

### 1. When To Use

Use this skill on the opportunity-scout heartbeat when:

- the pool has grown beyond `~15` entries and needs a freshness pass
- it has been more than 14 days since the last hygiene sweep (check via `last-handoff.md`)
- a heartbeat lands during a vision walk where the operator may want a refreshed catalog-promotion candidate list
- a recent ledger event, scenario shipping milestone, or competitor move could fire one or more revisit triggers

Do not use this skill for:
- intake / triage of the inbox — that's `monetization-signal-classifier`
- editing CATALOG.md candidate files — propose via `catalog-promotion`-class decision; catalog-strategist + operator do the writing

---

### 2. Required Reading

Read first:

- pool view: `prompt-manager team knowledge-list monetization --topic-prefix=monetization/opportunity/ --json`
- recent `monetization/market-scan/*` entries — competitor moves and benchmarks may fire triggers
- `docs/monetization/CATALOG.md` — current SKU lifecycle; some pool entries may already have graduated and be redundant
- recent `decisions.jsonl` for `catalog-promotion`, `services-activation`, `runway-warning` contexts
- recent `ledger.jsonl` events — financial state may invalidate some bets
- `scenarios/prompt-manager/store/teams/monetization/members/opportunity-scout/last-handoff.md`

Read as needed:

- `docs/monetization/STRATEGY.md`
- `docs/monetization/REVENUE_LINES.md`
- `docs/strategy/idea-pipeline/README.md` — broader-than-SKU ideas may belong in operator-curated staging instead

---

### 3. Hygiene Process

For each entry in the pool:

1. **Parse the front-matter.** Confirm it has the required fields (`kind`, `catalog.proposed_sku`, `catalog.parent_bundle`, `revisit_trigger`, `acquisition_hypothesis`, `retention_hypothesis`, `capability_reuse`, `tam`, `effort`, `status`). If any are missing or malformed, flag for repair (do not silently fix; surface in the output).

2. **Evaluate the revisit trigger** against current state:

| Trigger state | Action |
|---|---|
| **Trigger fully fired** (all conditions met) | Raise a `catalog-promotion` decision proposing this SKU enter CATALOG.md as `candidate`. Update entry's front-matter `status: trigger-met`. |
| **Trigger partially fired** (≥1 condition met, others outstanding) | Append a dated note to the entry body (`## Trigger evaluation log`) explaining what fired and what's missing. Do not change status. |
| **Trigger contradicted by evidence** (e.g., dependency scenario retired, target metric moved away from threshold) | Retire — see step 3. |
| **Trigger unchanged** | Leave entry alone. |
| **Trigger phrased ambiguously / unmeasurable** | Flag for repair: propose a sharper trigger in the output. The owning agent (or operator) rewrites; this skill does not silently rewrite triggers. |

3. **Retirement criteria.** Retire (delete or status=`retired`) an entry when **any** of:

   - The Vrooli capability the bet depends on has been retired or de-scoped (check scenarios/ inventory).
   - Ledger evidence has disproved either hypothesis (e.g., zero acquisition signal after 90+ days at a meaningful surface).
   - A neighboring SKU has shipped and absorbed the proposed scope (the bet is now redundant).
   - The bet has been superseded by a better-shaped sibling already in the pool.
   - The entry has been in the pool >180 days with no trigger movement and no recent reinforcing signal.

   For retirement: prefer `--content` update with `status: retired` + a `## Retirement note` body section over hard `knowledge-delete`. Hard-delete only when the entry was a duplicate or malformed. Retired entries remain searchable for institutional memory.

4. **Pool-size pressure.** If the pool has >25 active entries (`status` ∈ {`idea`, `candidate`, `trigger-met`}):

   - First, surface the oldest entries with no trigger movement as retirement candidates.
   - Second, look for clusters that should consolidate (e.g., three "creator-tooling" entries that overlap → merge into one).
   - Do not propose blanket cuts; the cap is a forcing function for review, not a hard ceiling.

5. **Handoff.** For each `catalog-promotion`-class decision raised, link the source opportunity entry's id in the decision body so catalog-strategist can trace.

---

### 4. CLI Reference

List active pool:
```bash
prompt-manager team knowledge-list monetization --topic-prefix=monetization/opportunity/ --json
```

Update an entry's front-matter (e.g., to flip `status`):
```bash
prompt-manager team knowledge-update monetization "<id>" --content="<full new content with updated front-matter>"
```

Retire an entry softly (preferred):
```bash
# Update content with status: retired + ## Retirement note section
prompt-manager team knowledge-update monetization "<id>" --content="<...>"
```

Delete an entry hard (duplicates / malformed only):
```bash
prompt-manager team knowledge-delete monetization "<id>"
```

Raise a catalog-promotion decision:
```bash
prompt-manager team decision-add monetization \
  --by=opportunity-scout \
  --context=catalog-promotion \
  --proposal="Promote <slug> to CATALOG candidate; trigger fired: <evidence>" \
  --notes="Source opportunity: <knowledge-entry-id>"
```

---

### 5. Output Contract

```markdown
### Opportunity Pool Hygiene Summary

**Pool size:** <active count> active / <retired count> retired

**Triggers fired:**
- `<slug>` (`<kind>`, sku=`<>`) — evidence: <one line>; decision raised: <id>

**Triggers partially fired (logged, not promoted):**
- `<slug>` — what fired, what's outstanding

**Retired:**
- `<slug>` — reason (capability gone / hypothesis disproved / superseded / >180d stale)

**Repair flags:**
- `<slug>` — missing front-matter / unmeasurable trigger / etc.

**Consolidation candidates:**
- `<slug-a>` + `<slug-b>` overlap on <theme>; suggest merge.

**Pool size after sweep:** <active count> active / <retired count> retired.
```

No known operational edge cases for standard usage.
