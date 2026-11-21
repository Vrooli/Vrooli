# Forward operator-caught UX defects per UI review

UI Health owns a local-only measurement procedure for the EX1 / EX4 scorecard. It discovers correction-like user messages directly from local Claude Code, Codex, Web Console and (when present) Agent Manager imported-history files. It does not use semantic conversation search and does not send transcript text to a service.

The discovery pass is retrieval only. It labels messages `strong-candidate` or `ambiguous-candidate`; both start with `unknown`. An agent reviewer adjudicates each candidate against the transcript and private EX4 review evidence as `confirmed`, `ambiguous`, or `exclude`; this does not require a separate operator labeling task. Confirmed defects need a stable `defect_key`, defect class, and an anchor into the completed review. The published count includes only confirmed unique defect keys. Repeated or cross-surface copies can carry several transcript anchors but contribute one defect when assigned the same key. Unknown and ambiguous items stay visible and make the count provisional.

## Inputs

Create a private review manifest, outside the checkout, for each completed EX4 batch review:

```json
{
  "review_id": "EX4-2026-11-12-batch-01",
  "scenario": "web-console",
  "status": "completed",
  "review_started_at": "2026-11-12T14:00:00-05:00",
  "review_ended_at": "2026-11-12T15:00:00-05:00",
  "review_evidence": [
    {"label": "review packet", "uri": "private/ex4-review/review-packet.json"},
    {"label": "served mobile evidence", "uri": "private/ex4-review/mobile-gallery/index.html"}
  ]
}
```

The interval should include the review and its immediate written feedback. When a correction arrives later, create or amend the local manifest's end time only when that feedback is demonstrably part of the same review, then rerun discovery. The review anchors must identify the actual completed review and served evidence, not a mock review made to exercise this tool.

## Procedure

```bash
python3 scenarios/ui-health/scripts/count-forward-operator-corrections.py discover \
  --review /path/to/private/review-manifest.json \
  --output /path/to/private/candidates.jsonl
```

Discovery writes a companion `candidates.coverage.json` beside the JSONL, recording how many user messages each adapter yielded, matched source counts and source availability. Keep both files.

Review candidate rows against the source transcript anchor and EX4 evidence. Write adjudications as private JSONL, one per candidate:

```json
{"evidence_id":"...","review_id":"EX4-2026-11-12-batch-01","decision":"confirmed","defect_key":"web-console.drawer.mobile-actions","defect_class":"mobile-layout","review_anchor":"gallery/mobile/drawer.png#actions"}
{"evidence_id":"...","review_id":"EX4-2026-11-12-batch-01","decision":"ambiguous","review_anchor":"feedback/item-7"}
{"evidence_id":"...","review_id":"EX4-2026-11-12-batch-01","decision":"exclude"}
```

Then publish the evidence-linked count:

```bash
python3 scenarios/ui-health/scripts/count-forward-operator-corrections.py publish \
  --review /path/to/private/review-manifest.json \
  --candidates /path/to/private/candidates.jsonl \
  --adjudications /path/to/private/adjudications.jsonl \
  --output /path/to/private/published-count.json
```

The summary prints the number, completeness status, candidate source coverage, unknown/ambiguous coverage and local source availability. The private report includes a transcript anchor and a review anchor for each counted defect. Store manifests, candidate text, adjudications and reports under the run's private directory with restrictive permissions. Never commit that data.

## Measure definition and limits

`count = number of distinct confirmed defect_key values linked to one completed UI review`. The review and transcript evidence are required for every counted key. The denominator is one eligible completed UI review; report `count / 1` as operator-caught defects per review. Across multiple reviews, keep one report per `review_id` and calculate the mean from those reports; do not combine reviews before deduplication.

The direct source adapters identify local files/databases and only read user-role messages. Exact normalized duplicate message text is merged across adapters, while its source anchors are retained. Paraphrases are not safely auto-merged; the adjudicator must assign a shared defect key when they describe the same issue. The candidate rules are intentionally recall-oriented and do not establish that a mention was an actual defect, was caught during that review, or maps to the shown UI. Review-window matching can miss delayed feedback. Unreadable or absent sources are surfaced as coverage limits and never treated as zero. A report with unknown or ambiguous candidate messages is provisional; it must not be presented as a final count.

No baseline count is published by this method until a historical item is shown to be a completed UI review with directly linkable transcript and served-review evidence. Existing EX1 transcript candidates alone are not completed UI reviews.
