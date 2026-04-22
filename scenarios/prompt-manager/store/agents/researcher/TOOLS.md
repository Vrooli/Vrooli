# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`
`prompt-manager team decision-*`
`prompt-manager team knowledge-*`
Filesystem reads on `docs/marketing/`, `shared/audience-scans.jsonl`, `shared/knowledge.jsonl`.
Filesystem writes on `shared/audience-scans.jsonl` (append-only), `docs/marketing/notebook/AUDIENCE_OBSERVATIONS.md` (append-only, for raw unstructured notes).

## Primary Skills
- **seo-optimizer** — competitor SEO and keyword scanning.
- **systematic-exploration** — structured approach to scanning.
- **funnel-builder** — (dormant pre-telemetry) conversion analytics once real data exists.
- **documentation-health** — scans stay concrete and citation-grounded.

## Primary Surfaces
- **Read-canon:** `docs/marketing/AUDIENCES.md` (personas I propose updates to), `STRATEGY.md` (positioning)
- **Read-monetization-adjacent:** `docs/monetization/BENCHMARKS.md` (market-validator's curated benchmarks — know what's already captured so I don't duplicate)
- **Read-own:** `shared/audience-scans.jsonl`, `shared/knowledge.jsonl` (prior scans + cross-team entries)
- **Write-append:** `shared/audience-scans.jsonl`, `docs/marketing/notebook/AUDIENCE_OBSERVATIONS.md`

## Audience-scan entry schema

```
{
  "id": "as-<unix-nanos>",
  "at": "YYYY-MM-DDTHH:MM:SSZ",
  "by": "researcher",
  "scope": "audience | competitor | trend | monetization-benchmark-adjacent",
  "persona_key": "<key-from-AUDIENCES.md-or-null>",
  "observation": "<concrete-text>",
  "source_refs": ["<url | repo-link | post-ref>"],
  "interpretation": "<optional-interpretive-note>",
  "interpretation_flag": "observation | light-interpretation | heavy-interpretation",
  "honesty_flags": {
    "reach_claim": "measured | estimate | pending-telemetry",
    "audience_size": "measured | estimate | pending-telemetry"
  },
  "cross_team": "<null | monetization-benchmark-adjacent/<topic>>"
}
```

## Monetization-benchmark-adjacent knowledge entry shape

```
{
  "id": "k-<unix-nanos>",
  "at": "YYYY-MM-DDTHH:MM:SSZ",
  "by": "researcher",
  "topic": "monetization-benchmark-adjacent/<topic>",
  "body": "<observation>. Source: <link>. For monetization market-validator consumption.",
  "source_scan_ids": ["as-..."],
  "supersedes": null
}
```

`supersedes` is always null for cross-team entries — they are historical record.

## Analytical Approaches
- **Triangulate before propose.** An `audience-update` requires ≥3 converging scans. Single observations stay in audience-scans only.
- **Observation before interpretation.** Record the observation first; annotate interpretation separately with the appropriate flag.
- **No engagement math without source.** If a metric doesn't have a cited data source, it's `pending-telemetry`.
- **Know what monetization has.** Read `docs/monetization/BENCHMARKS.md` so cross-team entries add signal rather than duplicate.

## Usage Rules
- Propose personas; never edit `AUDIENCES.md` directly.
- Never invent engagement numbers.
- Cross-team entries carry `monetization-benchmark-adjacent/<topic>` — that's the grep handle for market-validator.
- Cap at 1 audience-update + 1 capability-gap per heartbeat (2 new decisions max). Scan entries and knowledge entries are unlimited.
- No duplicate entries for the same observation within 7 heartbeats — check prior scans.
- Silent workarounds violate operating rule 11 — raise `capability-gap` + notebook note together.
