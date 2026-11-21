# Intent recall: can an agent find the capability?

Provider suites (`search-hub evals`) grade one corpus. The router suite grades
provider selection. Neither answers the question an operator actually asks: when
I type a plain request ("show me a screenshot of my TV"), does the search a cold
agent runs surface the scenario that can do it? A scenario can pass every suite
and still be invisible because its skill, program and command descriptions share
no vocabulary with how requests are phrased. That is how an agent told the
operator it could not screenshot a TV while `device-control` could.

`search-hub intents` measures that end result.

## Measure

```bash
search-hub intents run                       # every corpus, --type library,skill,command, top 5
search-hub intents run --scenario device-control --json
search-hub intents run --assert              # CI gate: exit non-zero unless every case passes
search-hub intents coverage                  # scenarios with leaves but no intent case
```

A case passes when a leaf owned by the case's scenario, of an accepted type
(`expect_types`, e.g. `skill`), is in the top `--limit` results. The report also
gives `any_leaf`, the rate at which any leaf of the scenario appears, so a miss
reads as either "scenario invisible" (`any-leaf-rank=0`: fix the descriptions) or
"visible but not the skill" (rank > 0: make the skill outrank the thin program).
Queries that fail are reported as unavailable, never as misses.

## Corpus format

`scenarios/search-hub/intents/<scenario>.json` (shared, cross-scenario) or
`scenarios/<scenario>/.vrooli/intents.json` (scenario-owned):

```json
{"scenario": "device-control", "expect_types": ["skill"],
 "cases": [{"id": "tv-screenshot", "query": "take a screenshot of my TV", "tags": ["tv"]}]}
```

`scenario: "*"` means each case names its `owner`. Unknown fields, duplicate ids,
missing queries and missing owners are load errors, not skips. Write queries in
the operator's words, not the scenario's vocabulary; a case is a reviewed intent,
so a failing case is a defect in the owner's descriptions, never a reason to edit
the case to match the ranking.

## The standing guard

`prompt-manager`'s `skill-set` Test Genie phase now measures the narrower question
for every scenario that ships a skill: does the scenario's *own skill* surface in
its skill search for the intents it declares? It reads `.vrooli/intents.json` and
the shared corpus files here, and emits advisory warnings (they never fail the
phase): `skill-set.intent_missing` when a scenario ships a skill and declares no
intent, and `skill-set.intent_recall` listing the missed requests. Run it with
`vrooli scenario test <scenario> --phases skill-set`. The router-level measure in
this guide stays the cross-type check.

## Why a skipped miss is silent in the underlying provider

`prompt-manager.skill` is dense retrieval with a cosine floor (default 0.5,
`aisearch.NewService`). A request that shares no vocabulary with a skill
description scores below the floor and the provider returns zero rows with no
error: that is why `search-hub query "turn on my TV" --type skill` returned
nothing for `device-control` even though it was searched. The fix is in the
skill's `description` and tags, not in the router.

## Baseline and results

See [`../internal/INTENT-RECALL-LOG.md`](../internal/INTENT-RECALL-LOG.md).
