# Calibration Review Worksheet

Phase 10 requires a human operator to review 50 imported entries. This file
records the reproducible selection method and the decision fields; it does not
substitute an automated judgement.

## Selection

Run this query against the scenario database. It selects eight entries from
each current facet and two additional `gotcha` entries, ordered by immutable
entry ID (50 entries total).

```sql
WITH latest AS (
  SELECT entry_id, facet_id,
    ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY assigned_at DESC, id DESC) AS rn
  FROM facet_assignments
), ranked AS (
  SELECT e.id, l.facet_id, e.source_path, e.body,
    ROW_NUMBER() OVER (PARTITION BY l.facet_id ORDER BY e.id) AS facet_rank
  FROM entries e JOIN latest l ON l.entry_id=e.id AND l.rn=1
  WHERE e.source_harness<>''
)
SELECT id, facet_id, source_path, body FROM ranked
WHERE facet_rank<=8 OR (facet_id='gotcha' AND facet_rank BETWEEN 9 AND 10)
ORDER BY facet_id, facet_rank;
```

## Per-entry decision

For each row, record all three fields in the Phase 10 plan log:

1. Entry ID and current facet.
2. `correct`, or the one replacement facet from the closed six-facet set.
3. Whether the operator would nominate it for a pin; nomination is not an
   automatic pin and must remain false unless explicitly approved.

Then record sample size, correct count, every correction, nominated-pin count,
and the operator identity/date. Only then may the taxonomy fit, pin budget, and
review interval be calibrated in `DECISIONS.md` and configuration.

## Completed decisions

### Batch 1 — 2026-07-28

The operator accepted all ten entries except these corrections:

| Entry ID | Previous facet | Operator facet | Rationale |
|---|---|---|---|
| `14e98ce3-496e-4ed5-8b08-89b277a61bab` | entity-record | gotcha | The template's pre-existing test failure is a recurring trap. |
| `17cdea3a-efe5-4abf-911a-9b2568593788` | entity-record | environment-fact | Sandbox-aware CLI behavior is a durable tooling/environment fact. |
| `0eff4fd2-5d88-4407-985c-ea1dd3be752f` | environment-fact | entity-record | The structure-health plan is a named project artifact. |

`14948487-eb38-4197-b977-1f4634b9de28` (host environment) remains
`environment-fact`; the operator noted that another category could be
defensible, but did not request a change. No pin nomination was made in this
batch.
