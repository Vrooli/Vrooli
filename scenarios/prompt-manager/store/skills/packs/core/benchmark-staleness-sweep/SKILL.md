## Tools focus: Benchmark Staleness Sweep

Periodically sweep the monetization market-scan canon — knowledge entries under `monetization/market-scan/<slug>` — and auto-populate `validation-inbox/benchmark-staleness/<slug>` entries for any scan that has aged past its dimension-aware threshold. Complement to `market-validation-triage` (which triages the queue) and `pricing-comp-capture` (which does the actual re-fetch).

> **Status:** v1. The sweep does NOT fetch sources or write decisions; it only marks scans as needing re-fetch by enqueuing them. This keeps the inbox-as-unrouted-set invariant uniform across all entry types.

---

### 1. When To Use

Run this skill at the **start** of a market-validator heartbeat, before triaging the queue. Frequency:

- **Every heartbeat**: scan only the pricing-dimension entries (cheap; ~20 entries today).
- **Weekly cadence** (track via `last-handoff.md` notes): full sweep across all dimensions.
- **On-demand**: when the operator says "are our benchmarks current?" during a vision walk.

Do not run this skill during a heartbeat where the validation queue already has >10 unresolved entries — clear the queue first.

---

### 2. Required Reading

- all market-scans: `prompt-manager team knowledge-list monetization --topic-prefix=monetization/market-scan/ --json`
- existing queue: `prompt-manager team knowledge-list monetization --topic-prefix=validation-inbox/ --json` (avoid duplicate enqueues)
- `scenarios/prompt-manager/store/teams/monetization/team.json` — `taskParameters.staleBenchmarkAfterMonths` (current default: 12 months)
- `last-handoff.md` for the prior sweep date

---

### 3. Staleness Thresholds

Dimension-aware. The team's declared default (`staleBenchmarkAfterMonths: 12`) is the upper bound; pricing moves faster.

| Dimension | Threshold | Rationale |
|---|---|---|
| `pricing` | **90 days** | SaaS pricing pages change roughly monthly; tier names and quotas drift faster than 12 months. |
| `retention` / `churn` | **180 days** | Aggregate cohort metrics move slowly; quarterly check is sufficient. |
| `attach-rate` / `activation` | **180 days** | Same reasoning. |
| `channel-cac` | **120 days** | Channel economics move with platform changes; 4-month check is reasonable. |
| `other` | **365 days** (team default) | Conservative ceiling for facts that age slowly. |

Override per-entry by adding `refresh_after_days: <n>` to the front-matter; the sweep respects entry-level overrides over dimension defaults.

---

### 4. Sweep Process

1. **Read every market-scan.** Parse front-matter; extract `dimension`, `date_observed`, optional `refresh_after_days`, and `supersedes` (skip entries that already supersede a current scan — they ARE current).

2. **Compute age.** `age_days = today - date_observed`. Apply threshold:
   - `age_days >= threshold` → stale.
   - `age_days >= 0.8 * threshold` → flag-only (not enqueued, but surfaced in the sweep summary so a human can opt-in to early refresh).
   - otherwise → fresh.

3. **De-duplicate.** Before enqueuing, check `validation-inbox/benchmark-staleness/<slug>` for the same scan slug. If a queue entry already exists, skip — the prior sweep enqueued it and triage hasn't gotten to it yet.

4. **Enqueue stale scans.**
   ```bash
   prompt-manager team knowledge-add monetization \
     --by=benchmark-staleness-sweep \
     --topic="validation-inbox/benchmark-staleness/<scan-slug>" \
     --content="$(cat <<EOF
   request_type: benchmark-staleness
   source: benchmark-staleness-sweep
   target: monetization/market-scan/<scan-slug>
   target_id: <knw-id-of-stale-scan>
   urgency: staleness-sweep
   age_days: <computed>
   threshold_days: <applied>
   dimension: <pricing|retention|...>
   prior_source_url: <url-from-prior-scan>

   Re-fetch via pricing-comp-capture (or peer method for non-pricing dimensions). Supersede the prior scan if the value changed beyond the material-change threshold.
   EOF
   )"
   ```

5. **Do NOT raise decisions in this skill.** Decisions get raised by the validation-router after re-fetch reveals an actual change. Staleness alone is not a decision — it's a queue entry.

6. **Do NOT delete the stale scan.** It remains the current evidence until something supersedes it. Deletion happens only when the re-fetch confirms a structural removal (comp killed the SKU entirely) — and that's the router's call, not the sweep's.

---

### 5. CLI Reference

List scans with their dates (jq for parsing):

```bash
prompt-manager team knowledge-list monetization --topic-prefix=monetization/market-scan/ --json \
  | jq -r '.[] | "\(.id)\t\(.topic)\t\(.content | capture("date_observed: (?<d>[0-9-]+)") | .d // "unknown")"'
```

Add a queue entry (see step 4).

Check existing queue to avoid dupes:

```bash
prompt-manager team knowledge-list monetization --topic-prefix=validation-inbox/benchmark-staleness/ --json
```

---

### 6. Output Contract

```markdown
### Benchmark Staleness Sweep

**Scans inspected:** <count>
**Stale (enqueued):** <count>
**Aging (>=80% threshold, not enqueued):** <count>
**Fresh:** <count>
**Skipped (already in queue):** <count>

**Newly enqueued:**
- `<scan-slug>` (dim=`<>`, age=`<n>d`, threshold=`<n>d`) -> queue entry `<knw-...>`

**Aging — surface for awareness:**
- `<scan-slug>` (dim=`<>`, age=`<n>d`, threshold=`<n>d`)

**Notes:**
- <e.g., "first sweep since 2026-05-15" or "skipped full sweep, pricing-only this heartbeat">
```

After the sweep, proceed to `market-validation-triage` to triage the queue (which now includes the newly-enqueued staleness items alongside any operator-fed or peer-fed requests).

No known operational edge cases for standard usage.
