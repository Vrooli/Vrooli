# Discovery Pipeline

`prompt-manager discover` is the engine behind every implementation plan's
curated **skill setup context**. Given 2–5 concept queries it returns the set of
skills (and optionally Actions) an agent should read before doing the work, sized
against a complexity budget. Plan Manager stores accepted setup through
`relevant_context[]`; legacy `Required Reading` markdown is import-only
compatibility. This doc explains how the pipeline ranks results
(the block-aware ranking model and its tunable levers), how the topic hierarchy
contributes curated packs, the two other knobs that bound output (similarity
**threshold** and per-tier **budget**), the telemetry that makes them tunable,
and the rubric for changing them.

Audience: developers working on the `aisearch` package and anyone diagnosing
"discover returned the wrong few skills."

---

## 1. Two modes — curated vs. operational

Discover serves two opposite needs, selected by `--type`:

| Mode | `--type` | Want | Callers |
|---|---|---|---|
| **Curated / plan-authoring** | `skill` (default) | the skill setup bundle for development work — strong direct matches **plus** the curated topic packs for that kind of work | `plan-skill-discovery`, `implementation-plan-authoring`, `ecosystem-fit` |
| **Operational** | `all`, `action` | the *single best-matching* skill or action to avoid hand-rolling — pure relevance, no packs | the AGENTS.md "Discover → Use → Capture" reflex, `skill-improvement-suggestions`, meta-optimization, etc. |

**INVARIANT I8 — mode-aware ranking.** Curated-pack behavior (I1–I7 below)
applies **only in skill mode**. In operational mode topics are **not**
force-included; skills and actions rank by pure relevance. This is why a "find
an existing tool for X" query never gets a curated dev pack injected, and why
plan authoring stays in skill mode (operational mode won't *give* it the pack).

---

## 2. The pipeline, end to end

`Service.DiscoverTyped` (`api/aisearch/service.go`) runs these steps:

1. **Topic search → packs** (skill mode only; `SearchTopics`, 3 topics/query).
   Topics scoring ≥ `topicGate` are *selected*; each selected topic's accumulated
   skills (its own + its ancestors', via `TopicStoreReader.AccumulateSkills`)
   become a **pack**. Selected packs are deduped by topic, sorted by topic score
   desc, and bounded by `topicSkillCap` (whole-pack, skip-and-continue). Each
   pack skill carries its topic's score (I1).
2. **Skill search** (`Search`, one vector search per query). Direct semantic
   matches against the skill corpus — the *individuals*.
3. **Action search** (operational modes only). Topics are skill-only.
4. **Dedup (I6).** A skill found by both a pack and direct search is included once
   via the pack and ranks by `max(individualScore, topicScore)`.
5. **Filter.** Drafts and any excluded ids/modes/tags are removed.
6. **Compose** (the block-aware ranking — §3).
7. **Limit.** Truncated to `--limit` (default 10).
8. **Budget.** Combined content size vs. the complexity budget; when over, a
   block-aware trimmed "recommended" command is produced (§6).
9. **Telemetry.** Every call is recorded (§7).

The CLI renders the result table plus a `prompt-manager skill read …` command.

---

## 3. The ranking model (block-aware composition)

The composition replaced an older **topic-first absolute-priority sort** that
ranked *all* topic-sourced skills ahead of *all* search-sourced skills regardless
of score — so eight weak (~0.51) seed-bundle skills filled the top slots and a
0.81 exact match landed at #9, often cut by the limit. The model below ranks by
relevance *with topics as curated packs*: strong direct matches surface at the
top, relevant packs are always included, and neither crowds out the other.

When a query selects at least one pack, results compose as three blocks:

```
[ ≤ N high-confidence non-pack individuals ]   (sorted by score)
[ selected pack block(s) ]                      (packs by topic score; skills whole)
[ remaining individuals ]                       (the tail, sorted by score)
```

When **no** pack is selected (or in operational mode), results rank purely by
score up to the limit — a clean fallback to embedding search.

### The invariants

| # | Invariant | Justification |
|---|---|---|
| **I1** | A selected topic's whole accumulated pack is included; each skill carries the **topic's** score, not its own embedding score. | A topic is an explicit human declaration of "for this task, include these," even when an individual skill is a weak embedding match. |
| **I2** | A topic's pack is force-included only if the topic's own score ≥ `topicGate` (a higher bar than the skill threshold). | A pack is a larger commitment than one skill, so it clears a higher bar; stops generic seed topics from dumping into everything. |
| **I3** | A non-pack skill scoring ≥ `highConfidenceBar` ranks **above** the pack block. | A 0.81 exact match must not be buried under a 0.51 pack. |
| **I4** | A selected pack is never fully crowded out under limit/budget. | Room is reserved so curated packs survive even when many individuals out-score them. |
| **I5** | At most `maxIndividualsAbovePack` individuals sit above the pack. **With no pack selected the cap does not apply** — pure score ranking to the limit. | Bounds how far a flood of strong individuals pushes packs down, without distorting the no-pack fallback. |
| **I6** | A skill in both a pack and direct search is included once (via the pack) and ranks by `max(individual, topic)`. | One row, scored by the stronger signal; pack inclusion stays authoritative. |
| **I7** | Over budget, keep the top-N individuals + selected packs (whole) and trim the embedding tail first; never amputate a pack or top-N individual — report honest "over" instead. | The curated core is the dependable part of the answer; the weak tail is the disposable part. |

Plus the **topic-skill cap** (`topicSkillCap`): when multiple topics clear the
gate, add packs whole in descending topic relevance; include a pack if it fits
the remaining cap, else skip it and try the next (smaller) one. Prefers several
precise small packs over one large loose one without unbounded topic dumping.

Each invariant has a focused unit test in `api/aisearch/discover_ranking_test.go`
(`TestRanking_I1_I2_…` through `TestRanking_I8_…`) and a `// INVARIANT:` tag at
its enforcement site in `service.go`.

---

## 4. The topic hierarchy (curated packs)

Topics form a tree; each topic **owns a skill list** (the SSOT — skills do not
self-declare topics). A leaf inherits its folder's and the root's skills via
**upward accumulation** (`FileTopicStore.AccumulateSkills` walks the topic +
ancestors, deduped). So matching a leaf yields *leaf + folder + root* skills.

- **Root** holds the always-relevant development practices
  (`documentation-health`, `test`, `intent-clarification`).
- **Folders** (only `Architecture & code structure` today) hold cross-cutting
  skills shared by their leaves and add one level of depth.
- **Leaves** hold the skills specific to one kind of work.

The locked tree lives in `store/topics/` (one directory per topic, file-backed)
and is rebuilt reproducibly by `store/topics/rebuild-taxonomy.sh` (deletes then
recreates the tree parent-before-child via the `topic` CLI). After a rebuild,
restart the scenario so the reconciler re-embeds the topics — **topic
discoverability depends on each topic's description matching how agents phrase
that kind of work** (activity + surface, e.g. "refactor and clean up
duplication"), so descriptions are written that way, not as bare nouns.

Verify accumulation for any leaf:

```bash
prompt-manager topic skills refactoring-cleanup   # leaf + Architecture folder + root
prompt-manager topic tree                          # full tree
```

### Curation policy — what belongs in a topic

- A topic is a **kind of development work**, not a tag. If you can't phrase it as
  "doing X," it isn't a topic.
- Put a skill in the **most specific** topic that always applies to that work;
  let accumulation carry shared skills up to a folder or the root. Don't repeat a
  root/folder skill in a leaf.
- The **root bundle** is only skills relevant to *every* dev task — keep it to a
  handful. Seam/boundary/invariant skills live in the Architecture folder, not
  the root, because they're architecture work, not universal.
- A skill may belong to several topics (it's then included via whichever pack
  scores highest; see I6). A skill in **no** topic (e.g. `r3f-coherence`) is
  reachable only by direct search — that's fine for rare work.

---

## 5. The similarity threshold

Both topic and skill search pass a **score threshold** to Qdrant; candidates
below it are never returned.

- Source of truth: `aiSearchThreshold` in `api/main.go`, default **0.5**,
  overridable via `AI_SEARCH_THRESHOLD`.
- The **topic gate** (`topicGate`, §6) sits *above* this threshold by
  construction — a pack clears a higher bar than a single skill, and config
  validation enforces `topicGate > threshold`.

Relevant skill scores often cluster in **0.51–0.59**, right on the 0.5 floor; a
0.48 match is dropped silently. Anything that shifts the distribution down
(re-embedding, model change, corpus growth, differently-phrased queries) pushes
more under 0.5. This is one cause of under-returning — but it is **not** the
historical "returns the wrong few skills" bug, which was the topic-first ordering
fixed by §3.

---

## 6. Budget & ranking levers (config)

### Ranking levers — `store/config/discover-ranking.json`

Four tunable levers (file-backed `DiscoverRankingConfigStore`, hot-loadable like
budgets; defaults from `DefaultDiscoverRankingConfig()`):

| Lever | Default | Bounds | Effect (monotonic) |
|---|---|---|---|
| `topicGate` | 0.55 | `(threshold, 1]` | Higher → fewer, more-relevant packs force-included. |
| `highConfidenceBar` | 0.65 | `(0, 1]` | Higher → packs dominate the top; lower → strong direct matches surface first. |
| `maxIndividualsAbovePack` | 3 | `≥ 0` | Caps how many strong individuals sit above the pack block (no-pack fallback ignores it). |
| `topicSkillCap` | 12 | `> 0` | Caps total skills all selected packs contribute. |

Defaults are the documented starting points; change values from telemetry, not
hunches (§8). **Current status: all four levers sit at these defaults pending a
representative window of `discovery-metrics`** — they were chosen to satisfy the
invariants (gate > threshold; bar > gate so a strong individual clears a higher
bar than a pack), not tuned from data. Re-tune per §8 once data exists.
Validation lives in `ValidateDiscoverRankingConfig` (and rejects a gate that
doesn't exceed the skill threshold). Tests: `discover_ranking_config_test.go`.

### Budget — unit, tiers, trim

A result's `ContentChars` is `len(content)` of **exactly that skill's own
`SKILL.md`** — references inside a skill are not counted, and there is
deliberately **no** transitive budget.

| Complexity | Live budget (`store/config/budgets.json`) | Code default (fallback only) |
|---|---|---|
| `minor` | 50,000 | 4,000 |
| `moderate` | 75,000 | 8,000 |
| `major` | 100,000 | 12,000 |
| `architectural` | 150,000 | 18,000 |

`DefaultBudgetConfig()` returns the small code defaults **only when the file is
missing** — reason from the live config. At the live values the budget rarely
binds.

**Trim is block-aware (I7).** When over budget, `DiscoverTyped` keeps the
protected core — the top-N high-confidence individuals and the selected packs,
whole, in display order — and never trims them; it fills the remaining budget
from the tail (skip-oversized, not hard-stop). If the protected core alone
exceeds budget it is reported honestly as "over" rather than amputating a pack.
With no packs (operational / curated-no-pack) the protected set is empty and the
trim degrades to a greedy skip-oversized pack by score. Regression tests:
`TestRanking_I7_BlockAwareTrim`, `TestDiscover_OverBudget_SkipsOversizedKeepsSmaller`.

---

## 7. Telemetry

Two append-only JSONL logs under the runtime-data root (never the git-tracked
store tree), both bounded (5,000 entries) and time-windowed (30-day retention):

| Log | Store | Records | Read surface |
|---|---|---|---|
| `discovery-misses.jsonl` | `store.DiscoveryMissStore` | calls that returned nothing useful (top score < 0.45) | `discovery-gaps` |
| `discovery-calls.jsonl` | `store.DiscoveryCallStore` | **every** call | `discovery-metrics` |

The per-call record (`store.DiscoveryCall`) captures: queries, type, complexity,
active threshold, budget chars, total content chars, budget status, returned
count, trimmed count, optional clip count, and per-result
`{id, score, chars, source, type}`. The `source` field (`topic` vs `search`) is
how you confirm packs are firing: a curated skill-mode call should show `topic`
sources; if it shows only `search`, the relevant topic isn't clearing the gate
(rephrase the query toward the kind of work, or check the topic's description).

```bash
prompt-manager discovery-metrics --since 7d            # human summary
prompt-manager discovery-metrics --since 7d --json     # full report
```

A `DISCOVERY_PROBE_SAMPLE=N` opt-in re-runs 1-in-N skill searches with no floor
to count threshold-clipped results (`clippedBelowThreshold`); off by default
because it doubles embed+search cost on probed calls.

---

## 8. Tuning rubric (evidence-based)

Change a knob only from collected `discovery-metrics`, never from a hunch:

| Symptom in metrics | Likely cause | Move |
|---|---|---|
| Curated calls show mostly `search` sources, few `topic`; expected packs absent | Topics aren't clearing the gate (thin descriptions or `topicGate` too high) | Improve topic descriptions to match query phrasing; if broadly under-firing, lower `topicGate` (still > threshold) and re-measure |
| A strong direct match keeps landing below a pack | `highConfidenceBar` too high | Lower `highConfidenceBar` toward the observed strong-match band |
| Packs flood the top, burying direct matches | `highConfidenceBar` too low or `maxIndividualsAbovePack` too small | Raise `highConfidenceBar` / `maxIndividualsAbovePack` |
| Selected packs overflow / one loose pack dominates | `topicSkillCap` too high, or a leaf accumulates too much | Lower `topicSkillCap`; or re-curate (move a shared skill up a level) |
| High **near-threshold/clip rate**; scores cluster just above 0.5 | Threshold amputating individuals | Lower `AI_SEARCH_THRESHOLD` (candidate ~0.40); spot-check no junk appears |
| High **over-budget rate** at the live budgets | A tier genuinely binds | Raise that tier in `budgets.json` (ascending, ≤200,000) |
| Median returned count low but scores healthy | Corpus/curation gap | Author/curate skills or topics; not a tuning problem |

Guardrails:
- Levers trade precision for recall; all are config/env-reversible. Measure both
  sides of the trade.
- The budget unit stays per-skill own file. No transitive budget.
- Record any value change (and the metrics that justified it) in a
  `swarm-manager records create --kind execute` entry so future tuners inherit
  the rationale.

---

## 9. Related

- Seam registry: [`docs/internal/SEAMS.md`](../internal/SEAMS.md) —
  topic store, ranking/budget config providers, telemetry stores.
- Config reference: [`reference/configuration.md`](configuration.md) — the
  ranking levers and budgets as user-facing control surface.
- CLI reference: [`reference/cli-commands.md`](cli-commands.md) — `discover`,
  `topic`, `discovery-gaps`, `discovery-metrics`.
- Planning consumer: the `plan-skill-discovery` internal skill.
- Taxonomy rebuild: `store/topics/rebuild-taxonomy.sh`.
