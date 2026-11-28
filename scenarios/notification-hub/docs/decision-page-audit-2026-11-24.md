# Decision-page audit — 24 November 2026

Measured with the Hub's own check:

```bash
notification-hub pages check <page> --cover-file <cover.json>
```

The cover is built from the open ask: question, options, recommendation, reason,
big picture and technical details. The audit covers every open `research-desk`
decision page.

## First run (morning)

| Measure | Count |
|---|---:|
| Pages checked | 18 |
| Pages the Hub would refuse | **11** |
| `explainer_restates` (blocking) | 11 |
| `page_no_picture` | 6 |
| `explainer_no_viewport` | 5 |
| `unattached_path` | 18 |
| `packet_missing` | 18 |
| `internal_id_in_question` | 7 |

Page sizes were 693–3,444 bytes (median ~1.4 KB).

## Re-run after the page backfill (same day, later)

| Measure | Count |
|---|---:|
| Pages checked | 16 |
| Pages the Hub would refuse | **12** |
| `explainer_restates` (blocking) | 12 |
| `page_no_picture` | 4 |
| `explainer_no_viewport` | 7 |
| `unattached_path` | 16 |
| `packet_missing` | 16 |

The set changed (some decisions were answered, four new ones were raised), but
the failure rate did not improve. Six pages were written today; four of them are
in this open set:

| Page written today | Size | Result |
|---|---:|---|
| `OBS1-WEBCONSOLE-TELEMETRY` | 1,065 B | **accepted** |
| `M1-BAS-SESSION-ROUTE-20261124` | 1,814 B | refused — `explainer_restates` |
| `M7-PHONE-ACCESS-20261124` | 813 B | refused — `explainer_restates` |
| `SH3-FRESH-COHORT-SCOPE` | 867 B | refused — `explainer_restates` |

The older pages are unchanged on disk (mtime 23 Nov), and no decision-page
writer run is recorded in `~/.vrooli/state/vrooli/decision-pages/` for today.

## What this proves

1. A page writer alone does not fix the problem. Three of four pages written
   today still restate the cover; only the one that carried new evidence passed.
2. The gate must run **before** the ask is sent, not after. The Hub marks
   `explainer_restates` blocking only when `AgentSent` is true; a research
   decision is an event/service ask, so it receives advice and ships anyway.
3. The producer needs the finding, not just a refusal. The readiness projection
   in `docs/mockups/night-desk/PROPOSALS.md` §3 is the route: one taxonomy, one
   persisted report, a producer scorecard, and a decision that does not consume
   operator bandwidth until it is ready.

## Why the pages fail

The pages tell the decision in prose: they repeat the question, the options and
the recommendation, and add no picture, table or comparison. The Hub measures
new words against the cover's own words; a page that only paraphrases scores
below the floor.

The fix is not “more words”. It is one visual idea that makes the choice
obvious: options side by side against real costs, a before/after, a timeline
with its stop rule, or a table of measured numbers. The kit's starters
(`skills/notification-hub/kit/examples/`) and the `hub-*` blocks exist for this.

## Still blocking every page

- `packet_missing` — no brief a worker can start from.
- `unattached_path` — the page names a local path the phone cannot open.

Both are cover/packet findings, not page findings. The producer must fix them;
a page writer cannot.
